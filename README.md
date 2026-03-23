# RewindDB

> A time-travel state engine for codebases — like Git, but for runtime state.

## Install

```bash
git clone https://github.com/yourusername/rewinddb
cd rewinddb
go build -o rw ./cmd/rw
```

## Usage

```
rw save [message]         Save a snapshot of current state
rw goto <snapshot-id>     Restore to a previous snapshot
rw list                   List snapshots on the current branch
rw branches               List all branches
rw diff <snap-a> <snap-b> Show delta between two snapshots
rw gc                     Garbage-collect unreachable objects
```

## Repository Layout

```
.rewind/
├── objects/     Content-addressable blob store
├── snapshots/   Serialized snapshot metadata
├── branches/    Branch pointer files
└── index        Staging index
```

## Development

```bash
go test ./...
go vet ./...
```
