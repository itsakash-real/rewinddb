<div align="center">

# RewindDB

### Save your project state. Instantly rewind. Explore alternate timelines.

[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/itsakash-real/rewinddb?style=flat-square)](https://goreportcard.com/report/github.com/itsakash-real/rewinddb)

**Git is built for collaboration. RewindDB is built for control.**

<p>
  <a href="#installation">Installation</a> ·
  <a href="#quick-demo">Quick Demo</a> ·
  <a href="#commands">Commands</a> ·
  <a href="#how-it-works">How It Works</a> ·
  <a href="#go-sdk">Go SDK</a> ·
  <a href="#roadmap">Roadmap</a>
</p>

</div>

---

## Why This Exists

I kept breaking working code while experimenting. Git commits felt too heavy mid-experiment — I didn't want a commit, I wanted a quicksave. It got worse with AI-generated code: something worked, I tweaked it, it broke, and I couldn't get back. So I built this.

---

## What Is This?

RewindDB checkpoints your entire project directory. One command saves everything. Another brings it all back. Think of it like git, but for your whole project state at any moment — including files git would never track, like build artifacts, compiled binaries, and runtime configs.

It's not trying to replace git. Git is for collaboration and code history. RewindDB is for the messy middle: experiments, refactors, AI-assisted coding, and any time you need a safety net that's faster than a commit.

---

## Architecture

![System Architecture](./docs/diagrams/system-architecture.svg)

---

## How Timelines Work

![Timeline DAG](./docs/diagrams/timeline-dag.svg)

---

## What Happens When You Save

![Save Flow](./docs/diagrams/save-flow.svg)

---

## How Storage Works

![Object Store](./docs/diagrams/object-store.svg)

---

## What Happens When You Restore

![Goto Restore](./docs/diagrams/goto-restore.svg)

---

## Common Scenarios

![Use Cases](./docs/diagrams/use-cases.svg)

---

## Quick Demo

```
# Start fresh in your project
$ rw init

  ◆  initialized  ──────────────────────────────

     directory      /your/project/.rewind
     branch         main
```

```
# Save a checkpoint — message is optional, RewindDB writes one for you
$ rw save "auth working"

  ◆  checkpoint saved  ─────────────────────────

     id             a3f2b1c8
     message        "auth working"
     branch         main
     files          12 tracked  ·  0 changed
     saved          just now
```

```
# Make changes, save again — or just run rw save with no message
$ rw save

  ◆  checkpoint saved  ─────────────────────────

     id             b2e1a0f4
     message        "auto: auth.go (1 file(s) changed)"
     branch         main
     files          12 tracked  ·  1 changed
     saved          just now
```

```
# Browse your history
$ rw list

  ◆  main  ·  2 checkpoints  ──────────────────

  ◆  b2e1a0f4  HEAD   just now            "auto: auth.go (1 file(s) changed)"
  ○  a3f2b1c8            3 minutes ago    "auth working"
```

```
# Something broke — go back
$ rw goto a3f2b1c8
Restore to: "auth working"? [y/N]: y

  ◆  restored  ─────────────────────────────────

     checkpoint     a3f2b1c8
     message        "auth working"
     branch         main
     written        1 file(s)
     removed        0 file(s)
```

```
# Or just undo the last save — no IDs needed
$ rw undo

  ◆  undo  ·  1 step(s) back  ─────────────────

     restoring to   a3f2b1c8
     message        "auth working"

  ✓  restored to a3f2b1c8
```

```
# See what's changed since your last checkpoint
$ rw status

  ╭────────────────────────────────────────────────────╮
  │   ◆  rewinddb                                      │
  │                                                    │
  │   main  ·  a3f2b1c8  ·  5 minutes ago              │
  ╰────────────────────────────────────────────────────╯

     checkpoints    2 on branch  ·  2 total
     storage        6 objects  ·  4.2 KB

  ◆  working directory  ────────────────────────

  ~  src/auth.go
  +  src/newfile.go

  →  run rw save "message" to checkpoint
```

```
# Before running a risky command — checkpoint before, rollback on failure
$ rw run "npm run build"
✓ Saved pre-run checkpoint: c1d0e9f
  running: npm run build
✗ Command failed (exit 1). Rolling back...
✓ Rolled back to c1d0e9f
```

---

## Installation

<details>
<summary><strong>macOS (Homebrew)</strong></summary>

```bash
brew install itsakash-real/tap/rewinddb
```

</details>

<details>
<summary><strong>Linux (one-liner)</strong></summary>

```bash
curl -sSL https://raw.githubusercontent.com/itsakash-real/rewinddb/main/install.sh | bash
```

</details>

<details>
<summary><strong>Go install</strong></summary>

```bash
go install github.com/itsakash-real/rewinddb/cmd/rw@latest
```

</details>

<details>
<summary><strong>Build from source</strong></summary>

```bash
git clone https://github.com/itsakash-real/rewinddb
cd rewinddb
make build
```

</details>

<details>
<summary><strong>Windows</strong></summary>

Download the `.exe` from [Releases](https://github.com/itsakash-real/rewinddb/releases) or use `go install`. Colors work in Windows Terminal — not in the legacy CMD prompt.

</details>

---

## Core Concepts

### Checkpoint
A snapshot of your entire project at a point in time — not a diff, a full capture. Each checkpoint gets a short ID like `a3f2b1c8`. Create one whenever you reach something worth keeping: tests pass, a feature works, before you try something dangerous. If you don't provide a message, RewindDB generates one from what changed.

### Branch
Branches work automatically. If you restore an old checkpoint and save from there, RewindDB creates a new branch instead of overwriting your history. You end up with a forked timeline — both paths preserved, no manual branching required.

### Timeline
The timeline is a DAG (directed acyclic graph) — the same structure git uses internally. Each checkpoint points to its parent. Branches are named pointers to specific checkpoints. `rw list --all` shows the full picture.

### Object Store
Only changed files get stored. If 100 files haven't changed since the last checkpoint, they take zero extra space. Everything is deduplicated by content hash. Snapshots are gzip-compressed.

---

## Commands

### Core

| Command | What it does | Example |
|---|---|---|
| `rw init` | Set up a repo in the current directory | `rw init` |
| `rw save [msg]` | Save a checkpoint (message optional — auto-generated if omitted) | `rw save "login works"` |
| `rw goto <id>` | Restore to a checkpoint (warns if you have unsaved changes) | `rw goto a3f2b1c` |
| `rw undo [--n N]` | Go back N checkpoints without needing to know the ID | `rw undo --n 3` |
| `rw list` | List checkpoints on the current branch | `rw list` |
| `rw list --all` | Show all branches and their checkpoints | `rw list --all` |
| `rw diff <id1> [id2]` | Compare two checkpoints (supports `HEAD~N` syntax) | `rw diff HEAD HEAD~2` |
| `rw status` | Show branch, HEAD, and what's changed on disk | `rw status` |

### Branching

| Command | What it does | Example |
|---|---|---|
| `rw branches` | List all branches | `rw branches` |
| `rw branches branch <name>` | Create a new branch at HEAD | `rw branches branch experiment` |
| `rw branches switch <name>` | Switch to a branch and restore its state | `rw branches switch experiment` |
| `rw tag <name> [id]` | Label a checkpoint with a human-readable name | `rw tag v1.0` |

### Power Features

| Command | What it does | Example |
|---|---|---|
| `rw run "cmd"` | Auto-checkpoint before running, rollback if it fails | `rw run "npm run build"` |
| `rw watch [--interval]` | Auto-save checkpoints in the background on file change | `rw watch --interval 5m` |
| `rw bisect start` | Binary-search your timeline to find when a bug appeared | `rw bisect start` |
| `rw bisect good [id]` | Mark a checkpoint as bug-free | `rw bisect good` |
| `rw bisect bad [id]` | Mark a checkpoint as broken | `rw bisect bad` |
| `rw bisect reset` | End bisect session and restore original HEAD | `rw bisect reset` |
| `rw search <text>` | Search checkpoint messages and tags | `rw search "JWT"` |
| `rw session start "name"` | Begin a named work session | `rw session start "auth feature"` |
| `rw session end` | End the current session | `rw session end` |
| `rw session list` | List all sessions | `rw session list` |
| `rw session restore "name"` | Restore to a session's starting checkpoint | `rw session restore "auth feature"` |

### Storage & Export

| Command | What it does | Example |
|---|---|---|
| `rw gc` | Remove objects no checkpoint references | `rw gc` |
| `rw gc --dry-run` | Preview what GC would delete | `rw gc --dry-run` |
| `rw stats` | Show timeline and storage statistics | `rw stats` |
| `rw export <id>` | Export a checkpoint to a portable `.rwdb` file | `rw export a3f2b1c` |
| `rw import <file>` | Import a `.rwdb` file into the current repo | `rw import state.rwdb` |
| `rw ignore add "pattern"` | Add a pattern to `.rewindignore` | `rw ignore add "dist/"` |
| `rw ignore auto` | Auto-add ignore patterns based on project type | `rw ignore auto` |
| `rw ignore list` | Show current `.rewindignore` contents | `rw ignore list` |
| `rw version` | Print version, build time, Go version | `rw version` |

---

## Real-World Examples

**Before a risky refactor**
```bash
rw save "before auth refactor"
# do the refactor...
# if it breaks:
rw undo
```

**When AI-generated code breaks something**
```bash
rw save "working state before AI edits"
# paste in the AI code, run tests...
# if tests explode:
rw undo
```

**Experimenting with two different approaches**
```bash
rw save "base: working login"

# try approach A
rw save "approach A: session tokens"

# go back to base and try approach B
rw goto <base-checkpoint-id>
# RewindDB auto-creates a new branch here
rw save "approach B: JWT"

rw list --all   # see both timelines side by side
```

**Debugging — find exactly when a bug appeared**
```bash
rw bisect start
rw bisect good <last-known-good-id>
rw bisect bad HEAD
# RewindDB jumps you to the midpoint checkpoint
# test your code, then:
rw bisect good   # or: rw bisect bad
# repeat until it pinpoints the exact checkpoint that broke things
```

**Grouping work into a named session**
```bash
rw session start "feature: payment flow"
# work for 2 hours, save checkpoints freely...
rw session end
# later, jump back to the start of that session:
rw session restore "feature: payment flow"
```

**Sharing a broken state with a teammate**
```bash
# export the exact state where the bug is reproducible
rw export a3f2b1c --output bug-repro.rwdb
# teammate imports it:
rw import bug-repro.rwdb
```

**Cleaning up old checkpoints**
```bash
rw gc --dry-run   # see what would be removed
rw gc             # actually do it
```

---

## Go SDK

```bash
go get github.com/itsakash-real/rewinddb
```

**Save a checkpoint from code**
```go
package main

import "github.com/itsakash-real/rewinddb/internal/sdk"

func main() {
    client, err := sdk.New("/path/to/project")
    if err != nil {
        panic(err)
    }

    // checkpoint before anything risky
    _, err = client.Save("before payment processing")
    if err != nil {
        panic(err)
    }
}
```

**Restore to a checkpoint**
```go
// restore by ID, tag name, or relative ref like "HEAD~2"
err := client.Goto("auth-working")
if err != nil {
    panic(err)
}
```

**Check what's changed**
```go
status, err := client.Status()
if err != nil {
    panic(err)
}
fmt.Printf("modified: %d  added: %d  removed: %d\n",
    len(status.ModifiedFiles),
    len(status.AddedFiles),
    len(status.RemovedFiles),
)
```

**CI/CD — checkpoint before deploying**
```yaml
# .github/workflows/deploy.yml
- name: Checkpoint before deploy
  run: rw save "pre-deploy: ${{ github.sha }}"
```

---

## How It Works

| Mechanism | Detail |
|---|---|
| **Content-addressable storage** | Every file stored by SHA-256 hash. Same content = same hash = zero duplication across checkpoints. |
| **DAG timeline** | Checkpoints form a directed graph. Each one points to its parent. Branching happens automatically when you save from a non-HEAD checkpoint. |
| **Delta restore** | Restoring only writes files that actually changed. If 90% of files are identical, 90% of the disk work is skipped. |
| **Atomic writes** | Everything goes to a temp file first, fsynced, then renamed into place. A crash mid-save leaves nothing corrupted. |
| **mtime cache** | `rw status` skips re-hashing files whose mtime and size haven't changed. Fast even on large projects. |
| **Crash recovery** | On startup, RewindDB checks for leftover temp files and incomplete renames from a previous crash and cleans them up. |

---

## Performance

| Operation | ~Time | Notes |
|---|---|---|
| Save (1,000 files) | ~180ms | Parallel SHA-256 hashing across all CPU cores |
| Restore (1,000 files, 10% changed) | ~40ms | Delta restore — unchanged files are skipped |
| Status check | ~60ms | mtime cache skips re-hashing unchanged files |
| GC (50 checkpoints) | ~90ms | Single pass over the object store |

> Measured on an Apple M2. Results vary with file sizes and disk speed.

---

## RewindDB vs Git

| Feature | Git | RewindDB |
|---|---|---|
| Tracks binary files | Poor (no delta compression) | ✅ Same as text files |
| Message required to save | Yes | ✅ No — auto-generates one |
| Tracks runtime artifacts | No | ✅ Yes |
| Works outside an editor | Yes | ✅ Yes — any terminal, CI/CD, scripts |
| Auto-branches on time-travel | No | ✅ Yes |
| Rollback on command failure | No | ✅ `rw run "cmd"` |
| Background auto-save daemon | No | ✅ `rw watch` |
| Binary search through history | `git bisect` | ✅ `rw bisect` |
| Collaboration / push / pull | ✅ Yes | No (local only) |
| Line-level history | ✅ Yes | File-level |

Use git for collaboration, code review, and sharing history with your team. Use RewindDB for local experiments, mid-session safety nets, and anything git wouldn't track. **Use both — they solve different problems.**

---

## Roadmap

### Done

- Content-addressable object store with deduplication
- DAG timeline with automatic branching
- Delta restore (only writes changed files)
- mtime-based fast status check
- Gzip compression for snapshots
- File locking with stale-lock detection
- Crash recovery (atomic writes + fsync)
- `.rewindignore` with `**` glob support
- Shell completions (bash, zsh, fish, PowerShell)
- Go SDK
- `rw undo` — go back N checkpoints without knowing IDs
- `rw run "cmd"` — auto-checkpoint before, rollback on failure
- `rw watch` — background auto-save daemon
- `rw bisect` — binary-search your timeline for a bad checkpoint
- `rw session` — group checkpoints into named work sessions
- `rw search` — search checkpoint messages and tags
- `rw stats` — storage and timeline dashboard
- `rw export` / `rw import` — share exact states as `.rwdb` files
- `rw ignore auto` — auto-detect ignore patterns by project type
- Auto-message generation when no message is given
- Auto-stash prompt before destructive restores
- Background GC every 10 saves
- Purple terminal UI with `◆` markers and box-drawing layout
- `--debug` flag for verbose logging (debug output hidden by default)
- Human-readable auto-branch names (`branch-2024-01-15-1423`)
- ANSI color detection (disabled on non-TTY, `NO_COLOR`, and Windows CMD)
- `rw diff HEAD~N HEAD` — relative refs work everywhere

### Planned

- [ ] Remote storage backend (S3, R2, local NAS)
- [ ] `rw interactive` — TUI checkpoint browser with arrow-key navigation
- [ ] Web UI for visualizing the checkpoint DAG
- [ ] `rw sync` — push/pull states between machines
- [ ] VSCode extension (status bar integration)
- [ ] Line-level diffs in the checkpoint viewer

---

## Contributing

PRs are welcome. Open an issue before starting anything large — I'd rather talk through the design first than have you spend a week on something that goes in a different direction.

Run `go test ./... -race` before submitting. I'll review within a few days.

See [CONTRIBUTING.md](CONTRIBUTING.md) for full details.

---

## License

MIT — see [LICENSE](LICENSE).
