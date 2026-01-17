/*
 * Copyright (c) 2026 Firefly Software Solutions Inc.
 * Licensed under the Apache License, Version 2.0
 */

package audit

import (
	"fmt"
	"strings"
	"time"
)

// SQLHelper provides SQL query generation for audit logs.
type SQLHelper struct {
	manager *Manager
}

// NewSQLHelper creates a new SQL helper.
func NewSQLHelper(manager *Manager) *SQLHelper {
	return &SQLHelper{manager: manager}
}

// BuildQuerySQL builds a SQL query for retrieving audit logs.
func (h *SQLHelper) BuildQuerySQL(opts QueryOptions) string {
	var conditions []string

	if !opts.StartTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp >= '%s'", opts.StartTime.Format("2006-01-02 15:04:05")))
	}
	if !opts.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp <= '%s'", opts.EndTime.Format("2006-01-02 15:04:05")))
	}
	if opts.Username != "" {
		conditions = append(conditions, fmt.Sprintf("username = '%s'", opts.Username))
	}
	if opts.Database != "" {
		conditions = append(conditions, fmt.Sprintf("database = '%s'", opts.Database))
	}
	if opts.EventType != "" {
		conditions = append(conditions, fmt.Sprintf("event_type = '%s'", opts.EventType))
	}
	if opts.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = '%s'", opts.Status))
	}
	if opts.ObjectType != "" {
		conditions = append(conditions, fmt.Sprintf("object_type = '%s'", opts.ObjectType))
	}
	if opts.ObjectName != "" {
		conditions = append(conditions, fmt.Sprintf("object_name = '%s'", opts.ObjectName))
	}

	query := "SELECT * FROM _audit_log"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY timestamp DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	return query
}

// GetAuditStats returns statistics about audit logs.
func (h *SQLHelper) GetAuditStats() (map[string]interface{}, error) {
	events, err := h.manager.QueryLogs(QueryOptions{})
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	stats["total_events"] = len(events)

	// Count by event type
	eventTypeCounts := make(map[EventType]int)
	statusCounts := make(map[Status]int)
	userCounts := make(map[string]int)

	var oldestEvent, newestEvent time.Time
	for i, event := range events {
		eventTypeCounts[event.EventType]++
		statusCounts[event.Status]++
		userCounts[event.Username]++

		if i == 0 || event.Timestamp.Before(oldestEvent) {
			oldestEvent = event.Timestamp
		}
		if i == 0 || event.Timestamp.After(newestEvent) {
			newestEvent = event.Timestamp
		}
	}

	stats["event_type_counts"] = eventTypeCounts
	stats["status_counts"] = statusCounts
	stats["user_counts"] = userCounts

	if !oldestEvent.IsZero() {
		stats["oldest_event"] = oldestEvent.Format("2006-01-02 15:04:05")
	}
	if !newestEvent.IsZero() {
		stats["newest_event"] = newestEvent.Format("2006-01-02 15:04:05")
	}

	return stats, nil
}

// GetRecentEvents returns the most recent audit events.
func (h *SQLHelper) GetRecentEvents(limit int) ([]Event, error) {
	return h.manager.QueryLogs(QueryOptions{
		Limit: limit,
	})
}

// GetEventsByUser returns audit events for a specific user.
func (h *SQLHelper) GetEventsByUser(username string, limit int) ([]Event, error) {
	return h.manager.QueryLogs(QueryOptions{
		Username: username,
		Limit:    limit,
	})
}

// GetEventsByType returns audit events of a specific type.
func (h *SQLHelper) GetEventsByType(eventType EventType, limit int) ([]Event, error) {
	return h.manager.QueryLogs(QueryOptions{
		EventType: eventType,
		Limit:     limit,
	})
}

// GetFailedEvents returns failed audit events.
func (h *SQLHelper) GetFailedEvents(limit int) ([]Event, error) {
	return h.manager.QueryLogs(QueryOptions{
		Status: StatusFailed,
		Limit:  limit,
	})
}

// GetEventsInTimeRange returns audit events within a time range.
func (h *SQLHelper) GetEventsInTimeRange(start, end time.Time, limit int) ([]Event, error) {
	return h.manager.QueryLogs(QueryOptions{
		StartTime: start,
		EndTime:   end,
		Limit:     limit,
	})
}

// GetEventsByDatabase returns audit events for a specific database.
func (h *SQLHelper) GetEventsByDatabase(database string, limit int) ([]Event, error) {
	return h.manager.QueryLogs(QueryOptions{
		Database: database,
		Limit:    limit,
	})
}

