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
Package disk provides asynchronous disk I/O for FlyDB.

Async I/O Overview:
===================

This module implements asynchronous disk I/O to improve throughput:

- Non-blocking read/write operations
- I/O request batching and coalescing
- Prioritized I/O scheduling
- Background flush and sync

Architecture:
=============

The async I/O system uses a worker pool model:

1. Requests are submitted to a queue
2. Worker goroutines process requests
3. Callbacks notify completion
4. Batching combines adjacent operations

Request Types:
==============

- Read: Async page read with callback
- Write: Async page write with callback
- Sync: Force data to disk
- Flush: Flush dirty pages

Benefits:
=========

- Higher throughput via parallelism
- Better CPU utilization
- Reduced latency for non-blocking callers
- Improved batching opportunities
*/
package disk

import (
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// I/O operation types
type IOOpType int

const (
	IORead IOOpType = iota
	IOWrite
	IOSync
	IOFlush
)

// IORequest represents an async I/O request
type IORequest struct {
	Type     IOOpType
	PageID   PageID
	Data     []byte
	Offset   int64
	Callback func(error)
	Priority int
	submittedAt time.Time
}

// IOResult represents the result of an I/O operation
type IOResult struct {
	Request *IORequest
	Error   error
	Latency time.Duration
}

// AsyncIOConfig holds configuration for async I/O
type AsyncIOConfig struct {
	NumWorkers     int           `json:"num_workers"`
	QueueSize      int           `json:"queue_size"`
	BatchSize      int           `json:"batch_size"`
	BatchTimeout   time.Duration `json:"batch_timeout"`
	EnableCoalesce bool          `json:"enable_coalesce"`
}

// DefaultAsyncIOConfig returns sensible defaults
func DefaultAsyncIOConfig() AsyncIOConfig {
	return AsyncIOConfig{
		NumWorkers:     4,
		QueueSize:      1024,
		BatchSize:      16,
		BatchTimeout:   1 * time.Millisecond,
		EnableCoalesce: true,
	}
}

// AsyncIO provides asynchronous I/O operations
type AsyncIO struct {
	config    AsyncIOConfig
	file      *os.File
	mu        sync.RWMutex
	
	// Request queue
	requestCh chan *IORequest
	
	// Worker management
	wg        sync.WaitGroup
	stopCh    chan struct{}
	
	// Statistics
	reads     atomic.Uint64
	writes    atomic.Uint64
	syncs     atomic.Uint64
	pending   atomic.Int64
	totalLatency atomic.Uint64
}

// NewAsyncIO creates a new async I/O manager
func NewAsyncIO(file *os.File, config AsyncIOConfig) *AsyncIO {
	aio := &AsyncIO{
		config:    config,
		file:      file,
		requestCh: make(chan *IORequest, config.QueueSize),
		stopCh:    make(chan struct{}),
	}

	// Start workers
	for i := 0; i < config.NumWorkers; i++ {
		aio.wg.Add(1)
		go aio.worker(i)
	}

	return aio
}

// Close shuts down the async I/O manager
func (aio *AsyncIO) Close() error {
	close(aio.stopCh)
	aio.wg.Wait()
	return nil
}

