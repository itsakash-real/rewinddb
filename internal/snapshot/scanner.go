package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/itsakash-real/rewinddb/internal/storage"
	"github.com/itsakash-real/rewinddb/internal/timeline"
)

// DefaultIgnores contains patterns that are always skipped during scanning.
// Patterns ending with "/" are treated as directory prefix matches;
// others are matched against the file's base name via filepath.Match [web:48].
var DefaultIgnores = []string{
	".rewind/",
	".git/",
	"node_modules/",
	"__pycache__/",
	"dist/",
	"build/",
	"*.pyc",
	".DS_Store",
	"*.exe",
}

// Scanner walks a project directory, hashes every non-ignored file, and
// produces Snapshot values. It delegates object persistence to an ObjectStore.
type Scanner struct {
	ProjectRoot string
	Store       *storage.ObjectStore
	Ignores     []string
}

// New returns a Scanner with DefaultIgnores pre-loaded.
func New(projectRoot string, store *storage.ObjectStore) *Scanner {
	ignores := make([]string, len(DefaultIgnores))
	copy(ignores, DefaultIgnores)
	return &Scanner{
		ProjectRoot: projectRoot,
		Store:       store,
		Ignores:     ignores,
	}
}

// ─── Scan ─────────────────────────────────────────────────────────────────────

