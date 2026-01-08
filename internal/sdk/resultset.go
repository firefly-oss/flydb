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
Result Set and Metadata Implementation
=======================================

This file defines the structures for representing query results and their
metadata. These structures are used by ODBC/JDBC drivers and language SDKs
to provide rich information about query results.

Result Set Metadata:
====================

Result set metadata describes the structure of query results:

  - Column count and names
  - Data types and sizes
  - Nullability and constraints
  - Source table information

This metadata is essential for:
  - Building dynamic UIs that adapt to query results
  - Type-safe data binding in strongly-typed languages
  - Generating reports with proper formatting

Column Information:
===================

Each column in a result set has rich metadata:

  Position:
    - Index: 0-based column position

  Names:
    - Name: Actual column name from schema
    - Label: Display label (may differ for aliases)
    - TableName: Source table
    - SchemaName: Source schema
    - CatalogName: Source catalog/database

  Type Information:
    - Type: FlyDB data type enum
    - TypeName: Database type string (e.g., "VARCHAR(255)")
    - Precision: Numeric precision or string max length
    - Scale: Decimal scale
    - DisplaySize: Suggested display width

  Constraints:
    - Nullable: Whether NULL values are allowed
    - AutoIncrement: Whether column auto-increments
    - ReadOnly: Whether column is read-only
    - Writable: Whether column is writable
    - Searchable: Whether column can be used in WHERE
    - CaseSensitive: Whether comparisons are case-sensitive
    - Signed: Whether numeric type is signed
    - Currency: Whether type represents currency

ODBC/JDBC Compatibility:
========================

The metadata structures are designed to be compatible with:

  - ODBC SQLDescribeCol and SQLColAttribute
  - JDBC ResultSetMetaData interface

This enables FlyDB drivers to provide full metadata support.

Thread Safety:
==============

ResultSetMetadata uses a read-write mutex for thread-safe access.
Multiple goroutines can read metadata concurrently.

References:
===========

  - ODBC SQLDescribeCol: https://docs.microsoft.com/en-us/sql/odbc/reference/syntax/sqldescribecol-function
  - JDBC ResultSetMetaData: https://docs.oracle.com/javase/8/docs/api/java/sql/ResultSetMetaData.html
*/
package sdk

import (
	"sync"
)

// ColumnInfo provides metadata about a result set column.
type ColumnInfo struct {
	// Position
	Index int // 0-based column index

	// Names
	Name       string // Column name
	Label      string // Display label (may differ from name for aliases)
	TableName  string // Source table name
	SchemaName string // Source schema name
	CatalogName string // Source catalog name

	// Type information
	Type        DataType
	TypeName    string // Database type name (e.g., "VARCHAR(255)")
	Precision   int    // Numeric precision or string max length
	Scale       int    // Decimal scale
	DisplaySize int    // Suggested display width

	// Constraints
	Nullable      bool // Whether NULL values are allowed
	AutoIncrement bool // Whether column auto-increments
	ReadOnly      bool // Whether column is read-only
	Writable      bool // Whether column is writable
	Searchable    bool // Whether column can be used in WHERE
	CaseSensitive bool // Whether comparisons are case-sensitive
	Signed        bool // Whether numeric type is signed
	Currency      bool // Whether type represents currency
}

// ResultSetMetadata provides metadata about a result set.
type ResultSetMetadata struct {
	mu sync.RWMutex

	// Column information
	Columns     []ColumnInfo
	ColumnCount int

	// Table information
	TableName   string
	SchemaName  string
	CatalogName string

	// Keys
	PrimaryKeys []string
}

// NewResultSetMetadata creates a new ResultSetMetadata.
func NewResultSetMetadata(columns []ColumnInfo) *ResultSetMetadata {
	return &ResultSetMetadata{
		Columns:     columns,
		ColumnCount: len(columns),
	}
}

// GetColumnInfo returns column info by index (0-based).
func (m *ResultSetMetadata) GetColumnInfo(index int) (*ColumnInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if index < 0 || index >= len(m.Columns) {
		return nil, false
	}
	return &m.Columns[index], true
}

// GetColumnByName returns column info by name.
func (m *ResultSetMetadata) GetColumnByName(name string) (*ColumnInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i := range m.Columns {
		if m.Columns[i].Name == name {
			return &m.Columns[i], true
		}
	}
	return nil, false
}

// GetColumnIndex returns the index of a column by name (-1 if not found).
func (m *ResultSetMetadata) GetColumnIndex(name string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i := range m.Columns {
		if m.Columns[i].Name == name {
			return i
		}
	}
	return -1
}

// Row represents a single row in a result set.
type Row struct {
	Values []Value
}

