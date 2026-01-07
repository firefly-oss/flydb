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

package protocol

import (
	"testing"
)

func TestQueryMessageEncodeDecode(t *testing.T) {
	original := &QueryMessage{Query: "SELECT * FROM users"}
	
	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	
	decoded, err := DecodeQueryMessage(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	
	if decoded.Query != original.Query {
		t.Errorf("Expected '%s', got '%s'", original.Query, decoded.Query)
	}
}

func TestQueryResultMessageEncodeDecode(t *testing.T) {
	original := &QueryResultMessage{
		Success:  true,
		Message:  "OK",
		Columns:  []string{"id", "name"},
		Rows:     [][]interface{}{{1, "Alice"}, {2, "Bob"}},
		RowCount: 2,
	}
	
	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	
	decoded, err := DecodeQueryResultMessage(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	
	if decoded.Success != original.Success {
		t.Errorf("Success mismatch")
	}
	if decoded.RowCount != original.RowCount {
		t.Errorf("RowCount mismatch: expected %d, got %d", original.RowCount, decoded.RowCount)
	}
}

func TestErrorMessageEncodeDecode(t *testing.T) {
	original := &ErrorMessage{Code: 404, Message: "Table not found"}
	
	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	
	decoded, err := DecodeErrorMessage(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	
	if decoded.Code != original.Code {
		t.Errorf("Code mismatch: expected %d, got %d", original.Code, decoded.Code)
	}
	if decoded.Message != original.Message {
		t.Errorf("Message mismatch")
	}
}

func TestAuthMessageEncodeDecode(t *testing.T) {
	original := &AuthMessage{Username: "admin", Password: "secret"}
	
	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	
	decoded, err := DecodeAuthMessage(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	
	if decoded.Username != original.Username {
		t.Errorf("Username mismatch")
	}
	if decoded.Password != original.Password {
		t.Errorf("Password mismatch")
	}
}

func TestPrepareMessageEncodeDecode(t *testing.T) {
	original := &PrepareMessage{Name: "get_user", Query: "SELECT * FROM users WHERE id = $1"}
	
	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	
	decoded, err := DecodePrepareMessage(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	
	if decoded.Name != original.Name {
		t.Errorf("Name mismatch")
	}
	if decoded.Query != original.Query {
		t.Errorf("Query mismatch")
	}
}

func TestBinaryEncoderDecoder(t *testing.T) {
	encoder := NewBinaryEncoder()
	
	encoder.WriteString("hello")
	encoder.WriteInt64(12345)
	encoder.WriteFloat64(3.14159)
	encoder.WriteBool(true)
	encoder.WriteBytes([]byte{1, 2, 3})
	
	decoder := NewBinaryDecoder(encoder.Bytes())
	
	str, err := decoder.ReadString()
	if err != nil || str != "hello" {
		t.Errorf("String mismatch: %v, %s", err, str)
	}
	
	i64, err := decoder.ReadInt64()
	if err != nil || i64 != 12345 {
		t.Errorf("Int64 mismatch: %v, %d", err, i64)
	}
	
	f64, err := decoder.ReadFloat64()
	if err != nil || f64 != 3.14159 {
		t.Errorf("Float64 mismatch: %v, %f", err, f64)
	}
	
	b, err := decoder.ReadBool()
	if err != nil || !b {
		t.Errorf("Bool mismatch: %v, %v", err, b)
	}
	
	bytes, err := decoder.ReadBytes()
	if err != nil || len(bytes) != 3 {
		t.Errorf("Bytes mismatch: %v, %v", err, bytes)
	}
}

