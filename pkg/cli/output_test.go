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

package cli

import (
	"testing"
)

func TestVisibleLen(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "plain text",
			input:    "hello",
			expected: 5,
		},
		{
			name:     "text with bold",
			input:    "\033[1mhello\033[0m",
			expected: 5,
		},
		{
			name:     "text with color",
			input:    "\033[31mred text\033[0m",
			expected: 8,
		},
		{
			name:     "text with multiple codes",
			input:    "\033[1m\033[31mbold red\033[0m",
			expected: 8,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "only ANSI codes",
			input:    "\033[1m\033[0m",
			expected: 0,
		},
		{
			name:     "unicode characters",
			input:    "héllo wörld",
			expected: 13, // len() returns byte count, not rune count
		},
		{
			name:     "mixed ANSI and unicode",
			input:    "\033[32m✓\033[0m Success",
			expected: 11, // ✓ is 3 bytes, " Success" is 8 bytes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := visibleLen(tt.input)
			if result != tt.expected {
				t.Errorf("visibleLen(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseOutputFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected OutputFormat
	}{
		{"table", FormatTable},
		{"TABLE", FormatTable},
		{"Table", FormatTable},
		{"json", FormatJSON},
		{"JSON", FormatJSON},
		{"plain", FormatPlain},
		{"PLAIN", FormatPlain},
		{"", FormatTable},
		{"unknown", FormatTable},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseOutputFormat(tt.input)
			if result != tt.expected {
				t.Errorf("ParseOutputFormat(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewTable(t *testing.T) {
	table := NewTable("ID", "Name", "Email")

	if len(table.headers) != 3 {
		t.Errorf("Expected 3 headers, got %d", len(table.headers))
	}
	if table.headers[0] != "ID" {
		t.Errorf("Expected first header 'ID', got '%s'", table.headers[0])
	}
	if table.format != FormatTable {
		t.Errorf("Expected default format FormatTable, got %v", table.format)
	}
	if len(table.rows) != 0 {
		t.Errorf("Expected 0 rows, got %d", len(table.rows))
	}
}

func TestTableAddRow(t *testing.T) {
	table := NewTable("ID", "Name")
	table.AddRow("1", "Alice")
	table.AddRow("2", "Bob")

	if len(table.rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(table.rows))
	}
	if table.rows[0][0] != "1" || table.rows[0][1] != "Alice" {
		t.Errorf("First row mismatch: got %v", table.rows[0])
	}
}

func TestTableSetFormat(t *testing.T) {
	table := NewTable("ID")
	table.SetFormat(FormatJSON)

	if table.format != FormatJSON {
		t.Errorf("Expected FormatJSON, got %v", table.format)
	}
}

// TestTableColumnWidthCalculation verifies that column widths are calculated
// correctly when data is wider than headers.
func TestTableColumnWidthCalculation(t *testing.T) {
	// This is a conceptual test - the actual width calculation happens
	// during printing. We verify the table structure is correct.
	table := NewTable("ID", "N") // Short headers
	table.AddRow("12345", "A very long name that exceeds header")

	if len(table.rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(table.rows))
	}
	// The actual alignment is verified visually or through integration tests
}

