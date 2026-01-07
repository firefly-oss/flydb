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

package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestFlyDBErrorBasic(t *testing.T) {
	err := NewSyntaxError("unexpected token")
	
	if err.Code != ErrCodeSyntax {
		t.Errorf("Expected code %d, got %d", ErrCodeSyntax, err.Code)
	}
	if err.Category != CategorySyntax {
		t.Errorf("Expected category %s, got %s", CategorySyntax, err.Category)
	}
	if !strings.Contains(err.Error(), "unexpected token") {
		t.Errorf("Expected error message to contain 'unexpected token', got: %s", err.Error())
	}
}

func TestFlyDBErrorWithDetail(t *testing.T) {
	err := NewExecutionError("query failed").WithDetail("table not found")
	
	if err.Detail != "table not found" {
		t.Errorf("Expected detail 'table not found', got: %s", err.Detail)
	}
	if !strings.Contains(err.Error(), "table not found") {
		t.Errorf("Expected error to contain detail, got: %s", err.Error())
	}
}

func TestFlyDBErrorWithHint(t *testing.T) {
	err := NewSyntaxError("missing keyword").WithHint("Add SELECT before column list")
	
	userMsg := err.UserMessage()
	if !strings.Contains(userMsg, "HINT:") {
		t.Errorf("Expected user message to contain HINT, got: %s", userMsg)
	}
	if !strings.Contains(userMsg, "Add SELECT") {
		t.Errorf("Expected hint in user message, got: %s", userMsg)
	}
}

func TestFlyDBErrorWithCause(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewStorageError("write failed").WithCause(cause)
	
	if err.Unwrap() != cause {
		t.Error("Expected Unwrap to return the cause")
	}
}

func TestSyntaxErrorConstructors(t *testing.T) {
	tests := []struct {
		name     string
		err      *FlyDBError
		code     ErrorCode
		category Category
	}{
		{"UnexpectedToken", UnexpectedToken("SELECT", "FROM"), ErrCodeUnexpectedToken, CategorySyntax},
		{"MissingKeyword", MissingKeyword("WHERE"), ErrCodeMissingKeyword, CategorySyntax},
		{"InvalidCommand", InvalidCommand("UNKNOWN"), ErrCodeInvalidCommand, CategorySyntax},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("Expected code %d, got %d", tt.code, tt.err.Code)
			}
			if tt.err.Category != tt.category {
				t.Errorf("Expected category %s, got %s", tt.category, tt.err.Category)
			}
		})
	}
}

func TestExecutionErrorConstructors(t *testing.T) {
	tests := []struct {
		name     string
		err      *FlyDBError
		code     ErrorCode
		category Category
	}{
		{"TableNotFound", TableNotFound("users"), ErrCodeTableNotFound, CategoryExecution},
		{"ColumnNotFound", ColumnNotFound("email", "users"), ErrCodeColumnNotFound, CategoryExecution},
		{"TypeMismatch", TypeMismatch("INT", "STRING", "age"), ErrCodeTypeMismatch, CategoryExecution},
		{"DuplicateKey", DuplicateKey("id=1", "users"), ErrCodeDuplicateKey, CategoryExecution},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("Expected code %d, got %d", tt.code, tt.err.Code)
			}
			if tt.err.Category != tt.category {
				t.Errorf("Expected category %s, got %s", tt.category, tt.err.Category)
			}
		})
	}
}

func TestErrorCategoryChecks(t *testing.T) {
	syntaxErr := NewSyntaxError("test")
	execErr := NewExecutionError("test")
	authErr := NewAuthError("test")

	if !IsSyntaxError(syntaxErr) {
		t.Error("Expected IsSyntaxError to return true for syntax error")
	}
	if IsSyntaxError(execErr) {
		t.Error("Expected IsSyntaxError to return false for execution error")
	}
	if !IsExecutionError(execErr) {
		t.Error("Expected IsExecutionError to return true for execution error")
	}
	if !IsAuthError(authErr) {
		t.Error("Expected IsAuthError to return true for auth error")
	}
}

func TestGetCode(t *testing.T) {
	err := TableNotFound("users")
	if GetCode(err) != ErrCodeTableNotFound {
		t.Errorf("Expected code %d, got %d", ErrCodeTableNotFound, GetCode(err))
	}

	regularErr := errors.New("regular error")
	if GetCode(regularErr) != 0 {
		t.Errorf("Expected code 0 for regular error, got %d", GetCode(regularErr))
	}
}

func TestFormatError(t *testing.T) {
	flyErr := NewSyntaxError("test error")
	formatted := FormatError(flyErr)
	if !strings.HasPrefix(formatted, "ERROR:") {
		t.Errorf("Expected formatted error to start with 'ERROR:', got: %s", formatted)
	}

	regularErr := errors.New("regular error")
	formatted = FormatError(regularErr)
	if !strings.Contains(formatted, "regular error") {
		t.Errorf("Expected formatted error to contain message, got: %s", formatted)
	}
}

