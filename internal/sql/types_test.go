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

		// INET tests
		{"INET", "192.168.1.1", true},
		{"INET", "::1", true},
		{"INET", "192.168.1.0/24", true},
		{"INET", "2001:db8::/32", true},
		{"INET", "256.0.0.1", false}, // Invalid IP
		{"INET", "invalid-ip", false},

		// INTERVAL tests
		{"INTERVAL", "1h", true},
		{"INTERVAL", "1.5h", true},
		{"INTERVAL", "300ms", true},
		{"INTERVAL", "P1Y2M", true},    // ISO-8601
		{"INTERVAL", "P1DT1H", true},   // ISO-8601
		{"INTERVAL", "1 year", true},   // Postgres style
		{"INTERVAL", "2mons", true},    // Postgres style simplified
		{"INTERVAL", "invalid", false}, // No number, no P prefix
		{"INTERVAL", "", false},        // Empty

		// SET tests
		{"SET", `[1, 2, 3]`, true},
		{"SET", `["a", "b"]`, true},
		{"SET", `[1, "a", true]`, true},
		{"SET", `{}`, false}, // Not an array
		{"SET", `123`, false},

		// ZSET tests
		{"ZSET", `[{"score": 1, "member": "a"}, {"score": 2, "member": "b"}]`, true},
		{"ZSET", `[{"score": 1.5, "member": "val"}]`, true},
		{"ZSET", `[]`, true},                                     // Empty array is valid
		{"ZSET", `[{"member": "missing_score"}]`, true},          // Field missing defaults to 0 value for float64, technically valid Unmarshal but maybe not logic? Unmarshal will zero-fill. Let's keep it lenient on strict types but strict on structure. actually json.Unmarshal won't error on missing fields unless DisallowUnknownFields. but we defined struct tags. It's fine.
		{"ZSET", `[{"score": "invalid", "member": "a"}]`, false}, // Score must be number
		{"ZSET", `["not_object"]`, false},                        // Must be array of objects
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

		// INET normalization
		{"INET", "192.168.1.1", "192.168.1.1"},
		{"INET", "192.168.1.0/24", "192.168.1.0/24"},

		// INTERVAL normalization
		{"INTERVAL", "1h", "1h0m0s"},
		{"INTERVAL", "60m", "1h0m0s"},

		// SET normalization (deduplicate and sort)
		{"SET", `[3, 1, 2, 2, 1]`, `[1,2,3]`},
		{"SET", `["b", "a", "c", "a"]`, `["a","b","c"]`},
		{"SET", `[{"b":2}, {"a":1}]`, `[{"a":1},{"b":2}]`}, // Sorting JSON objects

		// ZSET normalization (sort by score)
		{"ZSET",
			`[{"score": 2, "member": "b"}, {"score": 1, "member": "a"}]`,
			`[{"score":1,"member":"a"},{"score":2,"member":"b"}]`},
		{"ZSET",
			`[{"score": 1, "member": "b"}, {"score": 1, "member": "a"}]`,
			`[{"score":1,"member":"a"},{"score":1,"member":"b"}]`}, // Tie-break by member
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
		"MONEY", "INTERVAL", "CLOB", "NCHAR", "NVARCHAR", "NTEXT",
		"INET", "SET", "ZSET",
	}
	invalidTypes := []string{"INVALID", "ARRAY", "ENUM"}

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

		{"INET", "192.168.1.1", "192.168.1.2", -1},
		{"INET", "192.168.1.20", "192.168.1.3", 1}, // 20 > 3
		{"INET", "192.168.1.1", "192.168.1.1", 0},
		{"INET", "10.0.0.0/8", "192.168.1.1", -1}, // 10.x < 192.x

		// SET/ZSET comparison (lexicographical on normalized form)
		// Note: [1,2] > [1,2,3] because ']' (93) > ',' (44) in ASCII
		{"SET", `[1, 2]`, `[1, 2, 3]`, 1},
		{"SET", `[1, 3]`, `[1, 2]`, 1},
		// Note: Comparison of JSON strings is tricky but strictly standard.
	}

	for _, tt := range tests {
		result := CompareValues(tt.typeName, tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("CompareValues(%s, %s, %s) = %d, expected %d", tt.typeName, tt.a, tt.b, result, tt.expected)
		}
	}
}
