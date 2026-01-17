/*
 * Copyright (c) 2026 Firefly Software Solutions Inc.
 * Licensed under the Apache License, Version 2.0
 */

/*
Package audit provides comprehensive audit trail functionality for FlyDB.

The audit system tracks all database operations, authentication events, and
administrative actions for security, compliance, and debugging purposes.

Audit Event Types:
==================

  - Authentication: LOGIN, LOGOUT, AUTH_FAILED
  - DDL Operations: CREATE_TABLE, DROP_TABLE, ALTER_TABLE, CREATE_INDEX, DROP_INDEX
  - DML Operations: INSERT, UPDATE, DELETE, SELECT (configurable)
  - DCL Operations: GRANT, REVOKE, CREATE_USER, ALTER_USER, DROP_USER
  - Administrative: BACKUP, RESTORE, CHECKPOINT, VACUUM
  - Cluster Events: NODE_JOIN, NODE_LEAVE, LEADER_ELECTION, FAILOVER

Audit Log Storage:
==================

Audit logs are stored in a dedicated system table `_audit_log` with the following schema:

  - id: SERIAL PRIMARY KEY
  - timestamp: TIMESTAMP (when the event occurred)
  - event_type: TEXT (type of event)
  - username: TEXT (user who performed the action)
  - database: TEXT (database context)
  - object_type: TEXT (table, index, user, etc.)
  - object_name: TEXT (name of the object)
  - operation: TEXT (SQL statement or operation)
  - client_addr: TEXT (client IP address)
  - session_id: TEXT (session identifier)
  - status: TEXT (SUCCESS, FAILED)
  - error_message: TEXT (error details if failed)
  - duration_ms: INT (operation duration in milliseconds)
  - metadata: JSONB (additional context)

Configuration:
==============

Audit logging can be configured via:

  - audit_enabled: Enable/disable audit logging (default: true)
  - audit_log_ddl: Log DDL operations (default: true)
  - audit_log_dml: Log DML operations (default: true)
  - audit_log_select: Log SELECT queries (default: false, can be verbose)
  - audit_log_auth: Log authentication events (default: true)
  - audit_log_admin: Log administrative operations (default: true)
  - audit_retention_days: Days to retain audit logs (default: 90, 0 = forever)

Usage:
======

	// Initialize audit manager
	auditMgr := audit.NewAuditManager(store, config)

	// Log an event
	auditMgr.LogEvent(audit.Event{
	    EventType:  audit.EventTypeCreateTable,
	    Username:   "admin",
	    Database:   "mydb",
	    ObjectType: "table",
	    ObjectName: "users",
	    Operation:  "CREATE TABLE users (id INT, name TEXT)",
	    ClientAddr: "192.168.1.100",
	    SessionID:  "sess-123",
	    Status:     audit.StatusSuccess,
	})

	// Query audit logs
	logs, err := auditMgr.QueryLogs(audit.QueryOptions{
	    StartTime: time.Now().Add(-24 * time.Hour),
	    EndTime:   time.Now(),
	    Username:  "admin",
	    EventType: audit.EventTypeCreateTable,
	    Limit:     100,
	})

	// Export audit logs
	err := auditMgr.ExportLogs("audit_export.json", audit.FormatJSON, queryOpts)

Thread Safety:
==============

The audit manager is thread-safe and can be used concurrently from multiple
goroutines. All operations are protected by appropriate synchronization.

Performance:
============

Audit logging is designed to have minimal performance impact:
  - Asynchronous logging with buffered channel
  - Batch inserts for high-throughput scenarios
  - Configurable event filtering
  - Automatic log rotation and cleanup

Cluster Support:
================

In cluster mode, each node maintains its own audit log. Audit logs can be
queried from any node, and the cluster manager aggregates logs from all nodes
for comprehensive audit trail visibility.
*/
package audit

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"flydb/internal/logging"
	"flydb/internal/storage"
)

// EventType represents the type of audit event.
type EventType string

