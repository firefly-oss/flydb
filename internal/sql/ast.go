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
Package sql provides the SQL processing pipeline for FlyDB.

Abstract Syntax Tree (AST) Overview:
====================================

The AST is the intermediate representation of SQL statements after parsing.
It represents the structure of a SQL statement as a tree of nodes, where
each node type corresponds to a specific SQL construct.

AST Design Pattern:
===================

FlyDB uses the Visitor pattern for AST processing:

  1. All statement types implement the Statement interface
  2. The statementNode() method is a marker (no implementation needed)
  3. The Executor uses type switches to handle each statement type

This design allows:
  - Type-safe handling of different statement types
  - Easy addition of new statement types
  - Clear separation between parsing and execution

Supported SQL Statements:
=========================

Data Definition Language (DDL):
  - CREATE TABLE: Define new tables with columns and types
  - CREATE USER: Create database user accounts

Data Manipulation Language (DML):
  - SELECT: Query data with optional WHERE, JOIN, ORDER BY, LIMIT
  - INSERT: Add new rows to tables
  - UPDATE: Modify existing rows
  - DELETE: Remove rows from tables

Data Control Language (DCL):
  - GRANT: Assign permissions to users with optional RLS

AST Node Hierarchy:
===================

	Statement (interface)
	├── CreateTableStmt
	├── CreateUserStmt
	├── GrantStmt
	├── InsertStmt
	├── UpdateStmt
	├── DeleteStmt
	└── SelectStmt
	    ├── JoinClause
	    ├── OrderByClause
	    └── Condition (WHERE)

Example AST:
============

For the SQL: SELECT name FROM users WHERE id = 1

	SelectStmt{
	    TableName: "users",
	    Columns:   []string{"name"},
	    Where:     &Condition{Column: "id", Value: "1"},
	}
*/
package sql

// Statement represents a SQL statement node in the Abstract Syntax Tree (AST).
// All concrete statement types must implement this interface.
//
// The statementNode() method is a marker method that serves two purposes:
//  1. It ensures only intended types can be used as statements
//  2. It enables compile-time type checking
//
// This pattern is common in Go AST implementations (see go/ast package).
type Statement interface {
	statementNode()
}

// CreateUserStmt represents a CREATE USER statement.
// It creates a new database user with the specified credentials.
//
// SQL Syntax:
//
//	CREATE USER <username> IDENTIFIED BY '<password>'
//
// Example:
//
//	CREATE USER alice IDENTIFIED BY 'secret123'
//
// The password is stored as provided (cleartext in this demo).
// Production systems should hash passwords before storage.
type CreateUserStmt struct {
	Username string // The unique username for the new user
	Password string // The user's password
}

// statementNode implements the Statement interface.
func (s CreateUserStmt) statementNode() {}

// GrantStmt represents a GRANT statement.
// It assigns permissions to a user for accessing a specific table,
// optionally with Row-Level Security (RLS) restrictions.
//
// SQL Syntax:
//
//	GRANT SELECT ON <table> [WHERE <column> = <value>] TO <user>
//
// Examples:
//
//	GRANT SELECT ON products TO alice
//	GRANT SELECT ON orders WHERE user_id = 'alice' TO alice
//
// The optional WHERE clause enables RLS, restricting the user
// to only see rows matching the condition.
type GrantStmt struct {
	TableName string     // The table to grant access to
	Username  string     // The user receiving the permission
	Where     *Condition // Optional RLS condition (nil = full access)
}

// statementNode implements the Statement interface.
func (s GrantStmt) statementNode() {}

