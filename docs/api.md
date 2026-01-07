# FlyDB API Reference

This document provides a complete reference for FlyDB's SQL syntax, protocol commands, and configuration options.

## Table of Contents

1. [Protocol Commands](#protocol-commands)
2. [SQL Statements](#sql-statements)
3. [Data Types](#data-types)
4. [Operators and Functions](#operators-and-functions)
5. [Configuration](#configuration)
   - [Interactive Wizard](#interactive-wizard)
   - [Server Configuration](#server-configuration)
   - [Operative Modes](#operative-modes)
   - [Installation Script](#installation-script)
   - [Uninstallation Script](#uninstallation-script)

---

## Protocol Commands

FlyDB supports two protocols: a text protocol (port 8888) and a binary protocol (port 8889).

### Text Protocol Commands

Connect using: `telnet localhost 8888` or `nc localhost 8888`

#### PING

Test server connectivity.

```
PING
→ PONG
```

#### AUTH

Authenticate with username and password.

```
AUTH <username> <password>
→ AUTH OK
→ ERROR: invalid credentials
```

#### SQL

Execute a SQL statement.

```
SQL <statement>
→ <result>
→ ERROR: <message>
```

#### WATCH

Subscribe to table change notifications.

```
WATCH <table>
→ WATCH OK

# Then receive events:
→ EVENT INSERT users 42
→ EVENT UPDATE users 42
→ EVENT DELETE users 42
```

#### UNWATCH

Unsubscribe from table notifications.

```
UNWATCH <table>
→ UNWATCH OK
```

#### QUIT

Close the connection.

```
QUIT
→ BYE
```

### Binary Protocol

The binary protocol uses length-prefixed JSON messages:

```
┌──────────────┬─────────────────────────────────────┐
│ Length (4B)  │ JSON Payload (variable)             │
│ Big-endian   │                                     │
└──────────────┴─────────────────────────────────────┘
```

**Request Format:**
```json
{
  "type": "query",
  "sql": "SELECT * FROM users"
}
```

**Response Format:**
```json
{
  "success": true,
  "result": "id|name|age\n1|Alice|30",
  "error": ""
}
```

---

## SQL Statements

### Data Definition Language (DDL)

#### CREATE TABLE

```sql
CREATE TABLE table_name (
    column1 TYPE [PRIMARY KEY] [NOT NULL] [DEFAULT value],
    column2 TYPE,
    ...
)
```

**Supported Types:** `INT`, `INTEGER`, `TEXT`, `VARCHAR(n)`, `BOOLEAN`, `FLOAT`, `DOUBLE`, `DATE`, `TIMESTAMP`

**Examples:**
```sql
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT NOT NULL,
    email VARCHAR(255),
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP
)

CREATE TABLE orders (
    id INT PRIMARY KEY,
    user_id INT NOT NULL,
    total FLOAT,
    status TEXT DEFAULT 'pending'
)
```

#### DROP TABLE

```sql
DROP TABLE table_name
```

#### CREATE INDEX

```sql
CREATE INDEX index_name ON table_name (column_name)
```

**Example:**
```sql
CREATE INDEX idx_users_email ON users (email)
```

#### DROP INDEX

```sql
DROP INDEX index_name ON table_name
```

### Data Manipulation Language (DML)

#### SELECT

```sql
SELECT column1, column2, ...
FROM table_name
[WHERE condition]
[ORDER BY column [ASC|DESC]]
[LIMIT n]
[OFFSET m]
[GROUP BY column1, column2, ...]
[HAVING condition]
```

**Examples:**
```sql
-- Basic select
SELECT * FROM users

-- With WHERE clause
SELECT name, email FROM users WHERE active = true

-- With ORDER BY and LIMIT
SELECT * FROM users ORDER BY created_at DESC LIMIT 10

-- With GROUP BY and aggregate
SELECT status, COUNT(*) FROM orders GROUP BY status

-- With HAVING
SELECT user_id, SUM(total) as sum FROM orders
GROUP BY user_id HAVING SUM(total) > 100
```

#### JOINs

```sql
SELECT columns
FROM table1
[INNER|LEFT|RIGHT|FULL] JOIN table2 ON table1.col = table2.col
[WHERE condition]
```

**Examples:**
```sql
-- Inner join
SELECT users.name, orders.total
FROM users
INNER JOIN orders ON users.id = orders.user_id

-- Left join
SELECT users.name, orders.total
FROM users
LEFT JOIN orders ON users.id = orders.user_id

-- Multiple joins
SELECT u.name, o.total, p.name as product
FROM users u
INNER JOIN orders o ON u.id = o.user_id
INNER JOIN products p ON o.product_id = p.id
```

#### UNION

```sql
SELECT columns FROM table1
UNION [ALL]
SELECT columns FROM table2
```

**Example:**
```sql
SELECT name FROM customers
UNION
SELECT name FROM suppliers
```

#### Subqueries

```sql
-- In WHERE clause
SELECT * FROM users WHERE id IN (SELECT user_id FROM orders WHERE total > 100)

-- In FROM clause
SELECT * FROM (SELECT user_id, SUM(total) as sum FROM orders GROUP BY user_id) AS subq
WHERE sum > 500
```

#### INSERT

```sql
INSERT INTO table_name [(column1, column2, ...)]
VALUES (value1, value2, ...)
```

**Examples:**
```sql
-- Insert with all columns
INSERT INTO users VALUES (1, 'Alice', 'alice@example.com', true)

-- Insert with specific columns
INSERT INTO users (name, email) VALUES ('Bob', 'bob@example.com')

-- Insert multiple rows
INSERT INTO users (name, email) VALUES
    ('Charlie', 'charlie@example.com'),
    ('Diana', 'diana@example.com')
```

#### UPDATE

```sql
UPDATE table_name
SET column1 = value1, column2 = value2, ...
[WHERE condition]
```

**Examples:**
```sql
-- Update single row
UPDATE users SET email = 'newemail@example.com' WHERE id = 1

-- Update multiple columns
UPDATE users SET active = false, updated_at = '2024-01-15' WHERE id = 1

-- Update multiple rows
UPDATE orders SET status = 'shipped' WHERE status = 'pending'
```

#### DELETE

```sql
DELETE FROM table_name
[WHERE condition]
```

**Examples:**
```sql
-- Delete single row
DELETE FROM users WHERE id = 1

-- Delete multiple rows
DELETE FROM orders WHERE status = 'cancelled'

-- Delete all rows (use with caution!)
DELETE FROM temp_data
```

### Transactions

```sql
BEGIN
-- statements
COMMIT

-- or
BEGIN
-- statements
ROLLBACK
```

**Example:**
```sql
BEGIN
INSERT INTO accounts (id, balance) VALUES (1, 1000)
UPDATE accounts SET balance = balance - 100 WHERE id = 1
UPDATE accounts SET balance = balance + 100 WHERE id = 2
COMMIT
```

### Stored Procedures

#### CREATE PROCEDURE

```sql
CREATE PROCEDURE procedure_name AS
BEGIN
    statement1;
    statement2;
END
```

**Example:**
```sql
CREATE PROCEDURE deactivate_old_users AS
BEGIN
    UPDATE users SET active = false WHERE last_login < '2023-01-01';
    DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE active = false);
END
```

#### CALL

```sql
CALL procedure_name
```

### Views

#### CREATE VIEW

```sql
CREATE VIEW view_name AS
SELECT columns FROM table [WHERE condition]
```

**Example:**
```sql
CREATE VIEW active_users AS
SELECT id, name, email FROM users WHERE active = true
```

Views can be queried like tables:
```sql
SELECT * FROM active_users
```

### Triggers

Triggers are automatic actions that execute in response to INSERT, UPDATE, or DELETE operations on tables.

#### CREATE TRIGGER

```sql
CREATE TRIGGER trigger_name
  BEFORE|AFTER INSERT|UPDATE|DELETE ON table_name
  FOR EACH ROW
  EXECUTE action_sql
```

**Examples:**
```sql
-- Log all inserts to an audit table
CREATE TRIGGER log_insert AFTER INSERT ON users FOR EACH ROW EXECUTE INSERT INTO audit_log VALUES ( 'insert' , 'users' )

-- Validate before update
CREATE TRIGGER validate_update BEFORE UPDATE ON products FOR EACH ROW EXECUTE INSERT INTO validation_log VALUES ( 'validating' )

-- Log deletions
CREATE TRIGGER log_delete AFTER DELETE ON orders FOR EACH ROW EXECUTE INSERT INTO audit_log VALUES ( 'delete' , 'orders' )
```

**Timing:**
- `BEFORE`: Trigger executes before the operation
- `AFTER`: Trigger executes after the operation

**Events:**
- `INSERT`: Fires on INSERT operations
- `UPDATE`: Fires on UPDATE operations
- `DELETE`: Fires on DELETE operations

#### DROP TRIGGER

```sql
DROP TRIGGER trigger_name ON table_name
```

**Example:**
```sql
DROP TRIGGER log_insert ON users
```

### User Management

#### CREATE USER

```sql
CREATE USER 'username' WITH PASSWORD 'password'
```

#### GRANT

```sql
GRANT permission ON table TO user
```

**Permissions:** `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `ALL`

**Examples:**
```sql
GRANT SELECT ON users TO readonly_user
GRANT SELECT, INSERT, UPDATE ON orders TO sales_user
GRANT ALL ON products TO admin_user
```

#### REVOKE

```sql
REVOKE permission ON table FROM user
```

### Row-Level Security

#### CREATE POLICY

```sql
CREATE POLICY ON table USING (column operator value)
```

**Example:**
```sql
-- Users can only see their own orders
CREATE POLICY ON orders USING (user_id = CURRENT_USER)
```

### Prepared Statements

#### PREPARE

```sql
PREPARE stmt_name AS SELECT * FROM users WHERE id = ?
```

#### EXECUTE

```sql
EXECUTE stmt_name (value1, value2, ...)
```

**Example:**
```sql
PREPARE get_user AS SELECT * FROM users WHERE id = ?
EXECUTE get_user (42)
```

---

## Data Types

| Type | Description | Example |
|------|-------------|---------|
| `INT` / `INTEGER` | 64-bit signed integer | `42`, `-100` |
| `FLOAT` / `DOUBLE` | 64-bit floating point | `3.14`, `-0.001` |
| `TEXT` | Variable-length string | `'Hello World'` |
| `VARCHAR(n)` | String with max length n | `'Hello'` |
| `BOOLEAN` | True or false | `true`, `false` |
| `DATE` | Date (YYYY-MM-DD) | `'2024-01-15'` |
| `TIMESTAMP` | Date and time | `'2024-01-15 10:30:00'` |

---

## Operators and Functions

### Comparison Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `=` | Equal | `WHERE id = 1` |
| `<>` or `!=` | Not equal | `WHERE status <> 'deleted'` |
| `<` | Less than | `WHERE age < 18` |
| `>` | Greater than | `WHERE total > 100` |
| `<=` | Less than or equal | `WHERE quantity <= 10` |
| `>=` | Greater than or equal | `WHERE rating >= 4.0` |
| `LIKE` | Pattern matching | `WHERE name LIKE 'A%'` |
| `IN` | Value in list | `WHERE status IN ('active', 'pending')` |
| `BETWEEN` | Range check | `WHERE age BETWEEN 18 AND 65` |
| `IS NULL` | Null check | `WHERE email IS NULL` |
| `IS NOT NULL` | Not null check | `WHERE email IS NOT NULL` |

### Logical Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `AND` | Logical AND | `WHERE active = true AND age > 18` |
| `OR` | Logical OR | `WHERE status = 'admin' OR status = 'moderator'` |
| `NOT` | Logical NOT | `WHERE NOT deleted` |

### Aggregate Functions

| Function | Description | Example |
|----------|-------------|---------|
| `COUNT(*)` | Count rows | `SELECT COUNT(*) FROM users` |
| `COUNT(col)` | Count non-null values | `SELECT COUNT(email) FROM users` |
| `SUM(col)` | Sum of values | `SELECT SUM(total) FROM orders` |
| `AVG(col)` | Average of values | `SELECT AVG(age) FROM users` |
| `MIN(col)` | Minimum value | `SELECT MIN(price) FROM products` |
| `MAX(col)` | Maximum value | `SELECT MAX(created_at) FROM users` |

### String Functions

| Function | Description | Example |
|----------|-------------|---------|
| `UPPER(str)` | Convert to uppercase | `SELECT UPPER(name) FROM users` |
| `LOWER(str)` | Convert to lowercase | `SELECT LOWER(email) FROM users` |
| `LENGTH(str)` | String length | `SELECT LENGTH(name) FROM users` |

---

## Configuration

### Interactive Wizard

When FlyDB is started without any command-line arguments, an interactive configuration wizard is launched:

```bash
./flydb
```

The wizard guides you through:

1. **Operative Mode Selection**
   - **Standalone**: Single server for development or small deployments
   - **Master**: Leader node that accepts writes and replicates to slaves
   - **Slave**: Follower node that receives replication from master

2. **Network Configuration**
   - Text protocol port (default: 8888)
   - Binary protocol port (default: 8889)
   - Replication port (master only, default: 9999)

3. **Storage Configuration**
   - Database file path (default: flydb.wal)

4. **Logging Configuration**
   - Log level (debug, info, warn, error)
   - JSON logging option

5. **Configuration Summary and Confirmation**

The wizard displays sensible defaults in brackets. Press Enter to accept defaults.

### Server Configuration

FlyDB is configured via command-line flags:

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | `8888` | Text protocol port |
| `-binary-port` | `8889` | Binary protocol port |
| `-repl-port` | `9999` | Replication port (master only) |
| `-db` | `flydb.wal` | WAL file path |
| `-role` | `master` | Server role: `standalone`, `master`, or `slave` |
| `-master` | - | Master address for slave mode (host:port) |
| `-log-level` | `info` | Log level: debug, info, warn, error |
| `-log-json` | `false` | Enable JSON log output |

### Operative Modes

| Mode | Description | Replication |
|------|-------------|-------------|
| `standalone` | Single server mode | None |
| `master` | Leader node | Accepts slaves on repl-port |
| `slave` | Follower node | Connects to master |

### Example Startup

```bash
# Interactive wizard (recommended for first-time setup)
./flydb

# Standalone mode (development)
./flydb -role standalone -db dev.wal

# Master with replication
./flydb -port 8888 -repl-port 9999 -role master -db master.wal

# Slave connecting to master
./flydb -port 8889 -role slave -master localhost:9999 -db slave.wal

# With debug logging
./flydb -role standalone -log-level debug

# With JSON logging (for log aggregation)
./flydb -role standalone -log-json
```

### Installation Script

FlyDB includes an installation script for easy setup:

```bash
# Interactive installation (recommended)
./install.sh

# Non-interactive installation with defaults
./install.sh --yes

# Install to custom prefix
./install.sh --prefix /opt/flydb

# System-wide installation
sudo ./install.sh --prefix /usr/local

# Preview installation without making changes
./install.sh --dry-run
```

The script:
- Downloads pre-built binaries or builds from source
- Installs `flydb` daemon and `fly-cli` client
- Optionally sets up system service (systemd on Linux, launchd on macOS)
- Creates default configuration files
- Works on Linux and macOS

### Uninstallation Script

To remove FlyDB from your system:

```bash
# Interactive uninstallation
./uninstall.sh

# Preview what would be removed
./uninstall.sh --dry-run

# Non-interactive uninstallation
./uninstall.sh --yes

# Also remove data directories (WARNING: deletes databases!)
./uninstall.sh --remove-data --yes
```

The script automatically detects and removes:
- FlyDB binaries (`flydb`, `fly-cli`)
- System service files
- Configuration directories (use `--no-config` to preserve)
- Data directories (only with `--remove-data` flag)

---

## See Also

- [Architecture Overview](architecture.md) - High-level system design
- [Implementation Details](implementation.md) - Technical deep-dives
- [Design Decisions](design-decisions.md) - Rationale and trade-offs

