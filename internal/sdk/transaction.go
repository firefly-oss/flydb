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

// TransactionState represents the current state of a transaction.
type TransactionState int

const (
	// TxStateNone means no transaction is active.
	TxStateNone TransactionState = iota
	// TxStateActive means a transaction is in progress.
	TxStateActive
	// TxStateCommitted means the transaction was committed.
	TxStateCommitted
	// TxStateRolledBack means the transaction was rolled back.
	TxStateRolledBack
	// TxStateFailed means the transaction failed and must be rolled back.
	TxStateFailed
)

// String returns the string representation of the transaction state.
func (ts TransactionState) String() string {
	switch ts {
	case TxStateNone:
		return "NONE"
	case TxStateActive:
		return "ACTIVE"
	case TxStateCommitted:
		return "COMMITTED"
	case TxStateRolledBack:
		return "ROLLED_BACK"
	case TxStateFailed:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// Transaction represents an active database transaction.
type Transaction struct {
	mu sync.RWMutex

	// Identification
	ID        string
	SessionID string

	// Configuration
	IsolationLevel IsolationLevel
	ReadOnly       bool
	Deferrable     bool

	// State
	State TransactionState

	// Timing
	StartTime time.Time
	EndTime   time.Time

	// Savepoints
	Savepoints []string
}

// NewTransaction creates a new transaction.
func NewTransaction(sessionID string, isolationLevel IsolationLevel) *Transaction {
	return &Transaction{
		ID:             generateID("tx"),
		SessionID:      sessionID,
		IsolationLevel: isolationLevel,
		State:          TxStateActive,
		StartTime:      time.Now(),
		Savepoints:     make([]string, 0),
	}
}

// IsActive returns true if the transaction is active.
func (tx *Transaction) IsActive() bool {
	tx.mu.RLock()
	defer tx.mu.RUnlock()
	return tx.State == TxStateActive
}

// IsFailed returns true if the transaction has failed.
func (tx *Transaction) IsFailed() bool {
	tx.mu.RLock()
	defer tx.mu.RUnlock()
	return tx.State == TxStateFailed
}

// IsCompleted returns true if the transaction is completed (committed or rolled back).
func (tx *Transaction) IsCompleted() bool {
	tx.mu.RLock()
	defer tx.mu.RUnlock()
	return tx.State == TxStateCommitted || tx.State == TxStateRolledBack
}

// SetState sets the transaction state.
func (tx *Transaction) SetState(state TransactionState) {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	tx.State = state
	if state == TxStateCommitted || state == TxStateRolledBack {
		tx.EndTime = time.Now()
	}
}

// MarkFailed marks the transaction as failed.
func (tx *Transaction) MarkFailed() {
	tx.SetState(TxStateFailed)
}

// AddSavepoint adds a savepoint to the transaction.
func (tx *Transaction) AddSavepoint(name string) {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	tx.Savepoints = append(tx.Savepoints, name)
}

// RemoveSavepoint removes a savepoint from the transaction.
func (tx *Transaction) RemoveSavepoint(name string) bool {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	for i, sp := range tx.Savepoints {
		if sp == name {
			tx.Savepoints = append(tx.Savepoints[:i], tx.Savepoints[i+1:]...)
			return true
		}
	}
	return false
}

// HasSavepoint returns true if the savepoint exists.
func (tx *Transaction) HasSavepoint(name string) bool {
	tx.mu.RLock()
	defer tx.mu.RUnlock()
	for _, sp := range tx.Savepoints {
		if sp == name {
			return true
		}
	}
	return false
}

// RollbackToSavepoint removes all savepoints after the given one.
func (tx *Transaction) RollbackToSavepoint(name string) bool {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	for i, sp := range tx.Savepoints {
		if sp == name {
			tx.Savepoints = tx.Savepoints[:i+1]
			return true
		}
	}
	return false
}

// Duration returns the duration of the transaction.
func (tx *Transaction) Duration() time.Duration {
	tx.mu.RLock()
	defer tx.mu.RUnlock()
	if tx.EndTime.IsZero() {
		return time.Since(tx.StartTime)
	}
	return tx.EndTime.Sub(tx.StartTime)
}

// TransactionOptions configures transaction behavior.
type TransactionOptions struct {
	IsolationLevel IsolationLevel
	ReadOnly       bool
	Deferrable     bool
}

// DefaultTransactionOptions returns default transaction options.
func DefaultTransactionOptions() TransactionOptions {
	return TransactionOptions{
		IsolationLevel: IsolationReadCommitted,
		ReadOnly:       false,
		Deferrable:     false,
	}
}

// TransactionManager defines the interface for transaction management.
type TransactionManager interface {
	// Begin starts a new transaction.
	Begin(session *Session) (*Transaction, error)
	// BeginWithOptions starts a new transaction with options.
	BeginWithOptions(session *Session, opts TransactionOptions) (*Transaction, error)
	// Commit commits the transaction.
	Commit(tx *Transaction) error
	// Rollback rolls back the transaction.
	Rollback(tx *Transaction) error
	// Savepoint creates a savepoint.
	Savepoint(tx *Transaction, name string) error
	// RollbackToSavepoint rolls back to a savepoint.
	RollbackToSavepoint(tx *Transaction, name string) error
	// ReleaseSavepoint releases a savepoint.
	ReleaseSavepoint(tx *Transaction, name string) error
}

