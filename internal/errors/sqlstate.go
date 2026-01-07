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
Package errors provides SQLSTATE mappings for ODBC/JDBC compatibility.

SQLSTATE is a 5-character code defined by SQL standards (ISO/IEC 9075)
that provides standardized error codes across database systems.

Format: CCXXX where:
  - CC = Class (2 characters)
  - XXX = Subclass (3 characters)

Common Classes:
  - 00 = Successful completion
  - 01 = Warning
  - 02 = No data
  - 08 = Connection exception
  - 22 = Data exception
  - 23 = Integrity constraint violation
  - 24 = Invalid cursor state
  - 25 = Invalid transaction state
  - 28 = Invalid authorization specification
  - 42 = Syntax error or access rule violation
  - HY = CLI-specific condition (ODBC)
*/
package errors

// SQLSTATE represents a standard SQL state code.
type SQLSTATE string

// Standard SQLSTATE codes
const (
	// Success
	SQLStateSuccess SQLSTATE = "00000"

	// Warning (01xxx)
	SQLStateWarning           SQLSTATE = "01000"
	SQLStateWarningTruncation SQLSTATE = "01004"

	// No Data (02xxx)
	SQLStateNoData         SQLSTATE = "02000"
	SQLStateNoMoreData     SQLSTATE = "02001"

	// Connection Exception (08xxx)
	SQLStateConnectionError     SQLSTATE = "08000"
	SQLStateConnectionFailure   SQLSTATE = "08001"
	SQLStateConnectionNameInUse SQLSTATE = "08002"
	SQLStateNoConnection        SQLSTATE = "08003"
	SQLStateServerRejected      SQLSTATE = "08004"
	SQLStateConnectionLinkFail  SQLSTATE = "08S01"

	// Data Exception (22xxx)
	SQLStateDataException       SQLSTATE = "22000"
	SQLStateStringTruncation    SQLSTATE = "22001"
	SQLStateNumericOutOfRange   SQLSTATE = "22003"
	SQLStateInvalidDatetime     SQLSTATE = "22007"
	SQLStateDivisionByZero      SQLSTATE = "22012"
	SQLStateInvalidCharValue    SQLSTATE = "22018"
	SQLStateAssignmentError     SQLSTATE = "22005"

	// Integrity Constraint Violation (23xxx)
	SQLStateIntegrityConstraint SQLSTATE = "23000"
	SQLStateRestrictViolation   SQLSTATE = "23001"
	SQLStateNotNullViolation    SQLSTATE = "23502"
	SQLStateForeignKeyViolation SQLSTATE = "23503"
	SQLStateUniqueViolation     SQLSTATE = "23505"
	SQLStateCheckViolation      SQLSTATE = "23514"

	// Invalid Cursor State (24xxx)
	SQLStateCursorState     SQLSTATE = "24000"
	SQLStateCursorNotOpen   SQLSTATE = "24001"
	SQLStateCursorAlreadyOpen SQLSTATE = "24002"
	SQLStateNoCurrentRow    SQLSTATE = "24003"

	// Invalid Transaction State (25xxx)
	SQLStateTransactionState    SQLSTATE = "25000"
	SQLStateActiveTx            SQLSTATE = "25001"
	SQLStateBranchTxActive      SQLSTATE = "25002"
	SQLStateNoActiveTx          SQLSTATE = "25P01"
	SQLStateReadOnlyTx          SQLSTATE = "25006"
	SQLStateSchemaAndDataStmt   SQLSTATE = "25007"

	// Invalid Authorization (28xxx)
	SQLStateAuthError           SQLSTATE = "28000"
	SQLStateInvalidAuthSpec     SQLSTATE = "28P01"

	// Invalid Catalog Name (3Dxxx)
	SQLStateInvalidCatalog SQLSTATE = "3D000"

	// Invalid Schema Name (3Fxxx)
	SQLStateInvalidSchema SQLSTATE = "3F000"

	// Syntax Error or Access Rule Violation (42xxx)
	SQLStateSyntaxError         SQLSTATE = "42000"
	SQLStateTableNotFound       SQLSTATE = "42S02"
	SQLStateColumnNotFound      SQLSTATE = "42S22"
	SQLStateColumnAlreadyExists SQLSTATE = "42S21"
	SQLStateTableAlreadyExists  SQLSTATE = "42S01"
	SQLStateIndexNotFound       SQLSTATE = "42S12"
	SQLStateAmbiguousColumn     SQLSTATE = "42702"
	SQLStateUndefinedFunction   SQLSTATE = "42883"
	SQLStateInsufficientPriv    SQLSTATE = "42501"

	// CLI-specific Condition (HYxxx) - ODBC specific
	SQLStateCLIError            SQLSTATE = "HY000"
	SQLStateMemoryAlloc         SQLSTATE = "HY001"
	SQLStateInvalidHandle       SQLSTATE = "HY009"
	SQLStateInvalidArgument     SQLSTATE = "HY024"
	SQLStateFunctionSequence    SQLSTATE = "HY010"
	SQLStateAttrCannotBeSet     SQLSTATE = "HY011"
	SQLStateInvalidInfoType     SQLSTATE = "HY096"
	SQLStateInvalidDescriptor   SQLSTATE = "HY091"
	SQLStateOptionalFeature     SQLSTATE = "HYC00"
	SQLStateTimeout             SQLSTATE = "HYT00"
	SQLStateConnectTimeout      SQLSTATE = "HYT01"

	// Internal Error (XX)
	SQLStateInternalError  SQLSTATE = "XX000"
	SQLStateDataCorrupted  SQLSTATE = "XX001"
	SQLStateIndexCorrupted SQLSTATE = "XX002"
)

