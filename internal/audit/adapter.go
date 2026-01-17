/*
 * Copyright (c) 2026 Firefly Software Solutions Inc.
 * Licensed under the Apache License, Version 2.0
 */

package audit

import (
	"time"
)

// Adapter adapts the audit Manager to the SQL executor's AuditManager interface.
type Adapter struct {
	manager *Manager
}

// NewAdapter creates a new audit adapter.
func NewAdapter(manager *Manager) *Adapter {
	return &Adapter{manager: manager}
}

// SQLAuditEvent represents an audit log entry as seen by the SQL executor.
type SQLAuditEvent struct {
	ID           int64
	Timestamp    time.Time
	EventType    string
	Username     string
	Database     string
	ObjectType   string
	ObjectName   string
	Operation    string
	ClientAddr   string
	SessionID    string
	Status       string
	ErrorMessage string
	DurationMs   int64
	Metadata     map[string]string
}

// SQLAuditQueryOptions specifies options for querying audit logs.
type SQLAuditQueryOptions struct {
	StartTime  time.Time
	EndTime    time.Time
	Username   string
	Database   string
	EventType  string
	Status     string
	ObjectType string
	ObjectName string
	Limit      int
	Offset     int
}

// LogEvent logs an audit event from the SQL executor.
func (a *Adapter) LogEvent(event interface{}) {
	sqlEvent, ok := event.(SQLAuditEvent)
	if !ok {
		return
	}

	// Convert to audit Event
	auditEvent := Event{
		Timestamp:    sqlEvent.Timestamp,
		EventType:    EventType(sqlEvent.EventType),
		Username:     sqlEvent.Username,
		Database:     sqlEvent.Database,
		ObjectType:   sqlEvent.ObjectType,
		ObjectName:   sqlEvent.ObjectName,
		Operation:    sqlEvent.Operation,
		ClientAddr:   sqlEvent.ClientAddr,
		SessionID:    sqlEvent.SessionID,
		Status:       Status(sqlEvent.Status),
		ErrorMessage: sqlEvent.ErrorMessage,
		DurationMs:   sqlEvent.DurationMs,
		Metadata:     sqlEvent.Metadata,
	}

	if auditEvent.Timestamp.IsZero() {
		auditEvent.Timestamp = time.Now()
	}

	a.manager.LogEvent(auditEvent)
}

// QueryLogs retrieves audit logs matching the given criteria.
func (a *Adapter) QueryLogs(opts interface{}) ([]interface{}, error) {
	sqlOpts, ok := opts.(SQLAuditQueryOptions)
	if !ok {
		return nil, nil
	}

	// Convert to audit QueryOptions
	auditOpts := QueryOptions{
		StartTime:  sqlOpts.StartTime,
		EndTime:    sqlOpts.EndTime,
		Username:   sqlOpts.Username,
		Database:   sqlOpts.Database,
		EventType:  EventType(sqlOpts.EventType),
		Status:     Status(sqlOpts.Status),
		ObjectType: sqlOpts.ObjectType,
		ObjectName: sqlOpts.ObjectName,
		Limit:      sqlOpts.Limit,
		Offset:     sqlOpts.Offset,
	}

	events, err := a.manager.QueryLogs(auditOpts)
	if err != nil {
		return nil, err
	}

	// Convert back to SQLAuditEvent
	results := make([]interface{}, len(events))
	for i, event := range events {
		results[i] = SQLAuditEvent{
			ID:           event.ID,
			Timestamp:    event.Timestamp,
			EventType:    string(event.EventType),
			Username:     event.Username,
			Database:     event.Database,
			ObjectType:   event.ObjectType,
			ObjectName:   event.ObjectName,
			Operation:    event.Operation,
			ClientAddr:   event.ClientAddr,
			SessionID:    event.SessionID,
			Status:       string(event.Status),
			ErrorMessage: event.ErrorMessage,
			DurationMs:   event.DurationMs,
			Metadata:     event.Metadata,
		}
	}

	return results, nil
}

// ExportLogs exports audit logs to a file in the specified format.
func (a *Adapter) ExportLogs(filename string, format string, opts interface{}) error {
	sqlOpts, ok := opts.(SQLAuditQueryOptions)
	if !ok {
		return nil
	}

	auditOpts := QueryOptions{
		StartTime: sqlOpts.StartTime,
		EndTime:   sqlOpts.EndTime,
		Username:  sqlOpts.Username,
		Limit:     sqlOpts.Limit,
	}

	return a.manager.ExportLogs(filename, ExportFormat(format), auditOpts)
}

// GetStats returns statistics about audit logs.
func (a *Adapter) GetStats() (map[string]interface{}, error) {
	helper := NewSQLHelper(a.manager)
	return helper.GetAuditStats()
}

// getString safely gets a string from a map.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getInt64 safely gets an int64 from a map.
func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int64:
			return val
		case int:
			return int64(val)
		case float64:
			return int64(val)
		}
	}
	return 0
}
