// Package wal implements a minimal Write-Ahead Log for the RewindDB save
// pipeline. Its purpose is crash-safety: if the process dies mid-save, the
// next startup can detect the incomplete operation and clean up gracefully.
//
// WAL format (plain text, one record per line):
//
//	INTENT  <unix-timestamp> <message>
//	OBJECT  <sha256-hex>
//	COMMIT  <checkpoint-id>
//
// A complete, successful save writes INTENT … one or more OBJECT … COMMIT in
// order. Any WAL that does not end with COMMIT is considered incomplete.
// The objects already written to the content-addressable store are safe to
// leave (they are immutable and will be reclaimed by GC if no checkpoint
// references them), so recovery only requires removing the WAL file.
package wal

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const FileName = "WAL"

// Status describes the state of a WAL file found on disk.
type Status int

const (
	StatusClean      Status = iota // no WAL file present
	StatusIncomplete               // INTENT without COMMIT — crash mid-save
	StatusComplete                 // COMMIT present — safe to remove
)

// WAL is an open, append-only write-ahead log.
type WAL struct {
	path string
	f    *os.File
}

// Open creates or appends to the WAL file at path.
// Callers must call Close (or WriteCommit, which closes automatically).
func Open(path string) (*WAL, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("wal.Open: mkdir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("wal.Open: %w", err)
	}
	return &WAL{path: path, f: f}, nil
}

// WriteIntent records the start of a save operation.
func (w *WAL) WriteIntent(message string) error {
	// Sanitise newlines in message so the line-oriented format stays valid.
	safe := strings.ReplaceAll(message, "\n", " ")
	_, err := fmt.Fprintf(w.f, "INTENT %d %s\n", time.Now().Unix(), safe)
	if err != nil {
		return fmt.Errorf("wal.WriteIntent: %w", err)
	}
	return w.f.Sync()
}

// WriteObject records that one object has been successfully persisted.
func (w *WAL) WriteObject(hash string) error {
	_, err := fmt.Fprintf(w.f, "OBJECT %s\n", hash)
	if err != nil {
		return fmt.Errorf("wal.WriteObject: %w", err)
	}
	// Batch syncs for OBJECT lines — WriteCommit will do the final sync.
	return nil
}

// WriteCommit records successful completion of the save, then closes the file.
// After this call the WAL is complete and can be removed with Clear.
func (w *WAL) WriteCommit(checkpointID string) error {
	if _, err := fmt.Fprintf(w.f, "COMMIT %s\n", checkpointID); err != nil {
		return fmt.Errorf("wal.WriteCommit: write: %w", err)
	}
	if err := w.f.Sync(); err != nil {
		return fmt.Errorf("wal.WriteCommit: sync: %w", err)
	}
	return w.f.Close()
}

// Close closes the underlying file without marking the operation complete.
// Call this in error paths to ensure the file handle is released.
func (w *WAL) Close() error {
	return w.f.Close()
}

// ─── Static helpers ────────────────────────────────────────────────────────────

// Check inspects a WAL file (at <rewindDir>/WAL) and returns its status
// along with the INTENT message if one was found.
func Check(rewindDir string) (Status, string, error) {
	path := filepath.Join(rewindDir, FileName)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return StatusClean, "", nil
	}
	if err != nil {
		return StatusClean, "", fmt.Errorf("wal.Check: open: %w", err)
	}
	defer f.Close()

	var intentMsg string
	hasIntent := false
	hasCommit := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "INTENT "):
			hasIntent = true
			// Extract message (everything after "INTENT <timestamp> ")
			parts := strings.SplitN(line, " ", 3)
			if len(parts) == 3 {
				intentMsg = parts[2]
			}
		case strings.HasPrefix(line, "COMMIT "):
			hasCommit = true
		}
	}
	if err := scanner.Err(); err != nil {
		return StatusClean, "", fmt.Errorf("wal.Check: scan: %w", err)
	}

	if !hasIntent {
		return StatusClean, "", nil
	}
	if hasCommit {
		return StatusComplete, intentMsg, nil
	}
	return StatusIncomplete, intentMsg, nil
}

// Clear removes the WAL file. Safe to call even when the file doesn't exist.
func Clear(rewindDir string) error {
	path := filepath.Join(rewindDir, FileName)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
