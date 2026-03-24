package storage

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"
)

// ObjectStore is a content-addressable file store modelled after Git's
// objects/ directory. Objects are addressed by their SHA-256 hex digest and
// stored at <root>/<first2>/<remaining62>.
type ObjectStore struct {
	RootDir string

	// mu guards concurrent writes to the same shard directory.
	// Reads are lock-free because os.ReadFile is safe for concurrent use and
	// the underlying file is immutable once written.
	mu sync.Mutex
}

// ErrCorruptObject is returned by Read when stored bytes do not match the hash.
var ErrCorruptObject = errors.New("storage: corrupt object")

// New returns an ObjectStore rooted at rootDir. The directory must already
// exist (created by config.Init).
func New(rootDir string) *ObjectStore {
	return &ObjectStore{RootDir: rootDir}
}

// ─── Path helpers ─────────────────────────────────────────────────────────────

// objectPath returns the on-disk path for a given hex hash.
// Layout: <root>/<hash[0:2]>/<hash[2:]>
func (s *ObjectStore) objectPath(hash string) (string, error) {
	if len(hash) < 4 {
		return "", fmt.Errorf("storage: hash too short: %q", hash)
	}
	return filepath.Join(s.RootDir, hash[:2], hash[2:]), nil
}

// ─── Write ────────────────────────────────────────────────────────────────────

// Write hashes content with SHA-256, then stores the raw bytes at the
// content-addressed path. If an object with the same hash already exists the
// write is skipped (deduplication). Returns the hex hash.
//
// Writes are atomic: content is first written to a temp file in the shard
// directory, then renamed into place. A mutex serialises concurrent writes to
// the same shard directory so only one goroutine creates it.
func (s *ObjectStore) Write(content []byte) (string, error) {
	hash := hashBytes(content)
	return hash, s.writeObject(hash, content)
}

// WriteCompressed hashes the original content, gzip-compresses it, then
// stores the compressed bytes at the content-addressed path derived from the
// original hash. Returns the hex hash of the original content.
func (s *ObjectStore) WriteCompressed(content []byte) (string, error) {
	hash := hashBytes(content)
	compressed, err := gzipCompress(content)
	if err != nil {
		return "", fmt.Errorf("storage: compress: %w", err)
	}
	return hash, s.writeObject(hash, compressed)
}

// WriteFile reads the file at filePath, computes its SHA-256, and stores the
// raw bytes in the object store. Returns the hex hash.
func (s *ObjectStore) WriteFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("storage: WriteFile: read %s: %w", filePath, err)
	}
	return s.Write(content)
}

// writeObject is the shared implementation for Write and WriteCompressed.
func (s *ObjectStore) writeObject(hash string, data []byte) error {
	path, err := s.objectPath(hash)
	if err != nil {
		return err
	}

	// Fast path: object already exists, nothing to do.
	if s.Exists(hash) {
		return nil
	}

	shardDir := filepath.Dir(path)

	// Lock while we create the shard directory and write the temp file to
	// prevent two goroutines from racing on os.MkdirAll for the same shard.
	s.mu.Lock()
	defer s.mu.Unlock()

	// Re-check under the lock (another goroutine may have just written it).
	if fileExists(path) {
		return nil
	}

	if err := os.MkdirAll(shardDir, 0o755); err != nil {
		return fmt.Errorf("storage: create shard dir %s: %w", shardDir, err)
	}

	// Write to a temp file inside the same shard directory so that os.Rename
	// is guaranteed to be on the same filesystem (required for atomicity) [web:36].
	tmp, err := os.CreateTemp(shardDir, ".obj-*.tmp")
	if err != nil {
		return fmt.Errorf("storage: create temp object: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("storage: write temp object: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("storage: sync temp object: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("storage: close temp object: %w", err)
	}

	// Atomic rename: if we crash here the temp file is orphaned but the
	// destination is never partially written [web:40].
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("storage: rename object into place: %w", err)
	}

	// Make the object read-only, matching Git's behaviour.
	return os.Chmod(path, 0o444)
}

// ─── Read ─────────────────────────────────────────────────────────────────────

// Read returns the raw bytes stored for the given hash.
// After reading, the content is re-hashed and compared to the requested hash.
// A mismatch returns ErrCorruptObject [web:40].
func (s *ObjectStore) Read(hash string) ([]byte, error) {
	path, err := s.objectPath(hash)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("storage: object not found: %s", hash)
		}
		return nil, fmt.Errorf("storage: read object %s: %w", hash, err)
	}

	// Validate integrity: recompute SHA-256 and compare to the requested hash.
	actual := hashBytes(data)
	if actual != hash {
		log.Error().
			Str("expected", hash).
			Str("actual", actual).
			Str("path", path).
			Msg("storage: CORRUPT OBJECT detected")
		return nil, fmt.Errorf("%w: expected %s got %s (path: %s)",
			ErrCorruptObject, hash[:16]+"...", actual[:16]+"...", path)
	}

	return data, nil
}

