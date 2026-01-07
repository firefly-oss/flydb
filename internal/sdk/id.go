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

package sdk

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
)

// idCounter is used for generating unique IDs.
var idCounter uint64

// generateID generates a unique ID.
// Format: <prefix>-<counter>-<random_hex>
func generateID(prefix string) string {
	counter := atomic.AddUint64(&idCounter, 1)
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	return fmt.Sprintf("%s-%d-%s", prefix, counter, hex.EncodeToString(randomBytes))
}

// GenerateSessionID generates a unique session ID.
func GenerateSessionID() string {
	return generateID("sess")
}

// GenerateCursorID generates a unique cursor ID.
func GenerateCursorID() string {
	return generateID("cur")
}

// GenerateTransactionID generates a unique transaction ID.
func GenerateTransactionID() string {
	return generateID("tx")
}

// GenerateStatementID generates a unique prepared statement ID.
func GenerateStatementID() string {
	return generateID("stmt")
}

// GenerateResultSetID generates a unique result set ID.
func GenerateResultSetID() string {
	return generateID("rs")
}

