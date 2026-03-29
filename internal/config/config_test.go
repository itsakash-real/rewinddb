package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/itsakash-real/nimbi/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_CreatesExpectedLayout(t *testing.T) {
	// Arrange: isolated temp directory as working directory
	tmp := t.TempDir()
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { os.Chdir(orig) })

	// Act
	cfg, err := config.Init()

	// Assert
	require.NoError(t, err)
	assert.DirExists(t, cfg.RewindDir)
	assert.DirExists(t, cfg.ObjectsDir)
	assert.DirExists(t, cfg.SnapshotsDir)
	assert.DirExists(t, cfg.BranchesDir)
	assert.FileExists(t, cfg.IndexPath)
}

func TestInit_FailsIfAlreadyInitialized(t *testing.T) {
	tmp := t.TempDir()
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { os.Chdir(orig) })

	_, err := config.Init()
	require.NoError(t, err)

	_, err = config.Init()
	assert.Error(t, err, "second Init should fail")
}

func TestLoad_FindsRepoInParentDirectory(t *testing.T) {
	tmp := t.TempDir()
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { os.Chdir(orig) })

	// Init at root of tmp
	_, err := config.Init()
	require.NoError(t, err)

	// Descend into a nested subdirectory
	nested := filepath.Join(tmp, "a", "b", "c")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.Chdir(nested))

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Contains(t, cfg.RewindDir, config.RewindDirName)
}

func TestLoad_ReturnsErrWhenNoRepo(t *testing.T) {
	// Use LoadFrom with an isolated temp dir to avoid walking into parent
	// directories that might contain a real .rewind/ repo on the test machine.
	tmp := t.TempDir()
	_, err := config.LoadStrict(tmp)
	assert.ErrorIs(t, err, config.ErrNotInitialized)
}