// CreateTableStmt represents a CREATE TABLE statement.
// It defines a new table with the specified columns, types, and constraints.
//
// SQL Syntax:
//
//	CREATE TABLE <name> (
//	    <col1> <type1> [constraints],
//	    <col2> <type2> [constraints],
//	    [table_constraints]
//	)
//
// Examples:
//
//	CREATE TABLE users (id INT PRIMARY KEY, name TEXT NOT NULL, email TEXT UNIQUE)
//	CREATE TABLE orders (id SERIAL PRIMARY KEY, user_id INT REFERENCES users(id), amount DECIMAL)
//
// Supported column types: INT, TEXT, SERIAL, and others defined in types.go
// Supported constraints: PRIMARY KEY, FOREIGN KEY/REFERENCES, NOT NULL, UNIQUE, AUTO_INCREMENT, DEFAULT
type CreateTableStmt struct {
	TableName   string            // The name of the new table
	Columns     []ColumnDef       // Column definitions (name, type, and constraints)
	Constraints []TableConstraint // Table-level constraints (composite keys, etc.)
}

// statementNode implements the Statement interface.
func (s CreateTableStmt) statementNode() {}

// ConstraintType represents the type of column constraint.
type ConstraintType string

// Constraint type constants.
const (
	ConstraintPrimaryKey    ConstraintType = "PRIMARY KEY"
	ConstraintForeignKey    ConstraintType = "FOREIGN KEY"
	ConstraintNotNull       ConstraintType = "NOT NULL"
	ConstraintUnique        ConstraintType = "UNIQUE"
	ConstraintAutoIncrement ConstraintType = "AUTO_INCREMENT"
	ConstraintDefault       ConstraintType = "DEFAULT"
	ConstraintCheck         ConstraintType = "CHECK"
)

// ForeignKeyRef defines a foreign key reference to another table.
type ForeignKeyRef struct {
	Table  string // Referenced table name
	Column string // Referenced column name
}

// ColumnConstraint defines a constraint on a column.
type ColumnConstraint struct {
	Type         ConstraintType // The type of constraint
	ForeignKey   *ForeignKeyRef // For FOREIGN KEY: the referenced table and column
	DefaultValue string         // For DEFAULT: the default value
	CheckExpr    *CheckExpr     // For CHECK: the validation expression
}

// CheckExpr represents a CHECK constraint expression.
// It defines a validation condition that must be true for all rows.
//
// SQL Syntax:
//
//	CHECK (<column> <operator> <value>)
//	CHECK (<column> IN (<value1>, <value2>, ...))
//	CHECK (<column> BETWEEN <min> AND <max>)
//
// Examples:
//
//	CHECK (age >= 0)
//	CHECK (status IN ('active', 'inactive', 'pending'))
//	CHECK (price > 0 AND price < 10000)
type CheckExpr struct {
	Column   string   // Column being checked
	Operator string   // Comparison operator: =, <, >, <=, >=, <>, IN, BETWEEN
	Value    string   // Value for simple comparisons
	Values   []string // Values for IN clause
	MinValue string   // Min value for BETWEEN
	MaxValue string   // Max value for BETWEEN
	And      *CheckExpr // Optional AND condition
	Or       *CheckExpr // Optional OR condition
}

// ColumnDef defines a single column in a table schema.
// It specifies the column name, data type, and optional constraints.
//
// Supported Types:
//   - INT: Integer values (stored as strings internally)
//   - TEXT: String values
//   - SERIAL: Auto-incrementing integer (implies PRIMARY KEY)
//   - And other types defined in types.go
//
// Supported Constraints:
//   - PRIMARY KEY: Unique identifier for the row
//   - FOREIGN KEY: Reference to another table's primary key
//   - NOT NULL: Column cannot contain NULL values
//   - UNIQUE: Column values must be unique
//   - AUTO_INCREMENT: Automatically increment integer values
//   - DEFAULT: Default value when not specified
//
// Note: FlyDB stores all values as strings internally.
// Type information is used for validation and display purposes.
type ColumnDef struct {
	Name        string             // Column name (case-sensitive)
	Type        string             // Column type (INT, TEXT, SERIAL, etc.)
	Constraints []ColumnConstraint // Column constraints (PRIMARY KEY, NOT NULL, etc.)
}

