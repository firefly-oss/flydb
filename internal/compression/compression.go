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
Package compression provides configurable compression for FlyDB.

Compression Overview:
=====================

This module implements configurable compression for:
- WAL entries to reduce disk I/O
- Replication traffic to reduce network bandwidth
- Batch operations for better compression ratios

Supported Algorithms:
=====================

1. LZ4: Fast compression/decompression, moderate ratio
2. Snappy: Very fast, lower ratio, good for real-time
3. Zstd: Best ratio, configurable speed/ratio tradeoff

Batch Compression:
==================

Batching multiple entries before compression improves ratios:
1. Collect entries into a batch
2. Compress the entire batch
3. Store/transmit compressed batch
4. Decompress and split on read
*/
package compression

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
)

// Algorithm represents a compression algorithm
type Algorithm int

const (
	AlgorithmNone Algorithm = iota
	AlgorithmGzip
	AlgorithmLZ4
	AlgorithmSnappy
	AlgorithmZstd
)

func (a Algorithm) String() string {
	switch a {
	case AlgorithmNone:
		return "none"
	case AlgorithmGzip:
		return "gzip"
	case AlgorithmLZ4:
		return "lz4"
	case AlgorithmSnappy:
		return "snappy"
	case AlgorithmZstd:
		return "zstd"
	default:
		return "unknown"
	}
}

// ParseAlgorithm parses a compression algorithm from string
func ParseAlgorithm(s string) (Algorithm, error) {
	switch s {
	case "none", "":
		return AlgorithmNone, nil
	case "gzip":
		return AlgorithmGzip, nil
	case "lz4":
		return AlgorithmLZ4, nil
	case "snappy":
		return AlgorithmSnappy, nil
	case "zstd":
		return AlgorithmZstd, nil
	default:
		return AlgorithmNone, fmt.Errorf("unknown compression algorithm: %s", s)
	}
}

// Level represents compression level
type Level int

const (
	LevelFastest Level = 1
	LevelDefault Level = 5
	LevelBest    Level = 9
)

// Config holds compression configuration
type Config struct {
	Algorithm        Algorithm `json:"algorithm"`
	Level            Level     `json:"level"`
	MinSize          int       `json:"min_size"`           // Minimum size to compress
	BatchSize        int       `json:"batch_size"`         // Number of entries per batch
	BatchTimeout     int       `json:"batch_timeout_ms"`   // Max wait time for batch (ms)
	DictionaryEnable bool      `json:"dictionary_enable"`  // Use dictionary compression
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		Algorithm:        AlgorithmGzip,
		Level:            LevelDefault,
		MinSize:          256,
		BatchSize:        100,
		BatchTimeout:     10,
		DictionaryEnable: false,
	}
}

// Errors
var (
	ErrDataTooSmall     = errors.New("data too small to compress")
	ErrInvalidHeader    = errors.New("invalid compression header")
	ErrUnsupportedAlgo  = errors.New("unsupported compression algorithm")
	ErrDecompressFailed = errors.New("decompression failed")
)

// Compressor provides compression/decompression operations
type Compressor struct {
	config     Config
	gzipPool   sync.Pool
	bufferPool sync.Pool
}

// NewCompressor creates a new compressor
func NewCompressor(config Config) *Compressor {
	return &Compressor{
		config: config,
		gzipPool: sync.Pool{
			New: func() interface{} {
				return gzip.NewWriter(nil)
			},
		},
		bufferPool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

