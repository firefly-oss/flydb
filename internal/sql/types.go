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

Type Validation:
================

Each type has a validation function that checks if a string value
can be converted to that type. This is used during INSERT and UPDATE
operations to ensure data integrity.
*/
package sql

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ColumnType represents the supported column types in FlyDB.
type ColumnType string

// Column type constants.
const (
	TypeINT       ColumnType = "INT"
	TypeBIGINT    ColumnType = "BIGINT"
	TypeTEXT      ColumnType = "TEXT"
	TypeVARCHAR   ColumnType = "VARCHAR"
	TypeBOOLEAN   ColumnType = "BOOLEAN"
	TypeFLOAT     ColumnType = "FLOAT"
	TypeDECIMAL   ColumnType = "DECIMAL"
	TypeTIMESTAMP ColumnType = "TIMESTAMP"
	TypeDATE      ColumnType = "DATE"
	TypeTIME      ColumnType = "TIME"
	TypeBLOB      ColumnType = "BLOB"
	TypeUUID      ColumnType = "UUID"
	TypeJSONB     ColumnType = "JSONB"
	TypeSERIAL    ColumnType = "SERIAL"
)

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
	"BIGINT":    TypeBIGINT,
	"TEXT":      TypeTEXT,
	"VARCHAR":   TypeVARCHAR,
	"BOOLEAN":   TypeBOOLEAN,
	"FLOAT":     TypeFLOAT,
	"DECIMAL":   TypeDECIMAL,
	"NUMERIC":   TypeDECIMAL, // Alias for DECIMAL
	"TIMESTAMP": TypeTIMESTAMP,
	"DATE":      TypeDATE,
	"TIME":      TypeTIME,
	"BLOB":      TypeBLOB,
	"UUID":      TypeUUID,
	"JSONB":     TypeJSONB,
	"SERIAL":    TypeSERIAL,
}

// IsValidType checks if a type name is a valid column type.
func IsValidType(typeName string) bool {
	_, ok := ValidColumnTypes[strings.ToUpper(typeName)]
	return ok
}

// ValidateValue checks if a value is valid for the given column type.
// Returns an error if the value cannot be converted to the type.
func ValidateValue(typeName string, value string) error {
	colType := ColumnType(strings.ToUpper(typeName))
	// Handle NUMERIC as alias for DECIMAL
	if colType == "NUMERIC" {
		colType = TypeDECIMAL
	}

	switch colType {
	case TypeINT:
		_, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid INT value: %s", value)
		}

	case TypeBIGINT:
		_, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid BIGINT value: %s", value)
		}

	case TypeSERIAL:
		// SERIAL is auto-incrementing, but if a value is provided it must be a positive integer
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil || v < 0 {
			return fmt.Errorf("invalid SERIAL value: %s (must be a positive integer)", value)
		}

	case TypeFLOAT:
		_, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid FLOAT value: %s", value)
		}

	case TypeDECIMAL:
		if !decimalRegex.MatchString(value) {
			return fmt.Errorf("invalid DECIMAL value: %s", value)
		}

	case TypeBOOLEAN:
		upper := strings.ToUpper(value)
		if upper != "TRUE" && upper != "FALSE" && upper != "1" && upper != "0" {
			return fmt.Errorf("invalid BOOLEAN value: %s (use TRUE/FALSE)", value)
		}

	case TypeTIMESTAMP:
		_, err := time.Parse(time.RFC3339, value)
		if err != nil {
			// Also try common formats
			_, err = time.Parse("2006-01-02 15:04:05", value)
			if err != nil {
				return fmt.Errorf("invalid TIMESTAMP value: %s (use RFC3339 or YYYY-MM-DD HH:MM:SS)", value)
			}
		}

	case TypeDATE:
		if !dateRegex.MatchString(value) {
			return fmt.Errorf("invalid DATE value: %s (use YYYY-MM-DD)", value)
		}
		_, err := time.Parse("2006-01-02", value)
		if err != nil {
			return fmt.Errorf("invalid DATE value: %s", value)
		}

	case TypeTIME:
		if !timeRegex.MatchString(value) {
			return fmt.Errorf("invalid TIME value: %s (use HH:MM:SS)", value)
		}
		_, err := time.Parse("15:04:05", value)
		if err != nil {
			return fmt.Errorf("invalid TIME value: %s", value)
		}

	case TypeUUID:
		if !uuidRegex.MatchString(value) {
			return fmt.Errorf("invalid UUID value: %s", value)
		}

	case TypeBLOB:
		// BLOB values should be base64 encoded
		_, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return fmt.Errorf("invalid BLOB value: must be base64 encoded")
		}

	case TypeJSONB:
		if !json.Valid([]byte(value)) {
			return fmt.Errorf("invalid JSONB value: not valid JSON")
		}

	case TypeTEXT, TypeVARCHAR:
		// TEXT and VARCHAR accept any string value
		return nil

	default:
		return fmt.Errorf("unknown column type: %s", typeName)
	}

	return nil
}