// HasConstraint checks if the column has a specific constraint type.
func (c ColumnDef) HasConstraint(constraintType ConstraintType) bool {
	for _, constraint := range c.Constraints {
		if constraint.Type == constraintType {
			return true
		}
	}
	return false
}

// IsPrimaryKey returns true if this column is a primary key.
func (c ColumnDef) IsPrimaryKey() bool {
	return c.HasConstraint(ConstraintPrimaryKey)
}

// IsNotNull returns true if this column has a NOT NULL constraint.
func (c ColumnDef) IsNotNull() bool {
	return c.HasConstraint(ConstraintNotNull) || c.IsPrimaryKey()
}

// IsUnique returns true if this column has a UNIQUE constraint.
func (c ColumnDef) IsUnique() bool {
	return c.HasConstraint(ConstraintUnique) || c.IsPrimaryKey()
}

// IsAutoIncrement returns true if this column auto-increments.
func (c ColumnDef) IsAutoIncrement() bool {
	return c.HasConstraint(ConstraintAutoIncrement) || c.Type == "SERIAL"
}

// GetForeignKey returns the foreign key reference if this column has one.
func (c ColumnDef) GetForeignKey() *ForeignKeyRef {
	for _, constraint := range c.Constraints {
		if constraint.Type == ConstraintForeignKey && constraint.ForeignKey != nil {
			return constraint.ForeignKey
		}
	}
	return nil
}

// GetDefaultValue returns the default value if this column has one.
func (c ColumnDef) GetDefaultValue() (string, bool) {
	for _, constraint := range c.Constraints {
		if constraint.Type == ConstraintDefault {
			return constraint.DefaultValue, true
		}
	}
	return "", false
}

// GetCheckConstraint returns the CHECK constraint if this column has one.
func (c ColumnDef) GetCheckConstraint() *CheckExpr {
	for _, constraint := range c.Constraints {
		if constraint.Type == ConstraintCheck && constraint.CheckExpr != nil {
			return constraint.CheckExpr
		}
	}
	return nil
}

// TableConstraint defines a table-level constraint (e.g., composite primary key).
type TableConstraint struct {
	Name       string         // Optional constraint name
	Type       ConstraintType // The type of constraint
	Columns    []string       // Columns involved in the constraint
	ForeignKey *ForeignKeyRef // For FOREIGN KEY: the referenced table and column
}

// InsertStmt represents an INSERT INTO statement.
// It adds a new row to a table with the specified values.
//
// SQL Syntax:
//
//	INSERT INTO <table> VALUES (<val1>, <val2>, ...)
//
// Example:
//
//	INSERT INTO users VALUES (1, 'Alice', 'alice@example.com')
//
// The number of values must match the number of columns in the table.
// Values are provided in the same order as the table's column definitions.
type InsertStmt struct {
	TableName string   // The target table
	Values    []string // Values for each column (in order)
}

// statementNode implements the Statement interface.
func (s InsertStmt) statementNode() {}

// UpdateStmt represents an UPDATE statement.
// It modifies existing rows in a table that match the WHERE condition.
//
// SQL Syntax:
//
//	UPDATE <table> SET <col1>=<val1>, <col2>=<val2> [WHERE <col>=<val>]
//
// Example:
//
//	UPDATE products SET price=1200 WHERE id=1
//
// If no WHERE clause is provided, all rows are updated.
type UpdateStmt struct {
	TableName string            // The target table
	Updates   map[string]string // Column-to-value mapping for updates
	Where     *Condition        // Optional filter condition
}

// statementNode implements the Statement interface.
func (s UpdateStmt) statementNode() {}

// DeleteStmt represents a DELETE statement.
// It removes rows from a table that match the WHERE condition.
//
// SQL Syntax:
//
//	DELETE FROM <table> [WHERE <col>=<val>]
//
// Example:
//
//	DELETE FROM users WHERE id=5
//
// If no WHERE clause is provided, all rows are deleted.
type DeleteStmt struct {
	TableName string     // The target table
	Where     *Condition // Optional filter condition
}

