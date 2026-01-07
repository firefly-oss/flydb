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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Port != 8888 {
		t.Errorf("Expected default port 8888, got %d", cfg.Port)
	}
	if cfg.BinaryPort != 8889 {
		t.Errorf("Expected default binary port 8889, got %d", cfg.BinaryPort)
	}
	if cfg.ReplPort != 9999 {
		t.Errorf("Expected default replication port 9999, got %d", cfg.ReplPort)
	}
	if cfg.Role != "standalone" {
		t.Errorf("Expected default role 'standalone', got '%s'", cfg.Role)
	}
	if cfg.DBPath != "flydb.wal" {
		t.Errorf("Expected default db_path 'flydb.wal', got '%s'", cfg.DBPath)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected default log_level 'info', got '%s'", cfg.LogLevel)
	}
	if cfg.LogJSON != false {
		t.Errorf("Expected default log_json false, got %v", cfg.LogJSON)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "valid standalone config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "valid master config",
			cfg: &Config{
				Port:       8888,
				BinaryPort: 8889,
				ReplPort:   9999,
				Role:       "master",
				DBPath:     "test.wal",
				LogLevel:   "info",
			},
			wantErr: false,
		},
		{
			name: "valid slave config",
			cfg: &Config{
				Port:       8888,
				BinaryPort: 8889,
				ReplPort:   9999,
				Role:       "slave",
				MasterAddr: "localhost:9999",
				DBPath:     "test.wal",
				LogLevel:   "info",
			},
			wantErr: false,
		},
		{
			name: "invalid port - zero",
			cfg: &Config{
				Port:       0,
				BinaryPort: 8889,
				ReplPort:   9999,
				Role:       "standalone",
				DBPath:     "test.wal",
				LogLevel:   "info",
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			cfg: &Config{
				Port:       70000,
				BinaryPort: 8889,
				ReplPort:   9999,
				Role:       "standalone",
				DBPath:     "test.wal",
				LogLevel:   "info",
			},
			wantErr: true,
		},
		{
			name: "port conflict",
			cfg: &Config{
				Port:       8888,
				BinaryPort: 8888,
				ReplPort:   9999,
				Role:       "standalone",
				DBPath:     "test.wal",
				LogLevel:   "info",
			},
			wantErr: true,
		},
		{
			name: "invalid role",
			cfg: &Config{
				Port:       8888,
				BinaryPort: 8889,
				ReplPort:   9999,
				Role:       "invalid",
				DBPath:     "test.wal",
				LogLevel:   "info",
			},
			wantErr: true,
		},
		{
			name: "slave without master_addr",
			cfg: &Config{
				Port:       8888,
				BinaryPort: 8889,
				ReplPort:   9999,
				Role:       "slave",
				MasterAddr: "",
				DBPath:     "test.wal",
				LogLevel:   "info",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			cfg: &Config{
				Port:       8888,
				BinaryPort: 8889,
				ReplPort:   9999,
				Role:       "standalone",
				DBPath:     "test.wal",
				LogLevel:   "invalid",
			},
			wantErr: true,
		},
		{
			name: "empty db_path",
			cfg: &Config{
				Port:       8888,
				BinaryPort: 8889,
				ReplPort:   9999,
				Role:       "standalone",
				DBPath:     "",
				LogLevel:   "info",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir, err := os.MkdirTemp("", "flydb_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configContent := `# Test configuration
role = "master"
port = 9000
binary_port = 9001
replication_port = 9002
db_path = "/tmp/test.wal"
log_level = "debug"
log_json = true
master_addr = "localhost:9999"
`

	configPath := filepath.Join(tmpDir, "flydb.conf")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	mgr := NewManager()
	if err := mgr.LoadFromFile(configPath); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	cfg := mgr.Get()

	if cfg.Role != "master" {
		t.Errorf("Expected role 'master', got '%s'", cfg.Role)
	}
	if cfg.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", cfg.Port)
	}
	if cfg.BinaryPort != 9001 {
		t.Errorf("Expected binary_port 9001, got %d", cfg.BinaryPort)
	}
	if cfg.ReplPort != 9002 {
		t.Errorf("Expected replication_port 9002, got %d", cfg.ReplPort)
	}
	if cfg.DBPath != "/tmp/test.wal" {
		t.Errorf("Expected db_path '/tmp/test.wal', got '%s'", cfg.DBPath)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected log_level 'debug', got '%s'", cfg.LogLevel)
	}
	if cfg.LogJSON != true {
		t.Errorf("Expected log_json true, got %v", cfg.LogJSON)
	}
	if cfg.ConfigFile != configPath {
		t.Errorf("Expected ConfigFile '%s', got '%s'", configPath, cfg.ConfigFile)
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Save original env vars
	origPort := os.Getenv(EnvPort)
	origRole := os.Getenv(EnvRole)
	origLogLevel := os.Getenv(EnvLogLevel)
	origLogJSON := os.Getenv(EnvLogJSON)
	origAdminPass := os.Getenv(EnvAdminPassword)

	// Restore env vars after test
	defer func() {
		os.Setenv(EnvPort, origPort)
		os.Setenv(EnvRole, origRole)
		os.Setenv(EnvLogLevel, origLogLevel)
		os.Setenv(EnvLogJSON, origLogJSON)
		os.Setenv(EnvAdminPassword, origAdminPass)
	}()

	// Set test env vars
	os.Setenv(EnvPort, "7777")
	os.Setenv(EnvRole, "master")
	os.Setenv(EnvLogLevel, "debug")
	os.Setenv(EnvLogJSON, "true")
	os.Setenv(EnvAdminPassword, "testpassword")

	mgr := NewManager()
	mgr.LoadFromEnv()

	cfg := mgr.Get()

	if cfg.Port != 7777 {
		t.Errorf("Expected port 7777 from env, got %d", cfg.Port)
	}
	if cfg.Role != "master" {
		t.Errorf("Expected role 'master' from env, got '%s'", cfg.Role)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected log_level 'debug' from env, got '%s'", cfg.LogLevel)
	}
	if cfg.LogJSON != true {
		t.Errorf("Expected log_json true from env, got %v", cfg.LogJSON)
	}
	if cfg.AdminPassword != "testpassword" {
		t.Errorf("Expected admin_password 'testpassword' from env, got '%s'", cfg.AdminPassword)
	}
}

func TestConfigPrecedence(t *testing.T) {
	// Create a temporary config file
	tmpDir, err := os.MkdirTemp("", "flydb_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Config file sets port to 9000
	configContent := `port = 9000
role = "standalone"
db_path = "test.wal"
log_level = "info"
`
	configPath := filepath.Join(tmpDir, "flydb.conf")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Save and set env var to override port to 7777
	origPort := os.Getenv(EnvPort)
	defer os.Setenv(EnvPort, origPort)
	os.Setenv(EnvPort, "7777")

	mgr := NewManager()
	if err := mgr.LoadFromFile(configPath); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	mgr.LoadFromEnv()

	cfg := mgr.Get()

	// Env var should override file value
	if cfg.Port != 7777 {
		t.Errorf("Expected port 7777 (env override), got %d", cfg.Port)
	}
}

func TestToTOML(t *testing.T) {
	cfg := &Config{
		Port:       8888,
		BinaryPort: 8889,
		ReplPort:   9999,
		Role:       "master",
		MasterAddr: "localhost:9999",
		DBPath:     "/var/lib/flydb/data.wal",
		LogLevel:   "info",
		LogJSON:    false,
	}

	toml := cfg.ToTOML()

	// Check that key values are present
	if !contains(toml, "role = \"master\"") {
		t.Error("TOML output missing role")
	}
	if !contains(toml, "port = 8888") {
		t.Error("TOML output missing port")
	}
	if !contains(toml, "binary_port = 8889") {
		t.Error("TOML output missing binary_port")
	}
	if !contains(toml, "db_path = \"/var/lib/flydb/data.wal\"") {
		t.Error("TOML output missing db_path")
	}
}

func TestSaveToFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "flydb_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig()
	cfg.Port = 7777
	cfg.Role = "master"

	configPath := filepath.Join(tmpDir, "subdir", "flydb.conf")
	if err := cfg.SaveToFile(configPath); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load it back and verify
	mgr := NewManager()
	if err := mgr.LoadFromFile(configPath); err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	loaded := mgr.Get()
	if loaded.Port != 7777 {
		t.Errorf("Expected port 7777, got %d", loaded.Port)
	}
	if loaded.Role != "master" {
		t.Errorf("Expected role 'master', got '%s'", loaded.Role)
	}
}

func TestReload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "flydb_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initial config
	configContent := `port = 9000
role = "standalone"
db_path = "test.wal"
log_level = "info"
`
	configPath := filepath.Join(tmpDir, "flydb.conf")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	mgr := NewManager()
	if err := mgr.LoadFromFile(configPath); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	cfg := mgr.Get()
	if cfg.Port != 9000 {
		t.Errorf("Expected initial port 9000, got %d", cfg.Port)
	}

	// Track reload callback
	reloadCalled := false
	mgr.OnReload(func(c *Config) {
		reloadCalled = true
	})

	// Update config file
	newContent := `port = 8000
role = "standalone"
db_path = "test.wal"
log_level = "debug"
`
	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// Reload
	if err := mgr.Reload(); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	cfg = mgr.Get()
	if cfg.Port != 8000 {
		t.Errorf("Expected reloaded port 8000, got %d", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected reloaded log_level 'debug', got '%s'", cfg.LogLevel)
	}
	if !reloadCalled {
		t.Error("Reload callback was not called")
	}
}

func TestGlobalManager(t *testing.T) {
	mgr := Global()
	if mgr == nil {
		t.Error("Global() returned nil")
	}

	// Should return the same instance
	mgr2 := Global()
	if mgr != mgr2 {
		t.Error("Global() returned different instances")
	}
}

func TestConfigString(t *testing.T) {
	cfg := DefaultConfig()
	str := cfg.String()

	if !contains(str, "Role:") {
		t.Error("String() missing Role")
	}
	if !contains(str, "Port:") {
		t.Error("String() missing Port")
	}
	if !contains(str, "standalone") {
		t.Error("String() missing role value")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

