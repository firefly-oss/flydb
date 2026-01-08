/*
 * Copyright (c) 2026 Firefly Software Solutions Inc.
 * Licensed under the Apache License, Version 2.0
 */

package storage

import (
	"os"
	"testing"
	"time"
)

// setupTestEngine creates a test storage engine and returns a cleanup function.
// This is used by tests that need a storage engine.
func setupTestEngine(t *testing.T) (Engine, func()) {
	t.Helper()

	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "flydb-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	config := StorageConfig{
		DataDir:            tmpDir,
		BufferPoolSize:     256, // Small buffer pool for tests
		CheckpointInterval: 0,   // Disable checkpoints for tests
	}

	engine, err := NewStorageEngine(config)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create storage engine: %v", err)
	}

	cleanup := func() {
		engine.Close()
		os.RemoveAll(tmpDir)
	}

	return engine, cleanup
}

// setupTestEngineWithEncryption creates a test storage engine with encryption.
func setupTestEngineWithEncryption(t *testing.T, passphrase string) (Engine, func()) {
	t.Helper()

	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "flydb-test-enc-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	config := StorageConfig{
		DataDir:            tmpDir,
		BufferPoolSize:     256, // Small buffer pool for tests
		CheckpointInterval: 0,   // Disable checkpoints for tests
		Encryption: EncryptionConfig{
			Enabled:    true,
			Passphrase: passphrase,
		},
	}

	engine, err := NewStorageEngine(config)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create storage engine: %v", err)
	}

	cleanup := func() {
		engine.Close()
		os.RemoveAll(tmpDir)
	}

	return engine, cleanup
}

// setupTestEngineWithPath creates a test storage engine at a specific path.
// Returns the engine, path, and cleanup function.
func setupTestEngineWithPath(t *testing.T) (Engine, string, func()) {
	t.Helper()

	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "flydb-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	config := StorageConfig{
		DataDir:            tmpDir,
		BufferPoolSize:     256, // Small buffer pool for tests
		CheckpointInterval: time.Second * 60,
	}

	engine, err := NewStorageEngine(config)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create storage engine: %v", err)
	}

	cleanup := func() {
		engine.Close()
		os.RemoveAll(tmpDir)
	}

	return engine, tmpDir, cleanup
}

