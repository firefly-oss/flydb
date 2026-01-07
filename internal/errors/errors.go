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
Package errors provides comprehensive error handling for FlyDB.

The errors package implements a structured error system with:
  - Error categories (Syntax, Execution, Connection, Auth, etc.)
  - Error codes for programmatic handling
  - User-friendly error messages
  - Contextual information for debugging
  - Error wrapping for root cause analysis

Error Categories:
  - SyntaxError: Command parsing and SQL syntax errors
  - ExecutionError: Runtime failures during query execution
  - ConnectionError: Network and connection issues
  - AuthError: Authentication and authorization failures
  - StorageError: Persistence and storage issues
  - ValidationError: Input validation failures
*/
package errors

import (
	"fmt"
)

// ErrorCode represents a unique error identifier.
type ErrorCode int

const (
	// Syntax errors (1000-1999)
	ErrCodeSyntax           ErrorCode = 1000
	ErrCodeUnexpectedToken  ErrorCode = 1001
	ErrCodeMissingKeyword   ErrorCode = 1002
	ErrCodeInvalidCommand   ErrorCode = 1003
	ErrCodeMalformedQuery   ErrorCode = 1004
	ErrCodeInvalidLiteral   ErrorCode = 1005
	ErrCodeUnclosedString   ErrorCode = 1006
	ErrCodeInvalidOperator  ErrorCode = 1007

	// Execution errors (2000-2999)
	ErrCodeExecution        ErrorCode = 2000
	ErrCodeTableNotFound    ErrorCode = 2001
	ErrCodeColumnNotFound   ErrorCode = 2002
	ErrCodeTypeMismatch     ErrorCode = 2003
	ErrCodeConstraintViolation ErrorCode = 2004
	ErrCodeDuplicateKey     ErrorCode = 2005
	ErrCodeNullViolation    ErrorCode = 2006
	ErrCodeForeignKeyViolation ErrorCode = 2007
	ErrCodeDivisionByZero   ErrorCode = 2008
	ErrCodeOverflow         ErrorCode = 2009

	// Connection errors (3000-3999)
	ErrCodeConnection       ErrorCode = 3000
	ErrCodeConnectionLost   ErrorCode = 3001
	ErrCodeTimeout          ErrorCode = 3002
	ErrCodeProtocolError    ErrorCode = 3003
	ErrCodeServerUnavailable ErrorCode = 3004

	// Auth errors (4000-4999)
	ErrCodeAuth             ErrorCode = 4000
	ErrCodeAuthFailed       ErrorCode = 4001
	ErrCodePermissionDenied ErrorCode = 4002
	ErrCodeSessionExpired   ErrorCode = 4003
	ErrCodeInvalidCredentials ErrorCode = 4004

	// Storage errors (5000-5999)
	ErrCodeStorage          ErrorCode = 5000
	ErrCodeWALCorrupted     ErrorCode = 5001
	ErrCodeDiskFull         ErrorCode = 5002
	ErrCodeIOError          ErrorCode = 5003
	ErrCodeTransactionFailed ErrorCode = 5004

	// Validation errors (6000-6999)
	ErrCodeValidation       ErrorCode = 6000
	ErrCodeInvalidValue     ErrorCode = 6001
	ErrCodeValueOutOfRange  ErrorCode = 6002
	ErrCodeInvalidFormat    ErrorCode = 6003
	ErrCodeMissingRequired  ErrorCode = 6004
)

// Category represents the error category.
type Category string

const (
	CategorySyntax     Category = "SYNTAX"
	CategoryExecution  Category = "EXECUTION"
	CategoryConnection Category = "CONNECTION"
	CategoryAuth       Category = "AUTH"
	CategoryStorage    Category = "STORAGE"
	CategoryValidation Category = "VALIDATION"
)

// FlyDBError represents a structured error in FlyDB.
type FlyDBError struct {
	Code     ErrorCode
	Category Category
	Message  string
	Detail   string
	Hint     string
	Cause    error
}

