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

// LogEvent logs an audit event from the SQL executor.
func (a *Adapter) LogEvent(event interface{}) {
	// Type assert to get the event fields
	// The SQL executor passes a struct with these fields
	type SQLAuditEvent struct {
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

	sqlEvent, ok := event.(SQLAuditEvent)
	if !ok {
		// Try to handle as a map
		if eventMap, ok := event.(map[string]interface{}); ok {
			sqlEvent = SQLAuditEvent{
				EventType:    getString(eventMap, "EventType"),
				Username:     getString(eventMap, "Username"),
				Database:     getString(eventMap, "Database"),
				ObjectType:   getString(eventMap, "ObjectType"),
				ObjectName:   getString(eventMap, "ObjectName"),
				Operation:    getString(eventMap, "Operation"),
				ClientAddr:   getString(eventMap, "ClientAddr"),
				SessionID:    getString(eventMap, "SessionID"),
				Status:       getString(eventMap, "Status"),
				ErrorMessage: getString(eventMap, "ErrorMessage"),
				DurationMs:   getInt64(eventMap, "DurationMs"),
			}
		} else {
			return
		}
	}

	// Convert to audit Event
	auditEvent := Event{
		Timestamp:    time.Now(),
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

	a.manager.LogEvent(auditEvent)
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

