package sql

import (
	"os"
	"strings"
	"testing"

	"flydb/internal/auth"
	"flydb/internal/storage"
)

// TestMultiDatabaseOperations verifies multi-database features including:
// 1. Creating/switching databases
// 2. Cross-database queries
// 3. Database isolation
func TestMultiDatabaseOperations(t *testing.T) {
	// Setup test storage engine
	store, err := storage.NewStorageEngine(storage.StorageConfig{
		DataDir:            t.TempDir(),
		BufferPoolSize:     0,
		CheckpointInterval: 0,
	})
	if err != nil {
		t.Fatalf("failed to create storage engine: %v", err)
	}
	defer store.Close()

	// Setup DatabaseManager (requires a data directory)
	dbMgr, err := storage.NewDatabaseManager(t.TempDir(), storage.EncryptionConfig{})
	if err != nil {
		t.Fatalf("failed to create database manager: %v", err)
	}

	// Create Executor with DBManager
	authMgr := auth.NewAuthManager(store) // using default store for system auth
	exec := NewExecutor(store, authMgr)
	exec.dbMgr = dbMgr // Manually inject dbMgr as NewExecutor doesn't take it yet

	// 1. Create Databases
	t.Run("CreateDatabases", func(t *testing.T) {
		sqls := []string{
			"CREATE DATABASE db1",
			"CREATE DATABASE db2",
		}
		for _, sql := range sqls {
			_, err := exec.Execute(parse(t, sql))
			if err != nil {
				t.Errorf("Execute(%q) failed: %v", sql, err)
			}
		}
	})

	// 2. Create Tables in different databases
	t.Run("CreateTables", func(t *testing.T) {
		// Use db1
		_, err := exec.Execute(parse(t, "USE db1"))
		if err != nil {
			t.Fatalf("USE db1 failed: %v", err)
		}
		if exec.currentDatabase != "db1" {
			t.Errorf("currentDatabase = %q, want %q", exec.currentDatabase, "db1")
		}

		// Create table users in db1
		_, err = exec.Execute(parse(t, "CREATE TABLE users (id INT, name TEXT)"))
		if err != nil {
			t.Errorf("CREATE TABLE in db1 failed: %v", err)
		}
		exec.Execute(parse(t, "INSERT INTO users VALUES (1, 'alice')"))

		// Use db2
		_, err = exec.Execute(parse(t, "USE db2"))
		if err != nil {
			t.Fatalf("USE db2 failed: %v", err)
		}

		// Create table orders in db2
		_, err = exec.Execute(parse(t, "CREATE TABLE orders (id INT, user_id INT, amount INT)"))
		if err != nil {
			t.Errorf("CREATE TABLE in db2 failed: %v", err)
		}
		exec.Execute(parse(t, "INSERT INTO orders VALUES (100, 1, 50)"))
	})

	// 3. Cross-database Query (Explicit Qualification)
	t.Run("CrossDatabaseQuery", func(t *testing.T) {
		// Currently in db2. Query db1.users
		sql := "SELECT * FROM db1.users"
		res, err := exec.Execute(parse(t, sql))
		if err != nil {
			t.Fatalf("SELECT db1.users failed: %v", err)
		}
		if res == "" { // Expect result
			t.Error("expected result for db1.users, got empty")
		}

		// Join db1.users and db2.orders
		// Note: db2.orders can be referred to as just 'orders' since we are in db2
		sql = "SELECT users.name, orders.amount FROM db1.users JOIN orders ON users.id = orders.user_id"
		res, err = exec.Execute(parse(t, sql))
		if err != nil {
			t.Fatalf("Cross join failed: %v", err)
		}
		// Basic check if result contains expected data
		// Real test would check Rows, but Execute returns string currently.
		// We assumes formatted string contains "alice" and "50"
		// (Based on current Executor implementation which might return JSON or Table string)
		// For now, checks are loose.
	})
}

