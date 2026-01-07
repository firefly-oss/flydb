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
Package sdk provides core types and interfaces for FlyDB SDK and drivers.

This package defines the fundamental data types, cursor types, and result set
structures used by ODBC/JDBC drivers and language-specific SDKs.
*/
package sdk

import (
	"time"
)

// DataType represents a FlyDB data type.
type DataType int

const (
	// TypeNull represents a NULL value.
	TypeNull DataType = iota
	// TypeInt represents a 32-bit integer.
	TypeInt
	// TypeBigInt represents a 64-bit integer.
	TypeBigInt
	// TypeFloat represents a 64-bit floating point.
	TypeFloat
	// TypeDecimal represents an arbitrary precision decimal.
	TypeDecimal
	// TypeText represents variable-length text (unlimited).
	TypeText
	// TypeVarchar represents variable-length text with limit.
	TypeVarchar
	// TypeBoolean represents a boolean value.
	TypeBoolean
	// TypeTimestamp represents a timestamp with timezone.
	TypeTimestamp
	// TypeDate represents a date without time.
	TypeDate
	// TypeTime represents a time without date.
	TypeTime
	// TypeBlob represents binary large object.
	TypeBlob
	// TypeUUID represents a UUID.
	TypeUUID
	// TypeJSONB represents binary JSON.
	TypeJSONB
	// TypeSerial represents auto-incrementing integer.
	TypeSerial
)

// String returns the string representation of the data type.
func (dt DataType) String() string {
	switch dt {
	case TypeNull:
		return "NULL"
	case TypeInt:
		return "INT"
	case TypeBigInt:
		return "BIGINT"
	case TypeFloat:
		return "FLOAT"
	case TypeDecimal:
		return "DECIMAL"
	case TypeText:
		return "TEXT"
	case TypeVarchar:
		return "VARCHAR"
	case TypeBoolean:
		return "BOOLEAN"
	case TypeTimestamp:
		return "TIMESTAMP"
	case TypeDate:
		return "DATE"
	case TypeTime:
		return "TIME"
	case TypeBlob:
		return "BLOB"
	case TypeUUID:
		return "UUID"
	case TypeJSONB:
		return "JSONB"
	case TypeSerial:
		return "SERIAL"
	default:
		return "UNKNOWN"
	}
}

// ODBCType returns the ODBC SQL type code for this data type.
func (dt DataType) ODBCType() int {
	switch dt {
	case TypeNull:
		return 0 // SQL_UNKNOWN_TYPE
	case TypeInt:
		return 4 // SQL_INTEGER
	case TypeBigInt:
		return -5 // SQL_BIGINT
	case TypeFloat:
		return 8 // SQL_DOUBLE
	case TypeDecimal:
		return 3 // SQL_DECIMAL
	case TypeText, TypeVarchar:
		return 12 // SQL_VARCHAR
	case TypeBoolean:
		return -7 // SQL_BIT
	case TypeTimestamp:
		return 11 // SQL_TIMESTAMP
	case TypeDate:
		return 9 // SQL_DATE
	case TypeTime:
		return 10 // SQL_TIME
	case TypeBlob:
		return -4 // SQL_LONGVARBINARY
	case TypeUUID:
		return -11 // SQL_GUID
	case TypeJSONB:
		return -1 // SQL_LONGVARCHAR
	case TypeSerial:
		return -5 // SQL_BIGINT
	default:
		return 0
	}
}

// JDBCType returns the JDBC Types constant for this data type.
func (dt DataType) JDBCType() int {
	switch dt {
	case TypeNull:
		return 0 // Types.NULL
	case TypeInt:
		return 4 // Types.INTEGER
	case TypeBigInt, TypeSerial:
		return -5 // Types.BIGINT
	case TypeFloat:
		return 8 // Types.DOUBLE
	case TypeDecimal:
		return 3 // Types.DECIMAL
	case TypeText, TypeVarchar:
		return 12 // Types.VARCHAR
	case TypeBoolean:
		return 16 // Types.BOOLEAN
	case TypeTimestamp:
		return 93 // Types.TIMESTAMP
	case TypeDate:
		return 91 // Types.DATE
	case TypeTime:
		return 92 // Types.TIME
	case TypeBlob:
		return 2004 // Types.BLOB
	case TypeUUID:
		return 12 // Types.VARCHAR (UUID as string)
	case TypeJSONB:
		return 2005 // Types.CLOB
	default:
		return 0
	}
}

