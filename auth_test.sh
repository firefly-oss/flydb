#!/bin/bash
set -e

echo "Building flydb..."
go build -o flydb ./cmd/flydb

# Cleanup
rm -f auth.fdb

# Set admin password via environment variable for testing
export FLYDB_ADMIN_PASSWORD="testadmin123"

# Set encryption passphrase (encryption is enabled by default)
export FLYDB_ENCRYPTION_PASSPHRASE="test-encryption-passphrase"

echo "Starting Server..."
./flydb -port 8085 -db auth.fdb > auth.log 2>&1 &
PID=$!
sleep 2

echo "--- 1. Admin Bootstrap ---"
# Login as admin (password set via FLYDB_ADMIN_PASSWORD env var)
echo "AUTH admin $FLYDB_ADMIN_PASSWORD" | nc localhost 8085

echo "--- 2. Create Users & Tables ---"
(
echo "AUTH admin $FLYDB_ADMIN_PASSWORD"
echo "SQL CREATE TABLE confidential (id INT, level TEXT, data TEXT)"
echo "SQL INSERT INTO confidential VALUES (1, 'top', 'Nuclear codes')"
echo "SQL INSERT INTO confidential VALUES (2, 'low', 'Lunch menu')"
echo "SQL CREATE USER spy IDENTIFIED BY secret"
echo "SQL CREATE USER intern IDENTIFIED BY coffee"
) | nc localhost 8085

echo "--- 3. Verify Default Deny ---"
echo "Trying to read as intern (should fail)..."
RES=$(echo -e "AUTH intern coffee\nSQL SELECT id, data FROM confidential" | nc localhost 8085)
echo "$RES"
if [[ "$RES" != *"permission denied"* ]]; then
    echo "FAIL: Intern read confidential data without permission"
    exit 1
fi

echo "--- 4. Grant Privileges & RLS ---"
(
echo "AUTH admin $FLYDB_ADMIN_PASSWORD"
# Grant intern access only to 'low' level data
echo "SQL GRANT SELECT ON confidential WHERE level = 'low' TO intern"
# Grant spy access to everything (no WHERE clause means full access? Our parser might require WHERE for now based on AST? No, AST has *Condition, nil means no RLS)
# Wait, my Grant implementation requires WHERE clause in parser?
# "Simplified: GRANT ON table [WHERE col=val] TO user" - yes, optional.
echo "SQL GRANT SELECT ON confidential TO spy"
) | nc localhost 8085

echo "--- 5. Verify RLS (Intern) ---"
echo "Reading as intern (should see only row 2)..."
RES=$(echo -e "AUTH intern coffee\nSQL SELECT id, data FROM confidential" | nc localhost 8085)
echo "$RES"
if [[ "$RES" == *"Nuclear codes"* ]]; then
    echo "FAIL: Intern saw Top Secret data!"
    exit 1
fi
if [[ "$RES" != *"Lunch menu"* ]]; then
    echo "FAIL: Intern could not see Lunch menu"
    exit 1
fi

echo "--- 6. Verify Full Access (Spy) ---"
echo "Reading as spy (should see everything)..."
RES=$(echo -e "AUTH spy secret\nSQL SELECT id, data FROM confidential" | nc localhost 8085)
echo "$RES"
if [[ "$RES" != *"Nuclear codes"* ]]; then
    echo "FAIL: Spy could not see Nuclear codes"
    exit 1
fi

# Cleanup
kill $PID
wait $PID 2>/dev/null || true
rm -f auth.fdb
echo "Auth & RLS Tests Passed!"
