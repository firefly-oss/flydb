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
Metadata Provider Implementation
================================

This file implements the MetadataProvider interface for ODBC/JDBC driver support.
It provides database metadata such as tables, columns, primary keys, foreign keys,
and indexes.

The metadata provider uses the SQL catalog to retrieve schema information and
formats it in a way that's compatible with ODBC/JDBC metadata APIs.
*/
package server

import (
	"flydb/internal/sql"
	"flydb/internal/storage"
	"strings"
)

// serverMetadataProvider implements the protocol.MetadataProvider interface.
type serverMetadataProvider struct {
	srv *Server
}

// GetTables returns table metadata matching the specified pattern.
func (m *serverMetadataProvider) GetTables(catalog, schema, tablePattern string, tableTypes []string) ([][]interface{}, error) {
	executor := m.srv.executor
	if executor == nil {
		return nil, nil
	}

	// Get catalog from executor
	cat := executor.GetCatalog()
	if cat == nil {
		return nil, nil
	}

	var rows [][]interface{}
	for tableName, tableSchema := range cat.Tables {
		// Apply pattern matching
		if tablePattern != "" && tablePattern != "%" && !matchPattern(tableName, tablePattern) {
			continue
		}

		// Determine table type
		tableType := "TABLE"
		if _, isView := cat.GetView(tableName); isView {
			tableType = "VIEW"
		}

		// Filter by table types if specified
		if len(tableTypes) > 0 {
			found := false
			for _, t := range tableTypes {
				if strings.EqualFold(t, tableType) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		row := []interface{}{
			catalog,                // TABLE_CAT
			schema,                 // TABLE_SCHEM
			tableName,              // TABLE_NAME
			tableType,              // TABLE_TYPE
			tableSchema.Comment,    // REMARKS
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// GetColumns returns column metadata for the specified table.
func (m *serverMetadataProvider) GetColumns(catalog, schema, tablePattern, columnPattern string) ([][]interface{}, error) {
	executor := m.srv.executor
	if executor == nil {
		return nil, nil
	}

	cat := executor.GetCatalog()
	if cat == nil {
		return nil, nil
	}

	var rows [][]interface{}
	for tableName, tableSchema := range cat.Tables {
		// Apply table pattern matching
		if tablePattern != "" && tablePattern != "%" && !matchPattern(tableName, tablePattern) {
			continue
		}

		for i, col := range tableSchema.Columns {
			// Apply column pattern matching
			if columnPattern != "" && columnPattern != "%" && !matchPattern(col.Name, columnPattern) {
				continue
			}

			nullable := 1 // SQL_NULLABLE
			if col.NotNull {
				nullable = 0 // SQL_NO_NULLS
			}

			row := []interface{}{
				catalog,           // TABLE_CAT
				schema,            // TABLE_SCHEM
				tableName,         // TABLE_NAME
				col.Name,          // COLUMN_NAME
				getSQLType(col.Type), // DATA_TYPE
				col.Type,          // TYPE_NAME
				col.Size,          // COLUMN_SIZE
				0,                 // BUFFER_LENGTH
				col.Scale,         // DECIMAL_DIGITS
				10,                // NUM_PREC_RADIX
				nullable,          // NULLABLE
				col.Comment,       // REMARKS
				col.Default,       // COLUMN_DEF
				getSQLType(col.Type), // SQL_DATA_TYPE
				0,                 // SQL_DATETIME_SUB
				col.Size,          // CHAR_OCTET_LENGTH
				i + 1,             // ORDINAL_POSITION
				"YES",             // IS_NULLABLE
			}
			if col.NotNull {
				row[17] = "NO"
			}
			rows = append(rows, row)
		}
	}

	return rows, nil
}

// GetPrimaryKeys returns primary key information for the specified table.
func (m *serverMetadataProvider) GetPrimaryKeys(catalog, schema, table string) ([][]interface{}, error) {
	executor := m.srv.executor
	if executor == nil {
		return nil, nil
	}

	cat := executor.GetCatalog()
	if cat == nil {
		return nil, nil
	}

	tableSchema, ok := cat.Tables[table]
	if !ok {
		return nil, nil
	}

	pkCols := tableSchema.GetPrimaryKeyColumns()
	var rows [][]interface{}
	for i, colName := range pkCols {
		row := []interface{}{
			catalog,                    // TABLE_CAT
			schema,                     // TABLE_SCHEM
			table,                      // TABLE_NAME
			colName,                    // COLUMN_NAME
			i + 1,                      // KEY_SEQ
			table + "_pkey",            // PK_NAME
		}
		rows = append(rows, row)
	}

	return rows, nil
}

