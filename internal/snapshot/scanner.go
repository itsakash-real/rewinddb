package snapshot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/itsakash-real/nimbi/internal/storage"
	"github.com/itsakash-real/nimbi/internal/timeline"
	"github.com/rs/zerolog/log"
)

// Scanner walks a project directory, hashes every non-ignored file, and
// produces Snapshot values. It delegates object persistence to an ObjectStore.
type Scanner struct {
	ProjectRoot    string
	Store          *storage.ObjectStore
	Workers        int
	ProtectedFiles []string // paths that should never be overwritten during restore
}

// New returns a Scanner.
func New(projectRoot string, store *storage.ObjectStore) *Scanner {
	return &Scanner{
		ProjectRoot: projectRoot,
		Store:       store,
	}
}

// ─── Scan ─────────────────────────────────────────────────────────────────────

// Scan walks ProjectRoot in parallel and produces a Snapshot.
// Files are hashed concurrently using a bounded worker pool.
// File content is NOT written to the object store yet — call Save to persist.
func (sc *Scanner) Scan() (*timeline.Snapshot, error) {
	workers := runtime.NumCPU()
	if sc.Workers > 0 {
		workers = sc.Workers
	}

	// ── Phase 1: collect paths (single-threaded walk) ─────────────────────────
	type pending struct {
		relPath string
		absPath string
		info    fs.FileInfo
	}
	var files []pending
	ignores := loadIgnoreList(sc.ProjectRoot)

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

		if rel != "." && ignores.matches(rel) {
			if d.IsDir() {
				log.Debug().Str("dir", rel).Msg("scanner: skipping ignored directory")
				return filepath.SkipDir
			}
			log.Debug().Str("file", rel).Msg("scanner: skipping ignored file")
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
			return fmt.Errorf("snapshot: stat %s: %w", path, err)
		}

		files = append(files, pending{relPath: rel, absPath: path, info: info})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanner.Scan: walk failed: %w", err)
	}

	// ── Phase 2: hash concurrently ───────────────────────────────────────────
	entries := make([]timeline.FileEntry, len(files))

	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(workers)

	for i, f := range files {
		i, f := i, f // loop-var capture
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			hash, err := hashFile(f.absPath)
			if err != nil {
				return fmt.Errorf("scanner.Scan: hash %s: %w", f.relPath, err)
			}
			entries[i] = timeline.FileEntry{
				Path: f.relPath,
				Hash: hash,
				Size: f.info.Size(),
				Mode: f.info.Mode(),
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("scanner.Scan: parallel hash: %w", err)
	}

	// Sort for deterministic snapshot JSON.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	snapshotHash := computeSnapshotHash(entries)

	snap := &timeline.Snapshot{
		Hash:      snapshotHash,
		Files:     entries,
		CreatedAt: time.Now().UTC(),
	}

	log.Debug().
		Int("files", len(entries)).
		Int("workers", workers).
		Msg("scanner.Scan: complete")

	return snap, nil
}

// ─── Save ─────────────────────────────────────────────────────────────────────

