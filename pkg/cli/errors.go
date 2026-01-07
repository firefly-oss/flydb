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
	"fmt"
	"os"
)

// CLIError represents a CLI error with suggestions.
type CLIError struct {
	Message     string
	Detail      string
	Suggestions []string
	ExitCode    int
}

// Error implements the error interface.
func (e *CLIError) Error() string {
	return e.Message
}

// Print prints the error with formatting.
func (e *CLIError) Print() {
	fmt.Printf("\n%s %s\n", ErrorIcon(), Error(e.Message))
	
	if e.Detail != "" {
		fmt.Printf("  %s\n", Dimmed(e.Detail))
	}
	
	if len(e.Suggestions) > 0 {
		fmt.Println()
		fmt.Printf("  %s\n", Highlight("Suggestions:"))
		for _, s := range e.Suggestions {
			fmt.Printf("    • %s\n", s)
		}
	}
	fmt.Println()
}

// Exit prints the error and exits with the error code.
func (e *CLIError) Exit() {
	e.Print()
	os.Exit(e.ExitCode)
}

// NewCLIError creates a new CLI error.
func NewCLIError(message string) *CLIError {
	return &CLIError{
		Message:  message,
		ExitCode: 1,
	}
}

// WithDetail adds detail to the error.
func (e *CLIError) WithDetail(detail string) *CLIError {
	e.Detail = detail
	return e
}

// WithSuggestion adds a suggestion to the error.
func (e *CLIError) WithSuggestion(suggestion string) *CLIError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// WithExitCode sets the exit code.
func (e *CLIError) WithExitCode(code int) *CLIError {
	e.ExitCode = code
	return e
}

// Common CLI errors with helpful suggestions.

// ErrConnectionFailed creates a connection failed error.
func ErrConnectionFailed(host, port string, err error) *CLIError {
	return NewCLIError("Failed to connect to FlyDB server").
		WithDetail(fmt.Sprintf("Could not connect to %s:%s - %v", host, port, err)).
		WithSuggestion("Ensure the FlyDB daemon is running: ./flydb").
		WithSuggestion(fmt.Sprintf("Check if the server is listening on %s:%s", host, port)).
		WithSuggestion("Verify firewall settings allow the connection")
}

// ErrAuthFailed creates an authentication failed error.
func ErrAuthFailed() *CLIError {
	return NewCLIError("Authentication failed").
		WithDetail("Invalid username or password").
		WithSuggestion("Check your credentials and try again").
		WithSuggestion("Use AUTH <username> <password> to authenticate")
}

// ErrInvalidCommand creates an invalid command error.
func ErrInvalidCommand(cmd string) *CLIError {
	return NewCLIError(fmt.Sprintf("Unknown command: %s", cmd)).
		WithSuggestion("Type \\h or \\help for a list of available commands").
		WithSuggestion("SQL commands are auto-detected (SELECT, INSERT, etc.)")
}

// ErrMissingArgument creates a missing argument error.
func ErrMissingArgument(arg, usage string) *CLIError {
	return NewCLIError(fmt.Sprintf("Missing required argument: %s", arg)).
		WithSuggestion(fmt.Sprintf("Usage: %s", usage))
}

// ErrInvalidValue creates an invalid value error.
func ErrInvalidValue(field, value, reason string) *CLIError {
	return NewCLIError(fmt.Sprintf("Invalid value for %s: %s", field, value)).
		WithDetail(reason)
}

// ErrConfigNotFound creates a config file not found error.
func ErrConfigNotFound(path string) *CLIError {
	return NewCLIError("Configuration file not found").
		WithDetail(fmt.Sprintf("Could not find: %s", path)).
		WithSuggestion("Create a configuration file or use command-line flags").
		WithSuggestion("Run with --help to see available options")
}

// ErrPermissionDenied creates a permission denied error.
func ErrPermissionDenied(resource string) *CLIError {
	return NewCLIError("Permission denied").
		WithDetail(fmt.Sprintf("You don't have access to: %s", resource)).
		WithSuggestion("Contact your administrator to request access").
		WithSuggestion("Ensure you're authenticated with the correct user")
}

