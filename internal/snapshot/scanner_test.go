package snapshot_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/itsakash-real/rewinddb/internal/snapshot"
	"github.com/itsakash-real/rewinddb/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// testEnv sets up a fresh project root and object store in temp directories.
func testEnv(t *testing.T) (projectRoot string, sc *snapshot.Scanner) {
	t.Helper()
	root := t.TempDir()
	storeDir := t.TempDir()
	store := storage.New(storeDir)
	sc = snapshot.New(root, store)
	return root, sc
}

// writeFile writes content to a relative path inside root, creating dirs.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
	require.NoError(t, os.WriteFile(abs, []byte(content), 0o644))
}

// readFile reads a relative path inside root.
func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	require.NoError(t, err)
	return string(data)
}

// seedProject writes 5 canonical files.
func seedProject(t *testing.T, root string) {
	t.Helper()
	writeFile(t, root, "main.go", `package main`)
	writeFile(t, root, "go.mod", `module example`)
	writeFile(t, root, "README.md", `# Hello`)
	writeFile(t, root, "internal/util.go", `package internal`)
	writeFile(t, root, "internal/util_test.go", `package internal_test`)
}

// ─── Scan ─────────────────────────────────────────────────────────────────────

func TestScan_ReturnsCorrectFileCount(t *testing.T) {
	root, sc := testEnv(t)
	seedProject(t, root)

	snap, err := sc.Scan()
	require.NoError(t, err)
	assert.Len(t, snap.Files, 5)
}

func TestScan_FilesSortedByPath(t *testing.T) {
	root, sc := testEnv(t)
	seedProject(t, root)

	snap, err := sc.Scan()
	require.NoError(t, err)

	for i := 1; i < len(snap.Files); i++ {
		assert.Less(t, snap.Files[i-1].Path, snap.Files[i].Path,
			"files must be sorted ascending by path")
	}
}

func TestScan_HashChangesWhenFileChanges(t *testing.T) {
	root, sc := testEnv(t)
	seedProject(t, root)

	snap1, err := sc.Scan()
	require.NoError(t, err)

	writeFile(t, root, "main.go", `package main // modified`)

	snap2, err := sc.Scan()
	require.NoError(t, err)

	assert.NotEqual(t, snap1.Hash, snap2.Hash)
}

func TestScan_HashStableForIdenticalContent(t *testing.T) {
	root, sc := testEnv(t)
	seedProject(t, root)

	snap1, err := sc.Scan()
	require.NoError(t, err)
	snap2, err := sc.Scan()
	require.NoError(t, err)

	assert.Equal(t, snap1.Hash, snap2.Hash, "same content must produce same snapshot hash")
}

func TestScan_IgnoresDefaultPatterns(t *testing.T) {
	root, sc := testEnv(t)
	seedProject(t, root)

	// Write files that must be ignored
	writeFile(t, root, ".rewind/index", `{}`)
	writeFile(t, root, "node_modules/lodash/index.js", `module.exports={}`)
	writeFile(t, root, "cache/app.pyc", `bytecode`)
	writeFile(t, root, "dist/bundle.js", `bundled`)
	writeFile(t, root, "app.exe", `binary`)

	snap, err := sc.Scan()
	require.NoError(t, err)

	paths := make(map[string]bool)
	for _, f := range snap.Files {
		paths[f.Path] = true
	}

	assert.False(t, paths[".rewind/index"], ".rewind should be ignored")
	assert.False(t, paths["node_modules/lodash/index.js"], "node_modules should be ignored")
	assert.False(t, paths["app.exe"], "*.exe should be ignored")
	assert.False(t, paths["dist/bundle.js"], "dist/ should be ignored")
	assert.Len(t, snap.Files, 5, "only the 5 seed files must appear")
}

func TestScan_EmptyDirectory(t *testing.T) {
	_, sc := testEnv(t)
	snap, err := sc.Scan()
	require.NoError(t, err)
	assert.Empty(t, snap.Files)
	assert.NotEmpty(t, snap.Hash, "even empty snapshot has a deterministic hash")
}

// ─── Save / Load ──────────────────────────────────────────────────────────────

