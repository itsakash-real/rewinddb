## 🤝 Contributing

First of all — thank you for considering contributing to **RewindDB**.  
Every bug fix, feature, idea, or improvement genuinely makes this project better.

---

### 📜 Code of Conduct

This project follows the [Contributor Covenant v2.1](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).

In short:
- Be respectful  
- Assume good intent  
- Do not harass or escalate conflicts  

Everyone here is contributing their time — treat others accordingly.

---

### 🚀 Before You Start

A little coordination upfront prevents wasted effort.

**Check existing issues**
- Your idea or bug may already be reported  
- If yes, comment before starting work  

**Bug Fix vs Feature**
- 🐛 Bug fix → Open a PR directly (include a failing test if possible)  
- ✨ Feature → Open an issue first to align on direction  

**Understand the architecture**
- Read `ARCHITECTURE.md` before modifying core components  
- RewindDB relies on:
  - Content-addressable storage  
  - Atomic writes  
  - DAG-based structure  

Breaking these guarantees = PR will not be merged

**Core philosophy**
> Simple → Correct → Fast (in that order)

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

✔ If tests pass and build succeeds, you're ready.

Requirements

Go 1.22+
go version
🛠️ Making Changes

One feature per PR

Avoid bundling multiple changes
Smaller PRs = faster review + safer merges

Write tests (mandatory)

No test → No merge
Bug fix → include failing test
Feature → include meaningful coverage

Format before committing

go fmt ./...
go vet ./...

Zero output required.

✍️ Commit Message Guidelines

Good examples

Fix: race condition in TimelineEngine.Save
Feature: add rw export command with --format flag
Refactor: move SnapshotScanner to worker pool

Bad examples

fixed stuff
wip
misc changes

Your commit message is the first thing reviewers see — make it clear.

🧪 Testing

Run this before every push:

go test ./... -race -cover
go fmt ./...
go vet ./...
-race → detects concurrency issues
-cover → highlights missing test coverage

If any of these fail, the PR will not be reviewed.

📚 Documentation

Documentation is part of the codebase.

New feature → update GETTING_STARTED.md
Behavior change → update README.md
Exported functions → must include Go doc comments

If users notice it → document it.

🔁 Opening a Pull Request

Title format

Fix: race condition in SaveCheckpoint
Feature: add rw export command
Refactor: extract worker pool
Docs: update usage examples

Description must include

Why the change is needed
Trade-offs or alternatives considered
Related issue (e.g., Closes #42)
Screenshots / CLI output (if applicable)
✅ PR Checklist

Before submitting:

 Tests pass (go test ./... -race)
 Code formatted (go fmt ./...)
 No vet issues (go vet ./...)
 Tests added for new logic
 Documentation updated (if needed)
👀 Review Process
Reviews typically happen within 3–5 days
If no response in a week → politely follow up

Expect:

Feedback focused on code quality
At least one round of changes
Collaborative discussion

Once approved → your contribution will be merged 🎉

❓ Need Help?

If you're unsure, ask before investing time:

💬 GitHub Discussions → ideas & open questions
🐛 Issues → technical queries ([QUESTION])
📩 Direct message → quick clarifications

There are no bad questions — confusion often means docs can improve.

Thanks again for contributing to RewindDB 🚀
Every contribution moves the project forward.