// NormalizeValue converts a value to its canonical form for the given type.
// This ensures consistent storage and comparison.
func NormalizeValue(typeName string, value string) (string, error) {
	colType := ColumnType(strings.ToUpper(typeName))
	// Handle NUMERIC as alias for DECIMAL
	if colType == "NUMERIC" {
		colType = TypeDECIMAL
	}

	switch colType {
	case TypeBOOLEAN:
		upper := strings.ToUpper(value)
		if upper == "TRUE" || upper == "1" {
			return "true", nil
		}
		if upper == "FALSE" || upper == "0" {
			return "false", nil
		}
		return "", fmt.Errorf("invalid BOOLEAN value: %s", value)

	case TypeTIMESTAMP:
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05", value)
			if err != nil {
				return "", fmt.Errorf("invalid TIMESTAMP value: %s", value)
			}
		}
		return t.Format(time.RFC3339), nil

	case TypeDATE:
		t, err := time.Parse("2006-01-02", value)
		if err != nil {
			return "", fmt.Errorf("invalid DATE value: %s", value)
		}
		return t.Format("2006-01-02"), nil

	case TypeTIME:
		t, err := time.Parse("15:04:05", value)
		if err != nil {
			return "", fmt.Errorf("invalid TIME value: %s", value)
		}
		return t.Format("15:04:05"), nil

	case TypeUUID:
		// Normalize to lowercase
		return strings.ToLower(value), nil

	case TypeJSONB:
		// Compact the JSON
		var v interface{}
		if err := json.Unmarshal([]byte(value), &v); err != nil {
			return "", fmt.Errorf("invalid JSONB value: %s", value)
		}
		compact, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(compact), nil

	default:
		// INT, BIGINT, SERIAL, FLOAT, DECIMAL, TEXT, VARCHAR, BLOB - return as-is
		return value, nil
	}
}

// CompareValues compares two values of the given type.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func CompareValues(typeName string, a, b string) int {
	colType := ColumnType(strings.ToUpper(typeName))
	// Handle NUMERIC as alias for DECIMAL
	if colType == "NUMERIC" {
		colType = TypeDECIMAL
	}

	switch colType {
	case TypeINT, TypeBIGINT, TypeSERIAL:
		ai, _ := strconv.ParseInt(a, 10, 64)
		bi, _ := strconv.ParseInt(b, 10, 64)
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
		return 0

	case TypeFLOAT, TypeDECIMAL:
		af, _ := strconv.ParseFloat(a, 64)
		bf, _ := strconv.ParseFloat(b, 64)
		if af < bf {
			return -1
		}
		if af > bf {
			return 1
		}
		return 0

	case TypeTIMESTAMP, TypeDATE, TypeTIME:
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

	default:
		// String comparison for TEXT, VARCHAR, UUID, BLOB, JSONB
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}
}

