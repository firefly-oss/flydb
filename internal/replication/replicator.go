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
Package replication provides enhanced log replication for FlyDB.

Enhanced Log Replication Overview:
==================================

This package implements a robust log replication system with:
- Streaming replication from leader to followers
- Conflict resolution using vector clocks
- Consistency guarantees with configurable levels
- Automatic catch-up for lagging followers

Replication Flow:
=================

1. Leader receives write request
2. Leader appends to local WAL with sequence number
3. Leader streams entry to all followers
4. Followers acknowledge receipt
5. Leader commits when quorum acknowledges
6. Leader responds to client

Conflict Resolution:
====================

When conflicts occur (e.g., during network partitions), the system uses:
- Vector clocks for causality tracking
- Last-writer-wins with configurable tie-breakers
- Conflict detection and notification hooks

Consistency Levels:
===================

- Strong: Wait for all replicas before acknowledging
- Quorum: Wait for majority of replicas
- Eventual: Acknowledge immediately, replicate async
*/
package replication

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// ConsistencyLevel defines the replication consistency guarantee
type ConsistencyLevel int

const (
	// ConsistencyEventual acknowledges writes immediately
	ConsistencyEventual ConsistencyLevel = iota
	// ConsistencyQuorum waits for majority acknowledgment
	ConsistencyQuorum
	// ConsistencyStrong waits for all replicas
	ConsistencyStrong
)

func (c ConsistencyLevel) String() string {
	switch c {
	case ConsistencyEventual:
		return "EVENTUAL"
	case ConsistencyQuorum:
		return "QUORUM"
	case ConsistencyStrong:
		return "STRONG"
	default:
		return "UNKNOWN"
	}
}

// ReplicatorConfig holds configuration for the replicator
type ReplicatorConfig struct {
	NodeID           string           `json:"node_id"`
	ListenAddr       string           `json:"listen_addr"`
	Consistency      ConsistencyLevel `json:"consistency"`
	ReplicationPort  int              `json:"replication_port"`
	MaxBatchSize     int              `json:"max_batch_size"`
	FlushInterval    time.Duration    `json:"flush_interval"`
	AckTimeout       time.Duration    `json:"ack_timeout"`
	RetryInterval    time.Duration    `json:"retry_interval"`
	MaxRetries       int              `json:"max_retries"`
}

// DefaultReplicatorConfig returns sensible defaults
func DefaultReplicatorConfig(nodeID string) ReplicatorConfig {
	return ReplicatorConfig{
		NodeID:          nodeID,
		ListenAddr:      "0.0.0.0",
		Consistency:     ConsistencyQuorum,
		ReplicationPort: 9997,
		MaxBatchSize:    1000,
		FlushInterval:   10 * time.Millisecond,
		AckTimeout:      5 * time.Second,
		RetryInterval:   100 * time.Millisecond,
		MaxRetries:      10,
	}
}

// VectorClock tracks causality across nodes
type VectorClock struct {
	Clocks map[string]uint64 `json:"clocks"`
	mu     sync.RWMutex
}

// NewVectorClock creates a new vector clock
func NewVectorClock() *VectorClock {
	return &VectorClock{
		Clocks: make(map[string]uint64),
	}
}

// Increment increments the clock for a node
func (vc *VectorClock) Increment(nodeID string) uint64 {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.Clocks[nodeID]++
	return vc.Clocks[nodeID]
}

// Get returns the clock value for a node
func (vc *VectorClock) Get(nodeID string) uint64 {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.Clocks[nodeID]
}

// Merge merges another vector clock into this one
func (vc *VectorClock) Merge(other *VectorClock) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	other.mu.RLock()
	defer other.mu.RUnlock()

	for nodeID, clock := range other.Clocks {
		if clock > vc.Clocks[nodeID] {
			vc.Clocks[nodeID] = clock
		}
	}
}

// Copy returns a copy of the vector clock
func (vc *VectorClock) Copy() *VectorClock {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	copy := NewVectorClock()
	for k, v := range vc.Clocks {
		copy.Clocks[k] = v
	}
	return copy
}

