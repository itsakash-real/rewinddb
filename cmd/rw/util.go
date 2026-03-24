package main

import (
	"fmt"

	"github.com/itsakash-real/rewinddb/internal/config"
	"github.com/itsakash-real/rewinddb/internal/snapshot"
	"github.com/itsakash-real/rewinddb/internal/storage"
	"github.com/itsakash-real/rewinddb/internal/timeline"
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
		return nil, fmt.Errorf("not inside a RewindDB repository (run 'rw init' first): %w", err)
	}
	engine, err := timeline.New(cfg.IndexPath)
	if err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}
	store := storage.New(cfg.ObjectsDir)
	projectRoot := parentDir(cfg.RewindDir)
	sc := snapshot.New(projectRoot, store)
	return &repo{cfg: cfg, engine: engine, store: store, scanner: sc}, nil
}
