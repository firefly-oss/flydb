package sql

import (
	"os"
	"testing"

	"flydb/internal/auth"
	"flydb/internal/storage"
)

func setupTestExecutor(t *testing.T) (*Executor, func()) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "flydb_sql_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create a temporary storage
	kv, err := storage.NewKVStore(tmpDir + "/test.db")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create KVStore: %v", err)
	}

	authMgr := auth.NewAuthManager(kv)
	exec := NewExecutor(kv, authMgr)

	// Create a test table
	createStmt := &CreateTableStmt{
		TableName: "users",
		Columns: []ColumnDef{
			{Name: "id", Type: "INT"},
			{Name: "name", Type: "TEXT"},
			{Name: "email", Type: "TEXT"},
		},
	}
	_, err = exec.Execute(createStmt)
	if err != nil {
		kv.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create test table: %v", err)
	}

	cleanup := func() {
		kv.Close()
		os.RemoveAll(tmpDir)
	}

	return exec, cleanup
}

func TestPreparedStatementPrepare(t *testing.T) {
	exec, cleanup := setupTestExecutor(t)
	defer cleanup()

	// Test PREPARE statement
	prepareStmt := &PrepareStmt{
		Name:  "get_user",
		Query: "SELECT * FROM users WHERE id = $1",
	}

	result, err := exec.Execute(prepareStmt)
	if err != nil {
		t.Fatalf("PREPARE failed: %v", err)
	}

	if result != "PREPARE OK" {
		t.Errorf("Expected 'PREPARE OK', got '%s'", result)
	}

	// Verify the statement was stored
	mgr := exec.GetPreparedStatementManager()
	stmt, exists := mgr.Get("get_user")
	if !exists {
		t.Fatal("Prepared statement not found")
	}

	if stmt.ParamCount != 1 {
		t.Errorf("Expected 1 parameter, got %d", stmt.ParamCount)
	}
}

func TestPreparedStatementExecute(t *testing.T) {
	exec, cleanup := setupTestExecutor(t)
	defer cleanup()

	// Insert test data
	insertStmt := &InsertStmt{
		TableName: "users",
		Values:    []string{"1", "Alice", "alice@example.com"},
	}
	_, err := exec.Execute(insertStmt)
	if err != nil {
		t.Fatalf("INSERT failed: %v", err)
	}

	// Prepare a statement with specific columns (SELECT * is not fully supported)
	prepareStmt := &PrepareStmt{
		Name:  "get_user",
		Query: "SELECT name FROM users WHERE id = $1",
	}
	_, err = exec.Execute(prepareStmt)
	if err != nil {
		t.Fatalf("PREPARE failed: %v", err)
	}

	// Execute the prepared statement
	executeStmt := &ExecuteStmt{
		Name:   "get_user",
		Params: []string{"1"},
	}
	result, err := exec.Execute(executeStmt)
	if err != nil {
		t.Fatalf("EXECUTE failed: %v", err)
	}

	// Should return the user's name with header and row count
	if result == "" {
		t.Error("Expected non-empty result")
	}
	expected := "name\nAlice\n(1 rows)"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestPreparedStatementDeallocate(t *testing.T) {
	exec, cleanup := setupTestExecutor(t)
	defer cleanup()

	// Prepare a statement
	prepareStmt := &PrepareStmt{
		Name:  "test_stmt",
		Query: "SELECT * FROM users",
	}
	_, err := exec.Execute(prepareStmt)
	if err != nil {
		t.Fatalf("PREPARE failed: %v", err)
	}

	// Deallocate the statement
	deallocStmt := &DeallocateStmt{
		Name: "test_stmt",
	}
	result, err := exec.Execute(deallocStmt)
	if err != nil {
		t.Fatalf("DEALLOCATE failed: %v", err)
	}

	if result != "DEALLOCATE OK" {
		t.Errorf("Expected 'DEALLOCATE OK', got '%s'", result)
	}

	// Verify the statement was removed
	mgr := exec.GetPreparedStatementManager()
	_, exists := mgr.Get("test_stmt")
	if exists {
		t.Error("Prepared statement should have been deallocated")
	}
}

func TestPreparedStatementDuplicateName(t *testing.T) {
	exec, cleanup := setupTestExecutor(t)
	defer cleanup()

	// Prepare a statement
	prepareStmt := &PrepareStmt{
		Name:  "duplicate",
		Query: "SELECT * FROM users",
	}
	_, err := exec.Execute(prepareStmt)
	if err != nil {
		t.Fatalf("First PREPARE failed: %v", err)
	}

	// Try to prepare with the same name
	_, err = exec.Execute(prepareStmt)
	if err == nil {
		t.Error("Expected error for duplicate prepared statement name")
	}
}