// statementNode implements the Statement interface.
func (s DeleteStmt) statementNode() {}

// OrderByClause represents an ORDER BY clause in a SELECT statement.
// It specifies how to sort the result set.
//
// SQL Syntax:
//
//	ORDER BY <column> [ASC|DESC]
//
// Example:
//
//	SELECT * FROM products ORDER BY price DESC
//
// Default direction is ASC (ascending) if not specified.
type OrderByClause struct {
	Column    string // The column to sort by
	Direction string // Sort direction: "ASC" or "DESC"
}

// SelectStmt represents a SELECT statement.
// It queries data from one or more tables with optional filtering,
// joining, grouping, sorting, and limiting.
//
// SQL Syntax:
//
//	SELECT [DISTINCT] <columns> FROM <table>
//	  [JOIN <table2> ON <condition>]
//	  [WHERE <condition>]
//	  [GROUP BY <column1>, <column2>, ...]
//	  [HAVING <aggregate_condition>]
//	  [ORDER BY <column> [ASC|DESC]]
//	  [LIMIT <n>]
//
// Examples:
//
//	SELECT name, email FROM users
//	SELECT DISTINCT category FROM products
//	SELECT * FROM users WHERE id = 1
//	SELECT u.name, o.amount FROM users u JOIN orders o ON u.id = o.user_id
//	SELECT name FROM products ORDER BY price DESC LIMIT 10
//	SELECT COUNT(*), SUM(amount) FROM orders
//	SELECT category, COUNT(*) FROM products GROUP BY category
//	SELECT category, SUM(price) FROM products GROUP BY category HAVING SUM(price) > 100
type SelectStmt struct {
	TableName  string           // Primary table to query
	Columns    []string         // Columns to return (or "*" for all)
	Distinct   bool             // Whether to remove duplicate rows
	Aggregates []*AggregateExpr // Aggregate function expressions
	Where      *Condition       // Optional simple filter condition (backward compat)
	WhereExt   *WhereClause     // Optional extended WHERE clause with subquery support
	Join       *JoinClause      // Optional JOIN clause
	GroupBy    []string         // Optional GROUP BY columns
	Having     *HavingClause    // Optional HAVING clause for filtering groups
	OrderBy    *OrderByClause   // Optional ORDER BY clause
	Limit      int              // Maximum rows to return (0 = unlimited)
	Subquery   *SelectStmt      // Optional subquery for FROM clause
	FromAlias  string           // Alias for subquery or table
}

// statementNode implements the Statement interface.
func (s SelectStmt) statementNode() {}

// UnionStmt represents a UNION operation combining multiple SELECT statements.
// It combines the results of two or more SELECT queries into a single result set.
//
// SQL Syntax:
//
//	SELECT ... UNION [ALL] SELECT ...
//	SELECT ... UNION [ALL] SELECT ... UNION [ALL] SELECT ...
//
// Examples:
//
//	SELECT name FROM employees UNION SELECT name FROM contractors
//	SELECT id, name FROM table1 UNION ALL SELECT id, name FROM table2
//
// UNION removes duplicates by default. UNION ALL keeps all rows including duplicates.
type UnionStmt struct {
	Left     *SelectStmt // Left SELECT statement
	Right    *SelectStmt // Right SELECT statement
	All      bool        // If true, keep duplicates (UNION ALL)
	NextUnion *UnionStmt // For chaining multiple UNIONs
}

// statementNode implements the Statement interface.
func (u UnionStmt) statementNode() {}

// JoinClause represents an INNER JOIN operation in a SELECT statement.
// It combines rows from two tables based on a matching condition.
//
// SQL Syntax:
//
//	JOIN <table> ON <left_col> = <right_col>
//
// Example:
//
//	SELECT users.name, orders.amount
//	FROM users JOIN orders ON users.id = orders.user_id
//
// FlyDB implements a Nested Loop Join algorithm, which iterates
// through all combinations of rows and filters by the ON condition.
type JoinClause struct {
	TableName string     // The table to join with
	On        *Condition // The join condition (left_col = right_col)
}

