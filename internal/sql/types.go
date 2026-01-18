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

/*
Package sql contains type definitions and validation for FlyDB column types.

Supported Column Types:
=======================

  - INT: Integer values (64-bit signed)
  - TEXT: Variable-length string values
  - BOOLEAN: True/false values
  - FLOAT: 64-bit floating-point numbers
  - TIMESTAMP: Date and time with timezone (RFC3339 format)
  - DATE: Date only (YYYY-MM-DD format)
  - BLOB: Binary data (base64 encoded in storage)
  - UUID: Universally unique identifier (RFC 4122 format)
  - JSONB: Binary JSON for structured data
  - INET: IPv4/IPv6 address or CIDR
  - SET: Unordered collection of unique elements
  - ZSET: Sorted set with scores

Type Validation:
================

Each type has a validation function that checks if a string value
can be converted to that type. This is used during INSERT and UPDATE
operations to ensure data integrity.
*/
package sql

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	ferrors "flydb/internal/errors"
)

// ColumnType represents the supported column types in FlyDB.
type ColumnType string

// Column type constants.
const (
	TypeINT       ColumnType = "INT"
	TypeBIGINT    ColumnType = "BIGINT"
	TypeSMALLINT  ColumnType = "SMALLINT"
	TypeTEXT      ColumnType = "TEXT"
	TypeVARCHAR   ColumnType = "VARCHAR"
	TypeCHAR      ColumnType = "CHAR"
	TypeBOOLEAN   ColumnType = "BOOLEAN"
	TypeFLOAT     ColumnType = "FLOAT"
	TypeDOUBLE    ColumnType = "DOUBLE"
	TypeREAL      ColumnType = "REAL"
	TypeDECIMAL   ColumnType = "DECIMAL"
	TypeTIMESTAMP ColumnType = "TIMESTAMP"
	TypeDATETIME  ColumnType = "DATETIME"
	TypeDATE      ColumnType = "DATE"
	TypeTIME      ColumnType = "TIME"
	TypeBLOB      ColumnType = "BLOB"
	TypeBYTEA     ColumnType = "BYTEA"
	TypeBINARY    ColumnType = "BINARY"
	TypeVARBINARY ColumnType = "VARBINARY"
	TypeUUID      ColumnType = "UUID"
	TypeJSONB     ColumnType = "JSONB"
	TypeSERIAL    ColumnType = "SERIAL"
	TypeMONEY     ColumnType = "MONEY"
	TypeINTERVAL  ColumnType = "INTERVAL"
	TypeCLOB      ColumnType = "CLOB"
	TypeNCHAR     ColumnType = "NCHAR"
	TypeNVARCHAR  ColumnType = "NVARCHAR"
	TypeNTEXT     ColumnType = "NTEXT"
	TypeINET      ColumnType = "INET"
	TypeSET       ColumnType = "SET"
	TypeZSET      ColumnType = "ZSET"
)

// zsetMember represents a member in a sorted set (ZSET).
type zsetMember struct {
	Score  float64     `json:"score"`
	Member interface{} `json:"member"`
}

