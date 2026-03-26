package wal_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/itsakash-real/rewinddb/internal/wal"
)

func TestWAL_CompleteRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, wal.FileName)

	w, err := wal.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := w.WriteIntent("test save"); err != nil {
		t.Fatalf("WriteIntent: %v", err)
	}
	if err := w.WriteObject("abc123"); err != nil {
		t.Fatalf("WriteObject: %v", err)
	}
	if err := w.WriteCommit("cp-id-1"); err != nil {
		t.Fatalf("WriteCommit: %v", err)
	}

	status, msg, err := wal.Check(dir)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if status != wal.StatusComplete {
		t.Errorf("expected StatusComplete, got %v", status)
	}
	if msg != "test save" {
		t.Errorf("expected intent msg 'test save', got %q", msg)
	}
}

func TestWAL_IncompleteDetected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, wal.FileName)

	w, err := wal.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := w.WriteIntent("crash mid-save"); err != nil {
		t.Fatalf("WriteIntent: %v", err)
	}
	if err := w.WriteObject("deadbeef"); err != nil {
		t.Fatalf("WriteObject: %v", err)
	}
	// Simulate crash: close without WriteCommit.
	w.Close()

	status, msg, err := wal.Check(dir)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if status != wal.StatusIncomplete {
		t.Errorf("expected StatusIncomplete, got %v", status)
	}
	if msg != "crash mid-save" {
		t.Errorf("expected intent msg 'crash mid-save', got %q", msg)
	}
}

func TestWAL_CleanWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	status, _, err := wal.Check(dir)
	if err != nil {
		t.Fatalf("Check on empty dir: %v", err)
	}
	if status != wal.StatusClean {
		t.Errorf("expected StatusClean, got %v", status)
	}
}

func TestWAL_Clear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, wal.FileName)

	// Write a WAL file.
	if err := os.WriteFile(path, []byte("INTENT 123 foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := wal.Clear(dir); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("WAL file should not exist after Clear")
	}

	// Clear on non-existent file should not error.
	if err := wal.Clear(dir); err != nil {
		t.Fatalf("Clear on missing file: %v", err)
	}
}
