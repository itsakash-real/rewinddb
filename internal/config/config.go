package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

const (
	RewindDirName    = ".rewind"
	ObjectsDirName   = "objects"
	SnapshotsDirName = "snapshots"
	BranchesDirName  = "branches"
	IndexFileName    = "index.json"
)

// Config holds all resolved paths for a Nimbi repository.
type Config struct {
	// RewindDir is the root .rewind/ directory (analogous to .git/).
	RewindDir string

	// ObjectsDir is the content-addressable object store (.rewind/objects/).
	ObjectsDir string

	// SnapshotsDir stores serialized snapshot metadata (.rewind/snapshots/).
	SnapshotsDir string

	// BranchesDir stores branch pointer files (.rewind/branches/).
	BranchesDir string

	// IndexPath is the path to the staging index file (.rewind/index).
	IndexPath string
}

// Load locates an existing .rewind/ directory by walking up from cwd.
// Returns ErrNotInitialized if no repository is found.
func Load() (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("config: cannot determine working directory: %w", err)
	}
	return LoadFrom(cwd)
}

// LoadStrict checks only startDir itself for a .rewind/ directory (no upward
// walk). Returns ErrNotInitialized if startDir does not contain .rewind/.
// Useful in tests and tools that must not inherit a parent repo.
func LoadStrict(startDir string) (*Config, error) {
	candidate := filepath.Join(startDir, RewindDirName)
	if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
		log.Debug().Str("rewind_dir", candidate).Msg("found existing repository")
		return buildConfig(candidate), nil
	}
	return nil, ErrNotInitialized
}

// LoadFrom locates an existing .rewind/ directory by walking up from startDir.
// Returns ErrNotInitialized if no repository is found.
func LoadFrom(startDir string) (*Config, error) {
	dir := startDir
	for {
		candidate := filepath.Join(dir, RewindDirName)
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			log.Debug().Str("rewind_dir", candidate).Msg("found existing repository")
			return buildConfig(candidate), nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding .rewind/
			break
		}
		dir = parent
	}

	return nil, ErrNotInitialized
}

// Init creates a new .rewind/ directory structure in the current working
// directory. Returns an error if the repository already exists.
func Init() (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("config: cannot determine working directory: %w", err)
	}

	rewindDir := filepath.Join(cwd, RewindDirName)

	if _, err := os.Stat(rewindDir); err == nil {
		return nil, fmt.Errorf("config: repository already exists at %s", rewindDir)
	}

	cfg := buildConfig(rewindDir)

	dirs := []string{
		cfg.RewindDir,
		cfg.ObjectsDir,
		cfg.SnapshotsDir,
		cfg.BranchesDir,
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, fmt.Errorf("config: failed to create directory %s: %w", d, err)
		}
		log.Debug().Str("dir", d).Msg("created directory")
	}

	// Create an empty index file
	f, err := os.OpenFile(cfg.IndexPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return nil, fmt.Errorf("config: failed to create index file: %w", err)
	}
	f.Close()

	log.Info().Str("rewind_dir", rewindDir).Msg("initialized new Nimbi repository")
	return cfg, nil
}

// buildConfig constructs a Config from a known .rewind/ root path.
func buildConfig(rewindDir string) *Config {
	return &Config{
		RewindDir:    rewindDir,
		ObjectsDir:   filepath.Join(rewindDir, ObjectsDirName),
		SnapshotsDir: filepath.Join(rewindDir, SnapshotsDirName),
		BranchesDir:  filepath.Join(rewindDir, BranchesDirName),
		IndexPath:    filepath.Join(rewindDir, IndexFileName),
	}
}

// ErrNotInitialized is returned when no .rewind/ directory can be located.
var ErrNotInitialized = errors.New("not a Nimbi repository (no .rewind/ directory found)")