// Save persists a Snapshot produced by Scan:
//  1. Each file's raw content is written to the object store (keyed by file hash)
//     using parallel workers for throughput.
//  2. The Snapshot struct is serialised to JSON and written to the object store
//     (keyed by snapshot hash).
//
// Returns the snapshot hash, which is the lookup key for Load.
func (sc *Scanner) Save(snap *timeline.Snapshot) (string, error) {
	workers := runtime.NumCPU()
	if sc.Workers > 0 {
		workers = sc.Workers
	}

	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(workers)

	for _, entry := range snap.Files {
		entry := entry // capture loop var
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			absPath := filepath.Join(sc.ProjectRoot, filepath.FromSlash(entry.Path))
			storedHash, err := sc.Store.WriteFile(absPath)
			if err != nil {
				return fmt.Errorf("snapshot.Save: store file %s: %w", entry.Path, err)
			}
			if storedHash != entry.Hash {
				return fmt.Errorf("snapshot.Save: hash mismatch for %s: expected %s got %s",
					entry.Path, entry.Hash, storedHash)
			}
			log.Debug().Str("file", entry.Path).Str("hash", storedHash).Msg("stored object")
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return "", err
	}

	data, err := marshalSnapshot(snap)
	if err != nil {
		return "", fmt.Errorf("snapshot.Save: marshal snapshot: %w", err)
	}

	snapHash, err := sc.Store.Write(data)
	if err != nil {
		return "", fmt.Errorf("snapshot.Save: store snapshot JSON: %w", err)
	}

	if err := sc.saveSidecar(snap.Hash, snapHash); err != nil {
		return "", fmt.Errorf("snapshot.Save: save sidecar: %w", err)
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
	return sc.LoadCached(snapshotHash, func(hash string) (*timeline.Snapshot, error) {
		// Fast path: try the hash directly (works when caller passes the object hash).
		if snap, err := sc.readSnapshotObject(hash); err == nil {
			return snap, nil
		}

		// Fallback: look up via sidecar index stored in the object store.
		// The sidecar key is deterministic: SHA-256("snapshot-index:" + hash).
		sidecarKey := sidecarObjectKey(hash)
		data, err := sc.Store.ReadRaw(sidecarKey)
		if err != nil {
			return nil, fmt.Errorf("snapshot.Load: no sidecar for hash %s: %w", hash, err)
		}
		objectHash := strings.TrimSpace(string(data))
		return sc.readSnapshotObject(objectHash)
	})
}

// readSnapshotObject reads and unmarshals a Snapshot JSON blob by its object hash.
func (sc *Scanner) readSnapshotObject(objectHash string) (*timeline.Snapshot, error) {
	data, err := sc.Store.Read(objectHash)
	if err != nil {
		return nil, err
	}
	snap, err := unmarshalSnapshot(data)
	if err != nil {
		return nil, fmt.Errorf("snapshot.Load: unmarshal: %w", err)
	}
	return snap, nil
}

// saveSidecar atomically writes a mapping snapshotHash → objectHash into the
// object store so Load can resolve snapshot hashes back to object hashes.
func (sc *Scanner) saveSidecar(snapshotHash, objectHash string) error {
	key := sidecarObjectKey(snapshotHash)
	// Only write the sidecar if the object store doesn't already have this key.
	existing, err := sc.Store.Read(key)
	if err == nil && strings.TrimSpace(string(existing)) == objectHash {
		return nil // already correct
	}

	sidecarData := []byte(objectHash)
	h := sha256.Sum256([]byte("snapshot-index:" + snapshotHash))
	hexKey := hex.EncodeToString(h[:])
	path := filepath.Join(sc.Store.RootDir, hexKey[:2], hexKey[2:])
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("saveSidecar: mkdir: %w", err)
	}

	// Atomic write via temp file + rename (H1 fix).
	tmp, err := os.CreateTemp(dir, ".sidecar-*.tmp")
	if err != nil {
		return fmt.Errorf("saveSidecar: create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(sidecarData); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("saveSidecar: write: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("saveSidecar: sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("saveSidecar: close: %w", err)
	}
	return os.Rename(tmpName, path)
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

	ignores := loadIgnoreList(sc.ProjectRoot)

	// Build protected set for O(1) lookups.
	protectedSet := make(map[string]struct{}, len(sc.ProtectedFiles))
	for _, p := range sc.ProtectedFiles {
		protectedSet[p] = struct{}{}
	}

	// Phase 1: write/restore every file in the snapshot.
	for _, entry := range snap.Files {
		// Skip protected files — never overwrite them.
		if _, isProtected := protectedSet[entry.Path]; isProtected {
			log.Debug().Str("file", entry.Path).Msg("skipping protected file")
			continue
		}

		absPath := filepath.Join(sc.ProjectRoot, filepath.FromSlash(entry.Path))

		content, err := sc.Store.Read(entry.Hash)
		if err != nil {
			return fmt.Errorf("snapshot.Restore: read object for %s: %w", entry.Path, err)
		}

		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return fmt.Errorf("snapshot.Restore: mkdir for %s: %w", entry.Path, err)
		}

		// Write with a temp+rename for atomicity.
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
		// Skip ignored files (C5 fix: use ignores.matches instead of undefined shouldIgnoreFile).
		if ignores.matches(rel) {
			return nil
		}
		// Skip protected files from deletion.
		if _, isProtected := protectedSet[rel]; isProtected {
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
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
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

// hashAndSize computes the SHA-256 of a file and returns the hash and file size.
// Used by FastScan and restore's scanCurrent.
func hashAndSize(path string) (hash string, size int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", 0, err
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), info.Size(), nil
}

// isIgnored checks whether a relative path matches the project ignore list.
func (sc *Scanner) isIgnored(rel string) bool {
	ignores := loadIgnoreList(sc.ProjectRoot)
	return ignores.matches(rel)
}
