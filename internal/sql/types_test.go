package sql

import (
	"testing"
)

func TestValidateValue(t *testing.T) {
	tests := []struct {
		typeName string
		value    string
		valid    bool
	}{
		// INT tests
		{"INT", "42", true},
		{"INT", "-100", true},
		{"INT", "0", true},
		{"INT", "abc", false},
		{"INT", "12.5", false},

		// FLOAT tests
		{"FLOAT", "3.14", true},
		{"FLOAT", "-2.5", true},
		{"FLOAT", "100", true},
		{"FLOAT", "abc", false},

		// BOOLEAN tests
		{"BOOLEAN", "TRUE", true},
		{"BOOLEAN", "FALSE", true},
		{"BOOLEAN", "true", true},
		{"BOOLEAN", "false", true},
		{"BOOLEAN", "1", true},
		{"BOOLEAN", "0", true},
		{"BOOLEAN", "yes", false},
		{"BOOLEAN", "no", false},

		// TIMESTAMP tests
		{"TIMESTAMP", "2026-01-06T10:30:00Z", true},
		{"TIMESTAMP", "2026-01-06 10:30:00", true},
		{"TIMESTAMP", "2026-01-06", false},
		{"TIMESTAMP", "invalid", false},

		// DATE tests
		{"DATE", "2026-01-06", true},
		{"DATE", "2026-12-31", true},
		{"DATE", "2026-13-01", false},
		{"DATE", "invalid", false},

		// UUID tests
		{"UUID", "550e8400-e29b-41d4-a716-446655440000", true},
		{"UUID", "550E8400-E29B-41D4-A716-446655440000", true},
		{"UUID", "invalid-uuid", false},
		{"UUID", "550e8400e29b41d4a716446655440000", false},

		// BLOB tests (base64)
		{"BLOB", "SGVsbG8gV29ybGQ=", true},
		{"BLOB", "dGVzdA==", true},
		{"BLOB", "not-valid-base64!!!", false},

		// JSONB tests
		{"JSONB", `{"name": "Alice", "age": 30}`, true},
		{"JSONB", `[1, 2, 3]`, true},
		{"JSONB", `"string"`, true},
		{"JSONB", `42`, true},
		{"JSONB", `{invalid}`, false},
		{"JSONB", ``, false},

		// TEXT tests (always valid)
		{"TEXT", "any string", true},
		{"TEXT", "", true},
		{"TEXT", "123", true},

		// BIGINT tests
		{"BIGINT", "9223372036854775807", true},
		{"BIGINT", "-9223372036854775808", true},
		{"BIGINT", "abc", false},

		// DECIMAL tests
		{"DECIMAL", "123.456", true},
		{"DECIMAL", "-99.99", true},
		{"DECIMAL", "100", true},
		{"DECIMAL", "abc", false},

		// NUMERIC tests (alias for DECIMAL)
		{"NUMERIC", "123.456", true},
		{"NUMERIC", "abc", false},

		// TIME tests
		{"TIME", "10:30:00", true},
		{"TIME", "23:59:59", true},
		{"TIME", "25:00:00", false},
		{"TIME", "invalid", false},

		// VARCHAR tests (same as TEXT)
		{"VARCHAR", "any string", true},
		{"VARCHAR", "", true},

		// SERIAL tests
		{"SERIAL", "1", true},
		{"SERIAL", "100", true},
		{"SERIAL", "-1", false},
		{"SERIAL", "abc", false},
	}

	for _, tt := range tests {
		err := ValidateValue(tt.typeName, tt.value)
		if tt.valid && err != nil {
			t.Errorf("ValidateValue(%s, %s) should be valid, got error: %v", tt.typeName, tt.value, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateValue(%s, %s) should be invalid, got no error", tt.typeName, tt.value)
		}
	}
}

func TestNormalizeValue(t *testing.T) {
	tests := []struct {
		typeName string
		value    string
		expected string
	}{
		// BOOLEAN normalization
		{"BOOLEAN", "TRUE", "true"},
		{"BOOLEAN", "FALSE", "false"},
		{"BOOLEAN", "1", "true"},
		{"BOOLEAN", "0", "false"},

		// UUID normalization (lowercase)
		{"UUID", "550E8400-E29B-41D4-A716-446655440000", "550e8400-e29b-41d4-a716-446655440000"},

		// JSONB normalization (compact)
		{"JSONB", `{ "name" : "Alice" }`, `{"name":"Alice"}`},

		// TEXT (no change)
		{"TEXT", "hello world", "hello world"},

		// INT (no change)
		{"INT", "42", "42"},
	}

	for _, tt := range tests {
		result, err := NormalizeValue(tt.typeName, tt.value)
		if err != nil {
			t.Errorf("NormalizeValue(%s, %s) failed: %v", tt.typeName, tt.value, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("NormalizeValue(%s, %s) = %s, expected %s", tt.typeName, tt.value, result, tt.expected)
		}
	}
}

func TestIsValidType(t *testing.T) {
	validTypes := []string{
		"INT", "INTEGER", "BIGINT", "SMALLINT", "TINYINT",
		"TEXT", "VARCHAR", "CHAR", "CHARACTER",
		"BOOLEAN", "BOOL",
		"FLOAT", "DOUBLE", "REAL", "DECIMAL", "NUMERIC",
		"TIMESTAMP", "DATETIME", "DATE", "TIME",
		"BLOB", "BYTEA", "BINARY", "VARBINARY",
		"UUID", "JSONB", "JSON", "SERIAL",
	}
	invalidTypes := []string{"INVALID", "ARRAY", "CLOB", "INTERVAL"}

	for _, typeName := range validTypes {
		if !IsValidType(typeName) {
			t.Errorf("IsValidType(%s) should be true", typeName)
		}
	}

	for _, typeName := range invalidTypes {
		if IsValidType(typeName) {
			t.Errorf("IsValidType(%s) should be false", typeName)
		}
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		typeName string
		a, b     string
		expected int
	}{
		{"INT", "10", "20", -1},
		{"INT", "20", "10", 1},
		{"INT", "10", "10", 0},

		{"FLOAT", "1.5", "2.5", -1},
		{"FLOAT", "2.5", "1.5", 1},

		{"TEXT", "apple", "banana", -1},
		{"TEXT", "banana", "apple", 1},
		{"TEXT", "apple", "apple", 0},

		{"BOOLEAN", "false", "true", -1},
		{"BOOLEAN", "true", "false", 1},
	}

	for _, tt := range tests {
		result := CompareValues(tt.typeName, tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("CompareValues(%s, %s, %s) = %d, expected %d", tt.typeName, tt.a, tt.b, result, tt.expected)
		}
	}
}

