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
  9d1e4f72  refactored auth — something's wrong   (branch:
