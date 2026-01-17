/*
 * Copyright (c) 2026 Firefly Software Solutions Inc.
 * Licensed under the Apache License, Version 2.0
 */

package audit

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// exportJSON exports audit logs to JSON format.
func (m *Manager) exportJSON(filename string, events []Event) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(events); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	m.logger.Info("Exported audit logs to JSON", "filename", filename, "count", len(events))
	return nil
}

// exportCSV exports audit logs to CSV format.
func (m *Manager) exportCSV(filename string, events []Event) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"ID", "Timestamp", "EventType", "Username", "Database",
		"ObjectType", "ObjectName", "Operation", "ClientAddr",
		"SessionID", "Status", "ErrorMessage", "DurationMs", "Metadata",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for _, event := range events {
		metadata := ""
		if len(event.Metadata) > 0 {
			metaJSON, _ := json.Marshal(event.Metadata)
			metadata = string(metaJSON)
		}

		row := []string{
			strconv.FormatInt(event.ID, 10),
			event.Timestamp.Format("2006-01-02 15:04:05"),
			string(event.EventType),
			event.Username,
			event.Database,
			event.ObjectType,
			event.ObjectName,
			event.Operation,
			event.ClientAddr,
			event.SessionID,
			string(event.Status),
			event.ErrorMessage,
			strconv.FormatInt(event.DurationMs, 10),
			metadata,
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	m.logger.Info("Exported audit logs to CSV", "filename", filename, "count", len(events))
	return nil
}

// exportSQL exports audit logs to SQL format.
func (m *Manager) exportSQL(filename string, events []Event) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintln(file, "-- FlyDB Audit Log Export")
	fmt.Fprintf(file, "-- Generated: %s\n", events[0].Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "-- Total Events: %d\n\n", len(events))

	// Write CREATE TABLE statement
	fmt.Fprintln(file, "CREATE TABLE IF NOT EXISTS _audit_log (")
	fmt.Fprintln(file, "  id BIGINT PRIMARY KEY,")
	fmt.Fprintln(file, "  timestamp TIMESTAMP NOT NULL,")
	fmt.Fprintln(file, "  event_type TEXT NOT NULL,")
	fmt.Fprintln(file, "  username TEXT,")
	fmt.Fprintln(file, "  database TEXT,")
	fmt.Fprintln(file, "  object_type TEXT,")
	fmt.Fprintln(file, "  object_name TEXT,")
	fmt.Fprintln(file, "  operation TEXT,")
	fmt.Fprintln(file, "  client_addr TEXT,")
	fmt.Fprintln(file, "  session_id TEXT,")
	fmt.Fprintln(file, "  status TEXT,")
	fmt.Fprintln(file, "  error_message TEXT,")
	fmt.Fprintln(file, "  duration_ms BIGINT,")
	fmt.Fprintln(file, "  metadata JSONB")
	fmt.Fprintln(file, ");")
	fmt.Fprintln(file)

	// Write INSERT statements
	for _, event := range events {
		metadata := "NULL"
		if len(event.Metadata) > 0 {
			metaJSON, _ := json.Marshal(event.Metadata)
			metadata = fmt.Sprintf("'%s'", strings.ReplaceAll(string(metaJSON), "'", "''"))
		}

		errorMsg := "NULL"
		if event.ErrorMessage != "" {
			errorMsg = fmt.Sprintf("'%s'", strings.ReplaceAll(event.ErrorMessage, "'", "''"))
		}

		operation := strings.ReplaceAll(event.Operation, "'", "''")

		fmt.Fprintf(file, "INSERT INTO _audit_log VALUES (%d, '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', %s, %d, %s);\n",
			event.ID,
			event.Timestamp.Format("2006-01-02 15:04:05"),
			event.EventType,
			event.Username,
			event.Database,
			event.ObjectType,
			event.ObjectName,
			operation,
			event.ClientAddr,
			event.SessionID,
			event.Status,
			errorMsg,
			event.DurationMs,
			metadata,
		)
	}

	m.logger.Info("Exported audit logs to SQL", "filename", filename, "count", len(events))
	return nil
}