// ReadCompressed reads a gzip-compressed object and returns the decompressed
// bytes. Pair with WriteCompressed.
//
// Note: the object is stored at the hash of the ORIGINAL (uncompressed) content
// but the on-disk bytes are compressed, so we cannot use Read() (which validates
// the hash of the stored bytes). Instead we read raw, decompress, then validate
// the decompressed hash against the filename hash.
func (s *ObjectStore) ReadCompressed(hash string) ([]byte, error) {
	path, err := s.objectPath(hash)
	if err != nil {
		return nil, err
	}
	compressed, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("storage: object not found: %s", hash)
		}
		return nil, fmt.Errorf("storage: read object %s: %w", hash, err)
	}
	decompressed, err := gzipDecompress(compressed)
	if err != nil {
		return nil, fmt.Errorf("storage: decompress object %s: %w", hash, err)
	}
	// Validate integrity against the original-content hash (the filename).
	actual := hashBytes(decompressed)
	if actual != hash {
		log.Error().
			Str("expected", hash).
			Str("actual", actual).
			Str("path", path).
			Msg("storage: CORRUPT OBJECT detected")
		return nil, fmt.Errorf("%w: expected %s got %s (path: %s)",
			ErrCorruptObject, hash[:16]+"...", actual[:16]+"...", path)
	}
	return decompressed, nil
}

// ─── Exists ───────────────────────────────────────────────────────────────────

// ReadRaw returns the raw bytes stored at the given hash path without
// validating the content hash. Use this only for non-content-addressed
// metadata (e.g. sidecar index files) where the stored bytes are not
// expected to hash to the key.
func (s *ObjectStore) ReadRaw(hash string) ([]byte, error) {
	path, err := s.objectPath(hash)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("storage: object not found: %s", hash)
		}
		return nil, fmt.Errorf("storage: read object %s: %w", hash, err)
	}
	return data, nil
}

// Exists returns true if an object with the given hash is present in the store.
func (s *ObjectStore) Exists(hash string) bool {
	path, err := s.objectPath(hash)
	if err != nil {
		return false
	}
	return fileExists(path)
}

// ─── Stats ────────────────────────────────────────────────────────────────────

// Stats walks the entire object store and returns the total object count and
// the sum of all object file sizes in bytes.
func (s *ObjectStore) Stats() (objectCount int, totalBytes int64, err error) {
	err = filepath.WalkDir(s.RootDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		objectCount++
		totalBytes += info.Size()
		return nil
	})
	if err != nil {
		return 0, 0, fmt.Errorf("storage: stats walk: %w", err)
	}
	return objectCount, totalBytes, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// hashBytes returns the lowercase hex SHA-256 digest of data.
func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// fileExists is a cheap stat-based existence check.
func fileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// gzipCompress compresses src with gzip at default compression level [web:33][web:41].
func gzipCompress(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(src); err != nil {
		return nil, err
	}
	// Close must be called to flush gzip footer before reading buf [web:37].
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// gzipDecompress decompresses gzip data [web:41].
func gzipDecompress(src []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
