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
	"regexp"
	"strings"
)

// ansiRegex matches ANSI escape sequences for stripping from strings.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// VisibleLen returns the visible length of a string, excluding ANSI escape codes.
// This is essential for proper alignment when strings contain color codes.
func VisibleLen(s string) int {
	return len(ansiRegex.ReplaceAllString(s, ""))
}

// visibleLen is an internal alias for VisibleLen for backward compatibility.
func visibleLen(s string) int {
	return VisibleLen(s)
}

// PadRight pads a string to the specified visible width, accounting for ANSI codes.
// If the string (excluding ANSI codes) is already >= width, returns the original string.
func PadRight(s string, width int) string {
	visible := VisibleLen(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

// PadCenter centers a string within the specified visible width, accounting for ANSI codes.
func PadCenter(s string, width int) string {
	visible := VisibleLen(s)
	if visible >= width {
		return s
	}
	totalPadding := width - visible
	leftPadding := totalPadding / 2
	rightPadding := totalPadding - leftPadding
	return strings.Repeat(" ", leftPadding) + s + strings.Repeat(" ", rightPadding)
}

// BoxLine creates a line for a box with proper padding, accounting for ANSI codes.
// It creates a line like "│  content...padding  │" with the specified total width.
func BoxLine(content string, totalWidth int, leftBorder, rightBorder string) string {
	// totalWidth is the inner width (excluding borders)
	contentWidth := totalWidth - 4 // 2 spaces on each side
	padded := PadRight(content, contentWidth)
	return leftBorder + "  " + padded + "  " + rightBorder
}

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

	// Determine the number of columns (max of headers and all rows)
	numCols := len(t.headers)
	for _, row := range t.rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}

	// Calculate column widths considering both headers and data
	// Use visible length to handle ANSI color codes properly
	colWidths := make([]int, numCols)
	for i, h := range t.headers {
		if visibleLen(h) > colWidths[i] {
			colWidths[i] = visibleLen(h)
		}
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < numCols && visibleLen(cell) > colWidths[i] {
				colWidths[i] = visibleLen(cell)
			}
		}
	}

	// Ensure minimum column width of 3 for aesthetics
	for i := range colWidths {
		if colWidths[i] < 3 {
			colWidths[i] = 3
		}
	}

	// Unicode box-drawing characters for professional grid
	const (
		topLeft     = "┌"
		topRight    = "┐"
		bottomLeft  = "└"
		bottomRight = "┘"
		horizontal  = "─"
		vertical    = "│"
		topT        = "┬"
		bottomT     = "┴"
		leftT       = "├"
		rightT      = "┤"
		cross       = "┼"
	)

	// Build border strings
	var topParts, sepParts, bottomParts []string
	for _, width := range colWidths {
		topParts = append(topParts, strings.Repeat(horizontal, width+2))
		sepParts = append(sepParts, strings.Repeat(horizontal, width+2))
		bottomParts = append(bottomParts, strings.Repeat(horizontal, width+2))
	}
	topBorder := topLeft + strings.Join(topParts, topT) + topRight
	separator := leftT + strings.Join(sepParts, cross) + rightT
	bottomBorder := bottomLeft + strings.Join(bottomParts, bottomT) + bottomRight

	fmt.Println()
	fmt.Println(Dimmed(topBorder))

	// Print headers if present
	if len(t.headers) > 0 {
		var headerParts []string
		for i := 0; i < numCols; i++ {
			val := ""
			if i < len(t.headers) {
				val = t.headers[i]
			}
			// Pad based on visible length to handle ANSI codes
			padding := colWidths[i] - visibleLen(val)
			padded := " " + val + strings.Repeat(" ", padding) + " "
			headerParts = append(headerParts, padded)
		}
		// Apply bold to header content, not the borders
		headerLine := Dimmed(vertical)
		for i, part := range headerParts {
			headerLine += colorize(Bold, part)
			if i < len(headerParts)-1 {
				headerLine += Dimmed(vertical)
			}
		}
		headerLine += Dimmed(vertical)
		fmt.Println(headerLine)
		fmt.Println(Dimmed(separator))
	}

	// Print data rows
	for _, row := range t.rows {
		var rowParts []string
		for i := 0; i < numCols; i++ {
			val := ""
			if i < len(row) {
				val = row[i]
			}
			// Pad based on visible length to handle ANSI codes
			padding := colWidths[i] - visibleLen(val)
			padded := " " + val + strings.Repeat(" ", padding) + " "
			rowParts = append(rowParts, padded)
		}
		fmt.Println(Dimmed(vertical) + strings.Join(rowParts, Dimmed(vertical)) + Dimmed(vertical))
	}

	fmt.Println(Dimmed(bottomBorder))
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

// Box prints text in a styled section (wizard-style, no box borders).
// Properly handles ANSI color codes in title and content.
func Box(title, content string) {
	lines := strings.Split(content, "\n")

	// Print title with separator
	fmt.Println()
	fmt.Println("  " + Highlight(title))
	fmt.Println("  " + Separator(40))
	fmt.Println()

	// Print each content line with indentation
	for _, line := range lines {
		if line == "" {
			fmt.Println()
		} else {
			fmt.Println("    " + line)
		}
	}
	fmt.Println()
}

// KeyValue prints a key-value pair with alignment.
func KeyValue(key, value string, keyWidth int) {
	fmt.Printf("  %-*s %s\n", keyWidth, key+":", value)
}
