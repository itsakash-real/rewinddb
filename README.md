<div align="center">

<img src="logo.png" alt="Nimbi" width="220" />

**Your code works. You change one thing. Now it doesn't. Get it back instantly.**

[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen?style=flat-square)](CONTRIBUTING.md)

[What It Does](#what-it-does) · [The Killer Feature](#the-killer-feature) · [What Git Misses](#what-git-misses) · [Install](#install) · [Commands](#commands)

</div>

---

## What It Does

You're experimenting with code. Something works. You change one thing. Everything breaks and you can't get back.

Nimbi is a single binary that saves your **entire project folder** as a checkpoint — so you can go back to any working state instantly.

```
rw save "everything works"       ← save the current state
rw save "trying new auth flow"   ← keep experimenting
rw undo                          ← broke it? go back instantly
```

**It's not a replacement for git.** Git is for sharing code with your team. Nimbi is for the messy part *before* that — experimenting, breaking things, and getting back safely.

---

## The Killer Feature

### `rw run` — auto-rollback on failure

This is the thing no other tool does. Wrap any risky command and Nimbi protects you automatically:

```bash
rw run "npm run build"
```

```
✓ checkpoint saved
  running: npm run build...
─────────────────────────────────────
  ... build output ...
─────────────────────────────────────
✗ command failed (exit 1)
  rolling back to pre-run checkpoint...
✓ rolled back to a3f2b1c8
```

**What just happened:**
1. Nimbi saved a checkpoint *before* running your command
2. Your command ran and failed
3. Nimbi automatically restored your project to the pre-run state

No git workflow does this. No rclone does this. **Your failed database migration never leaves your project half-broken.**

Works for anything:
```bash
rw run "python migrate.py"       # failed migration → auto-rollback
rw run "cargo build"             # broken build → auto-rollback
rw run "npm run deploy"          # bad deploy → auto-rollback
```

---

## What Git Misses

Git only tracks files you tell it to. Nimbi tracks your **full project state** — including the files git ignores that break everything when they change:

```
rw status

  tracking   847 files
  ignoring   43,821 files (node_modules, .git, dist)

  worth saving:
    .env.local          ← not in git (API keys)
    build/server        ← compiled binary (10-min build)
    config/local.json   ← local config (not in git)
```

When you `rw diff`, you see what git can't:

```
rw diff HEAD~1

  files git would show:
    ~ src/auth.js

  files ONLY nimbi tracks:
    ~ .env.local         ← API key changed
    ~ build/server       ← binary updated
    + config/local.json  ← new local config
```

**That `.env.local` change is why your app is broken. Git wouldn't have shown you that.**

---

## vs Git / vs Backup Tools

| | Git | rclone / rsync | **Nimbi** |
|---|---|---|---|
| Named checkpoints with messages | ✓ | ✗ | ✓ |
| Diff between any two states | ✓ | ✗ | ✓ |
| Tracks `.env`, binaries, build output | ✗ | ✓ | ✓ |
| Auto-rollback on failed commands | ✗ | ✗ | **✓** |
| Auto-branches when you time-travel | ✗ | ✗ | **✓** |
| Timeline with named sessions | ✗ | ✗ | **✓** |

**Git** is for sharing code with your team. **rclone/rsync** are backup tools — no timeline, no diff, no rollback. **Nimbi** gives you named checkpoints, diff between any two states, and auto-rollback on failed commands. Different problems entirely.

---

## Install

**macOS / Linux**
```bash
curl -sSL https://raw.githubusercontent.com/itsakash-real/nimbi/main/install.sh | bash
```

**Go (any platform)**
```bash
go install github.com/itsakash-real/nimbi/cmd/rw@latest
```

**Windows** — download `nimbi-windows-amd64.exe` from [Releases](https://github.com/itsakash-real/nimbi/releases).

**Build from source**
```bash
git clone https://github.com/itsakash-real/nimbi
cd nimbi
go build -o rw ./cmd/rw
```

Verify it works:
```bash
rw version
```

```
Nimbi v0.1.0
  go1.22 · darwin/arm64
  built 2026-03-30
```

---

## Your First 5 Minutes

### 1. Initialize

```bash
cd my-project
rw init
```

```
  ◆  initialized  ──────────────────────────────

     directory      /my-project/.rewind
     branch         main

  ✓ Detected: Node.js project
    Auto-ignoring: node_modules/, dist/, .next/, coverage/
    Tracking: .env, .env.local

  run 'rw save "first checkpoint"' to get started
```

Nimbi auto-detects your project type and ignores the right things — so you never have to think about it.

### 2. Save a checkpoint

```bash
rw save "everything working before I touch auth"
```

### 3. Break something, go back

```bash
rw undo           # go back 1 checkpoint
rw goto a3f2b1c8  # go back to a specific checkpoint
```

That's the core loop: **save → work → break → undo.**

---

## Commands

### Save & Restore

| Command | What it does | So that... |
|---|---|---|
| `rw save "msg"` | Save the current project state | ...you have a checkpoint to return to |
| `rw save` | Save with auto-generated message | ...you never skip saving because of a message |
| `rw undo` | Go back 1 checkpoint | ...you can recover from a mistake instantly |
| `rw undo --n 3` | Go back 3 checkpoints | ...you can jump further back in time |
| `rw goto <id>` | Restore to a specific checkpoint | ...you can pick exactly which state to return to |

### Inspect

| Command | What it does | So that... |
|---|---|---|
| `rw status` | Show tracked files, changes, and what git misses | ...you know exactly what nimbi is protecting |
| `rw list` | Show your checkpoint history | ...you can pick which one to go back to |
| `rw diff <a> <b>` | Compare two checkpoints | ...you can see exactly what changed |
| `rw diff <a> <b> --categorize` | Diff split by git-tracked vs nimbi-only | ...you see what git would have missed |

### Protect

| Command | What it does | So that... |
|---|---|---|
| `rw run "cmd"` | Run command with auto-rollback on failure | ...a failed migration never leaves you half-broken |
| `rw watch` | Auto-save every 30s when files change | ...you never lose work even if you forget to save |
| `rw bisect start` | Binary search for the checkpoint that broke things | ...you find exactly when the bug was introduced |

### Organize

| Command | What it does | So that... |
|---|---|---|
| `rw save "msg" --tag v1` | Tag a checkpoint with a name | ...you can jump back to `rw goto v1` |
| `rw branches` | List experiment branches | ...you never lose an experiment trying a different approach |
| `rw session start "feature"` | Group work into a named session | ...you can restore an entire work session later |
| `rw search "JWT"` | Search checkpoint messages | ...you can find that one checkpoint you need |

### Manage

| Command | What it does | So that... |
|---|---|---|
| `rw ignore auto` | Auto-detect and ignore `node_modules` etc. | ...your checkpoints aren't bloated |
| `rw gc` | Clean up unreferenced objects | ...your `.rewind/` folder stays small |
| `rw export <id>` | Export a checkpoint to a `.rwdb` file | ...you can share "the exact state where it crashes" |
| `rw import <file>` | Import a checkpoint from a `.rwdb` file | ...you can load someone else's exact state |
| `rw health` | Check repository integrity (SHA-256 verification) | ...you know your checkpoints are intact |
| `rw repair` | Auto-fix corruption from interrupted saves | ...you recover from disk errors or crashes |
| `rw upgrade` | Self-update to the latest version | ...you're always on the latest |
| `rw version` | Print version and build info | ...you know exactly what you're running |

---

## FAQ

**Is this the same as git?**
No. Git tracks code history for collaboration. Nimbi tracks full project state (including files git ignores) for local safety nets. Use both.

**Does it slow my machine down?**
No. `rw watch` is lightweight. Saving 1,000 files takes ~180ms. Restoring only writes files that actually changed.

**What does it actually store?**
Only files that changed. If 50 files are the same as last checkpoint, they take zero extra space (stored once, referenced by hash). Everything is gzip-compressed.

**What if I save with no message?**
Nimbi auto-generates one: `"auto: auth.js, db.js (2 file(s) changed)"`.

**Can I use it in CI/CD?**
Yes. `rw save "pre-deploy: ${{ github.sha }}"` works in GitHub Actions.

**What if `.rewind/` gets corrupted?**
Nimbi uses crash-safe atomic writes. On startup it auto-recovers from interrupted saves. Run `rw status --verify` to check all objects.

**Can I ignore files like node_modules?**
On `rw init`, Nimbi auto-detects your project type and ignores the right things. You can also run `rw ignore auto` or edit `.rewindignore` manually.

---

## Performance

| Operation | ~Time | Notes |
|---|---|---|
| Save (1,000 files) | ~180ms | Parallel SHA-256 hashing |
| Restore (10% changed) | ~40ms | Only writes what actually changed |
| Status check | ~60ms | Skips re-hashing unchanged files |
| GC | ~90ms | One pass over the object store |

Measured on Apple M2. Varies by disk speed and file sizes.

---

## Architecture

For deep dives into how Nimbi stores data, handles content-defined chunking (deduplication), and manages timelines, see [ARCHITECTURE.md](ARCHITECTURE.md).

---

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide.

- **Bug reports** — open an issue with steps to reproduce
- **Feature ideas** — open an issue first to align on direction
- **Pull requests** — fork, branch, run `go test ./... -race`, submit

---

## Security

If you discover a security vulnerability, please **do not** open a public issue. Instead, use [GitHub's private vulnerability reporting](https://github.com/itsakash-real/nimbi/security).

---

## License

MIT — see [LICENSE](LICENSE).