// Error implements the error interface.
func (e *FlyDBError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("ERROR %d (%s): %s - %s", e.Code, e.Category, e.Message, e.Detail)
	}
	return fmt.Sprintf("ERROR %d (%s): %s", e.Code, e.Category, e.Message)
}

// Unwrap returns the underlying cause.
func (e *FlyDBError) Unwrap() error {
	return e.Cause
}

// UserMessage returns a user-friendly error message.
func (e *FlyDBError) UserMessage() string {
	msg := fmt.Sprintf("ERROR: %s", e.Message)
	if e.Detail != "" {
		msg += fmt.Sprintf(" (%s)", e.Detail)
	}
	if e.Hint != "" {
		msg += fmt.Sprintf("\nHINT: %s", e.Hint)
	}
	return msg
}

// WithDetail adds detail to the error.
func (e *FlyDBError) WithDetail(detail string) *FlyDBError {
	e.Detail = detail
	return e
}

// WithHint adds a hint to the error.
func (e *FlyDBError) WithHint(hint string) *FlyDBError {
	e.Hint = hint
	return e
}

// WithCause adds a cause to the error.
func (e *FlyDBError) WithCause(cause error) *FlyDBError {
	e.Cause = cause
	return e
}

// ============================================================================
// Syntax Error Constructors
// ============================================================================

// NewSyntaxError creates a new syntax error.
func NewSyntaxError(message string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeSyntax,
		Category: CategorySyntax,
		Message:  message,
	}
}

// UnexpectedToken creates an error for unexpected tokens.
func UnexpectedToken(expected, got string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeUnexpectedToken,
		Category: CategorySyntax,
		Message:  fmt.Sprintf("unexpected token: expected %s, got %s", expected, got),
		Hint:     "Check your SQL syntax",
	}
}

// MissingKeyword creates an error for missing keywords.
func MissingKeyword(keyword string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeMissingKeyword,
		Category: CategorySyntax,
		Message:  fmt.Sprintf("missing keyword: %s", keyword),
		Hint:     fmt.Sprintf("Add the '%s' keyword to your statement", keyword),
	}
}

// InvalidCommand creates an error for invalid commands.
func InvalidCommand(command string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeInvalidCommand,
		Category: CategorySyntax,
		Message:  fmt.Sprintf("invalid command: %s", command),
		Hint:     "Supported commands: PING, AUTH, SQL, WATCH",
	}
}

// ============================================================================
// Execution Error Constructors
// ============================================================================

// NewExecutionError creates a new execution error.
func NewExecutionError(message string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeExecution,
		Category: CategoryExecution,
		Message:  message,
	}
}

// TableNotFound creates an error for missing tables.
func TableNotFound(table string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeTableNotFound,
		Category: CategoryExecution,
		Message:  fmt.Sprintf("table not found: %s", table),
		Hint:     "Use INTROSPECT TABLES to see available tables",
	}
}

// ColumnNotFound creates an error for missing columns.
func ColumnNotFound(column, table string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeColumnNotFound,
		Category: CategoryExecution,
		Message:  fmt.Sprintf("column '%s' not found in table '%s'", column, table),
		Hint:     fmt.Sprintf("Use INTROSPECT TABLE %s to see available columns", table),
	}
}

// TypeMismatch creates an error for type mismatches.
func TypeMismatch(expected, got, column string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeTypeMismatch,
		Category: CategoryExecution,
		Message:  fmt.Sprintf("type mismatch for column '%s': expected %s, got %s", column, expected, got),
	}
}

// ConstraintViolation creates an error for constraint violations.
func ConstraintViolation(constraint, detail string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeConstraintViolation,
		Category: CategoryExecution,
		Message:  fmt.Sprintf("constraint violation: %s", constraint),
		Detail:   detail,
	}
}

// DuplicateKey creates an error for duplicate key violations.
func DuplicateKey(key, table string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeDuplicateKey,
		Category: CategoryExecution,
		Message:  fmt.Sprintf("duplicate key value violates unique constraint on table '%s'", table),
		Detail:   fmt.Sprintf("Key: %s", key),
	}
}

