// Package merkle implements a binary Merkle tree over the RewindDB object
// store. The tree provides integrity verification: any single-byte corruption
// in any stored object changes the root hash, and a bisect over the tree
// identifies the exact object that was modified.
//
// Tree construction:
//  1. Collect all object hashes from the store, sort them lexicographically.
//  2. Pair adjacent hashes and hash each pair: H(left||right).
//     If the level has an odd count, the last hash is promoted unchanged.
//  3. Repeat until a single root hash remains.
//
// The root is stored in plain text at <rewindDir>/MERKLE.
package merkle

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const FileName = "MERKLE"

// Build computes the Merkle root from an already-sorted slice of hex hashes.
// Returns an empty string if hashes is empty.
func Build(hashes []string) string {
	if len(hashes) == 0 {
		return ""
	}

	level := make([]string, len(hashes))
	copy(level, hashes)

	for len(level) > 1 {
		var next []string
		for i := 0; i < len(level); i += 2 {
			if i+1 < len(level) {
				combined := level[i] + level[i+1]
				h := sha256.Sum256([]byte(combined))
				next = append(next, hex.EncodeToString(h[:]))
			} else {
				// Odd node out: promote unchanged.
				next = append(next, level[i])
			}
		}
		level = next
	}

	return level[0]
}

// Compute walks the object store at objectsDir, collects all object hashes
// (derived from file names, not by re-reading content), sorts them, and
// returns the Merkle root plus the sorted hash slice.
func Compute(objectsDir string) (root string, hashes []string, err error) {
	err = filepath.WalkDir(objectsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		// Object path layout: <objectsDir>/<2-char shard>/<62-char remainder>
		// The full hash = shard dir name + file name.
		shard := filepath.Base(filepath.Dir(path))
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			return nil // skip temp files
		}
		hashes = append(hashes, shard+name)
		return nil
	})
	if err != nil {
		return "", nil, fmt.Errorf("merkle.Compute: walk: %w", err)
	}

	sort.Strings(hashes)
	root = Build(hashes)
	return root, hashes, nil
}

// SaveRoot atomically writes the Merkle root to <rewindDir>/MERKLE.
func SaveRoot(rewindDir, root string) error {
	path := filepath.Join(rewindDir, FileName)
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".merkle-*.tmp")
	if err != nil {
		return fmt.Errorf("merkle.SaveRoot: create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := fmt.Fprintln(tmp, root); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("merkle.SaveRoot: write: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("merkle.SaveRoot: sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("merkle.SaveRoot: close: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("merkle.SaveRoot: rename: %w", err)
	}
	return nil
}

// LoadRoot reads the stored Merkle root from <rewindDir>/MERKLE.
// Returns an empty string (not an error) if the file doesn't exist yet.
func LoadRoot(rewindDir string) (string, error) {
	path := filepath.Join(rewindDir, FileName)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("merkle.LoadRoot: open: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	return "", nil
}

// Verify recomputes the Merkle root from the object store and compares it to
// the stored root. Returns (true, nil) if consistent, (false, nil) if not,
// or (false, err) if an I/O error occurred.
func Verify(rewindDir, objectsDir string) (bool, error) {
	stored, err := LoadRoot(rewindDir)
	if err != nil {
		return false, err
	}
	computed, _, err := Compute(objectsDir)
	if err != nil {
		return false, err
	}
	if stored == "" && computed == "" {
		return true, nil // empty repo — trivially consistent
	}
	return stored == computed, nil
}
