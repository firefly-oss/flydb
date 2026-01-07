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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// OutputFormat represents the output format type.
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatPlain OutputFormat = "plain"
)

// ParseOutputFormat parses a string into an OutputFormat.
func ParseOutputFormat(s string) OutputFormat {
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON
	case "plain":
		return FormatPlain
	default:
		return FormatTable
	}
}

// Table provides formatted table output.
type Table struct {
	headers []string
	rows    [][]string
	format  OutputFormat
}

// NewTable creates a new table with the given headers.
func NewTable(headers ...string) *Table {
	return &Table{
		headers: headers,
		rows:    make([][]string, 0),
		format:  FormatTable,
	}
}

// SetFormat sets the output format.
func (t *Table) SetFormat(format OutputFormat) {
	t.format = format
}

// AddRow adds a row to the table.
func (t *Table) AddRow(values ...string) {
	t.rows = append(t.rows, values)
}

// Print outputs the table in the configured format.
func (t *Table) Print() {
	switch t.format {
	case FormatJSON:
		t.printJSON()
	case FormatPlain:
		t.printPlain()
	default:
		t.printTable()
	}
}

func (t *Table) printTable() {
	if len(t.rows) == 0 {
		fmt.Println("(no results)")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print headers
	if len(t.headers) > 0 {
		headerLine := strings.Join(t.headers, "\t")
		fmt.Fprintln(w, colorize(Bold, headerLine))
		
		// Print separator
		seps := make([]string, len(t.headers))
		for i, h := range t.headers {
			seps[i] = strings.Repeat("─", len(h))
		}
		fmt.Fprintln(w, strings.Join(seps, "\t"))
	}

	// Print rows
	for _, row := range t.rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	w.Flush()
	fmt.Printf("\n(%d rows)\n", len(t.rows))
}

func (t *Table) printJSON() {
	result := make([]map[string]string, len(t.rows))
	for i, row := range t.rows {
		rowMap := make(map[string]string)
		for j, val := range row {
			if j < len(t.headers) {
				rowMap[t.headers[j]] = val
			} else {
				rowMap[fmt.Sprintf("col%d", j)] = val
			}
		}
		result[i] = rowMap
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		PrintError("Failed to format JSON: %v", err)
		return
	}
	fmt.Println(string(data))
}

func (t *Table) printPlain() {
	for _, row := range t.rows {
		fmt.Println(strings.Join(row, "\t"))
	}
}

// Box prints text in a box.
func Box(title, content string) {
	lines := strings.Split(content, "\n")
	maxLen := len(title)
	for _, line := range lines {
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}

	width := maxLen + 4
	fmt.Println("╔" + strings.Repeat("═", width) + "╗")
	fmt.Printf("║  %s%s  ║\n", colorize(Bold, title), strings.Repeat(" ", maxLen-len(title)))
	fmt.Println("╠" + strings.Repeat("═", width) + "╣")
	for _, line := range lines {
		fmt.Printf("║  %s%s  ║\n", line, strings.Repeat(" ", maxLen-len(line)))
	}
	fmt.Println("╚" + strings.Repeat("═", width) + "╝")
}

// KeyValue prints a key-value pair with alignment.
func KeyValue(key, value string, keyWidth int) {
	fmt.Printf("  %-*s %s\n", keyWidth, key+":", value)
}

