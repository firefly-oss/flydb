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

package sql

import (
	"testing"
)

func TestLexerKeywords(t *testing.T) {
	input := "SELECT FROM WHERE INSERT INTO VALUES CREATE TABLE"
	lexer := NewLexer(input)

	expected := []struct {
		tokenType TokenType
		value     string
	}{
		{TokenKeyword, "SELECT"},
		{TokenKeyword, "FROM"},
		{TokenKeyword, "WHERE"},
		{TokenKeyword, "INSERT"},
		{TokenKeyword, "INTO"},
		{TokenKeyword, "VALUES"},
		{TokenKeyword, "CREATE"},
		{TokenKeyword, "TABLE"},
		{TokenEOF, ""},
	}

	for _, exp := range expected {
		tok := lexer.NextToken()
		if tok.Type != exp.tokenType {
			t.Errorf("Expected token type %v, got %v", exp.tokenType, tok.Type)
		}
		if tok.Value != exp.value {
			t.Errorf("Expected value '%s', got '%s'", exp.value, tok.Value)
		}
	}
}

func TestLexerIdentifiers(t *testing.T) {
	input := "users user_name table1 users.id"
	lexer := NewLexer(input)

	expected := []string{"users", "user_name", "table1", "users.id"}

	for _, exp := range expected {
		tok := lexer.NextToken()
		if tok.Type != TokenIdent {
			t.Errorf("Expected TokenIdent, got %v", tok.Type)
		}
		if tok.Value != exp {
			t.Errorf("Expected '%s', got '%s'", exp, tok.Value)
		}
	}
}

func TestLexerNumbers(t *testing.T) {
	input := "123 456 789"
	lexer := NewLexer(input)

	expected := []string{"123", "456", "789"}

	for _, exp := range expected {
		tok := lexer.NextToken()
		if tok.Type != TokenNumber {
			t.Errorf("Expected TokenNumber, got %v", tok.Type)
		}
		if tok.Value != exp {
			t.Errorf("Expected '%s', got '%s'", exp, tok.Value)
		}
	}
}

func TestLexerStrings(t *testing.T) {
	input := "'hello' 'world' 'user@example.com'"
	lexer := NewLexer(input)

	expected := []string{"hello", "world", "user@example.com"}

	for _, exp := range expected {
		tok := lexer.NextToken()
		if tok.Type != TokenString {
			t.Errorf("Expected TokenString, got %v", tok.Type)
		}
		if tok.Value != exp {
			t.Errorf("Expected '%s', got '%s'", exp, tok.Value)
		}
	}
}

func TestLexerSymbols(t *testing.T) {
	input := "( ) , ="
	lexer := NewLexer(input)

	expected := []struct {
		tokenType TokenType
		value     string
	}{
		{TokenLParen, "("},
		{TokenRParen, ")"},
		{TokenComma, ","},
		{TokenEqual, "="},
		{TokenEOF, ""},
	}

	for _, exp := range expected {
		tok := lexer.NextToken()
		if tok.Type != exp.tokenType {
			t.Errorf("Expected token type %v, got %v", exp.tokenType, tok.Type)
		}
	}
}

func TestLexerParameterPlaceholders(t *testing.T) {
	input := "$1 $2 $10"
	lexer := NewLexer(input)

	expected := []string{"$1", "$2", "$10"}

	for _, exp := range expected {
		tok := lexer.NextToken()
		if tok.Type != TokenIdent {
			t.Errorf("Expected TokenIdent for placeholder, got %v", tok.Type)
		}
		if tok.Value != exp {
			t.Errorf("Expected '%s', got '%s'", exp, tok.Value)
		}
	}
}

func TestLexerCompleteQuery(t *testing.T) {
	input := "SELECT name, age FROM users WHERE id = 1"
	lexer := NewLexer(input)

	expected := []struct {
		tokenType TokenType
		value     string
	}{
		{TokenKeyword, "SELECT"},
		{TokenIdent, "name"},
		{TokenComma, ","},
		{TokenIdent, "age"},
		{TokenKeyword, "FROM"},
		{TokenIdent, "users"},
		{TokenKeyword, "WHERE"},
		{TokenIdent, "id"},
		{TokenEqual, "="},
		{TokenNumber, "1"},
		{TokenEOF, ""},
	}

	for i, exp := range expected {
		tok := lexer.NextToken()
		if tok.Type != exp.tokenType {
			t.Errorf("Token %d: Expected type %v, got %v", i, exp.tokenType, tok.Type)
		}
		if tok.Value != exp.value {
			t.Errorf("Token %d: Expected value '%s', got '%s'", i, exp.value, tok.Value)
		}
	}
}

