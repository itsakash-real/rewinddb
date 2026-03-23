package storage_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/itsakash-real/rewinddb/internal/storage"
)

// newStore creates an ObjectStore backed by a fresh temp directory.
func newStore(t *testing.T) *storage.ObjectStore {
	t.Helper()
	dir := t.TempDir()
	return storage.New(dir)
}

// expectedHash returns the SHA-256 hex digest of data, mirroring the store's
// internal logic so tests don't hard-code hashes.
func expectedHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ─── Write / Read roundtrip ───────────────────────────────────────────────────

func TestWrite_Read_Roundtrip(t *testing.T) {
	s := newStore(t)
	content := []byte("hello rewinddb object store")

	hash, err := s.Write(content)
	require.NoError(t, err)
	assert.Equal(t, expectedHash(content), hash)

	got, err := s.Read(hash)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestWrite_StoresAtCorrectPath(t *testing.T) {
	s := newStore(t)
	content := []byte("path layout test")
	hash, err := s.Write(content)
	require.NoError(t, err)

	// Verify Git-style shard layout: <root>/<hash[0:2]>/<hash[2:]>
	expectedPath := filepath.Join(s.RootDir, hash[:2], hash[2:])
	assert.FileExists(t, expectedPath)
}

func TestWrite_ObjectIsReadOnly(t *testing.T) {
	s := newStore(t)
	content := []byte("immutable object")
	hash, err := s.Write(content)
	require.NoError(t, err)

	path := filepath.Join(s.RootDir, hash[:2], hash[2:])
	info, err := os.Stat(path)
	require.NoError(t, err)
	// Verify that only read bits are set (0o444)
	assert.Equal(t, os.FileMode(0o444), info.Mode().Perm())
}

// ─── Deduplication ───────────────────────────────────────────────────────────

func TestWrite_Deduplication_SameContentWrittenTwice(t *testing.T) {
	s := newStore(t)
	content := []byte("deduplicated content")

	hash1, err := s.Write(content)
	require.NoError(t, err)

	hash2, err := s.Write(content)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2, "same content must produce same hash")

	// Confirm only one object file exists on disk
	count, _, err := s.Stats()
	require.NoError(t, err)
	assert.Equal(t, 1, count, "deduplication must result in exactly one stored object")
}

func TestWrite_DifferentContent_DifferentObjects(t *testing.T) {
	s := newStore(t)
	h1, err := s.Write([]byte("content A"))
	require.NoError(t, err)
	h2, err := s.Write([]byte("content B"))
	require.NoError(t, err)

	assert.NotEqual(t, h1, h2)

	count, _, err := s.Stats()
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

// ─── Exists ───────────────────────────────────────────────────────────────────

func TestExists_TrueAfterWrite(t *testing.T) {
	s := newStore(t)
	content := []byte("existence check")
	hash, err := s.Write(content)
	require.NoError(t, err)
	assert.True(t, s.Exists(hash))
}

func TestExists_FalseForUnknownHash(t *testing.T) {
	s := newStore(t)
	assert.False(t, s.Exists("0000000000000000000000000000000000000000000000000000000000000000"))
}

// ─── Read: error paths ────────────────────────────────────────────────────────

func TestRead_MissingObject_ReturnsError(t *testing.T) {
	s := newStore(t)
	_, err := s.Read("abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "object not found")
}

func TestRead_HashTooShort_ReturnsError(t *testing.T) {
	s := newStore(t)
	_, err := s.Read("ab")
	assert.Error(t, err)
}

// ─── WriteCompressed / ReadCompressed ─────────────────────────────────────────

func TestWriteCompressed_ReadCompressed_Roundtrip(t *testing.T) {
	s := newStore(t)
	original := []byte("compressed content for rewinddb — repeating repeating repeating")

	hash, err := s.WriteCompressed(original)
	require.NoError(t, err)
	assert.Equal(t, expectedHash(original), hash, "hash must reflect original content, not compressed bytes")

	got, err := s.ReadCompressed(hash)
	require.NoError(t, err)
	assert.Equal(t, original, got)
}

func TestWriteCompressed_StoredBytesAreSmallerForRepetitiveContent(t *testing.T) {
	s := newStore(t)
	// Repetitive content compresses well
	original := bytes.Repeat([]byte("abcdefgh"), 500)

	hash, err := s.WriteCompressed(original)
	require.NoError(t, err)

	path := filepath.Join(s.RootDir, hash[:2], hash[2:])
	info, err := os.Stat(path)
	require.NoError(t, err)

	assert.Less(t, info.Size(), int64(len(original)),
		"compressed object must be smaller than original for repetitive data")
}

func TestWriteCompressed_Deduplication(t *testing.T) {
	s := newStore(t)
	data := []byte("compress me twice")

	h1, err := s.WriteCompressed(data)
	require.NoError(t, err)
	h2, err := s.WriteCompressed(data)
	require.NoError(t, err)

	assert.Equal(t, h1, h2)
	count, _, err := s.Stats()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// ─── WriteFile ────────────────────────────────────────────────────────────────

func TestWriteFile_Roundtrip(t *testing.T) {
	s := newStore(t)

	// Write a real temp file
	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "test.txt")
	fileContent := []byte("file content for WriteFile test")
	require.NoError(t, os.WriteFile(fpath, fileContent, 0o644))

	hash, err := s.WriteFile(fpath)
	require.NoError(t, err)
	assert.Equal(t, expectedHash(fileContent), hash)

	got, err := s.Read(hash)
	require.NoError(t, err)
	assert.Equal(t, fileContent, got)
}

func TestWriteFile_MissingFile_ReturnsError(t *testing.T) {
	s := newStore(t)
	_, err := s.WriteFile("/nonexistent/path/to/file.txt")
	assert.Error(t, err)
}

// ─── Stats ────────────────────────────────────────────────────────────────────

func TestStats_EmptyStore(t *testing.T) {
	s := newStore(t)
	count, total, err := s.Stats()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, int64(0), total)
}

func TestStats_AfterMultipleWrites(t *testing.T) {
	s := newStore(t)
	payloads := [][]byte{
		[]byte("object one"),
		[]byte("object two"),
		[]byte("object three"),
	}
	for _, p := range payloads {
		_, err := s.Write(p)
		require.NoError(t, err)
	}

	count, total, err := s.Stats()
	require.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.Greater(t, total, int64(0))
}

// ─── Concurrent writes ────────────────────────────────────────────────────────

func TestWrite_ConcurrentSameContent_NoRaceNoDuplication(t *testing.T) {
	s := newStore(t)
	content := []byte("concurrent write target")
	var wg sync.WaitGroup
	const goroutines = 50

	errs := make(chan error, goroutines)
	hashes := make(chan string, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h, err := s.Write(content)
			if err != nil {
				errs <- err
				return
			}
			hashes <- h
		}()
	}
	wg.Wait()
	close(errs)
	close(hashes)

	for err := range errs {
		require.NoError(t, err)
	}

	// All goroutines must return the same hash
	var first string
	for h := range hashes {
		if first == "" {
			first = h
		}
		assert.Equal(t, first, h)
	}

	// Exactly one object must be on disk
	count, _, err := s.Stats()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestWrite_ConcurrentDifferentContent_AllObjectsStored(t *testing.T) {
	s := newStore(t)
	const goroutines = 20
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := s.Write([]byte(fmt.Sprintf("unique object %d", n)))
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	count, _, err := s.Stats()
	require.NoError(t, err)
	assert.Equal(t, goroutines, count)
}