const (
	// Authentication events
	EventTypeLogin      EventType = "LOGIN"
	EventTypeLogout     EventType = "LOGOUT"
	EventTypeAuthFailed EventType = "AUTH_FAILED"

	// DDL events
	EventTypeCreateTable   EventType = "CREATE_TABLE"
	EventTypeDropTable     EventType = "DROP_TABLE"
	EventTypeAlterTable    EventType = "ALTER_TABLE"
	EventTypeCreateIndex   EventType = "CREATE_INDEX"
	EventTypeDropIndex     EventType = "DROP_INDEX"
	EventTypeCreateView    EventType = "CREATE_VIEW"
	EventTypeDropView      EventType = "DROP_VIEW"
	EventTypeTruncateTable EventType = "TRUNCATE_TABLE"

	// DML events
	EventTypeInsert EventType = "INSERT"
	EventTypeUpdate EventType = "UPDATE"
	EventTypeDelete EventType = "DELETE"
	EventTypeSelect EventType = "SELECT"

	// DCL events
	EventTypeGrant      EventType = "GRANT"
	EventTypeRevoke     EventType = "REVOKE"
	EventTypeCreateUser EventType = "CREATE_USER"
	EventTypeAlterUser  EventType = "ALTER_USER"
	EventTypeDropUser   EventType = "DROP_USER"
	EventTypeCreateRole EventType = "CREATE_ROLE"
	EventTypeDropRole   EventType = "DROP_ROLE"
	EventTypeGrantRole  EventType = "GRANT_ROLE"
	EventTypeRevokeRole EventType = "REVOKE_ROLE"

	// Transaction events
	EventTypeBegin    EventType = "BEGIN"
	EventTypeCommit   EventType = "COMMIT"
	EventTypeRollback EventType = "ROLLBACK"

	// Administrative events
	EventTypeBackup     EventType = "BACKUP"
	EventTypeRestore    EventType = "RESTORE"
	EventTypeCheckpoint EventType = "CHECKPOINT"
	EventTypeVacuum     EventType = "VACUUM"

	// Cluster events
	EventTypeNodeJoin       EventType = "NODE_JOIN"
	EventTypeNodeLeave      EventType = "NODE_LEAVE"
	EventTypeLeaderElection EventType = "LEADER_ELECTION"
	EventTypeFailover       EventType = "FAILOVER"

	// Database events
	EventTypeCreateDatabase EventType = "CREATE_DATABASE"
	EventTypeDropDatabase   EventType = "DROP_DATABASE"
	EventTypeUseDatabase    EventType = "USE_DATABASE"
)

// Status represents the outcome of an audited event.
type Status string

const (
	StatusSuccess Status = "SUCCESS"
	StatusFailed  Status = "FAILED"
)

