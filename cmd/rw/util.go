package main

import (
	"fmt"

	"github.com/itsakash-real/nimbi/internal/config"
	"github.com/itsakash-real/nimbi/internal/snapshot"
	"github.com/itsakash-real/nimbi/internal/storage"
	"github.com/itsakash-real/nimbi/internal/timeline"
	"github.com/itsakash-real/nimbi/internal/wal"
)

// repo bundles the full loaded stack that most commands need.
type repo struct {
	cfg     *config.Config
	engine  *timeline.TimelineEngine
	store   *storage.ObjectStore
	scanner *snapshot.Scanner
}

func loadRepo() (*repo, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("not inside a Nimbi repository (run 'rw init' first): %w", err)
	}

	// ── WAL recovery: check for an incomplete prior save ──────────────────────
	switch status, msg, _ := wal.Check(cfg.RewindDir); status {
	case wal.StatusIncomplete:
		// A save was interrupted before the checkpoint was committed.
		// The objects it wrote are safe (COW store), but the checkpoint
		// was never added to the index — so the state is consistent.
		// We just remove the stale WAL so the next save starts clean.
		yellowP.Printf("⚠  previous save was interrupted (%q) — cleaning up WAL\n", msg)
		_ = wal.Clear(cfg.RewindDir)
	case wal.StatusComplete:
		// Normal: last save finished, WAL is just leftover. Remove it.
		_ = wal.Clear(cfg.RewindDir)
	}

	engine, err := timeline.New(cfg.IndexPath)
	if err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}
	store := storage.New(cfg.ObjectsDir)
	projectRoot := parentDir(cfg.RewindDir)
	sc := snapshot.New(projectRoot, store)

	// Load protected files so restores respect them.
	if protected, err := loadProtected(cfg.RewindDir); err == nil && len(protected) > 0 {
		sc.ProtectedFiles = protected
	}

	return &repo{cfg: cfg, engine: engine, store: store, scanner: sc}, nil
}
