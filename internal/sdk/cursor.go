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

package sdk

import (
	"sync"
	"time"
)

// CursorType defines the type of cursor behavior.
type CursorType int

const (
	// CursorForwardOnly is the default, most efficient cursor type.
	// Can only move forward through the result set.
	CursorForwardOnly CursorType = iota
	// CursorStatic creates a snapshot of the result set at open time.
	// Scrollable but does not see changes made after opening.
	CursorStatic
	// CursorKeyset has fixed keys but dynamic values.
	// Scrollable and sees value changes but not new/deleted rows.
	CursorKeyset
	// CursorDynamic is fully dynamic.
	// Scrollable and sees all changes including new/deleted rows.
	CursorDynamic
)

// String returns the string representation of the cursor type.
func (ct CursorType) String() string {
	switch ct {
	case CursorForwardOnly:
		return "FORWARD_ONLY"
	case CursorStatic:
		return "STATIC"
	case CursorKeyset:
		return "KEYSET"
	case CursorDynamic:
		return "DYNAMIC"
	default:
		return "UNKNOWN"
	}
}

// CursorConcurrency defines the concurrency model for cursors.
type CursorConcurrency int

const (
	// ConcurrencyReadOnly means the cursor cannot update data.
	ConcurrencyReadOnly CursorConcurrency = iota
	// ConcurrencyLock uses pessimistic locking for updates.
	ConcurrencyLock
	// ConcurrencyOptimistic uses optimistic concurrency control.
	ConcurrencyOptimistic
)

// String returns the string representation of the concurrency mode.
func (cc CursorConcurrency) String() string {
	switch cc {
	case ConcurrencyReadOnly:
		return "READ_ONLY"
	case ConcurrencyLock:
		return "LOCK"
	case ConcurrencyOptimistic:
		return "OPTIMISTIC"
	default:
		return "UNKNOWN"
	}
}

// CursorState represents the current state of a cursor.
type CursorState int

const (
	// CursorStateAllocated means the cursor is allocated but not open.
	CursorStateAllocated CursorState = iota
	// CursorStateOpen means the cursor is open and ready for fetching.
	CursorStateOpen
	// CursorStateFetching means the cursor is currently fetching data.
	CursorStateFetching
	// CursorStateExhausted means no more rows are available.
	CursorStateExhausted
	// CursorStateClosed means the cursor is closed.
	CursorStateClosed
)

// String returns the string representation of the cursor state.
func (cs CursorState) String() string {
	switch cs {
	case CursorStateAllocated:
		return "ALLOCATED"
	case CursorStateOpen:
		return "OPEN"
	case CursorStateFetching:
		return "FETCHING"
	case CursorStateExhausted:
		return "EXHAUSTED"
	case CursorStateClosed:
		return "CLOSED"
	default:
		return "UNKNOWN"
	}
}

// FetchDirection specifies the direction for scrollable cursor fetch.
type FetchDirection int

const (
	// FetchNext fetches the next row.
	FetchNext FetchDirection = iota
	// FetchPrior fetches the previous row.
	FetchPrior
	// FetchFirst fetches the first row.
	FetchFirst
	// FetchLast fetches the last row.
	FetchLast
	// FetchAbsolute fetches the row at an absolute position.
	FetchAbsolute
	// FetchRelative fetches the row at a relative offset.
	FetchRelative
)

// Cursor represents a server-side database cursor for result set navigation.
type Cursor struct {
	mu sync.RWMutex

	// Identification
	ID        string
	SessionID string

	// Configuration
	Type        CursorType
	Concurrency CursorConcurrency
	FetchSize   int

	// State
	State    CursorState
	Position int64 // Current row position (0-based, -1 = before first)
	RowCount int64 // Total rows (-1 if unknown)

	// Timing
	CreatedAt    time.Time
	LastAccessAt time.Time

	// Associated query
	Query      string
	Parameters []interface{}
}

// NewCursor creates a new cursor with a generated ID.
func NewCursor(sessionID string) *Cursor {
	return &Cursor{
		ID:           generateID("cur"),
		SessionID:    sessionID,
		Type:         CursorForwardOnly,
		Concurrency:  ConcurrencyReadOnly,
		FetchSize:    100,
		State:        CursorStateAllocated,
		Position:     -1, // Before first row
		RowCount:     -1, // Unknown
		CreatedAt:    time.Now(),
		LastAccessAt: time.Now(),
	}
}

// NewCursorWithID creates a new cursor with a specific ID.
func NewCursorWithID(id, sessionID string) *Cursor {
	return &Cursor{
		ID:           id,
		SessionID:    sessionID,
		Type:         CursorForwardOnly,
		Concurrency:  ConcurrencyReadOnly,
		FetchSize:    100,
		State:        CursorStateAllocated,
		Position:     -1, // Before first row
		RowCount:     -1, // Unknown
		CreatedAt:    time.Now(),
		LastAccessAt: time.Now(),
	}
}

// IsScrollable returns true if the cursor supports scrolling.
func (c *Cursor) IsScrollable() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Type != CursorForwardOnly
}

// IsUpdatable returns true if the cursor supports updates.
func (c *Cursor) IsUpdatable() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Concurrency != ConcurrencyReadOnly
}

// IsOpen returns true if the cursor is open.
func (c *Cursor) IsOpen() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State == CursorStateOpen || c.State == CursorStateFetching
}

// IsClosed returns true if the cursor is closed.
func (c *Cursor) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State == CursorStateClosed
}

// IsExhausted returns true if no more rows are available.
func (c *Cursor) IsExhausted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State == CursorStateExhausted
}

// SetState sets the cursor state.
func (c *Cursor) SetState(state CursorState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.State = state
	c.LastAccessAt = time.Now()
}

// GetPosition returns the current cursor position.
func (c *Cursor) GetPosition() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Position
}

// SetPosition sets the cursor position.
func (c *Cursor) SetPosition(pos int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Position = pos
	c.LastAccessAt = time.Now()
}

// IncrementPosition increments the cursor position by 1.
func (c *Cursor) IncrementPosition() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Position++
	c.LastAccessAt = time.Now()
	return c.Position
}

// SetRowCount sets the total row count.
func (c *Cursor) SetRowCount(count int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.RowCount = count
}

// GetRowCount returns the total row count (-1 if unknown).
func (c *Cursor) GetRowCount() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.RowCount
}

// CursorOptions configures cursor behavior.
type CursorOptions struct {
	Type        CursorType
	Concurrency CursorConcurrency
	FetchSize   int
	Scrollable  bool
	Holdable    bool // Whether cursor survives transaction commit
}

// DefaultCursorOptions returns default cursor options.
func DefaultCursorOptions() CursorOptions {
	return CursorOptions{
		Type:        CursorForwardOnly,
		Concurrency: ConcurrencyReadOnly,
		FetchSize:   100,
		Scrollable:  false,
		Holdable:    false,
	}
}