// Condition represents a simple equality condition.
// It is used in WHERE clauses, JOIN ON clauses, and RLS definitions.
//
// SQL Syntax:
//
//	<column> = <value>
//
// Examples:
//
//	WHERE id = 1
//	ON users.id = orders.user_id
//
// Note: FlyDB currently only supports equality conditions.
// Future versions may add support for other operators (<, >, LIKE, etc.).
type Condition struct {
	Column string // The column name (may include table prefix: "table.column")
	Value  string // The value to compare against (or column name for JOINs)
}

// WhereClause represents a WHERE clause that can contain subqueries.
// It extends the simple Condition to support IN, EXISTS, and other subquery operations.
//
// SQL Syntax:
//
//	WHERE <column> = <value>
//	WHERE <column> IN (SELECT ...)
//	WHERE EXISTS (SELECT ...)
//	WHERE <column> IN (<value1>, <value2>, ...)
//
// Examples:
//
//	WHERE id = 1
//	WHERE category IN (SELECT category FROM featured_products)
//	WHERE EXISTS (SELECT 1 FROM orders WHERE orders.user_id = users.id)
type WhereClause struct {
	Column     string       // The column name for comparison
	Operator   string       // Comparison operator: =, <, >, <=, >=, IN, EXISTS, NOT IN, NOT EXISTS
	Value      string       // Simple value for comparison
	Values     []string     // List of values for IN clause
	Subquery   *SelectStmt  // Subquery for IN/EXISTS
	IsSubquery bool         // True if this uses a subquery
	And        *WhereClause // Optional AND condition
	Or         *WhereClause // Optional OR condition
}

// HavingClause represents a HAVING clause for filtering grouped results.
// It applies a condition to aggregate function results after GROUP BY.
//
// SQL Syntax:
//
//	HAVING <aggregate_function>(<column>) <operator> <value>
//
// Examples:
//
//	HAVING COUNT(*) > 5
//	HAVING SUM(amount) > 1000
//	HAVING AVG(price) > 50
//
// The HAVING clause is evaluated after grouping and aggregation,
// unlike WHERE which filters rows before grouping.
type HavingClause struct {
	Aggregate *AggregateExpr // The aggregate function to evaluate
	Operator  string         // Comparison operator (>, <, =, >=, <=)
	Value     string         // The value to compare against
}

// BeginStmt represents a BEGIN statement to start a transaction.
//
// SQL Syntax:
//
//	BEGIN
//	BEGIN TRANSACTION
//
// After BEGIN, all subsequent statements are part of the transaction
// until COMMIT or ROLLBACK is executed.
type BeginStmt struct{}

// statementNode implements the Statement interface.
func (s BeginStmt) statementNode() {}

// CommitStmt represents a COMMIT statement to commit a transaction.
//
// SQL Syntax:
//
//	COMMIT
//
// COMMIT applies all changes made during the transaction to the database.
type CommitStmt struct{}

// statementNode implements the Statement interface.
func (s CommitStmt) statementNode() {}

// RollbackStmt represents a ROLLBACK statement to abort a transaction.
//
// SQL Syntax:
//
//	ROLLBACK
//
// ROLLBACK discards all changes made during the transaction.
type RollbackStmt struct{}

// statementNode implements the Statement interface.
func (s RollbackStmt) statementNode() {}

// CreateIndexStmt represents a CREATE INDEX statement.
//
// SQL Syntax:
//
//	CREATE INDEX <name> ON <table> (<column>)
//
// Example:
//
//	CREATE INDEX idx_users_email ON users (email)
//
// Indexes improve query performance for WHERE clause lookups
// on the indexed column.
type CreateIndexStmt struct {
	IndexName  string // Name of the index
	TableName  string // Table to create the index on
	ColumnName string // Column to index
}

