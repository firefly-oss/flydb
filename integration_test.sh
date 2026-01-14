#!/bin/bash
set -e

echo "Building flydb..."
go build -o flydb ./cmd/flydb

echo "Building flydb-shell..."
go build -o flydb-shell ./cmd/flydb-shell

# Cleanup
pkill -f flydb || true
rm -rf data1 data2
rm -f node1.log node2.log
rm -f flydb.conf

# Set admin password via environment variable for testing
export FLYDB_ADMIN_PASSWORD="testadmin123"

# Set encryption passphrase (encryption is enabled by default)
export FLYDB_ENCRYPTION_PASSPHRASE="test-encryption-passphrase"

echo "Starting Node 1 (Bootstrap)..."
mkdir -p data1
./flydb \
    -role cluster \
    -port 8081 \
    -repl-port 7081 \
    -cluster-port 9081 \
    -data-dir data1 \
    -cluster-peers "" \
    > node1.log 2>&1 &
NODE1_PID=$!
sleep 5

echo "Starting Node 2 (Join)..."
mkdir -p data2
./flydb \
    -role cluster \
    -port 8082 \
    -repl-port 7082 \
    -cluster-port 9082 \
    -data-dir data2 \
    -cluster-peers "localhost:9081" \
    > node2.log 2>&1 &
NODE2_PID=$!
sleep 10

# Helper function to run SQL using flydb-shell
run_sql() {
    local port=$1
    local sql=$2
    # Send AUTH command first, then the SQL
    (echo "AUTH admin testadmin123"; sleep 0.5; echo "$sql") | ./flydb-shell -H localhost -p $port
}

echo "Creating Database and Tables on Node 1..."
# Create database first
run_sql 8081 "CREATE DATABASE testdb;"
sleep 2

echo "Creating table..."
run_sql 8081 "USE testdb; CREATE TABLE users (id INT, name TEXT)"
run_sql 8081 "USE testdb; INSERT INTO users VALUES (1, 'Alice')"
run_sql 8081 "USE testdb; INSERT INTO users VALUES (2, 'Bob')"

echo "Creating user for replication test..."
run_sql 8081 "CREATE USER repl_user IDENTIFIED BY 'secret';"
sleep 5

echo "--- Verifying Replication on Node 2 ---"
echo "Query: SELECT id, name FROM users"
run_sql 8082 "USE testdb; SELECT id, name FROM users"

echo "--- Verifying User Replication on Node 2 ---"
run_sql 8082 "INSPECT USERS;"

# Cleanup PIDs
kill $NODE1_PID
kill $NODE2_PID
wait $NODE1_PID 2>/dev/null || true
wait $NODE2_PID 2>/dev/null || true

echo "Done."