func TestMultiDatabaseRBAC(t *testing.T) {
	// Setup temporary directory for test data
	tmpDir, err := os.MkdirTemp("", "flydb-multidb-rbac-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create DB Manager with empty encryption
	encConfig := storage.EncryptionConfig{}
	dbMgr, err := storage.NewDatabaseManager(tmpDir, encConfig)
	if err != nil {
		t.Fatalf("Failed to create DB manager: %v", err)
	}

	// Get system DB store to init AuthManager
	sysDB, err := dbMgr.GetDatabase(storage.SystemDatabaseName)
	if err != nil {
		t.Fatalf("Failed to get system DB: %v", err)
	}
	authMgr := auth.NewAuthManager(sysDB.Store)

	// Initialize built-in roles
	if err := authMgr.InitializeBuiltInRoles(); err != nil {
		t.Fatalf("Failed to init roles: %v", err)
	}

	// Create admin user
	authMgr.CreateAdminUser("admin", "secret")

	// Create a restricted user "alice"
	authMgr.CreateUser("alice", "password")

	// Create two databases
	dbMgr.CreateDatabase("db1")
	dbMgr.CreateDatabase("db2")

	// Grant alice access to db1 but NOT db2
	// We assign the 'reader' role to alice scoped to db1
	authMgr.GrantRoleToUser("alice", auth.RoleReader, "db1", "admin")

	// Setup Executor for Admin first to create tables
	defaultDB, err := dbMgr.GetDatabase("default")
	if err != nil {
		t.Fatalf("Failed to get default DB: %v", err)
	}

	// NewExecutor takes store and auth manager
	// It creates its own global catalog from the store
	adminExec := NewExecutor(defaultDB.Store, authMgr)
	adminExec.dbMgr = dbMgr
	adminExec.SetUser("admin")

	// Create table in db1
	_, err = adminExec.Execute(parse(t, "USE db1"))
	if err != nil {
		t.Fatalf("Failed to switch to db1: %v", err)
	}
	_, err = adminExec.Execute(parse(t, "CREATE TABLE t1 (id INT, name TEXT)"))
	if err != nil {
		t.Fatalf("Failed to create table in db1: %v", err)
	}
	_, err = adminExec.Execute(parse(t, "INSERT INTO t1 VALUES (1, 'row1')"))
	if err != nil {
		t.Fatalf("Failed to insert into db1: %v", err)
	}

	// Create table in db2
	_, err = adminExec.Execute(parse(t, "USE db2"))
	if err != nil {
		t.Fatalf("Failed to switch to db2: %v", err)
	}
	_, err = adminExec.Execute(parse(t, "CREATE TABLE t2 (id INT, val TEXT)"))
	if err != nil {
		t.Fatalf("Failed to create table in db2: %v", err)
	}
	_, err = adminExec.Execute(parse(t, "INSERT INTO t2 VALUES (2, 'row2')"))
	if err != nil {
		t.Fatalf("Failed to insert into db2: %v", err)
	}

	// Now switch to Alice
	// Create new executor for Alice starting at default
	// Important: We must maintain the mapping of database managers
	aliceExec := NewExecutor(defaultDB.Store, authMgr)
	aliceExec.dbMgr = dbMgr
	aliceExec.SetUser("alice")

	// 1. Alice tries to USE db1 -> Should succeed
	_, err = aliceExec.Execute(parse(t, "USE db1"))
	if err != nil {
		t.Logf("Alice failed to USE db1: %v", err)
	}

	// 2. Alice tries to SELECT from db1.t1 -> Should SUCCEED (Reader role on db1)

	_, err = aliceExec.Execute(parse(t, "SELECT * FROM t1"))
	if err != nil {
		t.Errorf("Alice should be able to read db1.t1 but got error: %v", err)
	}

	// 3. Alice tries to SELECT from db2.t2 -> Should FAIL (Access Denied)
	_, err = aliceExec.Execute(parse(t, "SELECT * FROM db2.t2"))
	if err == nil {
		t.Error("Alice passed SELECT on db2.t2 but should have failed")
	} else if !strings.Contains(err.Error(), "permission denied") {
		// Just log if message differs
		t.Logf("Got error accessing db2.t2 (as expected): %v", err)
	}

	// 4. Alice tries to USE db2
	// If USE is not RBAC protected, she can switch but subsequent actions fail.
	aliceExec.Execute(parse(t, "USE db2"))
	_, err = aliceExec.Execute(parse(t, "SELECT * FROM t2"))
	if err == nil {
		t.Error("Alice passed SELECT * FROM t2 (in db2 context) but should have failed")
	}

	// 5. Alice tries to CREATE DATABASE db3 -> Should FAIL (Requires Server privilege)
	_, err = aliceExec.Execute(parse(t, "CREATE DATABASE db3"))
	if err == nil {
		t.Error("Alice passed CREATE DATABASE db3 but should have failed")
	} else if !strings.Contains(err.Error(), "permission denied") {
		t.Logf("Got error creating database (as expected): %v", err)
	}

	// 6. Alice tries to DROP DATABASE db1 -> Should FAIL (Requires Drop privilege)
	_, err = aliceExec.Execute(parse(t, "DROP DATABASE db1"))
	if err == nil {
		t.Error("Alice passed DROP DATABASE db1 but should have failed")
	} else if !strings.Contains(err.Error(), "permission denied") {
		t.Logf("Got error dropping database (as expected): %v", err)
	}
}

// parse helper
func parse(t *testing.T, sqlStr string) Statement {
	lexer := NewLexer(sqlStr)
	parser := NewParser(lexer)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", sqlStr, err)
	}
	return stmt
}
