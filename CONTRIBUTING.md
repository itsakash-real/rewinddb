## 🤝 Contributing

First of all — thank you for considering contributing to **RewindDB**.
Every bug fix, feature, idea, or improvement genuinely makes this project better.

---

### 📜 Code of Conduct

This project follows the [Contributor Covenant v2.1](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).

**In short:**

* Be respectful
* Assume good intent
* Avoid harassment or escalation

Everyone here is contributing their time — treat others accordingly.

---

### 🚀 Before You Start

A little coordination upfront prevents wasted effort.

#### 🔎 Check Existing Issues

* Your idea or bug may already be reported
* If it exists, comment before starting work

#### 🐛 Bug Fix vs ✨ Feature

* **Bug fix** → Open a PR directly (include a failing test if possible)
* **Feature** → Open an issue first to align on direction

#### 🧠 Understand the Architecture

* Read `ARCHITECTURE.md` before modifying core components
* RewindDB relies on:

  * Content-addressable storage
  * Atomic writes
  * DAG-based structure

Breaking these guarantees means the PR will not be merged.

#### 🎯 Core Philosophy

> **Simple → Correct → Fast (in that order)**

---

### ⚙️ Development Setup

```bash
# Fork and clone
git clone https://github.com/YOUR_USERNAME/rewinddb.git
cd rewinddb

# Install dependencies
go mod tidy

# Run tests
go test ./...

# Build project
make build
```

✔ If tests pass and the build succeeds, you're ready.

**Requirements:**

* Go 1.22+

```bash
go version
```

---

### 🛠️ Making Changes

#### 📌 One Feature per PR

* Avoid bundling multiple changes
* Smaller PRs = faster review and safer merges

#### 🧪 Write Tests (Mandatory)

* No test → No merge
* Bug fix → include failing test
* Feature → include meaningful coverage

#### 🧹 Format Before Committing

```bash
go fmt ./...
go vet ./...
```

Zero output is expected.

---

### ✍️ Commit Message Guidelines

#### ✅ Good Examples

* `Fix: race condition in TimelineEngine.Save`
* `Feature: add rw export command with --format flag`
* `Refactor: move SnapshotScanner to worker pool`

#### ❌ Bad Examples

* `fixed stuff`
* `wip`
* `misc changes`

Your commit message is the first thing reviewers see — make it clear.

---

### 🧪 Testing

Run this before every push:

```bash
go test ./... -race -cover
go fmt ./...
go vet ./...
```

* `-race` → detects concurrency issues
* `-cover` → highlights missing test coverage

If any of these fail, the PR will not be reviewed.

---

### 📚 Documentation

Documentation is part of the codebase.

* New feature → update `GETTING_STARTED.md`
* Behavior change → update `README.md`
* Exported functions → include Go doc comments

If users notice it, document it.

---

### 🔁 Opening a Pull Request

#### 📝 Title Format

* `Fix: race condition in SaveCheckpoint`
* `Feature: add rw export command`
* `Refactor: extract worker pool`
* `Docs: update usage examples`

#### 📄 Description Must Include

* Why the change is needed
* Trade-offs or alternatives considered
* Related issue (e.g., `Closes #42`)
* Screenshots or CLI output (if applicable)

---

### ✅ PR Checklist

Before submitting:

* Tests pass (`go test ./... -race`)
* Code formatted (`go fmt ./...`)
* No vet issues (`go vet ./...`)
* Tests added for new logic
* Documentation updated (if needed)

---

### 👀 Review Process

* Reviews typically happen within **3–5 days**
* If no response in a week → politely follow up

Expect:

* Feedback focused on code quality
* At least one round of changes
* Collaborative discussion

Once approved, your contribution will be merged 🎉

---

### ❓ Need Help?

If you're unsure, ask before investing time:

* 💬 GitHub Discussions → ideas & questions
* 🐛 Issues → technical queries (`[QUESTION]`)
* 📩 Direct message → quick clarifications

There are no bad questions — confusion often means documentation can improve.

---

### 🚀 Final Note

Every contribution moves **RewindDB** forward.
Thank you for being part of the project.