// ============================================================================
// Connection Error Constructors
// ============================================================================

// NewConnectionError creates a new connection error.
func NewConnectionError(message string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeConnection,
		Category: CategoryConnection,
		Message:  message,
	}
}

// ConnectionLost creates an error for lost connections.
func ConnectionLost(reason string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeConnectionLost,
		Category: CategoryConnection,
		Message:  "connection lost",
		Detail:   reason,
		Hint:     "Check network connectivity and try reconnecting",
	}
}

// ProtocolError creates an error for protocol violations.
func ProtocolError(detail string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeProtocolError,
		Category: CategoryConnection,
		Message:  "protocol error",
		Detail:   detail,
	}
}

// ============================================================================
// Auth Error Constructors
// ============================================================================

// NewAuthError creates a new auth error.
func NewAuthError(message string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeAuth,
		Category: CategoryAuth,
		Message:  message,
	}
}

// AuthenticationFailed creates an error for failed authentication.
func AuthenticationFailed() *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeAuthFailed,
		Category: CategoryAuth,
		Message:  "authentication failed",
		Hint:     "Check your username and password",
	}
}

// PermissionDenied creates an error for permission denied.
func PermissionDenied(resource string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodePermissionDenied,
		Category: CategoryAuth,
		Message:  "permission denied",
		Detail:   fmt.Sprintf("Access to '%s' is not allowed", resource),
		Hint:     "Contact your administrator to request access",
	}
}

// ============================================================================
// Storage Error Constructors
// ============================================================================

// NewStorageError creates a new storage error.
func NewStorageError(message string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeStorage,
		Category: CategoryStorage,
		Message:  message,
	}
}

// WALCorrupted creates an error for corrupted WAL.
func WALCorrupted(detail string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeWALCorrupted,
		Category: CategoryStorage,
		Message:  "write-ahead log corrupted",
		Detail:   detail,
		Hint:     "Restore from backup or contact support",
	}
}

// TransactionFailed creates an error for failed transactions.
func TransactionFailed(reason string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeTransactionFailed,
		Category: CategoryStorage,
		Message:  "transaction failed",
		Detail:   reason,
	}
}

// ============================================================================
// Validation Error Constructors
// ============================================================================

// NewValidationError creates a new validation error.
func NewValidationError(message string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeValidation,
		Category: CategoryValidation,
		Message:  message,
	}
}

// InvalidValue creates an error for invalid values.
func InvalidValue(field, reason string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeInvalidValue,
		Category: CategoryValidation,
		Message:  fmt.Sprintf("invalid value for '%s'", field),
		Detail:   reason,
	}
}

// MissingRequired creates an error for missing required fields.
func MissingRequired(field string) *FlyDBError {
	return &FlyDBError{
		Code:     ErrCodeMissingRequired,
		Category: CategoryValidation,
		Message:  fmt.Sprintf("missing required field: %s", field),
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// IsSyntaxError checks if an error is a syntax error.
func IsSyntaxError(err error) bool {
	if e, ok := err.(*FlyDBError); ok {
		return e.Category == CategorySyntax
	}
	return false
}

// IsExecutionError checks if an error is an execution error.
func IsExecutionError(err error) bool {
	if e, ok := err.(*FlyDBError); ok {
		return e.Category == CategoryExecution
	}
	return false
}

// IsAuthError checks if an error is an auth error.
func IsAuthError(err error) bool {
	if e, ok := err.(*FlyDBError); ok {
		return e.Category == CategoryAuth
	}
	return false
}

// GetCode returns the error code if it's a FlyDBError, or 0 otherwise.
func GetCode(err error) ErrorCode {
	if e, ok := err.(*FlyDBError); ok {
		return e.Code
	}
	return 0
}

// FormatError formats an error for user display.
func FormatError(err error) string {
	if e, ok := err.(*FlyDBError); ok {
		return e.UserMessage()
	}
	return fmt.Sprintf("ERROR: %v", err)
}

