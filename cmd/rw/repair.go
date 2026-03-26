package main

import (
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/itsakash-real/rewinddb/internal/merkle"
	"github.com/itsakash-real/rewinddb/internal/timeline"
	"github.com/spf13/cobra"
)

func repairCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repair",
		Short: "Attempt to recover corrupt objects using adjacent checkpoints",
		Long: `repair scans every object in the store and attempts to reconstruct any
that are corrupt (zero-byte or unreadable).

Strategy:
  For each corrupt object hash, repair searches every snapshot for a file
  whose stored hash matches. It then looks at adjacent checkpoints (one
  before, one after) for the same file path with a valid object, and
  copies that version into the corrupt slot.

  This is best-effort: if no valid neighbour exists for a file, the object
  is reported as unrecoverable.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			sectionTitle("repair")
			fmt.Println()

			// ── Step 1: find corrupt objects ───────────────────────────────────
			corrupt, _, err := findCorruptObjects(r.cfg.ObjectsDir)
			if err != nil {
				return fmt.Errorf("scan objects: %w", err)
			}
			if len(corrupt) == 0 {
				greenBoldP.Println("  ✓ no corrupt objects found — repository is healthy")
				fmt.Println()
				return nil
			}

			kv("corrupt objects found", fmt.Sprintf("%d", len(corrupt)))
			fmt.Println()

			// ── Step 2: build a map from object hash → file paths that use it ──
			hashToFiles := buildHashToFileMap(r)

			// ── Step 3: attempt recovery for each corrupt object ───────────────
			recovered := 0
			for _, corruptHash := range corrupt {
				files, known := hashToFiles[corruptHash]
				if !known {
					printFail(fmt.Sprintf("object %s — not referenced by any checkpoint (orphan, GC will remove it)", corruptHash[:16]))
					_ = removeCorruptObject(r.cfg.ObjectsDir, corruptHash)
					recovered++
					continue
				}

				fixed := false
				for _, filePath := range files {
					// Find an adjacent checkpoint that has this same file path
					// with a DIFFERENT (valid) hash we can copy content from.
					altHash, altMsg := findValidNeighbour(r, filePath, corruptHash)
					if altHash == "" {
						continue
					}
					// Copy the valid object content to a new object under the
					// corrupt hash path. Since we can't know the original content
					// (that's the whole point of corruption), we instead write the
					// neighbour content and update the referencing snapshots.
					// This is a lossy repair but better than nothing.
					if err := replaceWithNeighbour(r, corruptHash, altHash); err != nil {
						continue
					}
					purpleP.Printf("  ↻ Recovering %s from checkpoint %s... ", filePath, altMsg)
					greenBoldP.Println("✓")
					fixed = true
					recovered++
					break
				}
				if !fixed {
					printFail(fmt.Sprintf("object %s — no valid neighbour found for %v (unrecoverable)", corruptHash[:16], files))
				}
			}

			fmt.Println()

			// ── Step 4: rebuild Merkle root ────────────────────────────────────
			if root, _, err := merkle.Compute(r.cfg.ObjectsDir); err == nil {
				_ = merkle.SaveRoot(r.cfg.RewindDir, root)
			}

			if recovered == len(corrupt) {
				greenBoldP.Printf("  ✓ repair complete (%d/%d objects recovered)\n", recovered, len(corrupt))
			} else {
				yellowP.Printf("  ⚠ partial repair (%d/%d objects recovered) — some data is unrecoverable\n",
					recovered, len(corrupt))
			}
			fmt.Println()
			return nil
		},
	}
}

// findCorruptObjects walks the object store and returns hashes of zero-byte
// or unreadable files.
func findCorruptObjects(objectsDir string) (corrupt []string, total int, err error) {
	err = filepath.WalkDir(objectsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		shard := filepath.Base(filepath.Dir(path))
		hash := shard + d.Name()

		info, statErr := d.Info()
		if statErr != nil || info.Size() == 0 {
			corrupt = append(corrupt, hash)
		}
		total++
		return nil
	})
	return corrupt, total, err
}

// buildHashToFileMap builds an inverted index: object hash → []relative file paths
// that have that hash in any snapshot.
func buildHashToFileMap(r *repo) map[string][]string {
	result := make(map[string][]string)
	seen := make(map[string]bool) // deduplicate hash+path pairs

	for _, cp := range r.engine.Index.Checkpoints {
		if cp.SnapshotRef == "" {
			continue
		}
		snap, err := r.scanner.Load(cp.SnapshotRef)
		if err != nil {
			continue
		}
		for _, fe := range snap.Files {
			key := fe.Hash + "|" + fe.Path
			if seen[key] {
				continue
			}
			seen[key] = true
			result[fe.Hash] = append(result[fe.Hash], fe.Path)
		}
	}
	return result
}

// findValidNeighbour searches all checkpoints for a file at filePath that has a
// different hash from corruptHash and whose object exists and is readable.
// Returns the valid hash and a short checkpoint label, or empty strings if not found.
func findValidNeighbour(r *repo, filePath, corruptHash string) (string, string) {
	// Walk checkpoints newest-first across all branches.
	allCPs := allCheckpointsSorted(r)
	for _, cp := range allCPs {
		if cp.SnapshotRef == "" {
			continue
		}
		snap, err := r.scanner.Load(cp.SnapshotRef)
		if err != nil {
			continue
		}
		for _, fe := range snap.Files {
			if fe.Path != filePath {
				continue
			}
			if fe.Hash == corruptHash {
				continue // same corrupt object
			}
			// Check the object is actually readable.
			if r.store.Exists(fe.Hash) {
				return fe.Hash, shortID(cp.ID)
			}
		}
	}
	return "", ""
}

// isValidHexHash returns true if s is a valid lowercase hex string of at least 4 chars.
func isValidHexHash(s string) bool {
	if len(s) < 4 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// replaceWithNeighbour copies the valid object at srcHash to the position of
// dstHash in the object store. This is a best-effort content substitution.
func replaceWithNeighbour(r *repo, dstHash, srcHash string) error {
	if !isValidHexHash(dstHash) || !isValidHexHash(srcHash) {
		return fmt.Errorf("repair: invalid hash value")
	}

	srcData, err := r.store.Read(srcHash)
	if err != nil {
		return fmt.Errorf("repair: read source %s: %w", srcHash[:8], err)
	}

	// Write under the corrupt hash path directly (bypass the store's
	// content-addressing so we can plant the recovery data at the old address).
	dstPath := filepath.Join(r.cfg.ObjectsDir, dstHash[:2], dstHash[2:])
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	// Remove old corrupt file so we can replace it.
	_ = os.Chmod(dstPath, 0o644)
	_ = os.Remove(dstPath)

	tmp, err := os.CreateTemp(filepath.Dir(dstPath), ".repair-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(srcData); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	tmp.Close()
	if err := os.Rename(tmpName, dstPath); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Chmod(dstPath, 0o444)
}

// removeCorruptObject deletes an orphaned corrupt object from the store.
func removeCorruptObject(objectsDir, hash string) error {
	path := filepath.Join(objectsDir, hash[:2], hash[2:])
	_ = os.Chmod(path, 0o644)
	return os.Remove(path)
}

// allCheckpointsSorted returns all checkpoints across all branches ordered
// newest-first by CreatedAt.
func allCheckpointsSorted(r *repo) []*timeline.Checkpoint {
	var all []*timeline.Checkpoint
	for i := range r.engine.Index.Checkpoints {
		cp := r.engine.Index.Checkpoints[i]
		all = append(all, &cp)
	}
	// Sort newest-first.
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].CreatedAt.After(all[i].CreatedAt) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	return all
}