// sqlstateMap maps FlyDB error codes to SQLSTATE codes.
var sqlstateMap = map[ErrorCode]SQLSTATE{
	// Syntax errors (1000-1999) -> 42xxx
	ErrCodeSyntax:          SQLStateSyntaxError,
	ErrCodeUnexpectedToken: SQLStateSyntaxError,
	ErrCodeMissingKeyword:  SQLStateSyntaxError,
	ErrCodeInvalidCommand:  SQLStateSyntaxError,
	ErrCodeMalformedQuery:  SQLStateSyntaxError,
	ErrCodeInvalidLiteral:  SQLStateSyntaxError,
	ErrCodeUnclosedString:  SQLStateSyntaxError,
	ErrCodeInvalidOperator: SQLStateSyntaxError,

	// Execution errors (2000-2999)
	ErrCodeExecution:           SQLStateCLIError,
	ErrCodeTableNotFound:       SQLStateTableNotFound,
	ErrCodeColumnNotFound:      SQLStateColumnNotFound,
	ErrCodeTypeMismatch:        SQLStateAssignmentError,
	ErrCodeConstraintViolation: SQLStateIntegrityConstraint,
	ErrCodeDuplicateKey:        SQLStateUniqueViolation,
	ErrCodeNullViolation:       SQLStateNotNullViolation,
	ErrCodeForeignKeyViolation: SQLStateForeignKeyViolation,
	ErrCodeDivisionByZero:      SQLStateDivisionByZero,
	ErrCodeOverflow:            SQLStateNumericOutOfRange,

	// Connection errors (3000-3999) -> 08xxx
	ErrCodeConnection:        SQLStateConnectionError,
	ErrCodeConnectionLost:    SQLStateConnectionLinkFail,
	ErrCodeTimeout:           SQLStateTimeout,
	ErrCodeProtocolError:     SQLStateConnectionError,
	ErrCodeServerUnavailable: SQLStateServerRejected,

	// Auth errors (4000-4999) -> 28xxx
	ErrCodeAuth:               SQLStateAuthError,
	ErrCodeAuthFailed:         SQLStateInvalidAuthSpec,
	ErrCodePermissionDenied:   SQLStateInsufficientPriv,
	ErrCodeSessionExpired:     SQLStateAuthError,
	ErrCodeInvalidCredentials: SQLStateInvalidAuthSpec,

	// Storage errors (5000-5999) -> XXxxx / HYxxx
	ErrCodeStorage:           SQLStateInternalError,
	ErrCodeWALCorrupted:      SQLStateDataCorrupted,
	ErrCodeDiskFull:          SQLStateInternalError,
	ErrCodeIOError:           SQLStateInternalError,
	ErrCodeTransactionFailed: SQLStateTransactionState,

	// Validation errors (6000-6999) -> 22xxx
	ErrCodeValidation:      SQLStateDataException,
	ErrCodeInvalidValue:    SQLStateDataException,
	ErrCodeValueOutOfRange: SQLStateNumericOutOfRange,
	ErrCodeInvalidFormat:   SQLStateInvalidCharValue,
	ErrCodeMissingRequired: SQLStateNotNullViolation,

	// Cursor errors (7000-7999) -> 24xxx
	ErrCodeCursor:            SQLStateCursorState,
	ErrCodeCursorNotOpen:     SQLStateCursorNotOpen,
	ErrCodeCursorAlreadyOpen: SQLStateCursorAlreadyOpen,
	ErrCodeCursorExhausted:   SQLStateNoData,
	ErrCodeInvalidCursorPos:  SQLStateNoCurrentRow,
	ErrCodeCursorClosed:      SQLStateCursorNotOpen,
	ErrCodeFetchOutOfRange:   SQLStateNoData,

	// Transaction errors (8000-8999) -> 25xxx
	ErrCodeTransaction:         SQLStateTransactionState,
	ErrCodeTxNotActive:         SQLStateNoActiveTx,
	ErrCodeTxAlreadyActive:     SQLStateActiveTx,
	ErrCodeTxIsolationError:    SQLStateTransactionState,
	ErrCodeTxDeadlock:          SQLStateTransactionState,
	ErrCodeTxSerializationFail: SQLStateTransactionState,
	ErrCodeTxReadOnly:          SQLStateReadOnlyTx,

	// Driver errors (9000-9999) -> HYxxx
	ErrCodeDriver:            SQLStateCLIError,
	ErrCodeDriverNotReady:    SQLStateFunctionSequence,
	ErrCodeInvalidHandle:     SQLStateInvalidHandle,
	ErrCodeFunctionSequence:  SQLStateFunctionSequence,
	ErrCodeMemoryAllocation:  SQLStateMemoryAlloc,
	ErrCodeInvalidDescriptor: SQLStateInvalidDescriptor,
}

