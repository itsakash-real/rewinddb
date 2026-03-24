# 🔁 RewindDB Architecture

This document is intended for contributors and developers who want to understand how **RewindDB** works internally. It assumes familiarity with **Go**, **content-addressable storage**, and **DAG data structures**.

---

## 📦 System Overview

RewindDB is organized into **five layers**, with strict dependency rules — lower layers never import from higher ones.

```
┌─────────────────────────────────────────────────────────┐
│  LAYER 1 — USER INTERFACE                               │
│  cmd/rw/   (Cobra CLI)  ·  internal/sdk/  (Go SDK)      │
└────────────────────────┬────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────┐
│  LAYER 2 — COMMAND HANDLER                              │
│  internal/commands/   parse · validate · dispatch        │
└──┬──────────┬──────────┬──────────┬──────────┬──────────┘
   │          │          │          │          │
┌──▼──┐  ┌───▼──┐  ┌────▼──┐  ┌───▼──┐  ┌───▼──────────┐
│LAYER 3 — CORE ENGINES                                   │
│Timeline  Snapshot  Diff    Branch   (internal/engine/)  │
│Engine    Engine    Engine  Manager                      │
└──┬──┘  └───┬──┘  └────┬──┘  └───┬──┘  └───────────────┘
   │          │          │          │
┌──▼──────────▼──────────▼──────────▼──────────────────┐
│  LAYER 4 — STORAGE                                     │
│  internal/storage/   ObjectStore · Index · Cache        │
└──────────────────────────┬─────────────────────────────┘
                           │
┌──────────────────────────▼─────────────────────────────┐
│  LAYER 5 — DISK                                         │
│  .rewind/objects/   .rewind/index.json   (mtime cache)  │
└─────────────────────────────────────────────────────────┘
```

---

## 📁 Package Layout

```
internal/
├── commands/     # CLI commands (rw save, rw goto, ...)
├── engine/
│   ├── timeline/ # DAG + checkpoints + branching
│   ├── snapshot/ # scanning + hashing + restore
│   ├── diff/     # delta computation
│   └── branch/   # branch management
├── storage/
│   ├── objects/  # content-addressable storage
│   ├── index/    # atomic metadata
│   └── cache/    # mtime-based optimization
└── sdk/          # public Go API
```

---

## ⚙️ Core Components

### 🧠 TimelineEngine

Handles the DAG (history graph).

**Key Responsibilities:**

* Append checkpoints
* Resolve references
* Track ancestry
* Maintain branch consistency

```go
type Checkpoint struct {
    ID         string
    ParentID   string
    BranchID   string
    SnapshotID string
    Message    string
    CreatedAt  time.Time
    Tags       []string
}
```

---

### 📸 SnapshotEngine

Captures and restores filesystem state.

```go
type FileEntry struct {
    Path    string
    Hash    string
    Size    int64
    Mode    os.FileMode
    ModTime time.Time
}
```

**Highlights:**

* Parallel hashing (CPU optimized)
* mtime cache to skip unchanged files
* Efficient restore via delta comparison

---

### 🔍 DiffEngine

* Uses **Myers algorithm**
* Detects binary vs text files
* Outputs unified diffs

---

### 📦 ObjectStore

Content-addressable immutable storage (like Git).

**Key properties:**

* SHA-256 hashing
* Deduplication
* Atomic writes
* Checksum validation on read

---

### 🌿 BranchManager

Handles branch operations:

* Create
* Switch
* List
* Update tip

---

## 🧬 Data Structures

```
Index
├── HEAD
├── CurrentBranch
└── Branches

Checkpoint → Snapshot → FileEntry → ObjectStore
```

* `index.json` → mutable (state)
* `objects/` → immutable (data)

---

## 🧮 Algorithms

### 🔀 Branching Logic

* If HEAD == branch tip → linear commit
* Else → automatic fork (new branch)

---

### 🧹 Garbage Collection (GC)

* Traverse DAG from all branch tips
* Mark reachable objects
* Delete unreachable ones

**Complexity:** `O(C + F)`

---

### 🔄 Restore

Efficient delta-based restore:

* Add missing files
* Modify changed files
* Remove extra files

---

## 🔐 Safety Guarantees

### ✅ Atomic Writes

* Uses temp file + `fsync` + rename
* Prevents corruption on crashes

### 🔒 File Locking

* `.rewind/LOCK` prevents concurrent writes
* Detects and removes stale locks

### 🧾 Checksum Validation

* Detects silent corruption
* Ensures data integrity

---

## ⚡ Performance

| Operation   | Complexity | Cost Driver  |
| ----------- | ---------- | ------------ |
| `rw save`   | O(N / CPU) | hashing      |
| `rw goto`   | O(N + M)   | file writes  |
| `rw status` | O(N)       | stat + cache |
| `rw gc`     | O(C + F)   | traversal    |

---

## 🧪 Testing Strategy

### Unit Tests

* Timeline logic
* Snapshot + hashing
* Diff engine
* Storage + index
* Branch management

### Integration Tests

* Save → Modify → Restore
* Auto-branching
* GC cleanup
* Crash recovery
* Concurrent execution

### Benchmarks

```bash
go test ./... -bench=. -benchmem
```

---

## 📏 Design Invariants

These rules are **non-negotiable**:

1. **Objects are immutable**
2. **Index writes are atomic**
3. **Layer separation is strict**
4. **Branching is automatic**
5. **Restore is delta-based**

---

## 🚀 Summary

RewindDB is designed as a **Git-like time machine for your filesystem**, with:

* Immutable storage
* Efficient snapshotting
* Automatic branching
* Strong safety guarantees
* High-performance operations

---

💡 If you're contributing: respect invariants, test thoroughly, and benchmark performance-critical changes.

---
