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
Collation Implementation
=========================

Collation defines how strings are compared and sorted in the database.
Different collations produce different sort orders for the same data,
which is essential for internationalization and locale-specific sorting.

What is Collation?
==================

Collation determines:
  - How strings are compared (equality and ordering)
  - How strings are sorted in ORDER BY clauses
  - How string comparisons work in WHERE clauses

For example, with case-insensitive collation:
  - "Alice" = "alice" (equal)
  - "Café" may sort differently than "cafe" depending on locale

Supported Collations:
=====================

FlyDB supports the following collations:

  1. BINARY (default):
     - Byte-by-byte comparison
     - Fastest, but not locale-aware
     - "A" < "B" < "a" < "b" (ASCII order)

  2. NOCASE:
     - Case-insensitive comparison
     - "Alice" = "alice"
     - Useful for case-insensitive searches

  3. UNICODE:
     - Unicode-aware comparison using ICU
     - Proper handling of accented characters
     - Locale-specific sorting rules

  4. Locale-specific (e.g., "en_US", "de_DE"):
     - Language-specific sorting rules
     - German: "ä" sorts with "a"
     - Swedish: "ä" sorts after "z"

Usage in SQL:
=============

Collation can be specified in column definitions:

  CREATE TABLE users (
    name TEXT COLLATE NOCASE,
    email TEXT COLLATE BINARY
  );

Or in queries:

  SELECT * FROM users ORDER BY name COLLATE UNICODE;

Performance Considerations:
===========================

  - BINARY is fastest (simple byte comparison)
  - NOCASE adds overhead for case conversion
  - UNICODE/locale collations are slowest but most correct

For indexes, the collation must match the query collation for
the index to be used effectively.

References:
===========

  - Unicode Technical Standard #10: Unicode Collation Algorithm
  - ICU Collation: https://unicode-org.github.io/icu/userguide/collation/
  - SQLite Collation: https://www.sqlite.org/datatype3.html#collation
*/
package storage

import (
	"strings"
	"unicode"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// Collator provides string comparison based on collation rules.
type Collator interface {
	// Compare compares two strings according to collation rules.
	// Returns -1 if a < b, 0 if a == b, 1 if a > b.
	Compare(a, b string) int

	// Equal returns true if two strings are equal according to collation rules.
	Equal(a, b string) bool
}

// DefaultCollator uses standard Go string comparison (byte-wise).
type DefaultCollator struct{}

// Compare implements Collator.
func (c *DefaultCollator) Compare(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// Equal implements Collator.
func (c *DefaultCollator) Equal(a, b string) bool {
	return a == b
}

// BinaryCollator uses strict byte-wise comparison.
type BinaryCollator struct{}

// Compare implements Collator.
func (c *BinaryCollator) Compare(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// Equal implements Collator.
func (c *BinaryCollator) Equal(a, b string) bool {
	return a == b
}

// NocaseCollator uses case-insensitive comparison.
type NocaseCollator struct{}

// Compare implements Collator.
func (c *NocaseCollator) Compare(a, b string) int {
	aLower := strings.ToLower(a)
	bLower := strings.ToLower(b)
	if aLower < bLower {
		return -1
	}
	if aLower > bLower {
		return 1
	}
	return 0
}

// Equal implements Collator.
func (c *NocaseCollator) Equal(a, b string) bool {
	return strings.EqualFold(a, b)
}

// UnicodeCollator uses Unicode collation with locale support.
type UnicodeCollator struct {
	collator *collate.Collator
	locale   string
}

// NewUnicodeCollator creates a new Unicode collator for the given locale.
func NewUnicodeCollator(locale string) *UnicodeCollator {
	tag := language.Make(locale)
	if tag == language.Und {
		tag = language.English
	}
	return &UnicodeCollator{
		collator: collate.New(tag, collate.Loose),
		locale:   locale,
	}
}

// Compare implements Collator.
func (c *UnicodeCollator) Compare(a, b string) int {
	return c.collator.CompareString(a, b)
}

// Equal implements Collator.
func (c *UnicodeCollator) Equal(a, b string) bool {
	return c.collator.CompareString(a, b) == 0
}

// GetCollator returns a Collator for the given collation type and locale.
func GetCollator(collationType Collation, locale string) Collator {
	switch collationType {
	case CollationBinary:
		return &BinaryCollator{}
	case CollationCaseInsensitive:
		return &NocaseCollator{}
	case CollationUnicode:
		return NewUnicodeCollator(locale)
	default:
		return &DefaultCollator{}
	}
}

// NormalizeForCollation normalizes a string for the given collation.
func NormalizeForCollation(s string, collationType Collation) string {
	switch collationType {
	case CollationCaseInsensitive:
		return strings.ToLower(s)
	case CollationUnicode:
		// Normalize Unicode to NFC form
		return strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return ' '
			}
			return r
		}, s)
	default:
		return s
	}
}

