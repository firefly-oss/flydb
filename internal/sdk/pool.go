/*
 * Copyright (c) 2026 Firefly Software Solutions Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
Connection Pool Implementation
==============================

This file provides a connection pool for FlyDB SDK and drivers. Connection
pooling is essential for production applications to:

  - Reduce connection establishment overhead
  - Limit the number of concurrent connections
  - Reuse connections efficiently
  - Handle connection failures gracefully

Pool Configuration:
===================

  MinConnections: Minimum idle connections to maintain
  MaxConnections: Maximum total connections allowed
  MaxIdleTime:    Maximum time a connection can be idle
  MaxLifetime:    Maximum lifetime of a connection
  AcquireTimeout: Maximum time to wait for a connection

Usage:
======

  pool := sdk.NewConnectionPool(config)
  conn, err := pool.Acquire(ctx)
  defer pool.Release(conn)
  // use conn...
*/
package sdk

import (
	"context"
	"sync"
	"time"
)

// PoolConfig configures the connection pool.
type PoolConfig struct {
	// Connection settings
	ConnectionConfig *ConnectionConfig

	// Pool size
	MinConnections int // Minimum idle connections (default: 1)
	MaxConnections int // Maximum total connections (default: 10)

	// Timeouts
	MaxIdleTime    time.Duration // Max idle time before closing (default: 5m)
	MaxLifetime    time.Duration // Max connection lifetime (default: 1h)
	AcquireTimeout time.Duration // Max time to acquire connection (default: 30s)

	// Health check
	HealthCheckInterval time.Duration // Interval for health checks (default: 30s)
}

// DefaultPoolConfig returns a pool configuration with sensible defaults.
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		ConnectionConfig:    NewConnectionConfig(),
		MinConnections:      1,
		MaxConnections:      10,
		MaxIdleTime:         5 * time.Minute,
		MaxLifetime:         1 * time.Hour,
		AcquireTimeout:      30 * time.Second,
		HealthCheckInterval: 30 * time.Second,
	}
}

// PooledConnection represents a connection managed by the pool.
type PooledConnection struct {
	ID         string
	Session    *Session
	CreatedAt  time.Time
	LastUsedAt time.Time
	InUse      bool
}

// ConnectionPool manages a pool of database connections.
type ConnectionPool struct {
	mu     sync.Mutex
	config *PoolConfig

	// Connection tracking
	connections []*PooledConnection
	available   chan *PooledConnection
	totalCount  int

	// State
	closed   bool
	closedCh chan struct{}
}

// NewConnectionPool creates a new connection pool.
func NewConnectionPool(config *PoolConfig) *ConnectionPool {
	if config == nil {
		config = DefaultPoolConfig()
	}

	pool := &ConnectionPool{
		config:      config,
		connections: make([]*PooledConnection, 0, config.MaxConnections),
		available:   make(chan *PooledConnection, config.MaxConnections),
		closedCh:    make(chan struct{}),
	}

	return pool
}

// Acquire gets a connection from the pool.
func (p *ConnectionPool) Acquire(ctx context.Context) (*PooledConnection, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, NewSDKError(ErrCodeConnectionClosed, "connection pool is closed")
	}
	p.mu.Unlock()

	// Try to get an available connection
	select {
	case conn := <-p.available:
		conn.InUse = true
		conn.LastUsedAt = time.Now()
		return conn, nil
	default:
		// No available connection, try to create one
	}

	p.mu.Lock()
	if p.totalCount < p.config.MaxConnections {
		conn := p.createConnection()
		p.mu.Unlock()
		return conn, nil
	}
	p.mu.Unlock()

	// Wait for an available connection
	timeout := p.config.AcquireTimeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	select {
	case conn := <-p.available:
		conn.InUse = true
		conn.LastUsedAt = time.Now()
		return conn, nil
	case <-time.After(timeout):
		return nil, NewSDKError(ErrCodeTimeout, "timeout waiting for connection")
	case <-ctx.Done():
		return nil, NewSDKErrorWithCause(ErrCodeTimeout, "context cancelled", ctx.Err())
	}
}

// Release returns a connection to the pool.
func (p *ConnectionPool) Release(conn *PooledConnection) {
	if conn == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	// Check if connection is still valid
	if time.Since(conn.CreatedAt) > p.config.MaxLifetime {
		p.removeConnection(conn)
		return
	}

	conn.InUse = false
	conn.LastUsedAt = time.Now()

	select {
	case p.available <- conn:
		// Connection returned to pool
	default:
		// Pool is full, close the connection
		p.removeConnection(conn)
	}
}

// Close closes the connection pool and all connections.
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	close(p.closedCh)

	// Close all connections
	for _, conn := range p.connections {
		if conn.Session != nil {
			conn.Session.Close()
		}
	}
	p.connections = nil
	p.totalCount = 0

	return nil
}

// Stats returns pool statistics.
func (p *ConnectionPool) Stats() PoolStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	inUse := 0
	for _, conn := range p.connections {
		if conn.InUse {
			inUse++
		}
	}

	return PoolStats{
		TotalConnections: p.totalCount,
		IdleConnections:  p.totalCount - inUse,
		InUseConnections: inUse,
		MaxConnections:   p.config.MaxConnections,
	}
}

// PoolStats contains connection pool statistics.
type PoolStats struct {
	TotalConnections int
	IdleConnections  int
	InUseConnections int
	MaxConnections   int
}

// createConnection creates a new pooled connection.
func (p *ConnectionPool) createConnection() *PooledConnection {
	conn := &PooledConnection{
		ID:         generateID("conn"),
		Session:    NewSession(p.config.ConnectionConfig.Username, p.config.ConnectionConfig.Database),
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
		InUse:      true,
	}
	p.connections = append(p.connections, conn)
	p.totalCount++
	return conn
}

// removeConnection removes a connection from the pool.
func (p *ConnectionPool) removeConnection(conn *PooledConnection) {
	for i, c := range p.connections {
		if c.ID == conn.ID {
			p.connections = append(p.connections[:i], p.connections[i+1:]...)
			p.totalCount--
			if conn.Session != nil {
				conn.Session.Close()
			}
			break
		}
	}
}