func TestSave_Load_Roundtrip(t *testing.T) {
	root, sc := testEnv(t)
	seedProject(t, root)

	snap, err := sc.Scan()
	require.NoError(t, err)

	snapshotHash, err := sc.Save(snap)
	require.NoError(t, err)
	assert.Equal(t, snap.Hash, snapshotHash)

	loaded, err := sc.Load(snapshotHash)
	require.NoError(t, err)

	assert.Equal(t, snap.Hash, loaded.Hash)
	assert.Len(t, loaded.Files, len(snap.Files))
	for i, f := range snap.Files {
		assert.Equal(t, f.Path, loaded.Files[i].Path)
		assert.Equal(t, f.Hash, loaded.Files[i].Hash)
		assert.Equal(t, f.Size, loaded.Files[i].Size)
	}
}

func TestLoad_UnknownHash_ReturnsError(t *testing.T) {
	_, sc := testEnv(t)
	_, err := sc.Load("0000000000000000000000000000000000000000000000000000000000000000")
	assert.Error(t, err)
}

// ─── Full roundtrip: Scan → Save → Modify → Restore → Verify ──────────────────

func TestRestore_MatchesOriginalExactly(t *testing.T) {
	root, sc := testEnv(t)
	seedProject(t, root)

	// Step 1: scan and save the original state
	original, err := sc.Scan()
	require.NoError(t, err)
	_, err = sc.Save(original)
	require.NoError(t, err)

	// Step 2: mutate the project — modify 2 files, add 1 new file
	writeFile(t, root, "main.go", `package main // MUTATED`)
	writeFile(t, root, "README.md", `# Mutated README`)
	writeFile(t, root, "extra_file.go", `package extra`)

	// Verify mutation took effect
	assert.Equal(t, `package main // MUTATED`, readFile(t, root, "main.go"))
	assert.FileExists(t, filepath.Join(root, "extra_file.go"))

	// Step 3: restore to original snapshot
	require.NoError(t, sc.Restore(original))

	// Step 4: verify every file matches the original exactly
	for _, entry := range original.Files {
		got := readFile(t, root, entry.Path)
		content, err := sc.StoreRead(entry.Hash)
		require.NoError(t, err, "object for %s must exist in store", entry.Path)
		assert.Equal(t, string(content), got,
			"file %s must match original content after restore", entry.Path)
	}

	// Step 5: verify the added file was deleted
	assert.NoFileExists(t, filepath.Join(root, "extra_file.go"),
		"extra file added after snapshot must be removed by restore")

	// Step 6: re-scan and compare hashes for bulletproof verification
	restored, err := sc.Scan()
	require.NoError(t, err)
	assert.Equal(t, original.Hash, restored.Hash,
		"snapshot hash of restored directory must equal original snapshot hash")
}

func TestRestore_PreservesFileModes(t *testing.T) {
	root, sc := testEnv(t)
	writeFile(t, root, "script.sh", `#!/bin/bash`)
	require.NoError(t, os.Chmod(filepath.Join(root, "script.sh"), 0o755))

	snap, err := sc.Scan()
	require.NoError(t, err)
	_, err = sc.Save(snap)
	require.NoError(t, err)

	// Overwrite with wrong permissions
	writeFile(t, root, "script.sh", `#!/bin/bash`)
	require.NoError(t, os.Chmod(filepath.Join(root, "script.sh"), 0o600))

	require.NoError(t, sc.Restore(snap))

	info, err := os.Stat(filepath.Join(root, "script.sh"))
	require.NoError(t, err)
	assert.Equal(t, snap.Files[0].Mode.Perm(), info.Mode().Perm(),
		"restore must reinstate original file permissions")
}

func TestRestore_CreatesSubdirectories(t *testing.T) {
	root, sc := testEnv(t)
	writeFile(t, root, "a/b/c/deep.go", `package deep`)

	snap, err := sc.Scan()
	require.NoError(t, err)
	_, err = sc.Save(snap)
	require.NoError(t, err)

	// Delete the entire subtree
	require.NoError(t, os.RemoveAll(filepath.Join(root, "a")))

	require.NoError(t, sc.Restore(snap))
	assert.FileExists(t, filepath.Join(root, "a", "b", "c", "deep.go"))
}

// ─── Custom ignores ───────────────────────────────────────────────────────────

func TestScanner_CustomIgnore(t *testing.T) {
	root, sc := testEnv(t)
	writeFile(t, root, ".rewindignore", "*.log\n")

	writeFile(t, root, "app.go", `package main`)
	writeFile(t, root, "debug.log", `log output`)

	snap, err := sc.Scan()
	require.NoError(t, err)

	paths := make(map[string]bool)
	for _, f := range snap.Files {
		paths[f.Path] = true
	}
	assert.True(t, paths["app.go"])
	assert.False(t, paths["debug.log"], "*.log must be ignored")
}
