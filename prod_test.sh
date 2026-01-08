#!/bin/bash
set -e

echo "Building flydb..."
go build -o flydb ./cmd/flydb

# Cleanup
rm -f prod_test.fdb

# Set admin password via environment variable for testing
export FLYDB_ADMIN_PASSWORD="testadmin123"

# Set encryption passphrase (encryption is enabled by default)
export FLYDB_ENCRYPTION_PASSPHRASE="test-encryption-passphrase"

echo "Starting Server..."
./flydb -port 8086 -db prod_test.fdb > prod.log 2>&1 &
PID=$!
sleep 2

echo "--- 1. Setup Data ---"
(
echo "AUTH admin $FLYDB_ADMIN_PASSWORD"
echo "SQL CREATE TABLE products (id INT, name TEXT, price INT)"
echo "SQL INSERT INTO products VALUES (1, 'Laptop', 1000)"
echo "SQL INSERT INTO products VALUES (2, 'Mouse', 50)"
echo "SQL INSERT INTO products VALUES (3, 'Keyboard', 80)"
echo "SQL INSERT INTO products VALUES (4, 'Monitor', 200)"
) | nc localhost 8086

echo "--- 2. Verify UPDATE ---"
echo "SQL UPDATE products SET price=1200 WHERE id=1" | nc localhost 8086
RES=$(echo "SQL SELECT price FROM products WHERE id=1" | nc localhost 8086)
if [[ "$RES" != *"1200"* ]]; then
    echo "FAIL: Update failed, expected 1200"
    exit 1
fi

echo "--- 3. Verify DELETE ---"
echo "SQL DELETE FROM products WHERE id=2" | nc localhost 8086
RES=$(echo "SQL SELECT name FROM products" | nc localhost 8086)
if [[ "$RES" == *"Mouse"* ]]; then
    echo "FAIL: Delete failed, Mouse still exists"
    exit 1
fi

echo "--- 4. Verify ORDER BY ---"
# Sort by price DESC: 1200 (Laptop), 200 (Monitor), 80 (Keyboard)
RES=$(echo "SQL SELECT price FROM products ORDER BY price DESC" | nc localhost 8086)
# Expected order: 1200, 200, 80
# Note: output is newline separated
# We just check if top is 1200
FIRST=$(echo "$RES" | head -n 1)
if [[ "$FIRST" != *"1200"* ]]; then
    echo "FAIL: ORDER BY DESC failed. First: $FIRST"
    exit 1
fi

echo "--- 5. Verify LIMIT ---"
RES=$(echo "SQL SELECT name FROM products LIMIT 1" | nc localhost 8086)
LINES=$(echo "$RES" | wc -l | xargs)
if [[ "$LINES" != "1" ]]; then
    echo "FAIL: LIMIT 1 returned $LINES lines"
    exit 1
fi

echo "Production Readiness Tests Passed!"

kill $PID
wait $PID 2>/dev/null || true
rm -f prod_test.fdb
