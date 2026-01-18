# FlyDB Architecture

This document describes the overall system architecture of FlyDB, including component relationships, data flow, and high-level design.

## Overview

FlyDB is a lightweight SQL database written in Go, designed to demonstrate core database concepts while remaining fully functional. It follows a layered architecture that separates concerns and enables modularity.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Client Applications                        │
│         (fsql CLI, ODBC/JDBC drivers, connection pools)         │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Binary Protocol (:8889)                       │
│   • Length-prefixed JSON payloads  • Efficient for drivers      │
│   • Full SQL support               • Cursor operations          │
│   • Transaction management         • Metadata queries           │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Server Layer                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ TCP Server  │  │ Replicator  │  │   Cluster Manager       │  │
│  │ (server.go) │  │(replication)│  │   (leader election)     │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Database Manager                           │    │
│  │  • Multi-database support    • Per-connection context   │    │
│  │  • Database lifecycle        • Metadata management      │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                         SQL Layer                               │
│  ┌─────────┐  ┌─────────┐  ┌──────────┐  ┌─────────────────┐    │
│  │  Lexer  │→ │ Parser  │→ │ Executor │→ │ Query Cache     │    │
│  └─────────┘  └─────────┘  └──────────┘  └─────────────────┘    │
│                                │                                │
│              ┌─────────────────┼─────────────────┐              │
│              ▼                 ▼                 ▼              │
│      ┌─────────────┐   ┌─────────────┐   ┌─────────────┐        │
│      │   Catalog   │   │  Prepared   │   │  Triggers   │        │
│      │  (schemas)  │   │ Statements  │   │  Manager    │        │
│      └─────────────┘   └─────────────┘   └─────────────┘        │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Internationalization (I18N)                │    │
│  │  ┌───────────┐  ┌───────────┐  ┌───────────────────┐    │    │
│  │  │ Collator  │  │  Encoder  │  │  Locale Support   │    │    │
│  │  │ (sorting) │  │(validation│  │  (golang.org/x/   │    │    │
│  │  │           │  │           │  │   text/collate)   │    │    │
│  │  └───────────┘  └───────────┘  └───────────────────┘    │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Auth Layer                               │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │ Authentication  │  │  Authorization  │  │ Row-Level       │  │
│  │ (bcrypt)        │  │  (GRANT/REVOKE) │  │ Security (RLS)  │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Storage Layer                             │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Unified Disk Storage Engine                │    │
│  │   (Page-Based Storage with Intelligent Caching)         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Buffer Pool                          │    │
│  │   - LRU-K page replacement    - Auto-sized by memory    │    │
│  │   - Pin/unpin semantics       - Dirty page tracking     │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Heap File                            │    │
│  │   - 8KB pages               - Slotted page layout       │    │
│  │   - Free space management   - Record-level operations   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ Transaction │  │  Encryptor  │  │   Database Instance     │  │
│  │  Manager    │  │ (AES-256)   │  │   (per-db isolation)    │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Write-Ahead Log (WAL)                      │    │
│  │   - Durability guarantees   - Crash recovery            │    │
│  │   - Replication support     - Optional encryption       │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                       File System                               │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  data_dir/                                              │    │
│  │  ├── default/                                           │    │
│  │  │   ├── data.db      (heap file with pages)            │    │
│  │  │   └── wal.fdb      (write-ahead log)                 │    │
│  │  ├── myapp/                                             │    │
│  │  │   ├── data.db      (heap file with pages)            │    │
│  │  │   └── wal.fdb      (write-ahead log)                 │    │
│  │  └── _system/                                           │    │
│  │      ├── data.db      (system metadata)                 │    │
│  │      └── wal.fdb      (system WAL)                      │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## Component Overview

### Server Layer (`internal/server/`)

The server layer handles all network communication and coordination:

| Component | File | Purpose |
|-----------|------|---------|
| TCP Server | `server.go` | Multi-threaded connection handling, command dispatch |
| Replicator | `replication.go` | Leader-Follower WAL streaming replication |
| Cluster Manager | `unified.go` | Unified cluster management with Raft consensus |
| Raft Node | `raft.go` | Raft consensus implementation with pre-vote |

### SQL Layer (`internal/sql/`)

The SQL layer processes and executes SQL statements:

| Component | File | Purpose |
|-----------|------|---------|
| Lexer | `lexer.go` | Tokenizes SQL input into tokens |
| Parser | `parser.go` | Builds Abstract Syntax Tree (AST) |
| AST | `ast.go` | Statement and expression node definitions |
| Executor | `executor.go` | Executes AST against storage |
| Catalog | `catalog.go` | Manages table schemas |
| Types | `types.go` | SQL type validation and normalization |
| Prepared Statements | `prepared.go` | Parameterized query support |

### Auth Layer (`internal/auth/`)

The auth layer provides security features:

| Component | File | Purpose |
|-----------|------|---------|
| AuthManager | `auth.go` | User management, authentication, authorization |

### TLS Layer (`internal/tls/`)

The TLS layer provides transport security for client-server connections:

| Component | File | Purpose |
|-----------|------|---------|
| Certificate Manager | `certs.go` | TLS certificate generation, validation, and loading |

**Features:**
- **Auto-generation:** Self-signed certificates using ECDSA (P-256/P-384)
- **Validation:** Certificate expiration checking with 30-day warnings
- **Secure defaults:** TLS 1.2+ with modern cipher suites (ECDHE-ECDSA/RSA-AES-GCM, ChaCha20-Poly1305)
- **Automatic paths:** `/etc/flydb/certs/` for root, `~/.config/flydb/certs/` for users
- **Proper permissions:** 0644 for certificates, 0600 for private keys

### Storage Layer (`internal/storage/`)

The storage layer provides persistence and data management using a unified disk-based engine:

| Component | File | Purpose |
|-----------|------|---------|
| Engine | `engine.go` | Storage interface definition |
| StorageEngine | `storage_engine.go` | Extended interface with Sync/Stats |
| Factory | `factory.go` | Engine creation with auto-sizing buffer pool |
| WAL | `wal.go` | Write-Ahead Log for durability |
| B-Tree | `btree.go` | Index data structure |
| Index Manager | `index.go` | Index lifecycle management |
| Transaction | `transaction.go` | ACID transaction support |
| Encryptor | `encryption.go` | AES-256-GCM encryption |
| Database | `database.go` | Database struct with metadata |
| DatabaseManager | `database.go` | Multi-database lifecycle management |
| Collation | `collation.go` | String comparison rules |
| Encoding | `encoding.go` | Character encoding validation |

### Disk Storage Engine (`internal/storage/disk/`)

The disk storage engine provides PostgreSQL-style page-based storage with intelligent caching:

| Component | File | Purpose |
|-----------|------|---------|
| Page | `page.go` | 8KB page with slotted layout |
| HeapFile | `heap_file.go` | Page allocation and free list management |
| BufferPool | `buffer_pool.go` | LRU-K page caching with auto-sizing |
| DiskStorageEngine | `disk_engine.go` | Unified disk-based Engine implementation |
| Checkpoint | `checkpoint.go` | Periodic full database snapshots |

**Key Features:**
- **Auto-sized Buffer Pool**: Automatically sizes based on available system memory (25% of RAM)
- **LRU-K Replacement**: Intelligent page eviction based on access frequency
- **Dirty Page Tracking**: Efficient write-back of modified pages
- **Checkpoint Recovery**: Fast startup by loading from checkpoint + WAL replay

### CLI/Shell (`cmd/flydb-shell/`)

The interactive shell provides a PostgreSQL-like experience:

| Feature | Description |
|---------|-------------|
| Multi-mode | Normal mode (commands) and SQL mode (direct SQL) |
| Tab completion | Commands, keywords, table names |
| History | Persistent command history with readline |
| Database context | Tracks current database, shown in prompt |
| Backslash commands | `\dt`, `\du`, `\db`, `\c`, etc. |

**Shell Prompt Format:**
```
flydb>                    -- Normal mode (default database)
flydb/mydb>               -- Normal mode (non-default database)
flydb/mydb[sql]>          -- SQL mode (non-default database)
```

### Supporting Packages

| Package | Purpose |
|---------|---------|
| `internal/cache/` | LRU query result cache with TTL expiration |
| `internal/pool/` | Client-side connection pooling |
| `internal/protocol/` | Binary wire protocol for JDBC/ODBC drivers |
| `internal/sdk/` | SDK types for driver development (cursors, sessions, transactions) |
| `internal/errors/` | Structured error handling with codes |
| `internal/logging/` | Structured logging framework |
| `internal/banner/` | Startup banner display |
| `internal/config/` | Configuration management with YAML support |
| `internal/wizard/` | Interactive setup wizard |
| `internal/audit/` | Comprehensive audit trail system for compliance and security |
| `pkg/cli/` | CLI utilities (colors, formatting, prompts) |

## Data Flow

### Query Execution Flow

```
1. Client sends: "SQL SELECT * FROM users WHERE id = 1"
                            │
                            ▼
2. Server receives command, extracts SQL statement
                            │
                            ▼
3. Lexer tokenizes: [SELECT, *, FROM, users, WHERE, id, =, 1]
                            │
                            ▼
4. Parser builds AST: SelectStmt{
                        TableName: "users",
                        Columns: ["*"],
                        Where: {Column: "id", Value: "1"}
                      }
                            │
                            ▼
5. Executor checks permissions via AuthManager
                            │
                            ▼
6. Executor queries KVStore: Scan("row:users:")
                            │
                            ▼
7. KVStore returns matching rows from in-memory HashMap
                            │
                            ▼
8. Executor applies WHERE filter and RLS conditions
                            │
                            ▼
9. Results formatted and sent to client
```

### Write Path (Durability)

```
1. Client sends: "SQL INSERT INTO users VALUES ('alice', 25)"
                            │
                            ▼
2. Executor validates against schema (Catalog)
                            │
                            ▼
3. WAL.Write() appends operation to disk
   ┌─────────┬───────────┬─────────────┬─────────────┬─────────────┐
   │ Op (1B) │ KeyLen(4B)│ Key (var)   │ ValLen (4B) │ Value (var) │
   └─────────┴───────────┴─────────────┴─────────────┴─────────────┘
                            │
                            ▼
4. KVStore updates in-memory HashMap
                            │
                            ▼
5. IndexManager updates B-Tree indexes
                            │
                            ▼
6. OnInsert callback notifies WATCH subscribers
                            │
                            ▼
7. "INSERT 1" returned to client
```

### Replication Flow

```
┌─────────────────┐                    ┌─────────────────┐
│     LEADER      │                    │    FOLLOWER     │
│                 │                    │                 │
│  ┌───────────┐  │                    │  ┌───────────┐  │
│  │  KVStore  │  │                    │  │  KVStore  │  │
│  └─────┬─────┘  │                    │  └─────▲─────┘  │
│        │        │                    │        │        │
│  ┌─────▼─────┐  │   WAL Streaming    │  ┌─────┴─────┐  │
│  │    WAL    │──┼───────────────────►│  │    WAL    │  │
│  └───────────┘  │   (100ms polling)  │  └───────────┘  │
│                 │                    │                 │
└─────────────────┘                    └─────────────────┘

1. Follower connects to Leader's replication port
2. Follower sends current WAL offset
3. Leader streams WAL entries from that offset
4. Follower applies entries to local KVStore
5. Follower updates its WAL offset
```

## Multi-Database Architecture

FlyDB supports multiple isolated databases within a single server instance:

```
┌─────────────────────────────────────────────────────────────────┐
│                      DatabaseManager                            │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │   Database:     │  │   Database:     │  │   Database:     │  │
│  │   "default"     │  │   "analytics"   │  │   "staging"     │  │
│  │                 │  │                 │  │                 │  │
│  │  ┌───────────┐  │  │  ┌───────────┐  │  │  ┌───────────┐  │  │
│  │  │  KVStore  │  │  │  │  KVStore  │  │  │  │  KVStore  │  │  │
│  │  └───────────┘  │  │  └───────────┘  │  │  └───────────┘  │  │
│  │  ┌───────────┐  │  │  ┌───────────┐  │  │  ┌───────────┐  │  │
│  │  │  Executor │  │  │  │  Executor │  │  │  │  Executor │  │  │
│  │  └───────────┘  │  │  └───────────┘  │  │  └───────────┘  │  │
│  │  ┌───────────┐  │  │  ┌───────────┐  │  │  ┌───────────┐  │  │
│  │  │  Catalog  │  │  │  │  Catalog  │  │  │  │  Catalog  │  │  │
│  │  └───────────┘  │  │  └───────────┘  │  │  └───────────┘  │  │
│  │                 │  │                 │  │                 │  │
│  │  Metadata:      │  │  Metadata:      │  │  Metadata:      │  │
│  │  - Encoding     │  │  - Encoding     │  │  - Encoding     │  │
│  │  - Locale       │  │  - Locale       │  │  - Locale       │  │
│  │  - Collation    │  │  - Collation    │  │  - Collation    │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
│                                                                 │
│  File System:                                                   │
│  data_dir/                                                      │
│  ├── default.fdb                                                │
│  ├── analytics.fdb                                              │
│  └── staging.fdb                                                │
└─────────────────────────────────────────────────────────────────┘
```

### Database Isolation

Each database has:
- **Separate KVStore**: Complete data isolation
- **Own Executor**: Independent query execution
- **Own Catalog**: Separate table schemas
- **Own Indexes**: B-Tree indexes scoped to database
- **Metadata**: Encoding, locale, collation, owner

### Database Metadata

| Property | Description | Values |
|----------|-------------|--------|
| Owner | User who created the database | Username |
| Encoding | Character encoding | UTF8, LATIN1, ASCII, UTF16 |
| Locale | Locale for sorting | en_US, de_DE, fr_FR, etc. |
| Collation | String comparison rules | default, binary, nocase, unicode |
| Description | Optional description | Free text |
| ReadOnly | Read-only mode | true/false |

### Connection Database Context

Each client connection maintains its own database context:

```
┌─────────────────┐     ┌─────────────────┐
│  Connection 1   │     │  Connection 2   │
│  Database: app  │     │  Database: logs │
└────────┬────────┘     └────────┬────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│  Executor: app  │     │  Executor: logs │
└─────────────────┘     └─────────────────┘

### Cross-Database Queries

FlyDB supports querying data across multiple databases using fully qualified table names (`database.table`). This allows for:
- **Cross-Database Joins**: `SELECT ... FROM db1.users JOIN db2.orders ...`
- **Data Migration**: `INSERT INTO db2.archive SELECT * FROM db1.active ...`

Permissions are enforced for the accessing user on all accessed objects, regardless of the database they reside in.
```

## Collation, Encoding, and Locale System

FlyDB provides comprehensive internationalization support through its collation, encoding, and locale subsystems. These features enable proper handling of text data across different languages and character sets.

### Collation Architecture

Collation determines how strings are compared and sorted. FlyDB implements the `Collator` interface:

```go
type Collator interface {
    Compare(a, b string) int  // Returns -1, 0, or 1
    Equal(a, b string) bool   // Equality check
    SortKey(s string) []byte  // Key for efficient sorting
}
```

**Supported Collations:**

| Collation | Description | Use Case |
|-----------|-------------|----------|
| `default` | Go's native string comparison | General purpose, fast |
| `nocase` | Case-insensitive comparison | Email addresses, usernames |
| `binary` | Byte-by-byte comparison | Exact matching, hashes |
| `unicode` | Locale-aware Unicode collation | International text |

**Collation Flow in Query Execution:**

```
┌─────────────────────────────────────────────────────────────────┐
│                      SQL Query                                  │
│  SELECT * FROM users WHERE name = 'José' ORDER BY name          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Executor                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Collator (from database metadata)                       │    │
│  │ - Unicode collator with locale "es_ES"                  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                  │
│              ┌───────────────┴───────────────┐                  │
│              ▼                               ▼                  │
│  ┌─────────────────────┐         ┌─────────────────────┐        │
│  │ WHERE Evaluation    │         │ ORDER BY Sorting    │        │
│  │ collator.Equal()    │         │ collator.Compare()  │        │
│  └─────────────────────┘         └─────────────────────┘        │
└─────────────────────────────────────────────────────────────────┘
```

### Encoding Architecture

Encoding ensures data integrity by validating that text conforms to the database's character encoding:

```go
type Encoder interface {
    Validate(s string) error      // Check if string is valid
    Encode(s string) ([]byte, error)   // Convert to bytes
    Decode(b []byte) (string, error)   // Convert from bytes
    Name() string                 // Encoding name
}
```

**Supported Encodings:**

| Encoding | Description | Byte Range |
|----------|-------------|------------|
| `UTF8` | Unicode UTF-8 (default) | 1-4 bytes per char |
| `ASCII` | 7-bit ASCII | 1 byte per char |
| `LATIN1` | ISO-8859-1 | 1 byte per char |
| `UTF16` | Unicode UTF-16 | 2-4 bytes per char |

**Encoding Validation Flow:**

```
┌─────────────────────────────────────────────────────────────────┐
│  INSERT INTO users (name) VALUES ('Müller')                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Executor                                   │
│                              │                                  │
│              ┌───────────────┴───────────────┐                  │
│              ▼                               ▼                  │
│  ┌─────────────────────┐         ┌─────────────────────┐        │
│  │ Type Validation     │         │ Encoding Validation │        │
│  │ ValidateValue()     │         │ encoder.Validate()  │        │
│  └─────────────────────┘         └─────────────────────┘        │
│              │                               │                  │
│              │         ┌─────────────────────┘                  │
│              │         │                                        │
│              │    ┌────▼────┐                                   │
│              │    │ ASCII?  │──No──► Error: invalid encoding    │
│              │    │ UTF8?   │──Yes─► Continue                   │
│              │    └─────────┘                                   │
│              ▼                                                  │
│  ┌─────────────────────┐                                        │
│  │ Store in KVStore    │                                        │
│  └─────────────────────┘                                        │
└─────────────────────────────────────────────────────────────────┘
```

### Locale Support

Locale affects how the Unicode collator sorts and compares strings. FlyDB uses Go's `golang.org/x/text/collate` package for locale-aware operations:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Locale Examples                              │
├─────────────────────────────────────────────────────────────────┤
│  Locale: en_US                                                  │
│  Sort: A, B, C, ... Z, a, b, c, ... z                           │
│                                                                 │
│  Locale: de_DE                                                  │
│  Sort: A, Ä, B, C, ... (ä sorts with a)                         │
│  Special: ß = ss                                                │
│                                                                 │
│  Locale: sv_SE                                                  │
│  Sort: A, B, C, ... Z, Å, Ä, Ö                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Database Creation with I18N Options

```sql
CREATE DATABASE german_app
  ENCODING UTF8
  LOCALE de_DE
  COLLATION unicode;
```

This creates a database where:
- All text is validated as UTF-8
- String comparisons use German sorting rules
- Umlauts (ä, ö, ü) sort correctly with their base letters

## Key Storage Conventions

FlyDB uses a key prefix convention to organize data in the KVStore:

| Prefix | Purpose | Example |
|--------|---------|---------|
| `row:<table>:<id>` | Table row data (JSON) | `row:users:1` |
| `seq:<table>` | Auto-increment sequence | `seq:users` |
| `schema:<table>` | Table schema definition | `schema:users` |
| `_sys_users:<username>` | User credentials | `_sys_users:alice` |
| `_sys_privs:<user>:<table>` | User permissions | `_sys_privs:alice:orders` |
| `_sys_db_privs:<user>:<db>` | Database permissions | `_sys_db_privs:alice:analytics` |
| `_audit:<timestamp>:<id>` | Audit log entries | `_audit:1705507200000000000:1` |
| `_sys_db_meta` | Database metadata | `_sys_db_meta` |
| `index:<table>:<column>` | Index metadata | `index:users:email` |
| `proc:<name>` | Stored procedure | `proc:get_user` |
| `view:<name>` | View definition | `view:active_users` |

## Disk Storage Engine Architecture

The disk storage engine provides PostgreSQL-style page-based storage for datasets larger than RAM:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Disk Storage Engine                          │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   Buffer Pool (LRU)                     │    │
│  │   ┌─────────┐ ┌─────────┐ ┌─────────┐     ┌─────────┐   │    │
│  │   │ Frame 0 │ │ Frame 1 │ │ Frame 2 │ ... │ Frame N │   │    │
│  │   │ [Page]  │ │ [Page]  │ │ [Page]  │     │ [Page]  │   │    │
│  │   │ pin:2   │ │ pin:0   │ │ pin:1   │     │ pin:0   │   │    │
│  │   └─────────┘ └─────────┘ └─────────┘     └─────────┘   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Heap File                            │    │
│  │   ┌─────────┐ ┌─────────┐ ┌─────────┐     ┌─────────┐   │    │
│  │   │ Header  │ │ Page 1  │ │ Page 2  │ ... │ Page N  │   │    │
│  │   │ (8KB)   │ │ (8KB)   │ │ (8KB)   │     │ (8KB)   │   │    │
│  │   └─────────┘ └─────────┘ └─────────┘     └─────────┘   │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### Page Layout (Slotted Page)

Each 8KB page uses a slotted page layout for variable-length records:

```
┌─────────────────────────────────────────────────────────────────┐
│                        Page (8192 bytes)                        │
├─────────────────────────────────────────────────────────────────┤
│  Page Header (24 bytes)                                         │
│  ┌─────────┬──────────┬───────┬───────────┬───────────┬───────┐ │
│  │ PageID  │ PageType │ Flags │ SlotCount │ FreeSpace │  LSN  │ │
│  │  (4B)   │   (1B)   │ (1B)  │   (2B)    │   (4B)    │ (4B)  │ │
│  └─────────┴──────────┴───────┴───────────┴───────────┴───────┘ │
├─────────────────────────────────────────────────────────────────┤
│  Slot Array (grows down)                                        │
│  ┌──────────────┬──────────────┬──────────────┐                 │
│  │ Slot 0       │ Slot 1       │ Slot 2       │ ...             │
│  │ offset:len   │ offset:len   │ offset:len   │                 │
│  └──────────────┴──────────────┴──────────────┘                 │
├─────────────────────────────────────────────────────────────────┤
│                     Free Space                                  │
├─────────────────────────────────────────────────────────────────┤
│  Records (grow up from bottom)                                  │
│  ┌──────────────┬──────────────┬──────────────┐                 │
│  │ Record 2     │ Record 1     │ Record 0     │                 │
│  │ (variable)   │ (variable)   │ (variable)   │                 │
│  └──────────────┴──────────────┴──────────────┘                 │
└─────────────────────────────────────────────────────────────────┘
```

### Configuration

The storage engine is selected via configuration:

```json
# Storage engine: "memory" (default) or "disk"
storage_engine = "disk"

# Buffer pool size in pages (8KB each)
buffer_pool_size = 1024  # 8MB

# Checkpoint interval in seconds (0 = disabled)
checkpoint_secs = 60
```

Or via environment variables:
- `FLYDB_STORAGE_ENGINE`: "memory" or "disk"
- `FLYDB_BUFFER_POOL_SIZE`: Number of pages
- `FLYDB_CHECKPOINT_SECS`: Checkpoint interval

## Audit Trail System

FlyDB provides a comprehensive audit trail system for tracking all database operations, authentication events, and administrative actions. This is essential for security, compliance, and debugging.

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    SQL Executor                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  executeCreate(), executeInsert(), executeUpdate(), ...  │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │           logAuditEvent()                                │   │
│  │  • Captures operation details                            │   │
│  │  • Records username, database, client IP                 │   │
│  │  • Tracks duration and status                            │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Audit Manager                                │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Buffered Channel (async logging)                        │   │
│  │  • Non-blocking event submission                         │   │
│  │  • Batch processing for performance                      │   │
│  │  • Configurable buffer size                              │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Event Filtering                                         │   │
│  │  • Filter by event type (DDL, DML, DCL, etc.)            │   │
│  │  • Configurable SELECT logging                           │   │
│  │  • Authentication event tracking                         │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Storage (KVStore)                                       │   │
│  │  Key: _audit:<timestamp>:<id>                            │   │
│  │  Value: JSON-encoded audit event                         │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Event Types

| Category | Event Types | Default Logging |
|----------|-------------|-----------------|
| **Authentication** | LOGIN, LOGOUT, AUTH_FAILED | Enabled |
| **DDL** | CREATE_TABLE, DROP_TABLE, ALTER_TABLE, CREATE_INDEX, DROP_INDEX, CREATE_VIEW, DROP_VIEW, TRUNCATE_TABLE | Enabled |
| **DML** | INSERT, UPDATE, DELETE | Enabled |
| **DML (Read)** | SELECT | Disabled (can be verbose) |
| **DCL** | GRANT, REVOKE, CREATE_USER, ALTER_USER, DROP_USER, CREATE_ROLE, DROP_ROLE | Enabled |
| **Transactions** | BEGIN, COMMIT, ROLLBACK | Enabled |
| **Administrative** | BACKUP, RESTORE, CHECKPOINT, VACUUM | Enabled |
| **Cluster** | NODE_JOIN, NODE_LEAVE, LEADER_ELECTION, FAILOVER | Enabled |
| **Database** | CREATE_DATABASE, DROP_DATABASE, USE_DATABASE | Enabled |

### Audit Log Schema

Each audit log entry contains:

| Field | Type | Description |
|-------|------|-------------|
| `id` | INT64 | Unique event identifier |
| `timestamp` | TIMESTAMP | When the event occurred |
| `event_type` | TEXT | Type of event (see Event Types) |
| `username` | TEXT | User who performed the action |
| `database` | TEXT | Database context |
| `object_type` | TEXT | Type of object (table, index, user, etc.) |
| `object_name` | TEXT | Name of the object |
| `operation` | TEXT | Full SQL statement or operation |
| `client_addr` | TEXT | Client IP address |
| `session_id` | TEXT | Session identifier |
| `status` | TEXT | SUCCESS or FAILED |
| `error_message` | TEXT | Error details if failed |
| `duration_ms` | INT64 | Operation duration in milliseconds |
| `metadata` | JSONB | Additional context (e.g., node_id in cluster mode) |

### Configuration

Audit logging is configured via:

```yaml
audit_enabled: true              # Enable/disable audit logging
audit_log_ddl: true              # Log DDL operations
audit_log_dml: true              # Log DML operations
audit_log_select: false          # Log SELECT queries (can be verbose)
audit_log_auth: true             # Log authentication events
audit_log_admin: true            # Log administrative operations
audit_log_cluster: true          # Log cluster events
audit_retention_days: 90         # Days to retain logs (0 = forever)
audit_buffer_size: 1000          # Event buffer size
audit_flush_interval_sec: 5      # Flush interval in seconds
```

### Cluster Mode

In cluster mode, audit logs are distributed:

- Each node maintains its own audit log
- Audit events include `node_id` in metadata
- Queries can aggregate logs from all nodes
- Export functionality works across the cluster

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│   Node 1    │  │   Node 2    │  │   Node 3    │
│  Audit Mgr  │  │  Audit Mgr  │  │  Audit Mgr  │
│  Local Logs │  │  Local Logs │  │  Local Logs │
└──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │                │                │
       └────────────────┼────────────────┘
                        │
                        ▼
              ┌─────────────────┐
              │ Cluster Audit   │
              │ Aggregator      │
              │ • Query all     │
              │ • Export all    │
              │ • Statistics    │
              └─────────────────┘
```

### CLI Commands

| Command | Description |
|---------|-------------|
| `\audit` | Show recent audit logs |
| `\audit-user <username>` | Show logs for specific user |
| `\audit-export <file> [format]` | Export logs (json, csv, sql) |
| `\audit-stats` | Show audit statistics |
| `INSPECT AUDIT [WHERE ...] [LIMIT n]` | Query audit logs with filters |
| `INSPECT AUDIT STATS` | Get detailed statistics |
| `EXPORT AUDIT TO '<file>' FORMAT <format>` | Export audit logs |

### SDK Integration

The SDK provides a type-safe audit client:

```go
// Create audit client
auditClient := sdk.NewAuditClient(session)

// Query recent logs
logs, err := auditClient.GetRecentLogs(100)

// Query by user
logs, err := auditClient.GetLogsByUser("admin", 50)

// Query by time range
logs, err := auditClient.GetLogsInTimeRange(startTime, endTime, 100)

// Export logs
err := auditClient.ExportLogs("audit.json", sdk.AuditFormatJSON, queryOpts)

// Get statistics
stats, err := auditClient.GetStatistics()
```

### Performance

The audit system is designed for minimal performance impact:

- **Asynchronous logging**: Events are buffered and processed in background
- **Batch writes**: Multiple events are written together
- **Non-blocking**: Failed audit writes don't block operations
- **Configurable filtering**: Reduce volume by disabling verbose events
- **Automatic cleanup**: Old logs are purged based on retention policy

## Thread Safety Model

FlyDB uses Go's concurrency primitives for thread safety:

- **KVStore**: `sync.RWMutex` for concurrent read access, exclusive writes
- **BufferPool**: `sync.Mutex` for frame allocation and eviction
- **WAL**: `sync.Mutex` for serialized append operations
- **Server**: Per-connection goroutines with mutex-protected shared state
- **Subscribers**: `sync.Mutex` protects the WATCH subscription map
- **Transactions**: Per-connection transaction isolation

## Network Protocol

### Binary Protocol (Port 8889)

FlyDB uses a binary wire protocol for all client communication. The protocol provides:

- **Efficient framing**: 8-byte header with length-prefixed payloads
- **JSON payloads**: Human-readable for debugging, efficient for parsing
- **Full SQL support**: Queries, prepared statements, transactions
- **Driver support**: Cursors, metadata queries, session management

```
Message Format:
┌───────────┬─────────┬──────────┬───────────┬────────────┬─────────────────┐
│ Magic (1B)│ Ver (1B)│ Type (1B)│ Flags (1B)│ Length (4B)│ Payload (var)   │
└───────────┴─────────┴──────────┴───────────┴────────────┴─────────────────┘

Magic: 0xFD (FlyDB identifier)
Version: 0x01 (current protocol version)
Max payload: 16 MB
```

See [Driver Development Guide](driver-development.md) for complete protocol specification.

### Reactive Event System (WATCH)

FlyDB provides a reactive event system for real-time data change notifications:

**Event Types:**
| Event | Format | Description |
|-------|--------|-------------|
| INSERT | `EVENT INSERT <table> <json>` | Row inserted |
| UPDATE | `EVENT UPDATE <table> <old_json> <new_json>` | Row updated |
| DELETE | `EVENT DELETE <table> <json>` | Row deleted |
| SCHEMA | `EVENT SCHEMA <type> <object> <details>` | Schema changed |

**Schema Event Types:**
- `CREATE_TABLE`, `DROP_TABLE`, `ALTER_TABLE`
- `CREATE_VIEW`, `DROP_VIEW`
- `CREATE_TRIGGER`, `DROP_TRIGGER`
- `TRUNCATE_TABLE`

**Example Usage:**
```
WATCH users
WATCH SCHEMA
# Client receives events as they occur:
# EVENT INSERT users {"id":1,"name":"Alice"}
# EVENT UPDATE users {"id":1,"name":"Alice"} {"id":1,"name":"Alice Smith"}
# EVENT SCHEMA CREATE_TABLE orders {}
```

### Binary Protocol (Port 8889)

The binary protocol provides a complete wire protocol for developing external JDBC/ODBC drivers:

```
┌───────────┬─────────┬──────────┬───────────┬────────────┬─────────────────┐
│ Magic (1B)│ Ver (1B)│ Type (1B)│ Flags (1B)│ Length (4B)│ Payload (var)   │
│   0xFD    │   0x01  │          │           │ Big-endian │                 │
└───────────┴─────────┴──────────┴───────────┴────────────┴─────────────────┘
```

**Message Categories:**

| Category | Types | Purpose |
|----------|-------|---------|
| Core | Query, Prepare, Execute, Deallocate | SQL execution |
| Cursors | CursorOpen, CursorFetch, CursorClose | Large result sets |
| Metadata | GetTables, GetColumns, GetTypeInfo | Schema discovery |
| Transactions | BeginTx, CommitTx, RollbackTx | Transaction control |
| Sessions | SetOption, GetOption, GetServerInfo | Connection management |
| Database | UseDatabase, GetDatabases | Multi-database support |

### ODBC/JDBC Driver Support Components

The binary protocol handler uses two key interfaces to support ODBC/JDBC driver operations:

```
┌─────────────────────────────────────────────────────────────────┐
│                    BinaryHandler                                 │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              MetadataProvider                           │    │
│  │  - GetTables()      Schema discovery                    │    │
│  │  - GetColumns()     Column metadata                     │    │
│  │  - GetPrimaryKeys() Primary key info                    │    │
│  │  - GetForeignKeys() Foreign key relationships           │    │
│  │  - GetIndexes()     Index information                   │    │
│  │  - GetTypeInfo()    Supported SQL types                 │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              DatabaseManager                            │    │
│  │  - UseDatabase()     Switch database context            │    │
│  │  - DatabaseExists()  Check database availability        │    │
│  │  - ListDatabases()   Enumerate available databases      │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

**MetadataProvider** bridges the protocol handler with the SQL catalog, providing ODBC/JDBC-compatible metadata responses. It accesses the Executor's catalog and index manager to retrieve schema information.

**DatabaseManager** enables multi-database support for drivers, allowing connections to switch between databases and enumerate available databases.

See [Driver Development Guide](driver-development.md) for complete protocol specification.

## Cluster Architecture

FlyDB implements a **true distributed database** with horizontal scaling through data sharding and replication:

```
┌─────────────────────────────────────────────────────────────┐
│              FlyDB Distributed Cluster Architecture         │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                  Consistent Hash Ring                 │  │
│  │   ┌────┐  ┌────┐  ┌────┐  ┌────┐  ┌────┐  ┌────┐      │  │
│  │   │ P0 │→ │ P1 │→ │ P2 │→ │... │→ │P254│→ │P255│→     │  │
│  │   └────┘  └────┘  └────┘  └────┘  └────┘  └────┘      │  │
│  │      ↓       ↓       ↓                ↓       ↓       │  │
│  │   Node1   Node2   Node3            Node1   Node2      │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    │
│    │   Node 1    │    │   Node 2    │    │   Node 3    │    │
│    │  (Leader)   │◄──►│ (Follower)  │◄──►│ (Follower)  │    │
│    │  Term: 5    │    │  Term: 5    │    │  Term: 5    │    │
│    │             │    │             │    │             │    │
│    │ Partitions: │    │ Partitions: │    │ Partitions: │    │
│    │ 0-84 (P)    │    │ 85-169 (P)  │    │ 170-255 (P) │    │
│    │ 85-169 (R)  │    │ 170-255 (R) │    │ 0-84 (R)    │    │
│    │ 170-255 (R) │    │ 0-84 (R)    │    │ 85-169 (R)  │    │
│    │             │    │             │    │             │    │
│    │ Data: 33%   │    │ Data: 33%   │    │ Data: 33%   │    │
│    └─────────────┘    └─────────────┘    └─────────────┘    │
│                                                             │
│    P = Primary (leader for partition)                       │
│    R = Replica (follower for partition)                     │
└─────────────────────────────────────────────────────────────┘
```

### Horizontal Scaling Features

**Data Sharding:**
- 256 partitions by default (configurable)
- Consistent hashing with virtual nodes (150 per physical node)
- Even data distribution across cluster
- Automatic partition assignment and rebalancing
- **Per-partition key index** for optimized scanning and migration

**Partition Migration:**
- Live data migration between nodes during rebalance
- Zero-downtime partition ownership transfer
- Robust completion signaling between source and destination nodes
- Automatic recovery of interrupted migrations

**Partition-Aware Routing:**
```
Client Request: GET user:12345
       │
       ▼
┌──────────────────┐
│  Hash("user:...")│  → Partition 42
└──────────────────┘
       │
       ▼
┌──────────────────┐
│ Partition 42     │  → Node 1 (primary)
│ Replicas: N2, N3 │
└──────────────────┘
       │
       ▼
┌──────────────────┐
│ Route to Node 1  │  → Read from primary
└──────────────────┘
```

**Cross-Partition Queries:**
```
Client Request: SELECT * FROM users
       │
       ▼
┌──────────────────────────────────────┐
│  Scatter-Gather Query Coordinator    │
└──────────────────────────────────────┘
       │
       ├─────────────┬─────────────┐
       ▼             ▼             ▼
   ┌───────┐     ┌───────┐     ┌───────┐
   │ Node1 │     │ Node2 │     │ Node3 │
   │ P0-84 │     │P85-169│     │P170-  │
   │       │     │       │     │ 255   │
   └───┬───┘     └───┬───┘     └───┬───┘
       │             │             │
       └─────────────┴─────────────┘
                     │
                     ▼
              ┌─────────────┐
              │ Merge Results│
              └─────────────┘
```

**Raft Consensus Features:**
- Log replication with strong consistency guarantees
- Pre-vote protocol to prevent disruption from partitioned nodes
- Term-based leader election with randomized timeouts
- Quorum-based commit (majority acknowledgment required)
- Automatic leader election on failure
- Log compaction and snapshotting

**Failover Process:**
1. Followers detect missing heartbeat (election timeout)
2. Follower transitions to Candidate, increments term
3. Candidate requests votes from all peers
4. Majority vote grants leadership
5. New leader begins sending AppendEntries
6. Partitions rebalance if needed

### Consensus Algorithms

| Algorithm | Config | Description |
|-----------|--------|-------------|
| **Raft** (default) | `enable_raft: true` | Full Raft consensus with log replication |
| Bully (legacy) | `enable_raft: false` | Simple leader election based on node ID |

### Cluster Events

The cluster manager emits events for monitoring and integration:

| Event | Description |
|-------|-------------|
| `LEADER_ELECTED` | A new leader was elected |
| `LEADER_STEP_DOWN` | Leader voluntarily stepped down |
| `NODE_JOINED` | A new node joined the cluster |
| `NODE_LEFT` | A node left the cluster |
| `NODE_UNHEALTHY` | A node became unhealthy |
| `NODE_RECOVERED` | An unhealthy node recovered |
| `QUORUM_LOST` | Cluster lost quorum |
| `QUORUM_RESTORED` | Cluster regained quorum |
| `SPLIT_BRAIN_RISK` | Potential split-brain detected |

### Replication Modes

FlyDB supports multiple replication modes for different consistency requirements:

| Mode | Description | Use Case |
|------|-------------|----------|
| `ASYNC` | Return immediately, replicate in background | Maximum performance |
| `SEMI_SYNC` | Wait for at least one replica to acknowledge | Balanced |
| `SYNC` | Wait for all replicas to acknowledge | Maximum durability |

### Advanced Horizontal Scaling

FlyDB implements production-grade horizontal scaling with multiple routing strategies, comprehensive metadata management, and performance optimizations.

#### Routing Strategies

FlyDB supports **5 routing strategies** optimized for different workloads:

**1. Key-Based Routing (Default)**
- Uses consistent hashing for deterministic partition placement
- Ensures data locality for related keys
- Minimal data movement during rebalancing
- Best for: General-purpose workloads, range queries

**2. Round-Robin Routing**
- Distributes requests evenly across all nodes
- Perfect load balancing
- No data locality guarantees
- Best for: Write-heavy workloads, uniform distribution

**3. Least-Loaded Routing**
- Routes to node with lowest current load (CPU, memory, connections, QPS)
- Automatic load balancing
- Handles heterogeneous hardware
- Best for: Variable workloads, auto-scaling

**4. Locality-Aware Routing**
- Prefers nodes in same datacenter/rack/zone
- Reduces cross-DC traffic
- Lower latency for reads
- Best for: Multi-datacenter deployments, geo-distributed clusters

**5. Hybrid Routing (Recommended for Production)**
- Combines key-based routing with locality awareness
- Key-based for writes (consistency)
- Locality-aware for reads (performance)
- Least-loaded for replica selection
- Best for: Production deployments

Configuration:
```yaml
routing_strategy: "hybrid"  # key_based, round_robin, least_loaded, locality_aware, hybrid
datacenter: "us-east-1"
rack: "rack-1"
zone: "zone-a"
```

#### Metadata Management

FlyDB maintains comprehensive metadata for efficient cluster operations:

**Cluster Metadata:**
- Version tracking for optimistic concurrency control
- Node registry with capacity and load metrics
- Partition assignments with replication status
- Routing tables for O(1) partition lookups
- Persistent storage with CRC32 integrity verification

**Node Metadata:**
Each node tracks:
- Hardware capacity (CPU, memory, disk, network)
- Current load (CPU usage, memory usage, connections, QPS)
- Topology (datacenter, rack, zone)
- Health status (healthy, degraded, unhealthy)
- Partition ownership (primary and replica partitions)

**Partition Metadata:**
Each partition tracks:
- Leader and replica nodes
- State (healthy, migrating, degraded, unavailable)
- Data statistics (key count, data size)
- Replication lag per replica
- Migration progress (if migrating)
- Performance metrics (read/write QPS)

**Routing Table:**
Fast lookups for:
- Partition ID → Primary node (O(1))
- Partition ID → Replica nodes (O(1))
- Node ID → Primary partitions (O(1))
- Node ID → Replica partitions (O(1))

#### Performance Optimizations

**Zero-Copy I/O:**
- **sendfile()** on Linux for file-to-socket transfers (5-10x faster)
- **splice()** on Linux for socket-to-socket transfers
- **Memory-mapped I/O** for large file operations
- Eliminates user-space memory copies
- Reduces CPU usage by 50-70%

**Connection Pooling:**
- Per-node connection pools with configurable limits
- Automatic health checking and idle timeout
- Connection reuse across requests
- 3-5x reduction in connection overhead

Configuration:
```yaml
connection_pool:
  max_idle_per_node: 10
  max_open_per_node: 100
  idle_timeout: 5m
  dial_timeout: 2s
```

**Adaptive Buffering:**
- Automatically adjusts buffer size based on workload
- Tracks average write size and resizes dynamically
- Buffer pooling for reuse (4KB, 64KB, 1MB, 4MB)
- 20-30% reduction in memory usage
- 40-60% reduction in allocations

**Performance Metrics:**

| Optimization | Improvement |
|--------------|-------------|
| Zero-Copy I/O | 5-10x faster data migration |
| Connection Pooling | 3-5x reduction in overhead |
| Buffer Pooling | 40-60% fewer allocations |
| Adaptive Buffering | 20-30% less memory usage |

Configuration:
```yaml
enable_zero_copy: true
enable_adaptive_buffering: true
```

## Performance Features

### Zero-Copy Buffer Pooling

FlyDB uses zero-copy buffer pooling to minimize memory allocations and GC pressure:

- **Buffer Pool**: Reusable buffers in size classes (256B to 16MB)
- **Zero-Copy Reader**: Reads messages directly into pooled buffers
- **Scatter-Gather I/O**: Efficient network writes without copying

Enable with: `enable_zero_copy: true` (default)

### Compression

FlyDB supports configurable compression for WAL and replication traffic:

| Algorithm | Config Value | Description |
|-----------|--------------|-------------|
| gzip | `gzip` | Good compression ratio, moderate speed |
| LZ4 | `lz4` | Very fast, lower ratio |
| Snappy | `snappy` | Fast, good for real-time |
| Zstd | `zstd` | Best ratio, configurable speed |

Configuration:
```json
enable_compression = true
compression_algorithm = "gzip"
compression_min_size = 256
```

## SQL Command Summary

FlyDB supports a comprehensive SQL dialect:

### Data Definition Language (DDL)

| Command | Description |
|---------|-------------|
| `CREATE TABLE` | Create a new table with columns and constraints |
| `DROP TABLE` | Remove a table |
| `ALTER TABLE` | Modify table structure |
| `CREATE INDEX` | Create a B-Tree index on a column |
| `DROP INDEX` | Remove an index |
| `CREATE DATABASE` | Create a new database with options |
| `DROP DATABASE` | Remove a database |
| `USE <database>` | Switch to a database |

### Data Manipulation Language (DML)

| Command | Description |
|---------|-------------|
| `SELECT` | Query data with WHERE, ORDER BY, LIMIT, JOIN |
| `INSERT` | Insert rows (single or bulk) |
| `UPDATE` | Modify existing rows |
| `DELETE` | Remove rows |
| `INSERT ... ON CONFLICT` | Upsert support |

### Transaction Control

| Command | Description |
|---------|-------------|
| `BEGIN` | Start a transaction |
| `COMMIT` | Commit the transaction |
| `ROLLBACK` | Rollback the transaction |
| `SAVEPOINT` | Create a savepoint |
| `RELEASE SAVEPOINT` | Release a savepoint |

### Access Control

| Command | Description |
|---------|-------------|
| `CREATE USER` | Create a new user |
| `DROP USER` | Remove a user |
| `GRANT` | Grant permissions |
| `REVOKE` | Revoke permissions |

### Inspection

| Command | Description |
|---------|-------------|
| `INSPECT TABLES` | List all tables |
| `INSPECT TABLE <name>` | Show table details |
| `INSPECT USERS` | List all users |
| `INSPECT INDEXES` | List all indexes |
| `INSPECT DATABASES` | List all databases |
| `INSPECT DATABASE <name>` | Show database details |
| `INSPECT SERVER` | Show server information |
| `INSPECT STATUS` | Show server status |

## Shell Commands (fsql)

The `fsql` shell provides PostgreSQL-like backslash commands:

| Command | Description |
|---------|-------------|
| `\q`, `\quit` | Exit the shell |
| `\h`, `\help` | Show help |
| `\clear` | Clear the screen |
| `\s`, `\status` | Show connection status |
| `\v`, `\version` | Show version |
| `\timing` | Toggle query timing |
| `\x` | Toggle expanded output |
| `\o [file]` | Set output to file |
| `\! <cmd>` | Execute shell command |
| `\dt` | List tables |
| `\du` | List users |
| `\di` | List indexes |
| `\db`, `\l` | List databases |
| `\c`, `\connect <db>` | Switch database |
| `\sql` | Enter SQL mode |
| `\normal` | Return to normal mode |

## See Also

- [Implementation Details](implementation.md) - Technical deep-dives
- [Design Decisions](design-decisions.md) - Rationale and trade-offs
- [API Reference](api.md) - SQL syntax and commands
- [Driver Development Guide](driver-development.md) - Building database drivers