// uuidRegex matches valid UUID format (RFC 4122).
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// dateRegex matches YYYY-MM-DD format.
var dateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// timeRegex matches HH:MM:SS format.
var timeRegex = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}$`)

// decimalRegex matches decimal numbers with optional precision.
var decimalRegex = regexp.MustCompile(`^-?\d+(\.\d+)?$`)

// ValidColumnTypes is the set of all valid column type names.
var ValidColumnTypes = map[string]ColumnType{
	"INT":       TypeINT,
	"INTEGER":   TypeINT, // Alias for INT
	"BIGINT":    TypeBIGINT,
	"SMALLINT":  TypeSMALLINT,
	"TINYINT":   TypeSMALLINT, // Alias for SMALLINT
	"TEXT":      TypeTEXT,
	"VARCHAR":   TypeVARCHAR,
	"CHAR":      TypeCHAR,
	"CHARACTER": TypeCHAR, // Alias for CHAR
	"BOOLEAN":   TypeBOOLEAN,
	"BOOL":      TypeBOOLEAN, // Alias for BOOLEAN
	"FLOAT":     TypeFLOAT,
	"DOUBLE":    TypeDOUBLE,
	"REAL":      TypeREAL,
	"DECIMAL":   TypeDECIMAL,
	"NUMERIC":   TypeDECIMAL, // Alias for DECIMAL
	"TIMESTAMP": TypeTIMESTAMP,
	"DATETIME":  TypeDATETIME,
	"DATE":      TypeDATE,
	"TIME":      TypeTIME,
	"BLOB":      TypeBLOB,
	"BYTEA":     TypeBYTEA,
	"BINARY":    TypeBINARY,
	"VARBINARY": TypeVARBINARY,
	"UUID":      TypeUUID,
	"JSONB":     TypeJSONB,
	"JSON":      TypeJSONB, // Alias for JSONB
	"SERIAL":    TypeSERIAL,
	"MONEY":     TypeMONEY,
	"INTERVAL":  TypeINTERVAL,
	"CLOB":      TypeCLOB,
	"NCHAR":     TypeNCHAR,
	"NVARCHAR":  TypeNVARCHAR,
	"NTEXT":     TypeNTEXT,
	"INET":      TypeINET,
	"SET":       TypeSET,
	"ZSET":      TypeZSET,
	"SORTEDSET": TypeZSET, // Alias for ZSET
}

// IsValidType checks if a type name is a valid column type.
func IsValidType(typeName string) bool {
	_, ok := ValidColumnTypes[strings.ToUpper(typeName)]
	return ok
}

// ValidateValue checks if a value is valid for the given column type.
// Returns an error if the value cannot be converted to the type.
func ValidateValue(typeName string, value string) error {
	// Normalize type name using the alias map
	upperType := strings.ToUpper(typeName)
	if canonical, ok := ValidColumnTypes[upperType]; ok {
		upperType = string(canonical)
	}
	colType := ColumnType(upperType)

	switch colType {
	case TypeINT:
		_, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return ferrors.TypeMismatch("INT", value, "")
		}

	case TypeSMALLINT:
		_, err := strconv.ParseInt(value, 10, 16)
		if err != nil {
			return ferrors.TypeMismatch("SMALLINT", value, "")
		}

	case TypeBIGINT:
		_, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return ferrors.TypeMismatch("BIGINT", value, "")
		}

	case TypeSERIAL:
		// SERIAL is auto-incrementing, but if a value is provided it must be a positive integer
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil || v < 0 {
			return ferrors.TypeMismatch("SERIAL", value, "").WithDetail("must be a positive integer")
		}

	case TypeFLOAT, TypeDOUBLE, TypeREAL:
		_, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return ferrors.TypeMismatch(string(colType), value, "")
		}

	case TypeDECIMAL:
		if !decimalRegex.MatchString(value) {
			return ferrors.TypeMismatch("DECIMAL", value, "")
		}

	case TypeBOOLEAN:
		upper := strings.ToUpper(value)
		if upper != "TRUE" && upper != "FALSE" && upper != "1" && upper != "0" {
			return ferrors.TypeMismatch("BOOLEAN", value, "").WithDetail("use TRUE/FALSE")
		}

	case TypeTIMESTAMP, TypeDATETIME:
		_, err := time.Parse(time.RFC3339, value)
		if err != nil {
			// Also try common formats
			_, err = time.Parse("2006-01-02 15:04:05", value)
			if err != nil {
				return ferrors.TypeMismatch("TIMESTAMP", value, "").WithDetail("use RFC3339 or YYYY-MM-DD HH:MM:SS")
			}
		}

	case TypeDATE:
		if !dateRegex.MatchString(value) {
			return ferrors.TypeMismatch("DATE", value, "").WithDetail("use YYYY-MM-DD")
		}
		_, err := time.Parse("2006-01-02", value)
		if err != nil {
			return ferrors.TypeMismatch("DATE", value, "")
		}

	case TypeTIME:
		if !timeRegex.MatchString(value) {
			return ferrors.TypeMismatch("TIME", value, "").WithDetail("use HH:MM:SS")
		}
		_, err := time.Parse("15:04:05", value)
		if err != nil {
			return ferrors.TypeMismatch("TIME", value, "")
		}

	case TypeUUID:
		if !uuidRegex.MatchString(value) {
			return ferrors.TypeMismatch("UUID", value, "")
		}

	case TypeBLOB, TypeBYTEA, TypeBINARY, TypeVARBINARY:
		// BLOB/BYTEA/BINARY values should be base64 encoded
		_, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return ferrors.TypeMismatch(string(colType), value, "").WithDetail("must be base64 encoded")
		}

	case TypeJSONB:
		if !json.Valid([]byte(value)) {
			return ferrors.TypeMismatch("JSONB", value, "").WithDetail("not valid JSON")
		}

	case TypeTEXT, TypeVARCHAR, TypeCHAR, TypeCLOB, TypeNCHAR, TypeNVARCHAR, TypeNTEXT:
		// TEXT, VARCHAR, CHAR, CLOB, and Unicode variants accept any string value
		return nil

	case TypeMONEY:
		// MONEY accepts decimal values (may have currency symbol prefix)
		cleanValue := strings.TrimPrefix(value, "$")
		cleanValue = strings.TrimPrefix(cleanValue, "€")
		cleanValue = strings.TrimPrefix(cleanValue, "£")
		cleanValue = strings.ReplaceAll(cleanValue, ",", "")
		if !decimalRegex.MatchString(cleanValue) {
			return ferrors.TypeMismatch("MONEY", value, "")
		}

	case TypeINET:
		// Accept IPv4, IPv6, or CIDR
		if ip := net.ParseIP(value); ip != nil {
			return nil
		}
		if _, _, err := net.ParseCIDR(value); err == nil {
			return nil
		}
		return ferrors.TypeMismatch("INET", value, "").WithDetail("invalid IP address or CIDR")

	case TypeINTERVAL:
		// Enhanced interval validation
		// Supports:
		// 1. Postgres style: "1 year 2 months 3 days"
		// 2. ISO-8601 duration: "P1Y2M3D"
		// 3. Simple time units: "300ms", "1.5h" (Go time.Duration style)

		// First try standard Go duration parse for simple cases
		if _, err := time.ParseDuration(value); err == nil {
			return nil
		}

		// Check for ISO-8601 P-prefix
		if strings.HasPrefix(strings.ToUpper(value), "P") {
			// Basic ISO validation - must contain at least one designator
			validRunes := "P0123456789YMWDTHS.,"
			for _, r := range strings.ToUpper(value) {
				if !strings.ContainsRune(validRunes, r) {
					return ferrors.TypeMismatch("INTERVAL", value, "").WithDetail("invalid ISO-8601 duration characters")
				}
			}
			return nil
		}

		// Check for Postgres style (quantity unit pairs)
		// regex to match optional sign, number, optional whitespace, unit
		// This is a simple heuristic check
		postgresIntervalRegex := regexp.MustCompile(`^[-+]?\d+(\.\d+)?\s*[a-zA-Z]+`)

		// If it's a simple number (seconds), it's also valid (though usually treated as seconds in Postgres if implied)
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return nil
		}

		if postgresIntervalRegex.MatchString(value) {
			return nil
		}

		// Check if it starts with a number at least
		if len(value) > 0 {
			r := rune(value[0])
			if r >= '0' && r <= '9' {
				// Likely a compact format like "2mons" that regex didn't catch or nuanced format
				// "2mons" matches the regex above actually.
				// Let's rely on the regex.
				return nil
			}
		}

		return ferrors.TypeMismatch("INTERVAL", value, "").WithDetail("must start with quantity or be ISO-8601")

	case TypeSET:
		// Must be a valid JSON array
		var arr []interface{}
		if err := json.Unmarshal([]byte(value), &arr); err != nil {
			return ferrors.TypeMismatch("SET", value, "").WithDetail("must be a JSON array")
		}
		return nil

	case TypeZSET:
		// Must be a valid JSON array of zsetMember objects
		var members []zsetMember
		if err := json.Unmarshal([]byte(value), &members); err != nil {
			return ferrors.TypeMismatch("ZSET", value, "").WithDetail("must be a JSON array of objects with 'score' and 'member' fields")
		}
		return nil

	default:
		return ferrors.NewExecutionError("unknown column type: " + typeName)
	}

	return nil
}

// NormalizeValue converts a value to its canonical form for the given type.
// This ensures consistent storage and comparison.
func NormalizeValue(typeName string, value string) (string, error) {
	// Normalize type name using the alias map
	upperType := strings.ToUpper(typeName)
	if canonical, ok := ValidColumnTypes[upperType]; ok {
		upperType = string(canonical)
	}
	colType := ColumnType(upperType)

	switch colType {
	case TypeBOOLEAN:
		upper := strings.ToUpper(value)
		if upper == "TRUE" || upper == "1" {
			return "true", nil
		}
		if upper == "FALSE" || upper == "0" {
			return "false", nil
		}
		return "", ferrors.TypeMismatch("BOOLEAN", value, "")

	case TypeTIMESTAMP, TypeDATETIME:
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05", value)
			if err != nil {
				return "", ferrors.TypeMismatch("TIMESTAMP", value, "")
			}
		}
		return t.Format(time.RFC3339), nil

	case TypeDATE:
		t, err := time.Parse("2006-01-02", value)
		if err != nil {
			return "", ferrors.TypeMismatch("DATE", value, "")
		}
		return t.Format("2006-01-02"), nil

	case TypeTIME:
		t, err := time.Parse("15:04:05", value)
		if err != nil {
			return "", ferrors.TypeMismatch("TIME", value, "")
		}
		return t.Format("15:04:05"), nil

	case TypeUUID:
		// Normalize to lowercase
		return strings.ToLower(value), nil

	case TypeJSONB:
		// Compact the JSON
		var v interface{}
		if err := json.Unmarshal([]byte(value), &v); err != nil {
			return "", ferrors.TypeMismatch("JSONB", value, "")
		}
		compact, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(compact), nil

	case TypeINET:
		// Normalize IP/CIDR representation
		// 1. Try ParseCIDR first
		if _, ipNet, err := net.ParseCIDR(value); err == nil {
			return ipNet.String(), nil
		}
		// 2. Try ParseIP
		if ip := net.ParseIP(value); ip != nil {
			return ip.String(), nil
		}
		return "", ferrors.TypeMismatch("INET", value, "")

	case TypeINTERVAL:
		// For Go durations, normalize to standardized string
		if d, err := time.ParseDuration(value); err == nil {
			return d.String(), nil
		}
		// Other formats returned as-is
		return value, nil

	case TypeSET:
		// Normalization for SET:
		// 1. Unmarshal array
		// 2. Deduplicate elements
		// 3. Sort elements for canonical representation
		var arr []interface{}
		if err := json.Unmarshal([]byte(value), &arr); err != nil {
			return "", ferrors.TypeMismatch("SET", value, "")
		}

		uniqueMap := make(map[string]interface{})
		var keys []string

		for _, v := range arr {
			// Serialize each element to string for robust keying/sorting
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			key := string(b)
			if _, exists := uniqueMap[key]; !exists {
				uniqueMap[key] = v
				keys = append(keys, key)
			}
		}

		// Sort keys to ensure deterministic order
		sort.Strings(keys)

		// Rebuild array
		resArr := make([]interface{}, 0, len(keys))
		for _, k := range keys {
			resArr = append(resArr, uniqueMap[k])
		}

		b, err := json.Marshal(resArr)
		if err != nil {
			return "", err
		}
		return string(b), nil

	case TypeZSET:
		// Normalization for ZSET:
		// 1. Unmarshal to []zsetMember
		// 2. Sort by Score (asc), then by Member string rep (asc)
		var members []zsetMember
		if err := json.Unmarshal([]byte(value), &members); err != nil {
			return "", ferrors.TypeMismatch("ZSET", value, "")
		}

		sort.Slice(members, func(i, j int) bool {
			if members[i].Score != members[j].Score {
				return members[i].Score < members[j].Score
			}
			// Same score, tie-break by member string representation
			mi, _ := json.Marshal(members[i].Member)
			mj, _ := json.Marshal(members[j].Member)
			return string(mi) < string(mj)
		})

		b, err := json.Marshal(members)
		if err != nil {
			return "", err
		}
		return string(b), nil

	default:
		// INT, SMALLINT, BIGINT, SERIAL, FLOAT, DOUBLE, REAL, DECIMAL, TEXT, VARCHAR, CHAR, BLOB, BYTEA, BINARY, VARBINARY - return as-is
		return value, nil
	}
}

// CompareValues compares two values of the given type.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func CompareValues(typeName string, a, b string) int {
	// Normalize type name using the alias map
	upperType := strings.ToUpper(typeName)
	if canonical, ok := ValidColumnTypes[upperType]; ok {
		upperType = string(canonical)
	}
	colType := ColumnType(upperType)

	switch colType {
	case TypeINT, TypeSMALLINT, TypeBIGINT, TypeSERIAL:
		ai, _ := strconv.ParseInt(a, 10, 64)
		bi, _ := strconv.ParseInt(b, 10, 64)
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
		return 0

	case TypeFLOAT, TypeDOUBLE, TypeREAL, TypeDECIMAL:
		af, _ := strconv.ParseFloat(a, 64)
		bf, _ := strconv.ParseFloat(b, 64)
		if af < bf {
			return -1
		}
		if af > bf {
			return 1
		}
		return 0

	case TypeTIMESTAMP, TypeDATETIME, TypeDATE, TypeTIME:
		// Lexicographic comparison works for ISO format dates and times
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0

	case TypeBOOLEAN:
		// false < true
		if a == b {
			return 0
		}
		if a == "false" {
			return -1
		}
		return 1

	case TypeINET:
		// Compare IPs byte-wise
		// Try CIDR
		_, ipNetA, errA := net.ParseCIDR(a)
		_, ipNetB, errB := net.ParseCIDR(b)

		if errA == nil && errB == nil {
			// Compare CIDRs
			// Ideally compare IP then mask size
			ipA := ipNetA.IP
			ipB := ipNetB.IP
			cmp := bytes.Compare(ipA, ipB)
			if cmp != 0 {
				return cmp
			}
			// Same IP, compare mask sizes (longer mask = "larger/more specific" -> arbitrary choice but consistent)
			// Actually, string compare is probably fine for normalized CIDRs of same IP version
			if a < b {
				return -1
			}
			if a > b {
				return 1
			}
			return 0
		}

		// Try IP
		ipA := net.ParseIP(a)
		ipB := net.ParseIP(b)
		if ipA != nil && ipB != nil {
			// Ensure both are 16-byte representation for comparison
			ipA16 := ipA.To16()
			ipB16 := ipB.To16()
			return bytes.Compare(ipA16, ipB16)
		}

		// Fallback to string compare
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0

	case TypeSET, TypeZSET:
		// Since SET and ZSET are normalized to canonical sorted JSON strings,
		// we can simpler compare their string representations.
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0

	default:
		// String comparison for TEXT, VARCHAR, CHAR, UUID, BLOB, BYTEA, BINARY, VARBINARY, JSONB
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}
}