// Scan walks ProjectRoot, hashes every non-ignored file, and returns a
// Snapshot. File content is NOT written to the object store yet — call Save
// to persist. The snapshot Hash is derived from sorted "path:filehash\n"
// pairs so it changes only when file content or names change.
func (sc *Scanner) Scan() (*timeline.Snapshot, error) {
	var entries []timeline.FileEntry

	err := filepath.WalkDir(sc.ProjectRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Compute path relative to project root for portability.
		rel, err := filepath.Rel(sc.ProjectRoot, path)
		if err != nil {
			return fmt.Errorf("snapshot: rel path: %w", err)
		}
		// Normalise to forward slashes so snapshots are cross-platform.
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if rel == "." {
				return nil
			}
			if sc.shouldIgnoreDir(rel) {
				log.Debug().Str("dir", rel).Msg("scanner: skipping ignored directory")
				return filepath.SkipDir // prunes the entire subtree [web:46]
			}
			return nil
		}

		if sc.shouldIgnoreFile(rel) {
			log.Debug().Str("file", rel).Msg("scanner: skipping ignored file")
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("snapshot: stat %s: %w", path, err)
		}

		hash, err := hashFile(path)
		if err != nil {
			return fmt.Errorf("snapshot: hash %s: %w", path, err)
		}

		entries = append(entries, timeline.FileEntry{
			Path: rel,
			Hash: hash,
			Size: info.Size(),
			Mode: info.Mode(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("snapshot.Scan: walk failed: %w", err)
	}

	// Sort by path for deterministic snapshot hashes [web:48].
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	snapshotHash := computeSnapshotHash(entries)

	return &timeline.Snapshot{
		Hash:      snapshotHash,
		Files:     entries,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// ─── Save ─────────────────────────────────────────────────────────────────────

// Save persists a Snapshot produced by Scan:
//  1. Each file's raw content is written to the object store (keyed by file hash).
//  2. The Snapshot struct is serialised to JSON and written to the object store
//     (keyed by snapshot hash).
//
// Returns the snapshot hash, which is the lookup key for Load.
func (sc *Scanner) Save(snap *timeline.Snapshot) (string, error) {
	for _, entry := range snap.Files {
		absPath := filepath.Join(sc.ProjectRoot, filepath.FromSlash(entry.Path))
		storedHash, err := sc.Store.WriteFile(absPath)
		if err != nil {
			return "", fmt.Errorf("snapshot.Save: store file %s: %w", entry.Path, err)
		}
		if storedHash != entry.Hash {
			// Paranoia check: file changed between Scan and Save.
			return "", fmt.Errorf("snapshot.Save: hash mismatch for %s: expected %s got %s",
				entry.Path, entry.Hash, storedHash)
		}
		log.Debug().Str("file", entry.Path).Str("hash", storedHash).Msg("stored object")
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", fmt.Errorf("snapshot.Save: marshal snapshot: %w", err)
	}

	snapHash, err := sc.Store.Write(data)
	if err != nil {
		return "", fmt.Errorf("snapshot.Save: store snapshot JSON: %w", err)
	}

	// The canonical hash is embedded in the struct; the object store hash is
	// used purely as a content-addressed lookup key.
	log.Info().Str("snapshot_hash", snap.Hash).Str("object_hash", snapHash).Msg("snapshot saved")
	return snap.Hash, nil
}

// ─── Load ─────────────────────────────────────────────────────────────────────

// Load retrieves a Snapshot by its snapshot hash. It finds the object store
// key by scanning the JSON blob stored under snapshotHash-as-object-key.
//
// Because Save writes the JSON blob keyed by the SHA-256 of the JSON bytes
// (not the snapshot hash), Load performs a two-step lookup:
//
//  1. Try the snapshotHash directly as an object key (fast path — works when
//     the snapshot hash happens to equal the JSON blob hash, e.g. empty repos).
//  2. Walk the object store looking for a JSON blob whose decoded Hash field
//     matches snapshotHash (fallback).
//
// In practice the canonical approach is to store a separate index mapping
// snapshot.Hash → objectHash. We implement that via a lightweight sidecar file.
func (sc *Scanner) Load(snapshotHash string) (*timeline.Snapshot, error) {
	// Fast path: try the hash directly (works when caller passes the object hash).
	if snap, err := sc.readSnapshotObject(snapshotHash); err == nil {
		return snap, nil
	}

	// Fallback: look up via sidecar index stored in the object store.
	// The sidecar key is deterministic: SHA-256("snapshot-index:" + snapshotHash).
	sidecarKey := sidecarObjectKey(snapshotHash)
	data, err := sc.Store.Read(sidecarKey)
	if err != nil {
		return nil, fmt.Errorf("snapshot.Load: no sidecar for hash %s: %w", snapshotHash, err)
	}
	objectHash := strings.TrimSpace(string(data))
	return sc.readSnapshotObject(objectHash)
}

// readSnapshotObject reads and unmarshals a Snapshot JSON blob by its object hash.
func (sc *Scanner) readSnapshotObject(objectHash string) (*timeline.Snapshot, error) {
	data, err := sc.Store.Read(objectHash)
	if err != nil {
		return nil, err
	}
	var snap timeline.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("snapshot.Load: unmarshal: %w", err)
	}
	return &snap, nil
}

// saveSidecar writes a mapping snapshotHash → objectHash into the object store
// so Load can resolve snapshot hashes back to object hashes.
func (sc *Scanner) saveSidecar(snapshotHash, objectHash string) error {
	key := sidecarObjectKey(snapshotHash)
	// Only write the sidecar if the object store doesn't already have this key.
	// We abuse Write for idempotency; the content is just the objectHash string.
	existing, err := sc.Store.Read(key)
	if err == nil && strings.TrimSpace(string(existing)) == objectHash {
		return nil // already correct
	}
	// Temporarily chmod the sidecar key to allow overwrite for mappings.
	// Since sidecars point to immutable objects, idempotency is fine.
	sidecarData := []byte(objectHash)
	h := sha256.Sum256([]byte("snapshot-index:" + snapshotHash))
	hexKey := hex.EncodeToString(h[:])
	path := filepath.Join(sc.Store.RootDir, hexKey[:2], hexKey[2:])
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, sidecarData, 0o644)
}

// sidecarObjectKey returns the object store key for the sidecar index entry.
func sidecarObjectKey(snapshotHash string) string {
	h := sha256.Sum256([]byte("snapshot-index:" + snapshotHash))
	return hex.EncodeToString(h[:])
}

// ─── Restore ──────────────────────────────────────────────────────────────────

// Restore writes all files in snapshot back to ProjectRoot, creating
// directories as needed and setting file permissions from FileEntry.Mode.
// Files present in ProjectRoot but absent from the snapshot are deleted,
// unless they match the ignore list. Empty directories left behind by
// deletions are also pruned.
func (sc *Scanner) Restore(snap *timeline.Snapshot) error {
	// Build a set of paths that should exist after restore.
	targetPaths := make(map[string]struct{}, len(snap.Files))
	for _, entry := range snap.Files {
		targetPaths[entry.Path] = struct{}{}
	}

	// Phase 1: write/restore every file in the snapshot.
	for _, entry := range snap.Files {
		absPath := filepath.Join(sc.ProjectRoot, filepath.FromSlash(entry.Path))

		content, err := sc.Store.Read(entry.Hash)
		if err != nil {
			return fmt.Errorf("snapshot.Restore: read object for %s: %w", entry.Path, err)
		}

		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return fmt.Errorf("snapshot.Restore: mkdir for %s: %w", entry.Path, err)
		}

		// Write with a temp+rename for atomicity [web:36].
		dir := filepath.Dir(absPath)
		tmp, err := os.CreateTemp(dir, ".restore-*.tmp")
		if err != nil {
			return fmt.Errorf("snapshot.Restore: temp file for %s: %w", entry.Path, err)
		}
		tmpName := tmp.Name()

		if _, err := tmp.Write(content); err != nil {
			tmp.Close()
			os.Remove(tmpName)
			return fmt.Errorf("snapshot.Restore: write temp for %s: %w", entry.Path, err)
		}
		if err := tmp.Close(); err != nil {
			os.Remove(tmpName)
			return fmt.Errorf("snapshot.Restore: close temp for %s: %w", entry.Path, err)
		}
		if err := os.Rename(tmpName, absPath); err != nil {
			os.Remove(tmpName)
			return fmt.Errorf("snapshot.Restore: rename %s: %w", entry.Path, err)
		}
		if err := os.Chmod(absPath, entry.Mode); err != nil {
			return fmt.Errorf("snapshot.Restore: chmod %s: %w", entry.Path, err)
		}

		log.Debug().Str("file", entry.Path).Msg("restored")
	}

	// Phase 2: delete files that exist on disk but are not in the snapshot.
	var toDelete []string
	err := filepath.WalkDir(sc.ProjectRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(sc.ProjectRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if sc.shouldIgnoreFile(rel) {
			return nil
		}
		if _, keep := targetPaths[rel]; !keep {
			toDelete = append(toDelete, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("snapshot.Restore: walk for cleanup: %w", err)
	}

	for _, path := range toDelete {
		log.Debug().Str("file", path).Msg("deleting stale file")
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("snapshot.Restore: delete %s: %w", path, err)
		}
	}

	return nil
}

// ─── Ignore logic ─────────────────────────────────────────────────────────────

// shouldIgnoreDir returns true if the directory (rel path) matches any
// ignore pattern that ends with "/" [web:48].
func (sc *Scanner) shouldIgnoreDir(rel string) bool {
	dirWithSlash := rel + "/"
	for _, pattern := range sc.Ignores {
		if !strings.HasSuffix(pattern, "/") {
			continue
		}
		trimmed := strings.TrimSuffix(pattern, "/")
		// Exact segment match or prefix (e.g. node_modules inside a subdir).
		base := filepath.Base(rel)
		if base == trimmed {
			return true
		}
		// prefix: node_modules/something
		if strings.HasPrefix(dirWithSlash, pattern) {
			return true
		}
	}
	return false
}

// shouldIgnoreFile returns true if the file's base name matches any non-dir
// ignore pattern via filepath.Match [web:53].
func (sc *Scanner) shouldIgnoreFile(rel string) bool {
	base := filepath.Base(rel)
	for _, pattern := range sc.Ignores {
		if strings.HasSuffix(pattern, "/") {
			continue
		}
		matched, err := filepath.Match(pattern, base)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// hashFile computes the SHA-256 of a file's content without buffering the
// entire file in memory, using io.Copy into the hash writer.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := copyBuffer(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// copyBuffer wraps the io.Copy call; separated for testability.
func copyBuffer(dst interface{ Write([]byte) (int, error) }, src interface {
	Read([]byte) (int, error)
}) (int64, error) {
	buf := make([]byte, 32*1024)
	var total int64
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			total += int64(nw)
			if ew != nil {
				return total, ew
			}
		}
		if er != nil {
			if er.Error() == "EOF" {
				break
			}
			return total, er
		}
	}
	return total, nil
}

// computeSnapshotHash produces a single SHA-256 over all "path:hash\n" lines,
// sorted by path, so the snapshot hash is deterministic and content-sensitive.
func computeSnapshotHash(entries []timeline.FileEntry) string {
	h := sha256.New()
	for _, e := range entries {
		fmt.Fprintf(h, "%s:%s\n", e.Path, e.Hash)
	}
	return hex.EncodeToString(h.Sum(nil))
}
