# RewindDB

**Save where you are. Break things. Go back. It's that simple.**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/yourusername/rewinddb/ci.yml?branch=main)](https://github.com/yourusername/rewinddb/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/rewinddb)](https://goreportcard.com/report/github.com/yourusername/rewinddb)

I built this because I kept breaking working code mid-experiment and had no fast way back. Git commits felt too heavy — I didn't want to write a message, stage files, and create history just to checkpoint "auth kinda works now." And I was spending a lot of time with AI-generated code that would work, then stop working, and I had no idea what changed. So I built this.

---

## What is this?

RewindDB saves snapshots of your entire project directory. You call it a checkpoint. You can restore any checkpoint instantly. If you check out an old checkpoint and make changes, a new branch spins up automatically. No staging, no commit messages required.

Think of it like git, but for your entire project state at any moment — including build artifacts, config files, generated code, and anything else git would never track. It's not trying to replace git. It does something different.

---

## Quick demo

```bash
# Start fresh in your project directory
$ cd my-project
$ rw init
✓ Initialized .rewind/ in /home/user/my-project
```

```bash
# You've got auth working. Save it before touching anything else.
$ rw save "auth working, don't break it"
✓ a3f2b1c8  auth working, don't break it
```

```bash
# You mess with the auth middleware. Things break.
$ vim internal/auth/middleware.go
# ... 40 minutes of making it worse ...

$ rw save "refactored auth — something's wrong"
✓ 9d1e4f72  refactored auth — something's wrong
```

```bash
# See exactly what changed between those two points
$ rw diff a3f2b1c8 9d1e4f72
Modified (3)
  internal/auth/middleware.go  +182 B
  internal/auth/token.go       -44 B
  config/app.yaml              +12 B
```

```bash
# Nope. Go back to when it worked.
$ rw goto a3f2b1c8
✓ Restored 3 files, skipped 47 unchanged
  HEAD → a3f2b1c8  auth working, don't break it
```

```bash
# Check what's in your history
$ rw list
  9d1e4f72  refactored auth — something's wrong   (branch: experiment-1)
* a3f2b1c8  auth working, don't break it           (branch: main)
  b7c3a291  initial setup                          (branch: main)
```

---

## Installation

**macOS (Homebrew):**
```bash
brew install yourusername/tap/rewinddb
```

**Linux (one-liner):**
```bash
curl -sSfL https://raw.githubusercontent.com/yourusername/rewinddb/main/install.sh | sh
```

**Go install:**
```bash
go install github.com/yourusername/rewinddb/cmd/rw@latest
```

**Build from source:**
```bash
git clone https://github.com/yourusername/rewinddb
cd rewinddb
make install
```