// statementNode implements the Statement interface.
func (s CreateIndexStmt) statementNode() {}

// PrepareStmt represents a PREPARE statement for prepared statements.
//
// SQL Syntax:
//
//	PREPARE <name> AS <query>
//
// Example:
//
//	PREPARE get_user AS SELECT * FROM users WHERE id = $1
//
// Parameters are specified using $1, $2, etc. placeholders.
// The prepared statement can be executed multiple times with
// different parameter values using EXECUTE.
type PrepareStmt struct {
	Name  string // Name of the prepared statement
	Query string // The SQL query with parameter placeholders
}

// statementNode implements the Statement interface.
func (s PrepareStmt) statementNode() {}

// ExecuteStmt represents an EXECUTE statement for prepared statements.
//
// SQL Syntax:
//
//	EXECUTE <name> [USING <param1>, <param2>, ...]
//
// Example:
//
//	EXECUTE get_user USING 42
//
// The parameters are substituted for $1, $2, etc. in the prepared query.
type ExecuteStmt struct {
	Name   string   // Name of the prepared statement to execute
	Params []string // Parameter values to substitute
}

// statementNode implements the Statement interface.
func (s ExecuteStmt) statementNode() {}

// DeallocateStmt represents a DEALLOCATE statement for prepared statements.
//
// SQL Syntax:
//
//	DEALLOCATE <name>
//
// Example:
//
//	DEALLOCATE get_user
//
// This removes the prepared statement from memory.
type DeallocateStmt struct {
	Name string // Name of the prepared statement to deallocate
}

// statementNode implements the Statement interface.
func (s DeallocateStmt) statementNode() {}

// IntrospectStmt represents an INTROSPECT statement for database metadata inspection.
// It allows users to query information about database objects.
//
// SQL Syntax:
//
//	INTROSPECT USERS              - List all database users
//	INTROSPECT DATABASES          - List database information
//	INTROSPECT DATABASE <name>    - Detailed info for a specific database
//	INTROSPECT TABLES             - List all tables with their schemas
//	INTROSPECT TABLE <name>       - Detailed info for a specific table
//	INTROSPECT INDEXES            - List all indexes
//
// Examples:
//
//	INTROSPECT USERS
//	INTROSPECT TABLES
//	INTROSPECT TABLE employees
//	INTROSPECT DATABASE flydb
type IntrospectStmt struct {
	Target     string // The target to introspect: USERS, DATABASES, DATABASE, TABLES, TABLE, INDEXES
	ObjectName string // Optional: specific object name for TABLE or DATABASE targets
}

// statementNode implements the Statement interface.
func (s IntrospectStmt) statementNode() {}

// AggregateExpr represents an aggregate function call in a SELECT statement.
// Aggregate functions compute a single result from a set of input values.
//
// SQL Syntax:
//
//	<function>(<column>)
//	<function>(*)
//
// Supported Functions:
//   - COUNT: Returns the number of rows (COUNT(*) or COUNT(column))
//   - SUM: Returns the sum of numeric values
//   - AVG: Returns the average of numeric values
//   - MIN: Returns the minimum value
//   - MAX: Returns the maximum value
//
// Examples:
//
//	SELECT COUNT(*) FROM users
//	SELECT SUM(amount) FROM orders
//	SELECT AVG(price), MIN(price), MAX(price) FROM products
type AggregateExpr struct {
	Function string // The aggregate function name (COUNT, SUM, AVG, MIN, MAX)
	Column   string // The column to aggregate (or "*" for COUNT(*))
	Alias    string // Optional alias for the result column
}

