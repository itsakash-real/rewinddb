package timeline

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// RunRecovery performs startup consistency checks on rewindDir (.rewind/).
//
// It does three things:
//  1. Completes or discards an interrupted Index.Save (leftover .index-*.tmp).
//  2. Deletes partial object writes in objects/ whose content does not match
//     their expected hash (filename = shard prefix + hash suffix).
//  3. Returns a summary of what was repaired.
func RunRecovery(rewindDir string) error {
	log.Debug().Str("dir", rewindDir).Msg("recovery: starting")

	if err := recoverIndex(rewindDir); err != nil {
		return fmt.Errorf("recovery: index: %w", err)
	}
	if err := recoverObjects(filepath.Join(rewindDir, "objects")); err != nil {
		return fmt.Errorf("recovery: objects: %w", err)
	}

	log.Debug().Msg("recovery: complete")
	return nil
}

// recoverIndex handles leftover .index-*.tmp files.
//
// If both index.json and a tmp file exist: the rename succeeded before crash,
// so discard the tmp.
// If only a tmp file exists: the rename never happened; attempt to complete it.
func recoverIndex(rewindDir string) error {
	entries, err := os.ReadDir(rewindDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	indexPath := filepath.Join(rewindDir, "index.json")

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), ".index-") && strings.HasSuffix(e.Name(), ".tmp") {
			tmp := filepath.Join(rewindDir, e.Name())

			if _, statErr := os.Stat(indexPath); statErr == nil {
				// index.json already exists — the tmp is leftover from a completed
				// (or interrupted-after-rename) operation. Safe to discard.
				log.Warn().Str("tmp", tmp).Msg("recovery: discarding leftover index tmp (index.json exists)")
				os.Remove(tmp)
				continue
			}
			// index.json is missing — attempt to complete the rename.
			log.Warn().Str("tmp", tmp).Msg("recovery: completing interrupted index rename")
			if renameErr := os.Rename(tmp, indexPath); renameErr != nil {
				log.Error().Str("tmp", tmp).Err(renameErr).Msg("recovery: could not rename index tmp")
				os.Remove(tmp) // discard rather than leave corrupt state
			}
		}
	}
	return nil
}

// recoverObjects walks the objects/ directory and deletes any file whose
// SHA-256 content hash does not match its filename (shard path).
// These are the result of a crash mid-write before the atomic rename completed.
func recoverObjects(objectsDir string) error {
	if _, err := os.Stat(objectsDir); os.IsNotExist(err) {
		return nil // no objects directory yet — nothing to check
	}

	var repaired, corrupt int

	err := filepath.WalkDir(objectsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}

		// Skip tmp files from interrupted object writes — remove them directly.
		if strings.HasSuffix(d.Name(), ".tmp") {
			log.Warn().Str("file", path).Msg("recovery: removing orphaned temp object")
			os.Remove(path)
			repaired++
			return nil
		}

		// Reconstruct expected hash from shard path layout: <2>/<62>.
		rel, err := filepath.Rel(objectsDir, path)
		if err != nil {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) != 2 {
			return nil
		}
		expectedHash := parts[0] + parts[1]
		if len(expectedHash) != 64 {
			return nil // not a standard object file — skip
		}

		// Validate: compute actual content hash.
		actualHash, err := hashFilePath(path)
		if err != nil {
			log.Warn().Str("file", path).Err(err).Msg("recovery: could not hash object")
			return nil
		}

		if actualHash != expectedHash {
			corrupt++
			log.Error().
				Str("file", path).
				Str("expected", expectedHash[:16]+"...").
				Str("actual", actualHash[:16]+"...").
				Msg("recovery: corrupt object detected — removing")

			// Make writable before removal (objects are stored 0o444).
			os.Chmod(path, 0o644)
			os.Remove(path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("recovery: walk: %w", err)
	}

	if repaired > 0 || corrupt > 0 {
		log.Info().
			Int("partial_writes_removed", repaired).
			Int("corrupt_objects_removed", corrupt).
			Msg("recovery: repairs applied")
	}
	return nil
}

// hashFilePath streams a file and returns its lowercase hex SHA-256 digest.
func hashFilePath(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
