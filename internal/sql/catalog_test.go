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
	"os"
	"testing"
	"time"

	"flydb/internal/storage"
)

func setupCatalogTest(t *testing.T) (*Catalog, string, func()) {
	tmpDir, err := os.MkdirTemp("", "flydb_catalog_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	config := storage.StorageConfig{
		DataDir:            tmpDir,
		BufferPoolSize:     256,
		CheckpointInterval: 0,
	}
	store, err := storage.NewStorageEngine(config)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create storage engine: %v", err)
	}

	catalog := NewCatalog(store)

	cleanup := func() {
		store.Close()
		time.Sleep(10 * time.Millisecond)
		os.RemoveAll(tmpDir)
	}

	return catalog, tmpDir, cleanup
}

func TestCatalogCreateTable(t *testing.T) {
	catalog, _, cleanup := setupCatalogTest(t)
	defer cleanup()

	cols := []ColumnDef{
		{Name: "id", Type: "INT"},
		{Name: "name", Type: "TEXT"},
	}

	err := catalog.CreateTable("users", cols)
	if err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	// Verify table exists
	schema, ok := catalog.GetTable("users")
	if !ok {
		t.Fatal("Table not found after creation")
	}
	if schema.Name != "users" {
		t.Errorf("Expected table name 'users', got '%s'", schema.Name)
	}
	if len(schema.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(schema.Columns))
	}
}

func TestCatalogDuplicateTable(t *testing.T) {
	catalog, _, cleanup := setupCatalogTest(t)
	defer cleanup()

	cols := []ColumnDef{{Name: "id", Type: "INT"}}

	err := catalog.CreateTable("users", cols)
	if err != nil {
		t.Fatalf("First CreateTable failed: %v", err)
	}

	// Try to create duplicate
	err = catalog.CreateTable("users", cols)
	if err == nil {
		t.Error("Expected error when creating duplicate table")
	}
}

func TestCatalogGetNonExistentTable(t *testing.T) {
	catalog, _, cleanup := setupCatalogTest(t)
	defer cleanup()

	_, ok := catalog.GetTable("nonexistent")
	if ok {
		t.Error("Expected table not found")
	}
}

func TestCatalogPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "flydb_catalog_persist_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := storage.StorageConfig{
		DataDir:            tmpDir,
		BufferPoolSize:     256,
		CheckpointInterval: 0,
	}

	// Create store and catalog, add a table
	store1, err := storage.NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	catalog1 := NewCatalog(store1)
	catalog1.CreateTable("persistent_table", []ColumnDef{{Name: "data", Type: "TEXT"}})
	store1.Close()
	time.Sleep(10 * time.Millisecond)

	// Reopen store and catalog, verify table persisted
	store2, err := storage.NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to reopen storage engine: %v", err)
	}
	defer store2.Close()

	catalog2 := NewCatalog(store2)
	schema, ok := catalog2.GetTable("persistent_table")
	if !ok {
		t.Fatal("Table not found after reopen")
	}
	if schema.Name != "persistent_table" {
		t.Errorf("Expected 'persistent_table', got '%s'", schema.Name)
	}
}

func TestCatalogMultipleTables(t *testing.T) {
	catalog, _, cleanup := setupCatalogTest(t)
	defer cleanup()

	// Create multiple tables
	catalog.CreateTable("users", []ColumnDef{{Name: "id", Type: "INT"}})
	catalog.CreateTable("orders", []ColumnDef{{Name: "id", Type: "INT"}, {Name: "user_id", Type: "INT"}})
	catalog.CreateTable("products", []ColumnDef{{Name: "id", Type: "INT"}, {Name: "name", Type: "TEXT"}})

	// Verify all tables exist
	tables := []string{"users", "orders", "products"}
	for _, name := range tables {
		_, ok := catalog.GetTable(name)
		if !ok {
			t.Errorf("Table '%s' not found", name)
		}
	}
}