// CreateProcedureStmt represents a CREATE PROCEDURE statement.
// It defines a stored procedure with parameters and a body of SQL statements.
//
// SQL Syntax:
//
//	CREATE PROCEDURE <name>([<param1> <type1>, ...])
//	BEGIN
//	    <statements>
//	END
//
// Examples:
//
//	CREATE PROCEDURE get_user(user_id INT)
//	BEGIN
//	    SELECT * FROM users WHERE id = $1;
//	END
//
//	CREATE PROCEDURE update_status(id INT, status TEXT)
//	BEGIN
//	    UPDATE orders SET status = $2 WHERE id = $1;
//	END
type CreateProcedureStmt struct {
	Name       string           // Procedure name
	Parameters []ProcedureParam // Input parameters
	Body       []Statement      // SQL statements in the procedure body
	BodySQL    []string         // Raw SQL strings for the body (for storage)
}

// statementNode implements the Statement interface.
func (s CreateProcedureStmt) statementNode() {}

// ProcedureParam represents a parameter in a stored procedure.
type ProcedureParam struct {
	Name string // Parameter name
	Type string // Parameter type (INT, TEXT, etc.)
}

// CallStmt represents a CALL statement to execute a stored procedure.
//
// SQL Syntax:
//
//	CALL <procedure_name>([<arg1>, <arg2>, ...])
//
// Examples:
//
//	CALL get_user(1)
//	CALL update_status(42, 'completed')
type CallStmt struct {
	ProcedureName string   // Name of the procedure to call
	Arguments     []string // Arguments to pass to the procedure
}

// statementNode implements the Statement interface.
func (s CallStmt) statementNode() {}

// DropProcedureStmt represents a DROP PROCEDURE statement.
//
// SQL Syntax:
//
//	DROP PROCEDURE <name>
type DropProcedureStmt struct {
	Name string // Procedure name to drop
}

// statementNode implements the Statement interface.
func (s DropProcedureStmt) statementNode() {}

// StoredProcedure represents a stored procedure in the catalog.
type StoredProcedure struct {
	Name       string           // Procedure name
	Parameters []ProcedureParam // Input parameters
	BodySQL    []string         // SQL statements as strings
}

// CreateViewStmt represents a CREATE VIEW statement.
// It creates a virtual table based on a SELECT query.
//
// SQL Syntax:
//
//	CREATE VIEW <view_name> AS SELECT ...
//
// Examples:
//
//	CREATE VIEW active_users AS SELECT * FROM users WHERE status = 'active'
//	CREATE VIEW order_summary AS SELECT user_id, COUNT(*) FROM orders GROUP BY user_id
type CreateViewStmt struct {
	ViewName string      // The name of the view
	Query    *SelectStmt // The SELECT query that defines the view
}

// statementNode implements the Statement interface.
func (s CreateViewStmt) statementNode() {}

// DropViewStmt represents a DROP VIEW statement.
// It removes a view from the database.
//
// SQL Syntax:
//
//	DROP VIEW <view_name>
//
// Example:
//
//	DROP VIEW active_users
type DropViewStmt struct {
	ViewName string // The name of the view to drop
}

// statementNode implements the Statement interface.
func (s DropViewStmt) statementNode() {}

// AlterTableAction represents the type of ALTER TABLE operation.
type AlterTableAction string

// ALTER TABLE action constants.
const (
	AlterActionAddColumn    AlterTableAction = "ADD COLUMN"
	AlterActionDropColumn   AlterTableAction = "DROP COLUMN"
	AlterActionRenameColumn AlterTableAction = "RENAME COLUMN"
	AlterActionModifyColumn AlterTableAction = "MODIFY COLUMN"
	AlterActionAddConstraint    AlterTableAction = "ADD CONSTRAINT"
	AlterActionDropConstraint   AlterTableAction = "DROP CONSTRAINT"
)