// Event represents a single audit log entry.
type Event struct {
	ID           int64             `json:"id"`
	Timestamp    time.Time         `json:"timestamp"`
	EventType    EventType         `json:"event_type"`
	Username     string            `json:"username"`
	Database     string            `json:"database"`
	ObjectType   string            `json:"object_type"`
	ObjectName   string            `json:"object_name"`
	Operation    string            `json:"operation"`
	ClientAddr   string            `json:"client_addr"`
	SessionID    string            `json:"session_id"`
	Status       Status            `json:"status"`
	ErrorMessage string            `json:"error_message,omitempty"`
	DurationMs   int64             `json:"duration_ms"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Config holds audit configuration.
type Config struct {
	Enabled          bool `json:"enabled"`
	LogDDL           bool `json:"log_ddl"`
	LogDML           bool `json:"log_dml"`
	LogSelect        bool `json:"log_select"`
	LogAuth          bool `json:"log_auth"`
	LogAdmin         bool `json:"log_admin"`
	LogCluster       bool `json:"log_cluster"`
	RetentionDays    int  `json:"retention_days"`
	BufferSize       int  `json:"buffer_size"`
	FlushIntervalSec int  `json:"flush_interval_sec"`
}

// DefaultConfig returns default audit configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:          true,
		LogDDL:           true,
		LogDML:           true,
		LogSelect:        false, // Can be verbose
		LogAuth:          true,
		LogAdmin:         true,
		LogCluster:       true,
		RetentionDays:    90,
		BufferSize:       1000,
		FlushIntervalSec: 5,
	}
}

// Manager manages audit logging.
type Manager struct {
	config  Config
	store   storage.Engine
	logger  *logging.Logger
	buffer  chan Event
	stopCh  chan struct{}
	wg      sync.WaitGroup
	mu      sync.RWMutex
	enabled bool
}

// NewManager creates a new audit manager.
func NewManager(store storage.Engine, config Config) *Manager {
	m := &Manager{
		config:  config,
		store:   store,
		logger:  logging.NewLogger("audit"),
		buffer:  make(chan Event, config.BufferSize),
		stopCh:  make(chan struct{}),
		enabled: config.Enabled,
	}

	// Start background worker for async logging
	if config.Enabled {
		m.wg.Add(1)
		go m.worker()
	}

	return m
}

// worker processes audit events from the buffer.
func (m *Manager) worker() {
	defer m.wg.Done()

	ticker := time.NewTicker(time.Duration(m.config.FlushIntervalSec) * time.Second)
	defer ticker.Stop()

	batch := make([]Event, 0, 100)

	for {
		select {
		case event := <-m.buffer:
			batch = append(batch, event)
			if len(batch) >= 100 {
				m.flushBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				m.flushBatch(batch)
				batch = batch[:0]
			}

		case <-m.stopCh:
			// Flush remaining events
			for len(m.buffer) > 0 {
				batch = append(batch, <-m.buffer)
			}
			if len(batch) > 0 {
				m.flushBatch(batch)
			}
			return
		}
	}
}

// flushBatch writes a batch of events to storage.
func (m *Manager) flushBatch(events []Event) {
	for _, event := range events {
		if err := m.writeEvent(event); err != nil {
			m.logger.Error("Failed to write audit event", "error", err, "event_type", event.EventType)
		}
	}
}

// writeEvent writes a single event to storage.
func (m *Manager) writeEvent(event Event) error {
	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Serialize event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	// Store in audit log table with key: _audit:<timestamp>:<id>
	key := fmt.Sprintf("_audit:%d:%d", event.Timestamp.UnixNano(), event.ID)
	return m.store.Put(key, data)
}

// LogEvent logs an audit event asynchronously.
func (m *Manager) LogEvent(event Event) {
	m.mu.RLock()
	enabled := m.enabled
	m.mu.RUnlock()

	if !enabled {
		return
	}

	// Filter based on configuration
	if !m.shouldLog(event.EventType) {
		return
	}

	// Try to send to buffer, drop if full (non-blocking)
	select {
	case m.buffer <- event:
	default:
		m.logger.Warn("Audit buffer full, dropping event", "event_type", event.EventType)
	}
}

// shouldLog checks if an event type should be logged based on configuration.
func (m *Manager) shouldLog(eventType EventType) bool {
	switch eventType {
	case EventTypeLogin, EventTypeLogout, EventTypeAuthFailed:
		return m.config.LogAuth

	case EventTypeCreateTable, EventTypeDropTable, EventTypeAlterTable,
		EventTypeCreateIndex, EventTypeDropIndex, EventTypeCreateView,
		EventTypeDropView, EventTypeTruncateTable:
		return m.config.LogDDL

	case EventTypeInsert, EventTypeUpdate, EventTypeDelete:
		return m.config.LogDML

	case EventTypeSelect:
		return m.config.LogSelect

	case EventTypeBackup, EventTypeRestore, EventTypeCheckpoint, EventTypeVacuum:
		return m.config.LogAdmin

	case EventTypeNodeJoin, EventTypeNodeLeave, EventTypeLeaderElection, EventTypeFailover:
		return m.config.LogCluster

	default:
		return true
	}
}

// QueryOptions specifies options for querying audit logs.
type QueryOptions struct {
	StartTime  time.Time
	EndTime    time.Time
	Username   string
	Database   string
	EventType  EventType
	Status     Status
	ObjectType string
	ObjectName string
	Limit      int
	Offset     int
}

// QueryLogs retrieves audit logs matching the given criteria.
func (m *Manager) QueryLogs(opts QueryOptions) ([]Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var events []Event

	// Scan audit log entries
	prefix := "_audit:"
	results, err := m.store.Scan(prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to scan audit logs: %w", err)
	}

	for key, value := range results {
		var event Event
		if err := json.Unmarshal(value, &event); err != nil {
			m.logger.Warn("Failed to unmarshal audit event", "key", key, "error", err)
			continue
		}

		// Apply filters
		if !opts.StartTime.IsZero() && event.Timestamp.Before(opts.StartTime) {
			continue
		}
		if !opts.EndTime.IsZero() && event.Timestamp.After(opts.EndTime) {
			continue
		}
		if opts.Username != "" && event.Username != opts.Username {
			continue
		}
		if opts.Database != "" && event.Database != opts.Database {
			continue
		}
		if opts.EventType != "" && event.EventType != opts.EventType {
			continue
		}
		if opts.Status != "" && event.Status != opts.Status {
			continue
		}
		if opts.ObjectType != "" && event.ObjectType != opts.ObjectType {
			continue
		}
		if opts.ObjectName != "" && event.ObjectName != opts.ObjectName {
			continue
		}

		events = append(events, event)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to scan audit logs: %w", err)
	}

	// Apply limit and offset
	if opts.Offset > 0 {
		if opts.Offset >= len(events) {
			return []Event{}, nil
		}
		events = events[opts.Offset:]
	}
	if opts.Limit > 0 && opts.Limit < len(events) {
		events = events[:opts.Limit]
	}

	return events, nil
}

// ExportFormat represents the export format for audit logs.
type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
	FormatCSV  ExportFormat = "csv"
	FormatSQL  ExportFormat = "sql"
)

// ExportLogs exports audit logs to a file in the specified format.
func (m *Manager) ExportLogs(filename string, format ExportFormat, opts QueryOptions) error {
	events, err := m.QueryLogs(opts)
	if err != nil {
		return err
	}

	return m.ExportEvents(filename, format, events)
}

// ExportEvents exports a specific set of events to a file.
func (m *Manager) ExportEvents(filename string, format ExportFormat, events []Event) error {
	switch format {
	case FormatJSON:
		return m.exportJSON(filename, events)
	case FormatCSV:
		return m.exportCSV(filename, events)
	case FormatSQL:
		return m.exportSQL(filename, events)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

// Stop stops the audit manager and flushes pending events.
func (m *Manager) Stop() {
	m.mu.Lock()
	m.enabled = false
	m.mu.Unlock()

	close(m.stopCh)
	m.wg.Wait()
}

// Enable enables audit logging.
func (m *Manager) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = true
}

// Disable disables audit logging.
func (m *Manager) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = false
}

// IsEnabled returns whether audit logging is enabled.
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// CleanupOldLogs removes audit logs older than the retention period.
func (m *Manager) CleanupOldLogs() error {
	if m.config.RetentionDays <= 0 {
		return nil // Retention disabled
	}

	cutoff := time.Now().AddDate(0, 0, -m.config.RetentionDays)
	m.logger.Info("Cleaning up audit logs", "cutoff", cutoff, "retention_days", m.config.RetentionDays)

	count := 0
	prefix := "_audit:"
	results, err := m.store.Scan(prefix)
	if err != nil {
		return fmt.Errorf("failed to scan audit logs: %w", err)
	}

	for key, value := range results {
		var event Event
		if err := json.Unmarshal(value, &event); err != nil {
			continue
		}

		if event.Timestamp.Before(cutoff) {
			if err := m.store.Delete(key); err != nil {
				m.logger.Warn("Failed to delete old audit log", "key", key, "error", err)
			} else {
				count++
			}
		}
	}

	if err != nil {
		return fmt.Errorf("failed to cleanup audit logs: %w", err)
	}

	m.logger.Info("Audit log cleanup complete", "deleted_count", count)
	return nil
}