// ToSQLSTATE converts a FlyDB error code to a SQLSTATE code.
func ToSQLSTATE(code ErrorCode) SQLSTATE {
	if state, ok := sqlstateMap[code]; ok {
		return state
	}
	// Default to general error
	return SQLStateCLIError
}

// GetSQLSTATE returns the SQLSTATE for a FlyDBError.
func GetSQLSTATE(err error) SQLSTATE {
	if e, ok := err.(*FlyDBError); ok {
		return ToSQLSTATE(e.Code)
	}
	return SQLStateCLIError
}

// SQLSTATEClass returns the 2-character class of a SQLSTATE.
func SQLSTATEClass(state SQLSTATE) string {
	if len(state) >= 2 {
		return string(state[:2])
	}
	return "HY"
}

// IsSuccessSQLSTATE returns true if the SQLSTATE indicates success.
func IsSuccessSQLSTATE(state SQLSTATE) bool {
	return SQLSTATEClass(state) == "00"
}

// IsWarningSQLSTATE returns true if the SQLSTATE indicates a warning.
func IsWarningSQLSTATE(state SQLSTATE) bool {
	return SQLSTATEClass(state) == "01"
}

// IsNoDataSQLSTATE returns true if the SQLSTATE indicates no data.
func IsNoDataSQLSTATE(state SQLSTATE) bool {
	return SQLSTATEClass(state) == "02"
}

// IsErrorSQLSTATE returns true if the SQLSTATE indicates an error.
func IsErrorSQLSTATE(state SQLSTATE) bool {
	class := SQLSTATEClass(state)
	return class != "00" && class != "01" && class != "02"
}

