#!/bin/bash
set -e

echo "Building flydb..."
go build -o flydb ./cmd/flydb

# Cleanup
rm -f master.fdb slave.fdb

# Set admin password via environment variable for testing
export FLYDB_ADMIN_PASSWORD="testadmin123"

# Set encryption passphrase (encryption is enabled by default)
export FLYDB_ENCRYPTION_PASSPHRASE="test-encryption-passphrase"

echo "Starting Master..."
./flydb -port 8081 -repl-port 9091 -role master -db master.fdb > master.log 2>&1 &
MASTER_PID=$!
sleep 2

echo "Starting Slave..."
./flydb -port 8082 -role slave -master localhost:9091 -db slave.fdb > slave.log 2>&1 &
SLAVE_PID=$!
sleep 2

echo "Creating Tables on Master..."
echo "SQL CREATE TABLE users (id INT, name TEXT)" | nc localhost 8081
echo "SQL CREATE TABLE orders (id INT, user_id INT, amount INT)" | nc localhost 8081

echo "Inserting Data on Master..."
echo "SQL INSERT INTO users VALUES (1, 'Alice')" | nc localhost 8081
echo "SQL INSERT INTO users VALUES (2, 'Bob')" | nc localhost 8081
echo "SQL INSERT INTO orders VALUES (101, 1, 500)" | nc localhost 8081
echo "SQL INSERT INTO orders VALUES (102, 1, 300)" | nc localhost 8081
echo "SQL INSERT INTO orders VALUES (103, 2, 700)" | nc localhost 8081

sleep 2

echo "--- Verifying Replication on Slave ---"
echo "Query: SELECT id, name FROM users"
echo "SQL SELECT id, name FROM users" | nc localhost 8082

echo "--- Verifying JOIN on Master ---"
echo "Query: SELECT users.name, orders.amount FROM users JOIN orders ON users.id = orders.user_id"
echo "SQL SELECT users.name, orders.amount FROM users JOIN orders ON users.id = orders.user_id" | nc localhost 8081

echo "--- Verifying JOIN on Slave ---"
echo "SQL SELECT users.name, orders.amount FROM users JOIN orders ON users.id = orders.user_id" | nc localhost 8082

# Cleanup PIDs
kill $MASTER_PID
kill $SLAVE_PID
wait $MASTER_PID 2>/dev/null || true
wait $SLAVE_PID 2>/dev/null || true

echo "Done."
