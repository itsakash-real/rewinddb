<div align="center">

<img src="website/public/logo.svg" alt="Drift" width="220" />

**A time-travel state engine for codebases.**

[![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/itsakash-real/rewinddb?style=flat-square)](https://goreportcard.com/report/github.com/itsakash-real/rewinddb)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen?style=flat-square)](CONTRIBUTING.md)

[Install](#install) · [First 5 Minutes](#your-first-5-minutes) · [Daily Commands](#commands-youll-use-every-day) · [All Commands](#full-command-reference) · [Go SDK](#go-sdk) · [Contributing](#contributing)

</div>

---

## What is this?

You're experimenting. Something works. You change one thing. Now nothing works and you can't get back.

Drift is a single binary (`rw`) that saves your **entire project folder** as a checkpoint. You can go back to any checkpoint instantly. It works on any project — React, Python, Rust, anything — and it tracks files that git ignores, like build outputs, configs, and `.env` files.

**It is not a replacement for git.** Git is for sharing code with your team. Drift is for the messy part before that — experimenting, breaking things, and getting back safely.

---

## Install

**macOS**
```bash
brew install itsakash-real/tap/rewinddb
```

**Linux**
```bash
curl -sSL https://raw.githubusercontent.com/itsakash-real/rewinddb/main/install.sh | bash
```

**Go (any platform)**
```bash
go install github.com/itsakash-real/rewinddb/cmd/rw@latest
```

**Windows** — download `rw.exe` from [Releases](https://github.com/itsakash-real/rewinddb/releases). Colors work in Windows Terminal.

**Build from source**
```bash
git clone https://github.com/itsakash-real/rewinddb
cd rewinddb
make build
```

Verify it works:
```bash
rw version
```

### Staying up to date

Run this any time to upgrade to the latest release:
```bash
rw upgrade
```

Drift also checks for updates silently in the background. If a newer version exists, you'll see a one-liner after your next command — nothing intrusive:
```
⚡ rw 1.2.0 is available (you have 1.1.0)  →  run rw upgrade
```

It checks at most once every 24 hours and never blocks your commands.

---

## Your First 5 Minutes

Follow these steps in any project folder. Every command is explained.

### Step 1 — Set up Drift in your project

```bash
cd my-project
rw init
```

```
  ◆  initialized  ──────────────────────────────

     directory      /my-project/.rewind
     branch         main
```

This creates a hidden `.rewind/` folder where all your checkpoints are stored. You only do this once per project.

---

### Step 2 — Save your first checkpoint

```bash
rw save "everything working before I touch anything"
```

```
  ◆  checkpoint saved  ─────────────────────────

     id             a3f2b1c8
     message        "everything working before I touch anything"
     branch         main
     files          24 tracked  ·  0 changed
     saved          just now
```

The `id` (`a3f2b1c8`) is how you refer to this checkpoint later. You don't need to memorize it — you can always run `rw list` to see it.

---

### Step 3 — Make some changes, then save again

Edit a few files. Then:

```bash
rw save "added login form"
```

```
  ◆  checkpoint saved  ─────────────────────────

     id             b2e1a0f4
     message        "added login form"
     branch         main
     files          24 tracked  ·  3 changed
     saved          just now
```

You now have 2 checkpoints. Keep saving as you make progress — treat it like quicksaving a game.

> **No message? No problem.** Just run `rw save` with no message and Drift writes one for you based on what changed.

---

### Step 4 — See your checkpoints

```bash
rw list
```

```
  ◆  main  ·  2 checkpoints  ──────────────────

  ◆  b2e1a0f4  HEAD   2 minutes ago       "added login form"
  ○  a3f2b1c8          5 minutes ago       "everything working before I touch anything"
```

- `◆` means that's where you are right now (HEAD)
- `○` means an older checkpoint
- The short ID on the left is what you use to go back

---

### Step 5 — Break something, then go back

You made some changes that broke everything. Go back to any checkpoint:

```bash
rw goto a3f2b1c8
```

```
Restore to: "everything working before I touch anything"? [y/N]: y

  ◆  restored  ─────────────────────────────────

     checkpoint     a3f2b1c8
     message        "everything working before I touch anything"
     written        3 file(s)
     removed        0 file(s)
```

Your project is now exactly as it was when you saved that checkpoint. 3 files were restored, everything else was untouched.

**Even simpler — just go back one step without knowing any ID:**

```bash
rw undo
```

```
  ✓  restored to a3f2b1c8 (1 checkpoint(s) back)
```

---

That's the core loop: **save → work → break → undo.** Everything else is bonus.

---

## Commands You'll Use Every Day

### `rw save` — Save the current state

```bash
rw save "message describing what works right now"
```

Save with no message (auto-generates one from what changed):
```bash
rw save
```

Save and tag it with a name you can use later:
```bash
rw save "login complete" --tag v1.0
```

---

### `rw list` — See your history

```bash
rw list
```

Show all branches:
```bash
rw list --all
```

Show only the last 5:
```bash
rw list --n 5
```

---

### `rw goto` — Go back to any checkpoint

```bash
rw goto a3f2b1c8        # by short ID
rw goto v1.0            # by tag name
rw goto HEAD~3          # go back 3 steps from current
```

> Drift warns you if you have unsaved changes before overwriting anything.

---

### `rw undo` — Go back without knowing the ID

```bash
rw undo           # go back 1 checkpoint
rw undo --n 3     # go back 3 checkpoints
```

---

### `rw status` — See what's changed since your last save

```bash
rw status
```

```
  ╭────────────────────────────────────────────────────╮
  │   ◆  drift                                      │
  │                                                    │
  │   main  ·  a3f2b1c8  ·  5 minutes ago              │
  ╰────────────────────────────────────────────────────╯

     checkpoints    2 on branch  ·  2 total
     storage        6 objects  ·  4.2 KB

  ◆  working directory  ────────────────────────

  ~  src/auth.js         ← modified
  +  src/newfile.js      ← added (not yet saved)

  →  run rw save "message" to checkpoint
```

- `~` means modified
- `+` means new file not in any checkpoint yet
- `-` means deleted since last checkpoint

---

### `rw diff` — See exactly what changed between two checkpoints

```bash
rw diff a3f2b1c8 b2e1a0f4     # compare two specific checkpoints
rw diff HEAD HEAD~1             # compare current vs previous
rw diff HEAD~3 HEAD             # compare 3 steps ago vs now
```

---

## Protecting Yourself Before Risky Things

### Before running a command that might break things

```bash
rw run "npm run build"
```

This automatically:
1. Saves a checkpoint before running
2. Runs the command
3. If it **fails** → rolls back to the pre-run checkpoint automatically
4. If it **succeeds** → saves another checkpoint marked as passing

```
  ✓  pre-run checkpoint saved: a3f2b1c8
     running: npm run build
  ✗  command failed (exit 1)
     rolling back...
  ✓  rolled back to a3f2b1c8
```

Works for any command: `rw run "python migrate.py"`, `rw run "cargo build"`, etc.

---

### Auto-save while you work (set and forget)

```bash
rw watch
```

Drift watches your project for changes and auto-saves a checkpoint every 30 seconds when files change. You can change the interval:

```bash
rw watch --interval 5m    # save every 5 minutes
rw watch --interval 1m    # save every minute
```

Press `Ctrl+C` to stop. Good for long work sessions where you don't want to think about saving.

---

## Finding Bugs in Your History

If something broke and you don't know when, use `rw bisect`. It binary-searches your checkpoint history — like `git bisect` but for full project state.

```bash
# 1. Start bisecting
rw bisect start

# 2. Tell it a checkpoint you know was working
rw bisect good a3f2b1c8

# 3. Tell it a checkpoint you know is broken
rw bisect bad HEAD

# Drift jumps you to the middle checkpoint automatically.
# Test your code. Then:
rw bisect good    # if this middle checkpoint works
rw bisect bad     # if this middle checkpoint is broken

# Keep going. Drift narrows it down until it finds
# the exact checkpoint where the bug appeared.

# 4. When done:
rw bisect reset
```

---

## Branching — Trying Two Approaches at Once

When you restore an old checkpoint and save something new from there, Drift **automatically creates a new branch** so you don't lose either path.

```bash
# Save a base checkpoint
rw save "base: login working"         # → id: a3f2b1c8

# Try approach A
rw save "approach A: JWT tokens"      # → id: b2e1a0f4

# Go back to base and try something different
rw goto a3f2b1c8
# Drift auto-creates a new branch here

# Try approach B
rw save "approach B: session cookies" # on new branch

# See both timelines
rw list --all
```

Both approaches are preserved. You can switch between them:

```bash
rw branches                      # list all branches
rw branches switch main          # go back to main branch
rw branches branch my-experiment # create a named branch manually
```

---

## Keeping Things Organized

### Tag a checkpoint with a name

```bash
rw tag stable          # tag current checkpoint as "stable"
rw tag v2.0 b2e1a0f4   # tag a specific checkpoint
```

Then use the name instead of the ID:
```bash
rw goto stable
rw diff stable HEAD
```

---

### Group work into a named session

```bash
rw session start "feature: dark mode"
# ... work for 2 hours, save checkpoints freely ...
rw session end

# Later, jump back to where that session started
rw session restore "feature: dark mode"

# See all your sessions
rw session list
```

---

### Search your checkpoint history

```bash
rw search "JWT"          # find all checkpoints mentioning JWT
rw search "login"        # find all checkpoints mentioning login
```

---

## Ignoring Files

Drift ignores `node_modules/`, `.git/`, and common junk by default.

To auto-add ignores based on your project type (detects Node, Python, Go, Rust, etc.):
```bash
rw ignore auto
```

To add your own:
```bash
rw ignore add "dist/"
rw ignore add "*.log"
rw ignore add ".env.local"
```

To see what's currently ignored:
```bash
rw ignore list
```

You can also edit `.rewindignore` in your project root directly — same format as `.gitignore`.

---

## Sharing a Checkpoint With Someone Else

Export any checkpoint to a file:
```bash
rw export a3f2b1c8 --output bug-repro.rwdb
```

Someone else imports it into their copy of the project:
```bash
rw import bug-repro.rwdb
```

Useful for: "here's the exact state where it crashes on my machine."

---

## Storage & Cleanup

### Check what's in your repo

```bash
rw stats
```

```
  ◆  drift stats  ───────────────────────────

     repository     /my-project
     branch         main
     head           a3f2b1c8  "everything working"

  timeline
  ────────────────────────────────────────
     checkpoints    12
     branches       2

  storage
  ────────────────────────────────────────
     objects        84
     size           2.4 MB
     compression    gzip compressed
```

### Clean up unreferenced objects

Over time, deleted branches and old objects can take up space. GC removes them:

```bash
rw gc --dry-run    # see what WOULD be deleted (safe to run)
rw gc              # actually delete it
```

---

## Go SDK

If you want to use Drift from Go code — for example, to checkpoint automatically before risky operations in your app:

```bash
go get github.com/itsakash-real/rewinddb
```

```go
package main

import (
    "fmt"
    "github.com/itsakash-real/rewinddb/internal/sdk"
)

func main() {
    client, err := sdk.New("/path/to/project")
    if err != nil {
        panic(err)
    }

    // Save a checkpoint
    _, err = client.Save("before processing payment")
    if err != nil {
        panic(err)
    }

    // Restore to a checkpoint by ID, tag, or "HEAD~2"
    err = client.Goto("before processing payment")

    // Check what's changed
    status, _ := client.Status()
    fmt.Printf("modified: %d  added: %d  removed: %d\n",
        len(status.ModifiedFiles),
        len(status.AddedFiles),
        len(status.RemovedFiles),
    )
}
```

---

## FAQ

**Is this the same as git?**
No. Git tracks code history for collaboration. Drift tracks full project state (including files git ignores) for local safety nets. Use both.

**Does it slow my machine down?**
No. `rw watch` runs in the background and is lightweight. Saving 1000 files takes ~180ms.

**What does it actually store?**
Only files that changed. If 50 files are the same as last checkpoint, they take zero extra space (stored once, referenced by hash). Snapshots are gzip-compressed.

**What if I save with no message?**
Drift auto-generates one: `"auto: auth.js, db.js (2 file(s) changed)"`.

**Can I use it in CI/CD?**
Yes. `rw save "pre-deploy: ${{ github.sha }}"` works in GitHub Actions.

**Does it work without git?**
Yes. Drift has no dependency on git.

**What if `.rewind/` gets corrupted?**
Drift uses crash-safe atomic writes. On startup it auto-recovers from any interrupted saves. You can also run `rw status --verify` to check all objects.

**Can I ignore files like node_modules?**
Yes — `rw ignore auto` detects your project type and adds the right patterns. Or add them manually to `.rewindignore`.

---

## Full Command Reference

### Basics

| Command | What it does |
|---|---|
| `rw init` | Set up Drift in the current directory |
| `rw save [message]` | Save a checkpoint (message optional) |
| `rw save --tag v1.0` | Save and attach a tag |
| `rw list` | List checkpoints on current branch |
| `rw list --all` | List checkpoints on all branches |
| `rw list --n 5` | Show only the last 5 |
| `rw status` | Show what's changed since last save |
| `rw status --verify` | Also verify all stored objects |

### Going Back

| Command | What it does |
|---|---|
| `rw goto <id>` | Restore to a checkpoint by ID |
| `rw goto <tag>` | Restore by tag name |
| `rw goto HEAD~3` | Go back 3 steps from current |
| `rw undo` | Go back 1 checkpoint (no ID needed) |
| `rw undo --n 3` | Go back 3 checkpoints |
| `rw undo --force` | Skip confirmation |

### Comparing

| Command | What it does |
|---|---|
| `rw diff <id1> <id2>` | Show what changed between two checkpoints |
| `rw diff HEAD HEAD~1` | Compare current vs previous |
| `rw diff --stat` | Show summary only (no line diffs) |

### Branches & Tags

| Command | What it does |
|---|---|
| `rw branches` | List all branches |
| `rw branches branch <name>` | Create a named branch at HEAD |
| `rw branches switch <name>` | Switch to a branch |
| `rw tag <name>` | Tag the current checkpoint |
| `rw tag <name> <id>` | Tag any checkpoint |

### Power Features

| Command | What it does |
|---|---|
| `rw run "command"` | Checkpoint before, rollback on failure |
| `rw watch` | Auto-save on file changes |
| `rw watch --interval 5m` | Auto-save every 5 minutes |
| `rw bisect start` | Start binary-search for a bad checkpoint |
| `rw bisect good [id]` | Mark checkpoint as working |
| `rw bisect bad [id]` | Mark checkpoint as broken |
| `rw bisect reset` | End bisect, restore original HEAD |

### Sessions & Search

| Command | What it does |
|---|---|
| `rw session start "name"` | Start a named session |
| `rw session end` | End current session |
| `rw session list` | List all sessions |
| `rw session restore "name"` | Jump to session start |
| `rw search "keyword"` | Search checkpoint messages and tags |

### Files to Ignore

| Command | What it does |
|---|---|
| `rw ignore auto` | Auto-add patterns for your project type |
| `rw ignore add "pattern"` | Add a pattern to `.rewindignore` |
| `rw ignore list` | Show current patterns |

### Storage

| Command | What it does |
|---|---|
| `rw stats` | Show timeline and storage info |
| `rw gc` | Delete unreferenced objects |
| `rw gc --dry-run` | Preview what would be deleted |
| `rw export <id>` | Export checkpoint to `.rwdb` file |
| `rw import <file>` | Import a `.rwdb` file |

### Updates

| Command | What it does |
|---|---|
| `rw upgrade` | Upgrade `rw` to the latest version |
| `rw version` | Show current version info |

### Other

| Command | What it does |
|---|---|
| `rw completion bash` | Generate bash completion script |
| `rw --debug` | Show internal debug logs |

---

## Architecture

![System Architecture](./docs/diagrams/system-architecture.svg)

## How Timelines Work

![Timeline DAG](./docs/diagrams/timeline-dag.svg)

## What Happens When You Save

![Save Flow](./docs/diagrams/save-flow.svg)

## How Storage Works (Deduplication)

![Object Store](./docs/diagrams/object-store.svg)

## What Happens When You Restore

![Goto Restore](./docs/diagrams/goto-restore.svg)

---

## Performance

| Operation | ~Time | Notes |
|---|---|---|
| Save (1,000 files) | ~180ms | Parallel SHA-256 hashing |
| Restore (10% of files changed) | ~40ms | Only writes what actually changed |
| Status check | ~60ms | Skips re-hashing unchanged files |
| GC | ~90ms | One pass over the object store |

Measured on Apple M2. Varies by disk speed and file sizes.

---

## Drift vs Git

| | Git | Drift |
|---|---|---|
| Purpose | Share code with a team | Personal safety net while working |
| Requires a message | Yes, always | No — auto-generates one |
| Tracks `node_modules`, binaries, `.env` | No | Yes |
| Auto-branches when you time-travel | No | Yes |
| Rollback on script failure | No | `rw run "cmd"` |
| Works in CI/CD | Yes | Yes |
| Works without internet | Yes | Yes (fully local) |
| Collaboration | Yes | No |

---

## Contributing

Contributions are welcome and appreciated! Whether it's a bug report, feature request, or pull request — every bit helps.

- **Bug reports** — open an issue with steps to reproduce
- **Feature ideas** — open an issue to discuss before building
- **Pull requests** — fork, branch, make your change, run `go test ./... -race`, and submit

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide including coding standards, commit conventions, and review process.

---

## Security

If you discover a security vulnerability, please **do not** open a public issue. Instead, email [akashmaurya3160@gmail.com](mailto:akashmaurya3160@gmail.com) with details. We take security seriously and will respond promptly.

---

## License

MIT — see [LICENSE](LICENSE).
