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
Package protocol provides connection multiplexing for FlyDB.

Multiplexing Overview:
======================

This module implements connection multiplexing to allow multiple logical
connections (streams) over a single TCP connection:

- Reduces connection overhead
- Enables concurrent requests on one connection
- Supports flow control per stream
- Handles stream prioritization

Frame Format:
=============

Multiplexed frames add a stream ID to the standard protocol:

  +--------+--------+--------+--------+--------+--------+--------+--------+...
  | Magic  | Version| MsgType| Flags  | StreamID (4B)   |    Length (4B)   | Payload...
  +--------+--------+--------+--------+--------+--------+--------+--------+...

Stream Lifecycle:
=================

1. Client opens stream with unique ID
2. Messages are tagged with stream ID
3. Server routes responses to correct stream
4. Either side can close stream
*/
package protocol

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

// Multiplexing constants
const (
	MultiplexHeaderSize = 12 // Magic + Version + Type + Flags + StreamID + Length
	MaxStreams          = 65536
)

// Stream states
const (
	StreamOpen uint32 = iota
	StreamHalfClosed
	StreamClosed
)

// Errors
var (
	ErrStreamClosed    = errors.New("stream is closed")
	ErrTooManyStreams  = errors.New("too many streams")
	ErrStreamNotFound  = errors.New("stream not found")
	ErrInvalidStreamID = errors.New("invalid stream ID")
)

// MultiplexFrame represents a multiplexed message frame
type MultiplexFrame struct {
	Header   Header
	StreamID uint32
	Payload  []byte
}

// Stream represents a logical stream within a multiplexed connection
type Stream struct {
	ID       uint32
	state    uint32
	recvChan chan *MultiplexFrame
	sendChan chan *MultiplexFrame
	mu       sync.Mutex
	conn     *MultiplexConn
}

// MultiplexConn manages a multiplexed connection
type MultiplexConn struct {
	conn       io.ReadWriteCloser
	mu         sync.RWMutex
	streams    map[uint32]*Stream
	nextID     uint32
	isClient   bool
	closed     atomic.Bool
	closeChan  chan struct{}
	writeMu    sync.Mutex
	headerBuf  []byte
	bufferPool *BufferPool
}

// NewMultiplexConn creates a new multiplexed connection
func NewMultiplexConn(conn io.ReadWriteCloser, isClient bool) *MultiplexConn {
	mc := &MultiplexConn{
		conn:       conn,
		streams:    make(map[uint32]*Stream),
		isClient:   isClient,
		closeChan:  make(chan struct{}),
		headerBuf:  make([]byte, MultiplexHeaderSize),
		bufferPool: DefaultBufferPool,
	}

	// Client uses odd stream IDs, server uses even
	if isClient {
		mc.nextID = 1
	} else {
		mc.nextID = 2
	}

	// Start read loop
	go mc.readLoop()

	return mc
}

// OpenStream opens a new stream
func (mc *MultiplexConn) OpenStream() (*Stream, error) {
	if mc.closed.Load() {
		return nil, ErrStreamClosed
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	if len(mc.streams) >= MaxStreams {
		return nil, ErrTooManyStreams
	}

	streamID := mc.nextID
	mc.nextID += 2 // Increment by 2 to maintain odd/even

	stream := &Stream{
		ID:       streamID,
		state:    StreamOpen,
		recvChan: make(chan *MultiplexFrame, 64),
		sendChan: make(chan *MultiplexFrame, 64),
		conn:     mc,
	}

	mc.streams[streamID] = stream
	return stream, nil
}

