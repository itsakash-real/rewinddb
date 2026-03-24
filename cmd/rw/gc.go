package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/itsakash-real/rewinddb/internal/storage"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func gcCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "gc",
		Short: "Garbage-collect unreferenced objects from the object store",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}
			lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
			fl := storage.NewFileLock(lockPath)
			return fl.WithLock(func() error {
				// ── Phase 1: Collect all reachable hashes ──────────────────────────
				reachable := make(map[string]struct{})

				// Walk every checkpoint → load its snapshot JSON object, then collect
				// every file-content hash referenced in that snapshot.
				for _, cp := range r.engine.Index.Checkpoints {
					if cp.SnapshotRef == "" {
						continue
					}

					// The snapshot JSON itself is an object.
					reachable[cp.SnapshotRef] = struct{}{}

					// Load the snapshot to collect its file-object hashes.
					snap, err := r.scanner.Load(cp.SnapshotRef)
					if err != nil {
						log.Warn().Str("snapshot", cp.SnapshotRef).Err(err).
							Msg("gc: could not load snapshot; skipping")
						continue
					}
					for _, fe := range snap.Files {
						reachable[fe.Hash] = struct{}{}
					}

					// Also keep the sidecar index object for this snapshot.
					sidecarKey := snapshotSidecarKey(cp.SnapshotRef)
					reachable[sidecarKey] = struct{}{}
				}

				// ── Phase 2: Walk objects/ and find unreferenced objects ───────────
				type deadObj struct {
					path string
					size int64
				}
				var dead []deadObj

				err := filepath.WalkDir(r.cfg.ObjectsDir, func(path string, d fs.DirEntry, walkErr error) error {
					if walkErr != nil || d.IsDir() {
						return walkErr
					}
					// Reconstruct hash from the two-level shard path.
					rel, err := filepath.Rel(r.cfg.ObjectsDir, path)
					if err != nil {
						return err
					}
					parts := strings.Split(filepath.ToSlash(rel), "/")
					if len(parts) != 2 {
						return nil // skip unexpected layout
					}
					hash := parts[0] + parts[1]

					if _, ok := reachable[hash]; ok {
						return nil // referenced — keep
					}

					info, err := d.Info()
					if err != nil {
						return err
					}
					dead = append(dead, deadObj{path: path, size: info.Size()})
					return nil
				})
				if err != nil {
					return fmt.Errorf("gc: walk objects: %w", err)
				}

				// ── Phase 3: Report / delete ───────────────────────────────────────
				var totalFreed int64
				for _, obj := range dead {
					totalFreed += obj.size
				}

				if len(dead) == 0 {
					fmt.Printf("%s✓ Object store is clean%s — no unreferenced objects\n",
						colorGreen, colorReset)
					return nil
				}

				if dryRun {
					fmt.Printf("%s[dry-run] Would remove %d objects, freeing %s%s\n",
						colorYellow, len(dead), formatBytes(totalFreed), colorReset)
					for _, obj := range dead {
						rel, _ := filepath.Rel(r.cfg.ObjectsDir, obj.path)
						fmt.Printf("  would delete: %s  (%s)\n", rel, formatBytes(obj.size))
					}
					return nil
				}

				var removed int
				for _, obj := range dead {
					// Objects are stored 0o444; chmod before remove.
					if err := os.Chmod(obj.path, 0o644); err != nil {
						log.Warn().Str("path", obj.path).Err(err).Msg("gc: chmod failed, skipping")
						continue
					}
					if err := os.Remove(obj.path); err != nil {
						log.Warn().Str("path", obj.path).Err(err).Msg("gc: remove failed")
						continue
					}
					removed++
				}

				// Prune empty shard directories.
				pruneEmptyDirs(r.cfg.ObjectsDir)

				fmt.Printf("%s✓ GC complete%s: removed %d objects, freed %s\n",
					colorGreen, colorReset, removed, formatBytes(totalFreed))
				return nil
			})
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be deleted without removing anything")
	return cmd
}

// snapshotSidecarKey mirrors the key used by snapshot.saveSidecar so gc can
// mark sidecar objects as reachable.
func snapshotSidecarKey(snapshotHash string) string {
	// Reimplementation of snapshot.sidecarObjectKey to avoid a cyclic import.
	// The formula is: SHA-256("snapshot-index:" + hash).
	h := sha256.Sum256([]byte("snapshot-index:" + snapshotHash))
	return hex.EncodeToString(h[:])
}

// pruneEmptyDirs removes empty shard directories under root.
func pruneEmptyDirs(root string) {
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == root {
			return nil
		}
		entries, readErr := os.ReadDir(path)
		if readErr == nil && len(entries) == 0 {
			os.Remove(path)
		}
		return nil
	})
}

// ── Background GC (Feature 14) ────────────────────────────────────────────────

const saveCountFile = "save-count"

// runGCBackground is the actual GC logic extracted for programmatic use.
// It mirrors gcCmd's Phase 1-3 but runs silently (no output on success).
func runGCBackground(r *repo) {
	// Collect reachable hashes.
	reachable := make(map[string]struct{})
	for _, cp := range r.engine.Index.Checkpoints {
		if cp.SnapshotRef == "" {
			continue
		}
		reachable[cp.SnapshotRef] = struct{}{}
		reachable[snapshotSidecarKey(cp.SnapshotRef)] = struct{}{}

		snap, err := r.scanner.Load(cp.SnapshotRef)
		if err != nil {
			continue
		}
		for _, fe := range snap.Files {
			reachable[fe.Hash] = struct{}{}
		}
	}

	// Walk objects/ and delete unreferenced objects.
	_ = filepath.WalkDir(r.cfg.ObjectsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		rel, err := filepath.Rel(r.cfg.ObjectsDir, path)
		if err != nil {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) != 2 {
			return nil
		}
		hash := parts[0] + parts[1]
		if _, ok := reachable[hash]; ok {
			return nil
		}
		if err := os.Chmod(path, 0o644); err == nil {
			os.Remove(path)
		}
		return nil
	})
	pruneEmptyDirs(r.cfg.ObjectsDir)
	log.Debug().Msg("background GC complete")
}
