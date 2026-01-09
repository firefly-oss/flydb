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
SDK Error Types
===============

This file defines error types for FlyDB SDK and drivers. These errors provide
structured error information compatible with ODBC/JDBC error handling.

SQLSTATE Codes:
===============

FlyDB uses standard SQLSTATE codes for error classification:

  Class 00 - Successful Completion
  Class 01 - Warning
  Class 02 - No Data
  Class 08 - Connection Exception
  Class 22 - Data Exception
  Class 23 - Integrity Constraint Violation
  Class 25 - Invalid Transaction State
  Class 28 - Invalid Authorization Specification
  Class 42 - Syntax Error or Access Rule Violation

Error Hierarchy:
================

  SDKError (base)
    ├── ConnectionError
    ├── AuthenticationError
    ├── QueryError
    ├── TransactionError
    ├── CursorError
    └── MetadataError
*/
package sdk

import (
	"fmt"
)

// ErrorCode represents a FlyDB error code.
type ErrorCode int

// Error codes for SDK operations.
const (
	// General errors (1000-1099)
	ErrCodeUnknown          ErrorCode = 1000
	ErrCodeInternal         ErrorCode = 1001
	ErrCodeNotImplemented   ErrorCode = 1002
	ErrCodeInvalidArgument  ErrorCode = 1003
	ErrCodeTimeout          ErrorCode = 1004

	// Connection errors (1100-1199)
	ErrCodeConnectionFailed ErrorCode = 1100
	ErrCodeConnectionClosed ErrorCode = 1101
	ErrCodeConnectionReset  ErrorCode = 1102
	ErrCodeProtocolError    ErrorCode = 1103
	ErrCodeTLSError         ErrorCode = 1104

	// Authentication errors (1200-1299)
	ErrCodeAuthFailed       ErrorCode = 1200
	ErrCodeAuthRequired     ErrorCode = 1201
	ErrCodeAccessDenied     ErrorCode = 1202
	ErrCodeInvalidToken     ErrorCode = 1203

	// Query errors (1300-1399)
	ErrCodeSyntaxError      ErrorCode = 1300
	ErrCodeTableNotFound    ErrorCode = 1301
	ErrCodeColumnNotFound   ErrorCode = 1302
	ErrCodeTypeMismatch     ErrorCode = 1303
	ErrCodeConstraintViolation ErrorCode = 1304
	ErrCodeDuplicateKey     ErrorCode = 1305

	// Transaction errors (1400-1499)
	ErrCodeTxNotActive      ErrorCode = 1400
	ErrCodeTxAlreadyActive  ErrorCode = 1401
	ErrCodeTxRollback       ErrorCode = 1402
	ErrCodeDeadlock         ErrorCode = 1403
	ErrCodeSerializationFailure ErrorCode = 1404

	// Cursor errors (1500-1599)
	ErrCodeCursorNotFound   ErrorCode = 1500
	ErrCodeCursorClosed     ErrorCode = 1501
	ErrCodeCursorExhausted  ErrorCode = 1502
	ErrCodeInvalidPosition  ErrorCode = 1503

	// Database errors (1600-1699)
	ErrCodeDatabaseNotFound ErrorCode = 1600
	ErrCodeDatabaseExists   ErrorCode = 1601
)

// SQLSTATE returns the SQLSTATE code for this error code.
func (c ErrorCode) SQLSTATE() string {
	switch c {
	case ErrCodeConnectionFailed, ErrCodeConnectionClosed, ErrCodeConnectionReset:
		return "08000" // Connection exception
	case ErrCodeProtocolError:
		return "08P01" // Protocol violation
	case ErrCodeAuthFailed, ErrCodeAuthRequired:
		return "28000" // Invalid authorization specification
	case ErrCodeAccessDenied:
		return "42501" // Insufficient privilege
	case ErrCodeSyntaxError:
		return "42601" // Syntax error
	case ErrCodeTableNotFound:
		return "42P01" // Undefined table
	case ErrCodeColumnNotFound:
		return "42703" // Undefined column
	case ErrCodeTypeMismatch:
		return "42804" // Datatype mismatch
	case ErrCodeConstraintViolation:
		return "23000" // Integrity constraint violation
	case ErrCodeDuplicateKey:
		return "23505" // Unique violation
	case ErrCodeTxNotActive:
		return "25000" // Invalid transaction state
	case ErrCodeDeadlock:
		return "40P01" // Deadlock detected
	case ErrCodeSerializationFailure:
		return "40001" // Serialization failure
	case ErrCodeDatabaseNotFound:
		return "3D000" // Invalid catalog name
	default:
		return "HY000" // General error
	}
}

// SDKError is the base error type for all SDK errors.
type SDKError struct {
	Code     ErrorCode
	Message  string
	SQLSTATE string
	Cause    error
}

// Error implements the error interface.
func (e *SDKError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.SQLSTATE, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.SQLSTATE, e.Message)
}

// Unwrap returns the underlying cause.
func (e *SDKError) Unwrap() error {
	return e.Cause
}

// NewSDKError creates a new SDK error.
func NewSDKError(code ErrorCode, message string) *SDKError {
	return &SDKError{
		Code:     code,
		Message:  message,
		SQLSTATE: code.SQLSTATE(),
	}
}

// NewSDKErrorWithCause creates a new SDK error with a cause.
func NewSDKErrorWithCause(code ErrorCode, message string, cause error) *SDKError {
	return &SDKError{
		Code:     code,
		Message:  message,
		SQLSTATE: code.SQLSTATE(),
		Cause:    cause,
	}
}

