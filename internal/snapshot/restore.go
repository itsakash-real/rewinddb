package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/itsakash-real/nimbi/internal/timeline"
)

// RestoreResult holds metrics from a delta-based Restore operation.
type RestoreResult struct {
	Written int // files written to disk
	Skipped int // files already matching target hash (unchanged)
	Removed int // files present on disk but absent from target snapshot
}

// RestoreDelta applies the target snapshot to ProjectRoot using delta-only writes.
// It scans the current disk state, skips files that already match the target
// hash, writes only changed or new files, and removes files not in the target.
// This is more efficient than the full Restore method when few files changed.
func (s *Scanner) RestoreDelta(target *timeline.Snapshot) (*RestoreResult, error) {
	result := &RestoreResult{}

	// ── Build target map ──────────────────────────────────────────────────────
	targetMap := make(map[string]timeline.FileEntry, len(target.Files))
	for _, fe := range target.Files {
		targetMap[fe.Path] = fe
	}

	// ── Scan current disk state without storing ───────────────────────────────
	currentMap, err := s.scanCurrentDisk()
	if err != nil {
		return nil, fmt.Errorf("restore: scan current: %w", err)
	}

	// ── Write only changed / new files ────────────────────────────────────────
	for _, fe := range target.Files {
		cur, exists := currentMap[fe.Path]
		if exists && cur == fe.Hash {
			result.Skipped++
			continue // file is already correct — skip the write
		}

		data, readErr := s.Store.Read(fe.Hash)
		if readErr != nil {
			return nil, fmt.Errorf("restore: read object %s for %s: %w", fe.Hash[:8], fe.Path, readErr)
		}

		dest := filepath.Join(s.ProjectRoot, filepath.FromSlash(fe.Path))
		if mkErr := os.MkdirAll(filepath.Dir(dest), 0o755); mkErr != nil {
			return nil, fmt.Errorf("restore: mkdir for %s: %w", fe.Path, mkErr)
		}
		if writeErr := atomicWriteFile(dest, data, fe.Mode); writeErr != nil {
			return nil, fmt.Errorf("restore: write %s: %w", fe.Path, writeErr)
		}
		result.Written++
	}

	// ── Remove files not in target ────────────────────────────────────────────
	for path := range currentMap {
		if _, inTarget := targetMap[path]; !inTarget {
			abs := filepath.Join(s.ProjectRoot, filepath.FromSlash(path))
			if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
				log.Warn().Str("path", path).Err(err).Msg("restore: could not remove file")
				continue
			}
			result.Removed++
		}
	}

	log.Info().
		Int("written", result.Written).
		Int("skipped", result.Skipped).
		Int("removed", result.Removed).
		Msg("restore: complete")

	return result, nil
}

// scanCurrentDisk walks ProjectRoot and computes a hash map of existing files
// without writing anything to the object store. Returns map[relPath]hash.
func (s *Scanner) scanCurrentDisk() (map[string]string, error) {
	m := make(map[string]string)

	ignores := loadIgnoreList(s.ProjectRoot)

	err := filepath.WalkDir(s.ProjectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(s.ProjectRoot, path)
		if relErr != nil {
			return fmt.Errorf("restore: rel path: %w", relErr)
		}
		rel = filepath.ToSlash(rel)

		if rel != "." && ignores.matches(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if !d.Type().IsRegular() {
			return nil
		}
		hash, err := hashFileForRestore(path)
		if err != nil {
			log.Warn().Str("path", rel).Err(err).Msg("restore: cannot hash file, treating as absent")
			return nil
		}
		m[rel] = hash
		return nil
	})
	return m, err
}

// hashFileForRestore computes the SHA-256 of a file for restore comparison.
func hashFileForRestore(path string) (string, error) {
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

// atomicWriteFile writes data to path via a temp file + rename for safety.
func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".restore-*.tmp")
	if err != nil {
		return fmt.Errorf("atomicWriteFile: create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Chmod(path, mode)
}
