/*
 * Copyright (c) 2026 Firefly Software Solutions Inc.
 * Licensed under the Apache License, Version 2.0
 */

/*
Package sdk provides audit trail functionality for FlyDB SDK.

This module enables SDK clients to query, export, and manage audit logs
through a simple, type-safe API.

Usage:
======

  // Create audit client
  auditClient := sdk.NewAuditClient(session)

  // Query recent audit logs
  logs, err := auditClient.GetRecentLogs(100)

  // Query logs by user
  logs, err := auditClient.GetLogsByUser("admin", 50)

  // Query logs in time range
  logs, err := auditClient.GetLogsInTimeRange(startTime, endTime, 100)

  // Export audit logs
  err := auditClient.ExportLogs("audit.json", sdk.AuditFormatJSON, queryOpts)

  // Get audit statistics
  stats, err := auditClient.GetStatistics()

Thread Safety:
==============

The audit client is thread-safe and can be used concurrently from multiple
goroutines.
*/
package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

// AuditEventType represents the type of audit event.
type AuditEventType string

const (
	// Authentication events
	AuditEventLogin      AuditEventType = "LOGIN"
	AuditEventLogout     AuditEventType = "LOGOUT"
	AuditEventAuthFailed AuditEventType = "AUTH_FAILED"

	// DDL events
	AuditEventCreateTable AuditEventType = "CREATE_TABLE"
	AuditEventDropTable   AuditEventType = "DROP_TABLE"
	AuditEventAlterTable  AuditEventType = "ALTER_TABLE"
	AuditEventCreateIndex AuditEventType = "CREATE_INDEX"
	AuditEventDropIndex   AuditEventType = "DROP_INDEX"

	// DML events
	AuditEventInsert AuditEventType = "INSERT"
	AuditEventUpdate AuditEventType = "UPDATE"
	AuditEventDelete AuditEventType = "DELETE"
	AuditEventSelect AuditEventType = "SELECT"

	// Administrative events
	AuditEventBackup     AuditEventType = "BACKUP"
	AuditEventRestore    AuditEventType = "RESTORE"
	AuditEventCheckpoint AuditEventType = "CHECKPOINT"
	AuditEventVacuum     AuditEventType = "VACUUM"

	// Cluster events
	AuditEventNodeJoin       AuditEventType = "NODE_JOIN"
	AuditEventNodeLeave      AuditEventType = "NODE_LEAVE"
	AuditEventLeaderElection AuditEventType = "LEADER_ELECTION"
	AuditEventFailover       AuditEventType = "FAILOVER"
)

// AuditStatus represents the outcome of an audited event.
type AuditStatus string

const (
	AuditStatusSuccess AuditStatus = "SUCCESS"
	AuditStatusFailed  AuditStatus = "FAILED"
)

// AuditLog represents a single audit log entry.
type AuditLog struct {
	ID           int64              `json:"id"`
	Timestamp    time.Time          `json:"timestamp"`
	EventType    AuditEventType     `json:"event_type"`
	Username     string             `json:"username"`
	Database     string             `json:"database"`
	ObjectType   string             `json:"object_type"`
	ObjectName   string             `json:"object_name"`
	Operation    string             `json:"operation"`
	ClientAddr   string             `json:"client_addr"`
	SessionID    string             `json:"session_id"`
	Status       AuditStatus        `json:"status"`
	ErrorMessage string             `json:"error_message,omitempty"`
	DurationMs   int64              `json:"duration_ms"`
	Metadata     map[string]string  `json:"metadata,omitempty"`
}

// AuditQueryOptions specifies options for querying audit logs.
type AuditQueryOptions struct {
	StartTime  time.Time
	EndTime    time.Time
	Username   string
	Database   string
	EventType  AuditEventType
	Status     AuditStatus
	ObjectType string
	ObjectName string
	Limit      int
	Offset     int
}

// AuditFormat represents the export format for audit logs.
type AuditFormat string

const (
	AuditFormatJSON AuditFormat = "json"
	AuditFormatCSV  AuditFormat = "csv"
	AuditFormatSQL  AuditFormat = "sql"
)

// AuditStatistics contains statistics about audit logs.
type AuditStatistics struct {
	TotalEvents      int64                      `json:"total_events"`
	EventTypeCounts  map[AuditEventType]int64   `json:"event_type_counts"`
	StatusCounts     map[AuditStatus]int64      `json:"status_counts"`
	UserCounts       map[string]int64           `json:"user_counts"`
	OldestEvent      time.Time                  `json:"oldest_event"`
	NewestEvent      time.Time                  `json:"newest_event"`
}

// AuditClient provides methods for querying and managing audit logs.
type AuditClient struct {
	session *Session
}

// NewAuditClient creates a new audit client.
func NewAuditClient(session *Session) *AuditClient {
	return &AuditClient{session: session}
}

// GetRecentLogs retrieves the most recent audit logs.
func (c *AuditClient) GetRecentLogs(limit int) ([]AuditLog, error) {
	return c.QueryLogs(AuditQueryOptions{Limit: limit})
}

