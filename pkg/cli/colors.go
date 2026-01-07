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
Package cli provides shared CLI utilities for FlyDB applications.

This package includes:
  - Color and styling utilities for terminal output
  - Progress indicators and spinners
  - Prompt utilities for user input
  - Output formatting helpers
  - Common CLI patterns and helpers
*/
package cli

import (
	"fmt"
	"os"
	"strings"
)

// ANSI color codes for terminal output.
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"

	// Foreground colors
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	// Bright foreground colors
	BrightBlack   = "\033[90m"
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"

	// Background colors
	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

// colorsEnabled controls whether colors are output.
var colorsEnabled = true

func init() {
	// Disable colors if NO_COLOR env var is set or if not a terminal
	if os.Getenv("NO_COLOR") != "" {
		colorsEnabled = false
	}
	// Check if stdout is a terminal
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		colorsEnabled = false
	}
}

// SetColorsEnabled enables or disables color output.
func SetColorsEnabled(enabled bool) {
	colorsEnabled = enabled
}

// ColorsEnabled returns whether colors are enabled.
func ColorsEnabled() bool {
	return colorsEnabled
}

// colorize applies color codes if colors are enabled.
func colorize(color, text string) string {
	if !colorsEnabled {
		return text
	}
	return color + text + Reset
}

// Success formats text as a success message (green).
func Success(text string) string {
	return colorize(Green, text)
}

// Error formats text as an error message (red).
func Error(text string) string {
	return colorize(Red, text)
}

// Warning formats text as a warning message (yellow).
func Warning(text string) string {
	return colorize(Yellow, text)
}

// Info formats text as an info message (cyan).
func Info(text string) string {
	return colorize(Cyan, text)
}

// Highlight formats text as highlighted (bold).
func Highlight(text string) string {
	return colorize(Bold, text)
}

// Dim formats text as dimmed.
func Dimmed(text string) string {
	return colorize(Dim, text)
}

// SuccessIcon returns a green checkmark.
func SuccessIcon() string {
	return colorize(Green, "✓")
}

// ErrorIcon returns a red X.
func ErrorIcon() string {
	return colorize(Red, "✗")
}

// WarningIcon returns a yellow warning sign.
func WarningIcon() string {
	return colorize(Yellow, "⚠")
}

// InfoIcon returns a cyan info icon.
func InfoIcon() string {
	return colorize(Cyan, "ℹ")
}

// PrintSuccess prints a success message with icon.
func PrintSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", SuccessIcon(), Success(msg))
}

// PrintError prints an error message with icon.
func PrintError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", ErrorIcon(), Error(msg))
}

// PrintWarning prints a warning message with icon.
func PrintWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", WarningIcon(), Warning(msg))
}

// PrintInfo prints an info message with icon.
func PrintInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", InfoIcon(), Info(msg))
}

// Separator returns a horizontal line separator.
func Separator(width int) string {
	return strings.Repeat("─", width)
}

// DoubleSeparator returns a double horizontal line separator.
func DoubleSeparator(width int) string {
	return strings.Repeat("═", width)
}

