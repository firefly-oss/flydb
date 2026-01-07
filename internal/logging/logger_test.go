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

package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogLevels(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("Level.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"DEBUG", DEBUG},
		{"debug", DEBUG},
		{"INFO", INFO},
		{"info", INFO},
		{"WARN", WARN},
		{"warn", WARN},
		{"WARNING", WARN},
		{"ERROR", ERROR},
		{"error", ERROR},
		{"unknown", INFO}, // default
	}

	for _, tt := range tests {
		if got := ParseLevel(tt.input); got != tt.expected {
			t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestLoggerTextOutput(t *testing.T) {
	var buf bytes.Buffer
	SetGlobalOutput(&buf)
	SetGlobalLevel(DEBUG)
	SetJSONMode(false)

	logger := NewLogger("test")
	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "[INFO ]") {
		t.Errorf("Expected [INFO ] in output, got: %s", output)
	}
	if !strings.Contains(output, "[test]") {
		t.Errorf("Expected [test] in output, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected 'test message' in output, got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("Expected 'key=value' in output, got: %s", output)
	}
}

func TestLoggerJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	SetGlobalOutput(&buf)
	SetGlobalLevel(DEBUG)
	SetJSONMode(true)

	logger := NewLogger("test")
	logger.Info("test message", "key", "value")

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if entry.Level != "INFO" {
		t.Errorf("Expected level INFO, got: %s", entry.Level)
	}
	if entry.Component != "test" {
		t.Errorf("Expected component 'test', got: %s", entry.Component)
	}
	if entry.Message != "test message" {
		t.Errorf("Expected message 'test message', got: %s", entry.Message)
	}
	if entry.Fields["key"] != "value" {
		t.Errorf("Expected field key=value, got: %v", entry.Fields)
	}

	// Reset to text mode
	SetJSONMode(false)
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	SetGlobalOutput(&buf)
	SetGlobalLevel(WARN)
	SetJSONMode(false)

	logger := NewLogger("test")
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Error("DEBUG message should be filtered out")
	}
	if strings.Contains(output, "info message") {
		t.Error("INFO message should be filtered out")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("WARN message should be present")
	}
	if !strings.Contains(output, "error message") {
		t.Error("ERROR message should be present")
	}

	// Reset level
	SetGlobalLevel(INFO)
}

func TestContextLogger(t *testing.T) {
	var buf bytes.Buffer
	SetGlobalOutput(&buf)
	SetGlobalLevel(DEBUG)
	SetJSONMode(false)

	logger := NewLogger("test")
	ctxLogger := logger.With("request_id", "123", "user", "admin")
	ctxLogger.Info("context message")

	output := buf.String()
	if !strings.Contains(output, "request_id=123") {
		t.Errorf("Expected 'request_id=123' in output, got: %s", output)
	}
	if !strings.Contains(output, "user=admin") {
		t.Errorf("Expected 'user=admin' in output, got: %s", output)
	}
}

