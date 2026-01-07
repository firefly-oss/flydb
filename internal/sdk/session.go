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

// IsolationLevel defines transaction isolation levels.
type IsolationLevel int

const (
	// IsolationReadUncommitted allows dirty reads.
	IsolationReadUncommitted IsolationLevel = iota
	// IsolationReadCommitted prevents dirty reads (default).
	IsolationReadCommitted
	// IsolationRepeatableRead prevents non-repeatable reads.
	IsolationRepeatableRead
	// IsolationSerializable provides full isolation.
	IsolationSerializable
)

// String returns the string representation of the isolation level.
func (il IsolationLevel) String() string {
	switch il {
	case IsolationReadUncommitted:
		return "READ UNCOMMITTED"
	case IsolationReadCommitted:
		return "READ COMMITTED"
	case IsolationRepeatableRead:
		return "REPEATABLE READ"
	case IsolationSerializable:
		return "SERIALIZABLE"
	default:
		return "UNKNOWN"
	}
}

// ODBCValue returns the ODBC constant for this isolation level.
func (il IsolationLevel) ODBCValue() int {
	switch il {
	case IsolationReadUncommitted:
		return 1 // SQL_TXN_READ_UNCOMMITTED
	case IsolationReadCommitted:
		return 2 // SQL_TXN_READ_COMMITTED
	case IsolationRepeatableRead:
		return 4 // SQL_TXN_REPEATABLE_READ
	case IsolationSerializable:
		return 8 // SQL_TXN_SERIALIZABLE
	default:
		return 2
	}
}

// JDBCValue returns the JDBC constant for this isolation level.
func (il IsolationLevel) JDBCValue() int {
	switch il {
	case IsolationReadUncommitted:
		return 1 // Connection.TRANSACTION_READ_UNCOMMITTED
	case IsolationReadCommitted:
		return 2 // Connection.TRANSACTION_READ_COMMITTED
	case IsolationRepeatableRead:
		return 4 // Connection.TRANSACTION_REPEATABLE_READ
	case IsolationSerializable:
		return 8 // Connection.TRANSACTION_SERIALIZABLE
	default:
		return 2
	}
}

// SessionState represents the state of a session.
type SessionState int

const (
	// SessionStateActive means the session is active and usable.
	SessionStateActive SessionState = iota
	// SessionStateIdle means the session is idle.
	SessionStateIdle
	// SessionStateClosed means the session is closed.
	SessionStateClosed
)

// Session represents a database session with its own state.
type Session struct {
	mu sync.RWMutex

	// Identification
	ID       string
	Username string
	Database string

	// Timing
	CreatedAt      time.Time
	LastActivityAt time.Time
	Timeout        time.Duration

	// State
	State SessionState

	// Settings
	AutoCommit     bool
	IsolationLevel IsolationLevel
	ReadOnly       bool

	// Active resources
	ActiveTransaction *Transaction
	Cursors           map[string]*Cursor
	PreparedStmts     map[string]*PreparedStatement

	// Server info (populated after connection)
	ServerVersion   string
	ProtocolVersion int
}

// NewSession creates a new session.
func NewSession(username, database string) *Session {
	return &Session{
		ID:             generateID("sess"),
		Username:       username,
		Database:       database,
		CreatedAt:      time.Now(),
		LastActivityAt: time.Now(),
		Timeout:        30 * time.Minute,
		State:          SessionStateActive,
		AutoCommit:     true,
		IsolationLevel: IsolationReadCommitted,
		ReadOnly:       false,
		Cursors:        make(map[string]*Cursor),
		PreparedStmts:  make(map[string]*PreparedStatement),
	}
}

// Touch updates the last activity time.
func (s *Session) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActivityAt = time.Now()
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.LastActivityAt) > s.Timeout
}

// IsActive returns true if the session is active.
func (s *Session) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State == SessionStateActive
}

// Close closes the session.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = SessionStateClosed
	// Close all cursors
	for _, cursor := range s.Cursors {
		cursor.SetState(CursorStateClosed)
	}
	s.Cursors = make(map[string]*Cursor)
	s.PreparedStmts = make(map[string]*PreparedStatement)
	return nil
}

// AddCursor adds a cursor to the session.
func (s *Session) AddCursor(cursor *Cursor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Cursors[cursor.ID] = cursor
	s.LastActivityAt = time.Now()
}

// GetCursor returns a cursor by ID.
func (s *Session) GetCursor(id string) (*Cursor, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cursor, ok := s.Cursors[id]
	return cursor, ok
}

// RemoveCursor removes a cursor from the session.
func (s *Session) RemoveCursor(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Cursors, id)
}

// AddPreparedStatement adds a prepared statement to the session.
func (s *Session) AddPreparedStatement(stmt *PreparedStatement) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PreparedStmts[stmt.ID] = stmt
	s.LastActivityAt = time.Now()
}

// GetPreparedStatement returns a prepared statement by ID.
func (s *Session) GetPreparedStatement(id string) (*PreparedStatement, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stmt, ok := s.PreparedStmts[id]
	return stmt, ok
}

// RemovePreparedStatement removes a prepared statement from the session.
func (s *Session) RemovePreparedStatement(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.PreparedStmts, id)
}

// SetAutoCommit sets the auto-commit mode.
func (s *Session) SetAutoCommit(autoCommit bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AutoCommit = autoCommit
}

// GetAutoCommit returns the auto-commit mode.
func (s *Session) GetAutoCommit() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.AutoCommit
}

// SetIsolationLevel sets the transaction isolation level.
func (s *Session) SetIsolationLevel(level IsolationLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsolationLevel = level
}

// GetIsolationLevel returns the transaction isolation level.
func (s *Session) GetIsolationLevel() IsolationLevel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.IsolationLevel
}

// SetReadOnly sets the read-only mode.
func (s *Session) SetReadOnly(readOnly bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ReadOnly = readOnly
}

// IsReadOnly returns true if the session is read-only.
func (s *Session) IsReadOnly() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ReadOnly
}

// HasActiveTransaction returns true if there's an active transaction.
func (s *Session) HasActiveTransaction() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ActiveTransaction != nil && s.ActiveTransaction.IsActive()
}

// PreparedStatement represents a prepared SQL statement.
type PreparedStatement struct {
	mu sync.RWMutex

	ID         string
	SessionID  string
	SQL        string
	Parameters []ParameterInfo
	CreatedAt  time.Time
}

// ParameterInfo describes a parameter in a prepared statement.
type ParameterInfo struct {
	Index     int
	Name      string   // For named parameters
	Type      DataType
	Precision int
	Scale     int
	Nullable  bool
	Mode      ParameterMode
}

// ParameterMode defines the parameter direction.
type ParameterMode int

const (
	// ParameterModeIn is an input parameter.
	ParameterModeIn ParameterMode = iota
	// ParameterModeOut is an output parameter.
	ParameterModeOut
	// ParameterModeInOut is an input/output parameter.
	ParameterModeInOut
)

// NewPreparedStatement creates a new prepared statement.
func NewPreparedStatement(sessionID, sql string) *PreparedStatement {
	return &PreparedStatement{
		ID:        generateID("stmt"),
		SessionID: sessionID,
		SQL:       sql,
		CreatedAt: time.Now(),
	}
}

// SessionInfo provides session metadata for drivers.
type SessionInfo struct {
	SessionID       string
	Username        string
	Database        string
	ServerVersion   string
	ProtocolVersion int
	Capabilities    []string
	MaxStatementLen int
	MaxConnections  int
	ReadOnly        bool
	AutoCommit      bool
	IsolationLevel  IsolationLevel
}