// NewRow creates a new row with the given values.
func NewRow(values []Value) *Row {
	return &Row{Values: values}
}

// GetValue returns the value at the given index (0-based).
func (r *Row) GetValue(index int) (Value, bool) {
	if index < 0 || index >= len(r.Values) {
		return Value{}, false
	}
	return r.Values[index], true
}

// IsNull returns true if the value at the given index is NULL.
func (r *Row) IsNull(index int) bool {
	if index < 0 || index >= len(r.Values) {
		return true
	}
	return r.Values[index].IsNull
}

// ColumnCount returns the number of columns in the row.
func (r *Row) ColumnCount() int {
	return len(r.Values)
}

// ResultSet represents a database result set.
type ResultSet struct {
	mu sync.RWMutex

	// Identification
	ID string

	// Metadata
	Metadata *ResultSetMetadata

	// Data
	Rows         []*Row
	currentIndex int // Current row index for iteration

	// Counts
	RowCount     int64 // Total rows in result set
	AffectedRows int64 // Rows affected by UPDATE/DELETE/INSERT

	// State
	HasMoreRows bool   // Whether more rows are available (for streaming)
	CursorID    string // Associated cursor ID (if any)
	Closed      bool
}

// NewResultSet creates a new result set.
func NewResultSet(id string, metadata *ResultSetMetadata) *ResultSet {
	return &ResultSet{
		ID:           id,
		Metadata:     metadata,
		Rows:         make([]*Row, 0),
		currentIndex: -1, // Before first row
	}
}

// AddRow adds a row to the result set.
func (rs *ResultSet) AddRow(row *Row) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.Rows = append(rs.Rows, row)
	rs.RowCount = int64(len(rs.Rows))
}

// Next moves to the next row. Returns false if no more rows.
func (rs *ResultSet) Next() bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.Closed {
		return false
	}
	rs.currentIndex++
	return rs.currentIndex < len(rs.Rows)
}

// GetRow returns the current row.
func (rs *ResultSet) GetRow() (*Row, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	if rs.currentIndex < 0 || rs.currentIndex >= len(rs.Rows) {
		return nil, false
	}
	return rs.Rows[rs.currentIndex], true
}

// GetRowAt returns the row at the specified index.
func (rs *ResultSet) GetRowAt(index int) (*Row, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	if index < 0 || index >= len(rs.Rows) {
		return nil, false
	}
	return rs.Rows[index], true
}

// First moves to the first row.
func (rs *ResultSet) First() bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if len(rs.Rows) == 0 {
		return false
	}
	rs.currentIndex = 0
	return true
}

// Last moves to the last row.
func (rs *ResultSet) Last() bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if len(rs.Rows) == 0 {
		return false
	}
	rs.currentIndex = len(rs.Rows) - 1
	return true
}

// Absolute moves to an absolute row position (1-based).
func (rs *ResultSet) Absolute(row int) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if row < 1 || row > len(rs.Rows) {
		return false
	}
	rs.currentIndex = row - 1
	return true
}

// Relative moves relative to the current position.
func (rs *ResultSet) Relative(offset int) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	newIndex := rs.currentIndex + offset
	if newIndex < 0 || newIndex >= len(rs.Rows) {
		return false
	}
	rs.currentIndex = newIndex
	return true
}

// BeforeFirst moves before the first row.
func (rs *ResultSet) BeforeFirst() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.currentIndex = -1
}

// AfterLast moves after the last row.
func (rs *ResultSet) AfterLast() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.currentIndex = len(rs.Rows)
}

// IsBeforeFirst returns true if cursor is before the first row.
func (rs *ResultSet) IsBeforeFirst() bool {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.currentIndex < 0 && len(rs.Rows) > 0
}

// IsAfterLast returns true if cursor is after the last row.
func (rs *ResultSet) IsAfterLast() bool {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.currentIndex >= len(rs.Rows) && len(rs.Rows) > 0
}

// IsFirst returns true if cursor is on the first row.
func (rs *ResultSet) IsFirst() bool {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.currentIndex == 0 && len(rs.Rows) > 0
}

// IsLast returns true if cursor is on the last row.
func (rs *ResultSet) IsLast() bool {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.currentIndex == len(rs.Rows)-1 && len(rs.Rows) > 0
}

// GetCurrentRow returns the current row number (1-based, 0 if not on a row).
func (rs *ResultSet) GetCurrentRow() int {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	if rs.currentIndex < 0 || rs.currentIndex >= len(rs.Rows) {
		return 0
	}
	return rs.currentIndex + 1
}

// Close closes the result set.
func (rs *ResultSet) Close() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.Closed = true
	return nil
}

// IsClosed returns true if the result set is closed.
func (rs *ResultSet) IsClosed() bool {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.Closed
}

