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

package main

import (
	"strings"
	"testing"
)

// TestParseHosts tests the parseHosts function
func TestParseHosts(t *testing.T) {
	tests := []struct {
		name     string
		hostStr  string
		portStr  string
		expected []string
	}{
		{
			name:     "single host without port",
			hostStr:  "localhost",
			portStr:  "8889",
			expected: []string{"localhost:8889"},
		},
		{
			name:     "single host with port",
			hostStr:  "localhost:9999",
			portStr:  "8889",
			expected: []string{"localhost:9999"},
		},
		{
			name:     "multiple hosts without ports",
			hostStr:  "node1,node2,node3",
			portStr:  "8889",
			expected: []string{"node1:8889", "node2:8889", "node3:8889"},
		},
		{
			name:     "multiple hosts with mixed ports",
			hostStr:  "node1:8889,node2,node3:9999",
			portStr:  "8889",
			expected: []string{"node1:8889", "node2:8889", "node3:9999"},
		},
		{
			name:     "hosts with spaces",
			hostStr:  " node1 , node2 , node3 ",
			portStr:  "8889",
			expected: []string{"node1:8889", "node2:8889", "node3:8889"},
		},
		{
			name:     "empty string",
			hostStr:  "",
			portStr:  "8889",
			expected: []string{},
		},
		{
			name:     "only commas",
			hostStr:  ",,",
			portStr:  "8889",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHosts(tt.hostStr, tt.portStr)
			if len(result) != len(tt.expected) {
				t.Errorf("parseHosts(%q, %q) = %v, want %v", tt.hostStr, tt.portStr, result, tt.expected)
				return
			}
			for i, host := range result {
				if host != tt.expected[i] {
					t.Errorf("parseHosts(%q, %q)[%d] = %q, want %q", tt.hostStr, tt.portStr, i, host, tt.expected[i])
				}
			}
		})
	}
}

// TestIsConnectionError tests the isConnectionError function
func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{"connection refused", "dial tcp: connection refused", true},
		{"connection reset", "read: connection reset by peer", true},
		{"broken pipe", "write: broken pipe", true},
		{"EOF error", "unexpected EOF", true},
		{"timeout", "i/o timeout", true},
		{"auth error", "authentication failed", false},
		{"syntax error", "syntax error near SELECT", false},
		{"nil error message", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.errMsg != "" {
				err = &testError{msg: tt.errMsg}
			}
			result := isConnectionError(err)
			if result != tt.expected {
				t.Errorf("isConnectionError(%q) = %v, want %v", tt.errMsg, result, tt.expected)
			}
		})
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// TestFormatSQLValue tests the formatSQLValue function
func TestFormatSQLValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "hello", "'hello'"},
		{"string with quote", "it's", "'it''s'"},
		{"integer float", float64(42), "42"},
		{"decimal float", float64(3.14), "3.14"},
		{"boolean true", true, "TRUE"},
		{"boolean false", false, "FALSE"},
		{"nil", nil, "NULL"},
		{"other type", []int{1, 2, 3}, "'[1 2 3]'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSQLValue(tt.input)
			if result != tt.expected {
				t.Errorf("formatSQLValue(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSQLScanner tests the SQLScanner for parsing SQL statements
func TestSQLScanner(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single statement",
			input:    "SELECT * FROM users;",
			expected: []string{"SELECT * FROM users;"},
		},
		{
			name:     "multiple statements",
			input:    "SELECT * FROM users;\nINSERT INTO users VALUES (1);",
			expected: []string{"SELECT * FROM users;", "INSERT INTO users VALUES (1);"},
		},
		{
			name:     "statement with newlines",
			input:    "SELECT * FROM users WHERE id = 1;",
			expected: []string{"SELECT * FROM users WHERE id = 1;"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "comments only",
			input:    "-- This is a comment\n-- Another comment",
			expected: []string{},
		},
		{
			name:     "statement with comment",
			input:    "-- Comment\nSELECT * FROM users;",
			expected: []string{"SELECT * FROM users;"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewSQLScanner(strings.NewReader(tt.input))
			var results []string
			for scanner.Scan() {
				text := strings.TrimSpace(scanner.Text())
				if text != "" && !strings.HasPrefix(text, "--") {
					results = append(results, text)
				}
			}
			if err := scanner.Err(); err != nil {
				t.Errorf("SQLScanner error: %v", err)
				return
			}
			if len(results) != len(tt.expected) {
				t.Errorf("SQLScanner got %d statements, want %d", len(results), len(tt.expected))
				t.Errorf("Got: %v", results)
				t.Errorf("Want: %v", tt.expected)
				return
			}
			for i, stmt := range results {
				if stmt != tt.expected[i] {
					t.Errorf("Statement %d = %q, want %q", i, stmt, tt.expected[i])
				}
			}
		})
	}
}

// TestFormatFileSize tests the formatFileSize function
func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected string
	}{
		{"bytes", 500, "500 bytes"},
		{"kilobytes", 1024, "1.00 KB"},
		{"megabytes", 1024 * 1024, "1.00 MB"},
		{"gigabytes", 1024 * 1024 * 1024, "1.00 GB"},
		{"mixed KB", 2560, "2.50 KB"},
		{"mixed MB", 5 * 1024 * 1024, "5.00 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatFileSize(tt.size)
			if result != tt.expected {
				t.Errorf("formatFileSize(%d) = %q, want %q", tt.size, result, tt.expected)
			}
		})
	}
}

// TestHAClientHosts tests HAClient host management
func TestHAClientHosts(t *testing.T) {
	hosts := []string{"node1:8889", "node2:8889", "node3:8889"}
	client := NewHAClient(hosts)

	if len(client.hosts) != 3 {
		t.Errorf("HAClient hosts count = %d, want 3", len(client.hosts))
	}

	for i, h := range hosts {
		if client.hosts[i] != h {
			t.Errorf("HAClient hosts[%d] = %q, want %q", i, client.hosts[i], h)
		}
	}
}

// TestIsLocalMode tests the isLocalMode function
func TestIsLocalMode(t *testing.T) {
	// Save original value
	originalDataDir := *dataDir
	defer func() { *dataDir = originalDataDir }()

	*dataDir = ""
	if isLocalMode() {
		t.Error("isLocalMode() = true when dataDir is empty, want false")
	}

	*dataDir = "/var/lib/flydb"
	if !isLocalMode() {
		t.Error("isLocalMode() = false when dataDir is set, want true")
	}
}

func TestIsRemoteMode(t *testing.T) {
	// Save original value
	originalHost := *host
	defer func() { *host = originalHost }()

	*host = ""
	if isRemoteMode() {
		t.Error("isRemoteMode() = true when host is empty, want false")
	}

	*host = "localhost"
	if !isRemoteMode() {
		t.Error("isRemoteMode() = false when host is set, want true")
	}
}