**Windows:** Download the `.zip` from the [releases page](https://github.com/yourusername/rewinddb/releases), extract, and add `rw.exe` to your PATH.

---

## Core concepts

**Checkpoint** is just a snapshot of every file in your project at a point in time. You make one whenever you reach a state worth remembering — before a refactor, after a feature clicks, before running a script that might nuke your config. It's not a git commit. There's no staging area. You just run `rw save "message"` and it's done.

**Branch** happens automatically. If you restore an old checkpoint and then save, RewindDB sees you've diverged from the main line and creates a new branch to track it. You don't have to name it or think about it — it just happens. This means you can explore two different approaches from the same starting point without manually managing branches.

**Timeline** is the graph of all your checkpoints. Each one points to its parent. It's a tree, not a straight line, because branching creates forks. You can think of it as a tree of "project states" that you can jump between. Run `rw list --all` to see the whole thing.

**Object store** is where the actual file data lives. It only stores what changed — if the same file appears in 10 checkpoints unchanged, it's stored once. It's under `.rewind/objects/` and you never need to touch it directly.

---

## Command reference

| Command | What it does | Example |
|---|---|---|
| `rw init` | Set up `.rewind/` in the current directory | `rw init` |
| `rw save <message>` | Snapshot everything and create a checkpoint | `rw save "jwt auth done"` |
| `rw goto <ref>` | Restore your project to any checkpoint | `rw goto a3f2b1c8` |
| `rw list` | Show checkpoint history | `rw list --all` |
| `rw diff <id1> [id2]` | Show what changed between two checkpoints | `rw diff HEAD~3 HEAD` |
| `rw status` | Show what's changed since the last checkpoint | `rw status` |
| `rw status --verify` | Re-validate all object checksums | `rw status --verify` |
| `rw branches` | List all branches | `rw branches` |
| `rw branches branch <name>` | Create a branch at HEAD | `rw branches branch experiment` |
| `rw branches switch <name>` | Switch to a branch and restore its files | `rw branches switch main` |
| `rw tag <name> [id]` | Attach a label to a checkpoint | `rw tag v1.0` |
| `rw gc` | Delete objects that no checkpoint references | `rw gc --dry-run` |
| `rw version` | Print version info | `rw version` |
| `rw completion <shell>` | Generate shell completions | `rw completion zsh` |

### Real-world examples

**Before running a risky refactor:**
```bash
rw save "stable before splitting the monolith"
# go ahead and break things
# ...
rw goto <that id> # if it goes wrong
```

**When AI-generated code breaks something:**
```bash
# You accepted a big AI suggestion. It sort of works.
rw save "ai refactor applied — needs testing"

# Three prompts later it's a mess.
rw goto <previous id>
# Back to where you started, in under a second.
```

**Experimenting with two different approaches:**
```bash
rw save "before trying approach A"        # base point

# implement approach A
rw save "approach A — uses goroutines"    # saved on main

rw goto <base id>                          # go back to the fork point
# RewindDB auto-creates a new branch here

# implement approach B  
rw save "approach B — uses channels"      # saved on branch-1

rw list --all                              # compare both timelines
```

**Debugging — rewinding to when it worked:**
```bash
rw list
# find the last checkpoint before things broke

rw goto b7c3a291
# confirm it works

rw diff b7c3a291 HEAD
# see exactly what changed between then and now
```

**Cleaning up old checkpoints:**
```bash
rw gc --dry-run     # preview what would get deleted
# 12 objects (4.2 MB) would be freed

rw gc               # actually run it
# ✓ Freed 12 objects (4.2 MB)
```

---

## Go SDK

If you want to use RewindDB in your Go code:

```bash
go get github.com/yourusername/rewinddb
```

**Save a checkpoint:**
```go
import "github.com/yourusername/rewinddb/internal/sdk"

client, err := sdk.New("/path/to/project")
if err != nil {
    log.Fatal(err)
}

cp, err := client.Save("before migration")
if err != nil {
    log.Fatal(err)
}
fmt.Println(cp.ID) // a3f2b1c8...
```

**Restore a checkpoint:**
```go
result, err := client.Goto("a3f2b1c8")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("wrote %d files, skipped %d\n", result.Written, result.Skipped)
```

**Check what changed:**
```go
status, err := client.Status()
if err != nil {
    log.Fatal(err)
}
for _, f := range status.ModifiedFiles {
    fmt.Println("modified:", f)
}
```

---

## How it works

- **Content-addressable storage:** every file gets SHA-256 hashed and stored under that hash. Same content = same hash = stored once, regardless of how many checkpoints reference it.
- **DAG timeline:** checkpoints form a directed acyclic graph where each node points to its parent. Branching just means one parent has two children.
- **Delta restore:** `goto` scans your current files first, then only writes the ones that actually differ from the target snapshot. That's why it's fast even on large projects.
- **Atomic writes:** nothing goes directly to `index.json`. Everything goes to a temp file, gets `fsync`'d, then gets renamed into place. A crash can't leave a corrupt index.

See [ARCHITECTURE.md](ARCHITECTURE.md) for the full technical breakdown.

---

## Performance

Measured on a MacBook M2, SSD, default worker count. Your results will vary.

| Operation | ~Time | Notes |
|---|---|---|
| `rw save` (1,000 files) | ~300 ms | Parallel hashing, all CPUs |
| `rw goto` (1,000 files, 50 changed) | ~80 ms | Delta restore — only changed files written |
| `rw status` (cold) | ~120 ms | Full hash of all files |
| `rw status` (warm) | ~15 ms | mtime cache skips unchanged files |
| `rw gc` (1,000 objects) | ~40 ms | Single pass through object store |

---

## Compared to git

| Feature | git | RewindDB |
|---|---|---|
| Tracks binary / build artifacts | No (or badly) | Yes |
| Requires a commit message | Yes | Optional |
| Auto-branches on diverge | No | Yes |
| Snapshots runtime state | No | Yes |
| Collaboration / remotes | Yes | No (local only) |
| Speed for full-project restore | Slow for large binaries | Fast (delta, content-addressed) |
| Understands code semantics | Somewhat (3-way merge) | No |

Use git for code history, collaboration, and deployment. Use RewindDB for experiment checkpointing, safe rollbacks, and anything git won't track. Use both. They solve different problems.

---

## Roadmap

**Done:**
- [x] Content-addressable object store with SHA-256 deduplication
- [x] Full checkpoint + DAG timeline engine
- [x] Automatic branching on diverge
- [x] Delta-only restore
- [x] Fast status with mtime cache
- [x] Gzip-compressed snapshot objects
- [x] File locking (no concurrent writes)
- [x] Crash recovery (fsync + atomic rename)
- [x] Checksum validation + corruption detection
- [x] GC with dry-run
- [x] `.rewindignore` support
- [x] Shell completions (bash, zsh, fish)
- [x] Go SDK
- [x] Cross-platform builds (Linux, macOS, Windows)

**In progress:**
- [ ] `rw diff` with inline line-level diffs (not just file-level)
- [ ] `rw log --graph` timeline visualisation in the terminal
- [ ] `rw stash` — save without creating a named checkpoint

**Planned:**
- [ ] Remote storage backend (S3, local server)
- [ ] Watch mode: `rw watch` auto-saves on file change
- [ ] Web UI for browsing timeline

I won't promise dates. If something is in "planned" it means I want it, not that it's coming soon.

---

## Contributing

PRs are welcome. If you're planning something big, open an issue first so we're not duplicating effort. Run `go test ./... -race` before submitting — I won't merge anything that fails the race detector. I'll review within a few days. See [CONTRIBUTING.md](CONTRIBUTING.md) for the full setup guide.

---

## License

MIT — see [LICENSE](LICENSE).
