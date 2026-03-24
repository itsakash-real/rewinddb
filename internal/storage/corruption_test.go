package storage_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/itsakash-real/rewinddb/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRead_DetectsCorruptObject writes valid content then overwrites the stored
// file with garbage, simulating disk corruption or a partial write.
func TestRead_DetectsCorruptObject(t *testing.T) {
	dir := t.TempDir()
	s := storage.New(dir)

	content := []byte("valid object content for corruption test")
	hash, err := s.Write(content)
	require.NoError(t, err)

	// Corrupt the stored file by overwriting with different bytes.
	objPath := filepath.Join(dir, hash[:2], hash[2:])
	require.NoError(t, os.Chmod(objPath, 0o644))
	require.NoError(t, os.WriteFile(objPath, []byte("CORRUPTED GARBAGE"), 0o644))

	_, readErr := s.Read(hash)
	require.Error(t, readErr)
	assert.True(t, errors.Is(readErr, storage.ErrCorruptObject),
		"should return ErrCorruptObject, got: %v", readErr)
}

// TestRead_ValidObject_NoFalsePositive ensures a legitimately written object
// passes the checksum check.
func TestRead_ValidObject_NoFalsePositive(t *testing.T) {
	s := storage.New(t.TempDir())
	content := []byte("healthy object")
	hash, err := s.Write(content)
	require.NoError(t, err)

	got, err := s.Read(hash)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}
