//go:build integration
// +build integration

package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// generateLargeFile writes n bytes of patterned data to path.
// Using a deterministic pattern keeps tests reproducible across platforms
// without requiring crypto/rand for all files.
func generateLargeFile(t *testing.T, root, rel string, sizeBytes int) {
	t.Helper()
	data := make([]byte, sizeBytes)
	for i := range data {
		data[i] = byte(0x41 + (i % 52)) // A–Z a–z cycling
	}
	abs := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
	require.NoError(t, os.WriteFile(abs, data, 0o644))
}

// mustNotExist asserts that a path does not exist on disk.
func mustNotExist(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		t.Errorf("expected %q to not exist on disk, but it does (stat err: %v)", path, err)
	}
}

// countFilesUnder returns the number of regular files under dir.
func countFilesUnder(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			n++
		}
		return nil
	})
	require.NoError(t, err)
	return n
}

// writeBinaryFile writes random-looking binary content (non-UTF-8) to a file.
func writeBinaryFile(t *testing.T, root, rel string) {
	t.Helper()
	// Header bytes that look like a compiled binary.
	data := []byte{
		0x7f, 0x45, 0x4c, 0x46, // ELF magic
		0x02, 0x01, 0x01, 0x00,
	}
	for i := 8; i < 256; i++ {
		data = append(data, byte(i))
	}
	abs := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
	require.NoError(t, os.WriteFile(abs, data, 0o755))
}

// makeNFiles writes n files with unique content spread across subdirs.
func makeNFiles(t *testing.T, r *repo, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		subdir := fmt.Sprintf("gen/sub%03d", i/50)
		r.write(t, fmt.Sprintf("%s/gen_%04d.go", subdir, i),
			fmt.Sprintf("package gen\n\nconst Gen%04d = %d\n", i, i))
	}
}