// TypeInfo provides metadata about a data type.
type TypeInfo struct {
	Type        DataType
	Name        string
	Precision   int  // Max precision for numeric types
	Scale       int  // Max scale for decimal types
	MaxLength   int  // Max length for string/binary types
	Nullable    bool // Whether NULL is allowed
	CaseSensitive bool
	Searchable  int  // 0=none, 1=like only, 2=all except like, 3=all
	Unsigned    bool
	FixedPrecScale bool
	AutoIncrement bool
}

// DefaultTypeInfo returns default TypeInfo for a data type.
func DefaultTypeInfo(dt DataType) TypeInfo {
	info := TypeInfo{
		Type:       dt,
		Name:       dt.String(),
		Nullable:   true,
		Searchable: 3,
	}
	switch dt {
	case TypeInt:
		info.Precision = 10
	case TypeBigInt, TypeSerial:
		info.Precision = 19
		if dt == TypeSerial {
			info.AutoIncrement = true
		}
	case TypeFloat:
		info.Precision = 15
	case TypeDecimal:
		info.Precision = 38
		info.Scale = 16
		info.FixedPrecScale = true
	case TypeVarchar:
		info.MaxLength = 65535
		info.CaseSensitive = true
	case TypeText:
		info.MaxLength = -1 // Unlimited
		info.CaseSensitive = true
	case TypeBoolean:
		info.Precision = 1
	case TypeTimestamp:
		info.Precision = 29 // YYYY-MM-DD HH:MM:SS.FFFFFFFFF+TZ
	case TypeDate:
		info.Precision = 10 // YYYY-MM-DD
	case TypeTime:
		info.Precision = 15 // HH:MM:SS.FFFFFF
	case TypeBlob:
		info.MaxLength = -1
		info.Searchable = 0
	case TypeUUID:
		info.MaxLength = 36
	case TypeJSONB:
		info.MaxLength = -1
		info.CaseSensitive = true
	}
	return info
}

// Value represents a typed database value.
type Value struct {
	Type    DataType
	IsNull  bool
	Data    interface{}
}

// NewNullValue creates a NULL value.
func NewNullValue() Value {
	return Value{Type: TypeNull, IsNull: true}
}

// NewIntValue creates an integer value.
func NewIntValue(v int32) Value {
	return Value{Type: TypeInt, Data: v}
}

// NewBigIntValue creates a bigint value.
func NewBigIntValue(v int64) Value {
	return Value{Type: TypeBigInt, Data: v}
}

// NewFloatValue creates a float value.
func NewFloatValue(v float64) Value {
	return Value{Type: TypeFloat, Data: v}
}

// NewStringValue creates a string value.
func NewStringValue(v string) Value {
	return Value{Type: TypeText, Data: v}
}

// NewBoolValue creates a boolean value.
func NewBoolValue(v bool) Value {
	return Value{Type: TypeBoolean, Data: v}
}

// NewTimestampValue creates a timestamp value.
func NewTimestampValue(v time.Time) Value {
	return Value{Type: TypeTimestamp, Data: v}
}

// NewBlobValue creates a blob value.
func NewBlobValue(v []byte) Value {
	return Value{Type: TypeBlob, Data: v}
}

// AsInt64 returns the value as int64.
func (v Value) AsInt64() (int64, bool) {
	if v.IsNull {
		return 0, false
	}
	switch val := v.Data.(type) {
	case int32:
		return int64(val), true
	case int64:
		return val, true
	case int:
		return int64(val), true
	default:
		return 0, false
	}
}

// AsFloat64 returns the value as float64.
func (v Value) AsFloat64() (float64, bool) {
	if v.IsNull {
		return 0, false
	}
	switch val := v.Data.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	default:
		return 0, false
	}
}

// AsString returns the value as string.
func (v Value) AsString() (string, bool) {
	if v.IsNull {
		return "", false
	}
	if s, ok := v.Data.(string); ok {
		return s, true
	}
	return "", false
}

// AsBool returns the value as bool.
func (v Value) AsBool() (bool, bool) {
	if v.IsNull {
		return false, false
	}
	if b, ok := v.Data.(bool); ok {
		return b, true
	}
	return false, false
}

// AsTime returns the value as time.Time.
func (v Value) AsTime() (time.Time, bool) {
	if v.IsNull {
		return time.Time{}, false
	}
	if t, ok := v.Data.(time.Time); ok {
		return t, true
	}
	return time.Time{}, false
}

// AsBytes returns the value as []byte.
func (v Value) AsBytes() ([]byte, bool) {
	if v.IsNull {
		return nil, false
	}
	if b, ok := v.Data.([]byte); ok {
		return b, true
	}
	return nil, false
}

