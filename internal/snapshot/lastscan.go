package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
)

const lastScanFile = "last-scan.json"

// LastScanRecord maps relative file path → last-seen mtime + hash.
type LastScanRecord struct {
	UpdatedAt time.Time                `json:"updated_at"`
	Files     map[string]LastScanEntry `json:"files"`
}

// LastScanEntry records the mtime and hash observed during the previous scan.
type LastScanEntry struct {
	ModTime time.Time `json:"mtime"`
	Hash    string    `json:"hash"`
	Size    int64     `json:"size"`
}

// LoadLastScan reads the persisted last-scan record from rewindDir.
// Returns an empty record (not an error) if the file does not exist.
func LoadLastScan(rewindDir string) (*LastScanRecord, error) {
	path := filepath.Join(rewindDir, lastScanFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &LastScanRecord{Files: make(map[string]LastScanEntry)}, nil
		}
		return nil, err
	}
	var rec LastScanRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		log.Warn().Err(err).Msg("last-scan: corrupt cache, ignoring")
		return &LastScanRecord{Files: make(map[string]LastScanEntry)}, nil
	}
	return &rec, nil
}

// SaveLastScan atomically writes rec to rewindDir/last-scan.json.
func SaveLastScan(rewindDir string, rec *LastScanRecord) error {
	rec.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(rewindDir, lastScanFile)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// FastScan performs a status scan using cached mtimes.
// Files whose mtime is unchanged since the last scan reuse the cached hash.
// Only files with a newer mtime are re-hashed from disk [web:125].
//
// Returns (changedPaths, allEntries, error).
func (s *Scanner) FastScan(rewindDir string) (changed []string, entries map[string]LastScanEntry, err error) {
	rec, err := LoadLastScan(rewindDir)
	if err != nil {
		return nil, nil, err
	}

	current := make(map[string]LastScanEntry)
	ignores := loadIgnoreList(s.ProjectRoot)

	err = filepath.WalkDir(s.ProjectRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, _ := filepath.Rel(s.ProjectRoot, path)
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

		info, err := d.Info()
		if err != nil {
			return nil
		}

		cached, hasCached := rec.Files[rel]

		// mtime unchanged AND size unchanged → reuse cached hash (no disk read).
		if hasCached &&
			info.ModTime().Equal(cached.ModTime) &&
			info.Size() == cached.Size {
			current[rel] = cached
			return nil
		}

		// mtime changed or new file → must re-hash.
		hash, size, hashErr := hashAndSize(path)
		if hashErr != nil {
			return nil
		}
		entry := LastScanEntry{
			ModTime: info.ModTime().UTC(),
			Hash:    hash,
			Size:    size,
		}
		current[rel] = entry

		// Only flag as changed if hash is actually different from cache.
		if !hasCached || hash != cached.Hash {
			changed = append(changed, rel)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	// Detect deletions: files in last scan but no longer on disk.
	for rel := range rec.Files {
		if _, exists := current[rel]; !exists {
			changed = append(changed, rel)
		}
	}

	// Persist updated scan cache.
	rec.Files = current
	_ = SaveLastScan(rewindDir, rec) // best-effort; don't fail status on write error

	return changed, current, nil
}