// AlterTableStmt represents an ALTER TABLE statement.
// It modifies the structure of an existing table.
//
// SQL Syntax:
//
//	ALTER TABLE <table_name> ADD COLUMN <column_def>
//	ALTER TABLE <table_name> DROP COLUMN <column_name>
//	ALTER TABLE <table_name> RENAME COLUMN <old_name> TO <new_name>
//	ALTER TABLE <table_name> MODIFY COLUMN <column_name> <new_type>
//	ALTER TABLE <table_name> ADD CONSTRAINT <constraint_def>
//	ALTER TABLE <table_name> DROP CONSTRAINT <constraint_name>
//
// Examples:
//
//	ALTER TABLE users ADD COLUMN email TEXT NOT NULL
//	ALTER TABLE users DROP COLUMN email
//	ALTER TABLE users RENAME COLUMN email TO email_address
//	ALTER TABLE users MODIFY COLUMN age BIGINT
type AlterTableStmt struct {
	TableName      string           // The table to alter
	Action         AlterTableAction // The type of alteration
	ColumnDef      *ColumnDef       // For ADD COLUMN and MODIFY COLUMN
	ColumnName     string           // For DROP COLUMN, RENAME COLUMN, MODIFY COLUMN
	NewColumnName  string           // For RENAME COLUMN
	NewColumnType  string           // For MODIFY COLUMN
	Constraint     *TableConstraint // For ADD CONSTRAINT
	ConstraintName string           // For DROP CONSTRAINT
}

// statementNode implements the Statement interface.
func (s AlterTableStmt) statementNode() {}

// TriggerEvent represents the type of event that fires a trigger.
type TriggerEvent string

// Trigger event constants.
const (
	TriggerEventInsert TriggerEvent = "INSERT"
	TriggerEventUpdate TriggerEvent = "UPDATE"
	TriggerEventDelete TriggerEvent = "DELETE"
)

// TriggerTiming represents when the trigger fires relative to the event.
type TriggerTiming string

// Trigger timing constants.
const (
	TriggerTimingBefore TriggerTiming = "BEFORE"
	TriggerTimingAfter  TriggerTiming = "AFTER"
)

// CreateTriggerStmt represents a CREATE TRIGGER statement.
// It defines an automatic action that executes in response to INSERT, UPDATE, or DELETE operations.
//
// SQL Syntax:
//
//	CREATE TRIGGER <trigger_name>
//	  BEFORE|AFTER INSERT|UPDATE|DELETE ON <table_name>
//	  FOR EACH ROW
//	  EXECUTE <action_sql>
//
// Examples:
//
//	CREATE TRIGGER log_insert AFTER INSERT ON users FOR EACH ROW EXECUTE INSERT INTO audit_log VALUES ('insert', 'users')
//	CREATE TRIGGER validate_update BEFORE UPDATE ON products FOR EACH ROW EXECUTE SELECT validate_product()
//
// Triggers are executed automatically when the specified event occurs on the table.
// BEFORE triggers execute before the operation, AFTER triggers execute after.
type CreateTriggerStmt struct {
	TriggerName string        // The name of the trigger (unique per table)
	Timing      TriggerTiming // BEFORE or AFTER
	Event       TriggerEvent  // INSERT, UPDATE, or DELETE
	TableName   string        // The table the trigger is attached to
	ActionSQL   string        // The SQL statement to execute when the trigger fires
}

// statementNode implements the Statement interface.
func (s CreateTriggerStmt) statementNode() {}

// DropTriggerStmt represents a DROP TRIGGER statement.
// It removes a trigger from a table.
//
// SQL Syntax:
//
//	DROP TRIGGER <trigger_name> ON <table_name>
//
// Example:
//
//	DROP TRIGGER log_insert ON users
type DropTriggerStmt struct {
	TriggerName string // The name of the trigger to drop
	TableName   string // The table the trigger is attached to
}

// statementNode implements the Statement interface.
func (s DropTriggerStmt) statementNode() {}

// Trigger represents a stored trigger definition.
type Trigger struct {
	Name      string        // Trigger name
	Timing    TriggerTiming // BEFORE or AFTER
	Event     TriggerEvent  // INSERT, UPDATE, or DELETE
	TableName string        // The table the trigger is attached to
	ActionSQL string        // The SQL statement to execute
}
