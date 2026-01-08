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
Unified Storage Engine
======================

This file defines the StorageEngine interface for FlyDB's unified disk-based
storage engine. The engine combines page-based disk storage with intelligent
buffer pool caching to provide optimal performance for both small and large
datasets.

Architecture Overview:
======================

	┌─────────────────────────────────────────────────────────────────┐
	│                      SQL Executor                               │
	└─────────────────────────────────────────────────────────────────┘
	                              │
	                              ▼
	┌─────────────────────────────────────────────────────────────────┐
	│                   StorageEngine Interface                       │
	│         (Put, Get, Delete, Scan, Close, Sync, Stats)            │
	└─────────────────────────────────────────────────────────────────┘
	                              │
	                              ▼
	┌─────────────────────────────────────────────────────────────────┐
	│                  UnifiedStorageEngine                           │
	│                                                                 │
	│   ┌─────────────────────────────────────────────────────────┐   │
	│   │              Buffer Pool (LRU-K Caching)                │   │
	│   │   - Auto-sized based on available memory                │   │
	│   │   - Intelligent page prefetching                        │   │
	│   │   - Optimized for sequential and random access          │   │
	│   └─────────────────────────────────────────────────────────┘   │
	│                              │                                  │
	│   ┌─────────────────────────────────────────────────────────┐   │
	│   │              Page-Based Disk Storage                    │   │
	│   │   - Heap file organization                              │   │
	│   │   - Slotted page format                                 │   │
	│   │   - Efficient space management                          │   │
	│   └─────────────────────────────────────────────────────────┘   │
	│                              │                                  │
	│   ┌─────────────────────────────────────────────────────────┐   │
	│   │              Write-Ahead Log (WAL)                      │   │
	│   │   - Durability guarantees                               │   │
	│   │   - Crash recovery                                      │   │
	│   │   - Optional encryption                                 │   │
	│   └─────────────────────────────────────────────────────────┘   │
	└─────────────────────────────────────────────────────────────────┘

Performance Characteristics:
============================

	| Operation | Cached (in buffer pool) | Uncached (disk I/O)    |
	|-----------|-------------------------|------------------------|
	| Put       | O(1) + WAL              | O(1) + WAL             |
	| Get       | O(1)                    | O(disk)                |
	| Delete    | O(1) + WAL              | O(1) + WAL             |
	| Scan      | O(N) with prefetch      | O(N) + disk I/O        |

Buffer Pool Sizing:
===================

The buffer pool is automatically sized based on available system memory:
  - Uses 25% of available RAM
  - Minimum: 2MB (256 pages)
  - Maximum: 1GB (131,072 pages)

This ensures optimal performance without manual tuning.
*/
package storage

import (
	"errors"
	"fmt"
)

// StorageEngineType represents the type of storage engine.
// FlyDB uses a unified disk-based storage engine for all deployments.
type StorageEngineType string

const (
	// EngineTypeMemory is kept for backward compatibility but is not used.
	// All storage now uses the unified disk-based engine with buffer pool.
	EngineTypeMemory StorageEngineType = "memory"

	// EngineTypeDisk is the unified disk-based storage engine with buffer pool.
	// This is the only engine type used in production.
	EngineTypeDisk StorageEngineType = "disk"
)

// ErrEngineNotSupported is returned when an unsupported engine type is requested.
var ErrEngineNotSupported = errors.New("storage engine type not supported")

// StorageEngine extends the basic Engine interface with additional capabilities
// for the unified disk-based storage engine.
//
// This interface provides a unified abstraction that allows the SQL executor
// and other components to work with the storage backend without knowing
// the implementation details.
type StorageEngine interface {
	Engine // Embed the basic Engine interface (Put, Get, Delete, Scan, Close)

	// Sync forces all pending writes to be persisted to durable storage.
	// This flushes dirty pages from the buffer pool and syncs the WAL.
	//
	// This is called during checkpoint operations and before shutdown.
	Sync() error

	// Stats returns statistics about the storage engine.
	// This includes metrics like cache hit rate, page reads/writes, etc.
	Stats() EngineStats

	// Type returns the type of this storage engine.
	Type() StorageEngineType

	// WAL returns the underlying Write-Ahead Log.
	// This is needed for replication and recovery.
	WAL() *WAL

	// IsEncrypted returns true if the storage engine uses encryption.
	IsEncrypted() bool
}

// EngineStats contains statistics about the storage engine.
// These metrics are useful for monitoring and performance tuning.
type EngineStats struct {
	// Common stats
	KeyCount    int64 // Total number of keys in the store
	DataSize    int64 // Approximate size of all data in bytes
	WALSize     int64 // Size of the WAL file in bytes
	EngineType  StorageEngineType
	IsEncrypted bool

	// Buffer pool and disk I/O stats
	BufferPoolSize   int64   // Total buffer pool size in bytes
	BufferPoolUsed   int64   // Used buffer pool size in bytes
	CacheHitRate     float64 // Percentage of reads served from buffer pool (0-100)
	PageReads        int64   // Total page reads from disk
	PageWrites       int64   // Total page writes to disk
	DirtyPages       int64   // Number of dirty pages in buffer pool
	CheckpointCount  int64   // Number of checkpoints performed
	LastCheckpointAt int64   // Unix timestamp of last checkpoint
}

// String returns a human-readable representation of the engine stats.
func (s EngineStats) String() string {
	return fmt.Sprintf(
		"Engine: %s, Keys: %d, DataSize: %d bytes, WALSize: %d bytes, Encrypted: %v",
		s.EngineType, s.KeyCount, s.DataSize, s.WALSize, s.IsEncrypted,
	)
}
