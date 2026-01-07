# FlyDB Design Decisions

This document explains the key design decisions made in FlyDB, including the rationale, trade-offs, and alternatives considered.

## Table of Contents

1. [In-Memory Storage with WAL](#in-memory-storage-with-wal)
2. [Key-Value Foundation](#key-value-foundation)
3. [Simple Text Protocol](#simple-text-protocol)
4. [Bully Algorithm for Leader Election](#bully-algorithm-for-leader-election)
5. [Eventual Consistency for Replication](#eventual-consistency-for-replication)
6. [B-Tree for Indexes](#b-tree-for-indexes)
7. [bcrypt for Password Hashing](#bcrypt-for-password-hashing)
8. [AES-256-GCM for Encryption](#aes-256-gcm-for-encryption)
9. [Go as Implementation Language](#go-as-implementation-language)

---

## In-Memory Storage with WAL

### Decision

Store all data in memory using a Go `map[string][]byte`, with a Write-Ahead Log (WAL) for durability.

### Rationale

1. **Simplicity**: In-memory storage is straightforward to implement and understand
2. **Performance**: Memory access is orders of magnitude faster than disk I/O
3. **Educational Value**: Clearly demonstrates the WAL concept without complex buffer management
4. **Durability**: WAL ensures committed data survives crashes

### Trade-offs

| Advantage | Disadvantage |
|-----------|--------------|
| Fast reads (O(1) average) | Limited by available RAM |
| Simple implementation | Full dataset must fit in memory |
| Easy debugging | Startup time grows with data size |
| No buffer pool complexity | No partial loading of data |

### Alternatives Considered

1. **Disk-based storage (like SQLite)**: More complex, requires buffer pool management
2. **Memory-mapped files**: Platform-specific behavior, complex crash recovery
3. **LSM-Tree (like RocksDB)**: Better for write-heavy workloads, but more complex

### When to Reconsider

- Dataset exceeds available RAM
- Need for partial data loading
- Cold start time becomes unacceptable

---

## Key-Value Foundation

### Decision

Build the SQL layer on top of a simple key-value store abstraction.

### Rationale

1. **Separation of Concerns**: Storage logic is isolated from SQL logic
2. **Flexibility**: Easy to swap storage backends
3. **Simplicity**: Key-value operations are easy to reason about
4. **Educational**: Demonstrates how SQL databases can be built on simpler primitives

### Key Encoding Scheme

```
row:<table>:<id>     → Row data (JSON)
schema:<table>       → Table schema
seq:<table>          → Auto-increment counter
_sys_users:<name>    → User credentials
_sys_privs:<u>:<t>   → Permissions
```

### Trade-offs

| Advantage | Disadvantage |
|-----------|--------------|
| Clean abstraction | Prefix scans for table queries |
| Easy to test | No native range queries |
| Swappable backends | JSON serialization overhead |
| Simple transactions | No columnar storage benefits |

### Alternatives Considered

1. **Page-based storage**: More efficient for large tables, but complex
2. **Columnar storage**: Better for analytics, but complex for OLTP
3. **Document store**: Similar trade-offs, less educational value

---

## Simple Text Protocol

### Decision

Use a line-based text protocol for the primary interface (port 8888).

### Rationale

1. **Debuggability**: Can interact with database using telnet/netcat
2. **Simplicity**: Easy to implement clients in any language
3. **Educational**: Protocol is human-readable
4. **Testing**: Easy to write integration tests

### Protocol Format

```
Request:  COMMAND [args]\n
Response: RESULT\n

Examples:
  PING           → PONG
  AUTH user pass → AUTH OK
  SQL SELECT ... → id|name|age\n1|Alice|30
```

### Trade-offs

| Advantage | Disadvantage |
|-----------|--------------|
| Human-readable | Parsing overhead |
| Easy debugging | No binary data support |
| Simple clients | Escaping issues with special chars |
| Telnet-friendly | Less efficient than binary |

### Mitigation

A binary protocol is also available on port 8889 for clients that need efficiency.

---

## Bully Algorithm for Leader Election

### Decision

Use the Bully algorithm for automatic leader election in cluster mode.

### Rationale

1. **Simplicity**: Easy to understand and implement
2. **Deterministic**: Highest-ID node always wins
3. **Fast Convergence**: Election completes quickly
4. **Educational**: Classic distributed systems algorithm

### How It Works

```
1. Node detects leader failure (missed heartbeats)
2. Node sends ELECTION to all higher-ID nodes
3. If no response, node becomes leader (COORDINATOR)
4. If response received, wait for COORDINATOR from higher node
```

### Trade-offs

| Advantage | Disadvantage |
|-----------|--------------|
| Simple to implement | Highest-ID bias |
| Fast election | Network partition issues |
| Deterministic outcome | Not Byzantine fault tolerant |
| Well-understood | May elect unavailable node |

### Alternatives Considered

1. **Raft**: More robust, but significantly more complex
2. **Paxos**: Proven correct, but notoriously difficult
3. **ZAB (ZooKeeper)**: Production-ready, but external dependency

### When to Reconsider

- Need for stronger consistency guarantees
- Complex network topologies
- Byzantine fault tolerance requirements

---

## Eventual Consistency for Replication

### Decision

Use asynchronous WAL streaming with 100ms polling interval.

### Rationale

1. **Simplicity**: No complex consensus protocol
2. **Performance**: Leader doesn't wait for follower acknowledgment
3. **Availability**: Leader can continue if followers are slow
4. **Educational**: Demonstrates replication concepts clearly

### Replication Flow

```
Leader                          Follower
   │                               │
   │ ◄─── REPLICATE <offset> ──────│ (every 100ms)
   │                               │
   │ ──── WAL entries ────────────►│
   │                               │
   │                               │ (apply to local store)
```

### Trade-offs

| Advantage | Disadvantage |
|-----------|--------------|
| Simple implementation | Data loss on leader failure |
| High write throughput | Stale reads on followers |
| Leader availability | No read-your-writes guarantee |
| Low latency writes | Replication lag visible |

### Consistency Window

- **Maximum lag**: 100ms + network latency + apply time
- **Typical lag**: < 200ms under normal conditions

### Alternatives Considered

1. **Synchronous replication**: Stronger consistency, but higher latency
2. **Semi-synchronous**: Wait for one follower, balance of both
3. **Raft-based replication**: Strong consistency, complex implementation

---

## B-Tree for Indexes

### Decision

Use B-Tree data structure for secondary indexes.

### Rationale

1. **Balanced**: Guaranteed O(log N) operations
2. **Range Queries**: Efficient for ORDER BY and BETWEEN
3. **Well-Understood**: Classic database index structure
4. **Educational**: Demonstrates fundamental CS concepts

### Implementation Details

- **Minimum degree (t)**: Configurable, default 2
- **Node capacity**: 2t-1 keys per node
- **Leaf nodes**: Store actual row key references

### Trade-offs

| Advantage | Disadvantage |
|-----------|--------------|
| Balanced height | More complex than hash index |
| Range query support | Higher memory overhead |
| Ordered traversal | Split/merge complexity |
| Proven algorithm | Not cache-optimized |

### Alternatives Considered

1. **Hash Index**: O(1) lookups, but no range queries
2. **B+ Tree**: Better for disk, leaves linked for scans
3. **Skip List**: Simpler, probabilistic balance
4. **LSM Tree**: Better for writes, complex compaction

---

## bcrypt for Password Hashing

### Decision

Use bcrypt with default cost factor for password hashing.

### Rationale

1. **Security**: Designed specifically for password hashing
2. **Adaptive**: Cost factor can be increased over time
3. **Salt Built-in**: Automatic salt generation and storage
4. **Industry Standard**: Widely recommended and audited

### Implementation

```go
// Hashing
hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

// Verification
err := bcrypt.CompareHashAndPassword(hash, []byte(password))
```

### Trade-offs

| Advantage | Disadvantage |
|-----------|--------------|
| Purpose-built for passwords | Slower than SHA-256 |
| Automatic salting | CPU-intensive (by design) |
| Adjustable work factor | Fixed output format |
| Resistant to GPU attacks | Memory usage |

### Alternatives Considered

1. **Argon2**: Newer, memory-hard, but less library support
2. **scrypt**: Memory-hard, good alternative
3. **PBKDF2**: Widely supported, but less resistant to GPU attacks

---

## AES-256-GCM for Encryption

### Decision

Use AES-256-GCM for encrypting data at rest (WAL and stored values).

### Rationale

1. **Security**: 256-bit key provides strong encryption
2. **Authenticated**: GCM mode provides integrity checking
3. **Performance**: Hardware acceleration on modern CPUs
4. **Standard**: NIST-approved, widely audited

### Implementation

```go
type Encryptor struct {
    gcm cipher.AEAD
}

func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
    nonce := make([]byte, e.gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)
    return e.gcm.Seal(nonce, nonce, plaintext, nil), nil
}
```

### Trade-offs

| Advantage | Disadvantage |
|-----------|--------------|
| Strong encryption | Key management complexity |
| Integrity verification | Performance overhead |
| Hardware acceleration | Nonce management required |
| Industry standard | Increased storage size |

---

## Go as Implementation Language

### Decision

Implement FlyDB in Go.

### Rationale

1. **Concurrency**: Goroutines and channels for server workloads
2. **Simplicity**: Clean syntax, easy to read and understand
3. **Performance**: Compiled language with good performance
4. **Standard Library**: Excellent networking and crypto support
5. **Educational**: Popular language, accessible to learners

### Trade-offs

| Advantage | Disadvantage |
|-----------|--------------|
| Easy concurrency | Garbage collection pauses |
| Fast compilation | No generics (until Go 1.18) |
| Single binary deployment | Less control than C/Rust |
| Strong standard library | Verbose error handling |

### Alternatives Considered

1. **Rust**: Better performance, but steeper learning curve
2. **C++**: Maximum control, but complex memory management
3. **Java**: Good ecosystem, but JVM overhead
4. **Python**: Easy to write, but too slow for database

---

## See Also

- [Architecture Overview](architecture.md) - High-level system design
- [Implementation Details](implementation.md) - Technical deep-dives
- [API Reference](api.md) - SQL syntax and commands

