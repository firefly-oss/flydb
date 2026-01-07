# FlyDB Roadmap

This document outlines the development roadmap for FlyDB, including completed features and planned enhancements.

**Version:** 01.26.2
**Last Updated:** January 7, 2026

---

## Completed Features

### Core Database Features

| Feature | Description | Version |
|---------|-------------|---------|
| SQL Query Support | CREATE TABLE, INSERT, SELECT, UPDATE, DELETE | 01.26.1 |
| SELECT * FROM | Retrieve all columns from a table | 01.26.1 |
| INNER JOIN | Nested Loop algorithm for table joins | 01.26.1 |
| WHERE Clause | Filtering with equality and comparison operators | 01.26.1 |
| ORDER BY | ASC/DESC sorting | 01.26.1 |
| LIMIT/OFFSET | Result set restriction and pagination | 01.26.1 |
| Schema Persistence | Schemas survive server restarts | 01.26.1 |
| Transactions | BEGIN, COMMIT, ROLLBACK for atomic operations | 01.26.1 |
| B-Tree Indexing | CREATE INDEX for O(log N) lookups | 01.26.1 |
| Prepared Statements | PREPARE, EXECUTE, DEALLOCATE for query reuse | 01.26.1 |
| Aggregate Functions | COUNT, SUM, AVG, MIN, MAX | 01.26.1 |
| GROUP BY | Group rows for aggregate calculations | 01.26.1 |
| HAVING | Filter groups after aggregation | 01.26.1 |
| DISTINCT | Remove duplicate rows from SELECT results | 01.26.1 |
| UNION/UNION ALL | Combine results from multiple SELECT queries | 01.26.1 |
| Subqueries | Nested SELECT statements in WHERE clauses | 01.26.1 |
| Stored Procedures | CREATE PROCEDURE, CALL, DROP PROCEDURE | 01.26.1 |
| Views | Virtual tables (CREATE VIEW, DROP VIEW) | 01.26.1 |
| Triggers | Automatic actions on INSERT/UPDATE/DELETE (BEFORE/AFTER) | 01.26.1 |
| ALTER TABLE | ADD/DROP/RENAME/MODIFY COLUMN | 01.26.1 |
| INTROSPECT Command | Database metadata inspection | 01.26.1 |
| Row Count Information | All queries return affected row counts | 01.26.1 |
| Pretty Table Formatting | Formatted table output in CLI | 01.26.1 |

### Constraints

| Feature | Description | Version |
|---------|-------------|---------|
| PRIMARY KEY | Unique identifier with NOT NULL | 01.26.1 |
| FOREIGN KEY | REFERENCES constraint with referential integrity | 01.26.1 |
| NOT NULL | Prevent NULL values in columns | 01.26.1 |
| UNIQUE | Ensure column values are unique | 01.26.1 |
| AUTO_INCREMENT/SERIAL | Automatic sequence generation | 01.26.1 |
| DEFAULT | Default values for columns | 01.26.1 |
| CHECK | Custom validation expressions | 01.26.1 |

### Extended Column Types

| Type | Description | Version |
|------|-------------|---------|
| INT, BIGINT | Integer types | 01.26.1 |
| SERIAL | Auto-incrementing integer | 01.26.1 |
| FLOAT, DECIMAL/NUMERIC | Floating-point and decimal types | 01.26.1 |
| TEXT, VARCHAR | String types | 01.26.1 |
| BOOLEAN | True/false values | 01.26.1 |
| TIMESTAMP, DATE, TIME | Date and time types | 01.26.1 |
| UUID | Universally unique identifier | 01.26.1 |
| BLOB | Binary data (base64 encoded) | 01.26.1 |
| JSONB | Binary JSON for structured data | 01.26.1 |

### Storage Engine

| Feature | Description | Version |
|---------|-------------|---------|
| Write-Ahead Logging (WAL) | Durability through append-only log | 01.26.1 |
| In-Memory KV Store | Fast reads with prefix scanning | 01.26.1 |
| Automatic Recovery | State reconstruction from WAL on startup | 01.26.1 |
| Binary WAL Format | Efficient storage format | 01.26.1 |
| Transaction Support | Write buffering with commit/rollback | 01.26.1 |
| Data Encryption at Rest | AES-256-GCM encryption for WAL entries | 01.26.1 |

### Security

| Feature | Description | Version |
|---------|-------------|---------|
| User Authentication | Username/password credentials | 01.26.1 |
| bcrypt Password Hashing | Secure credential storage | 01.26.1 |
| Timing Attack Prevention | Constant-time password comparison | 01.26.1 |
| Table-Level Access Control | GRANT/REVOKE statements | 01.26.1 |
| Row-Level Security (RLS) | Predicate-based row filtering | 01.26.1 |
| Built-in Admin Account | Bootstrap operations | 01.26.1 |

### Distributed Features

| Feature | Description | Version |
|---------|-------------|---------|
| Leader-Follower Replication | WAL streaming to followers | 01.26.1 |
| Binary Replication Protocol | Efficient TCP-based replication | 01.26.1 |
| Offset-Based Sync | Replica catch-up from any position | 01.26.1 |
| Automatic Retry | Connection failure recovery | 01.26.1 |
| Automatic Failover | Leader election using Bully algorithm | 01.26.1 |

### Performance Features

| Feature | Description | Version |
|---------|-------------|---------|
| Connection Pooling | Efficient connection management | 01.26.1 |
| Query Caching | LRU cache with TTL and auto-invalidation | 01.26.1 |
| TLS Support | Encrypted client-server connections | 01.26.1 |

### Wire Protocol

| Feature | Description | Version |
|---------|-------------|---------|
| Binary Protocol | High-performance binary encoding (default) | 01.26.1 |
| Text Protocol | Human-readable for debugging | 01.26.1 |
| WATCH Command | Real-time table change notifications | 01.26.1 |

### Observability

| Feature | Description | Version |
|---------|-------------|---------|
| Structured Logging | DEBUG, INFO, WARN, ERROR levels | 01.26.1 |
| JSON Log Output | For log aggregation systems | 01.26.1 |
| Error Codes & Hints | Comprehensive error handling | 01.26.1 |

---

## Planned Features

### High Priority

| Feature | Description | Status |
|---------|-------------|--------|
| - | All high-priority features completed | ✅ |

### Medium Priority

| Feature | Description | Status |
|---------|-------------|--------|
| - | All medium-priority features completed | ✅ |

### Low Priority

| Feature | Description | Status |
|---------|-------------|--------|
| Window Functions | OVER, PARTITION BY, ROW_NUMBER | 📋 Planned |

---

## Version History

| Version | Release Date | Highlights |
|---------|--------------|------------|
| 01.26.1 | January 2026 | Initial public release with full SQL support, triggers, replication, security, and encryption |

---

## Contributing

We welcome contributions! If you'd like to work on a planned feature or propose a new one:

1. Check the [Issues](https://github.com/firefly-oss/flydb/issues) for existing discussions
2. Open a new issue to discuss your proposal
3. Submit a pull request with your implementation

---

## See Also

- [README](README.md) - Project overview and quick start
- [Architecture](docs/architecture.md) - System design
- [Changelog](CHANGELOG.md) - Detailed version history

