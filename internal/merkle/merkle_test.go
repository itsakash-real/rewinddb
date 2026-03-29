package merkle_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/itsakash-real/nimbi/internal/merkle"
)

func TestBuild_Empty(t *testing.T) {
	root := merkle.Build(nil)
	if root != "" {
		t.Errorf("expected empty root for empty input, got %q", root)
	}
}

func TestBuild_Single(t *testing.T) {
	root := merkle.Build([]string{"abc"})
	if root != "abc" {
		t.Errorf("expected single hash to be promoted, got %q", root)
	}
}

func TestBuild_Deterministic(t *testing.T) {
	hashes := []string{"aaa", "bbb", "ccc", "ddd"}
	r1 := merkle.Build(hashes)
	r2 := merkle.Build(hashes)
	if r1 != r2 {
		t.Errorf("Build is not deterministic: %q vs %q", r1, r2)
	}
	if r1 == "" {
		t.Error("root should not be empty")
	}
}

func TestBuild_ChangeSensitive(t *testing.T) {
	h1 := merkle.Build([]string{"aaa", "bbb"})
	h2 := merkle.Build([]string{"aaa", "ccc"}) // one hash changed
	if h1 == h2 {
		t.Error("Merkle root should change when a leaf changes")
	}
}

func TestSaveAndLoadRoot(t *testing.T) {
	dir := t.TempDir()
	const root = "deadbeef1234"

	if err := merkle.SaveRoot(dir, root); err != nil {
		t.Fatalf("SaveRoot: %v", err)
	}

	loaded, err := merkle.LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot: %v", err)
	}
	if loaded != root {
		t.Errorf("expected %q, got %q", root, loaded)
	}
}

func TestLoadRoot_MissingFile(t *testing.T) {
	dir := t.TempDir()
	root, err := merkle.LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot on missing file: %v", err)
	}
	if root != "" {
		t.Errorf("expected empty string for missing MERKLE, got %q", root)
	}
}

func TestCompute_ObjectStore(t *testing.T) {
	// Simulate a tiny object store layout: <dir>/<2-char>/<rest>
	dir := t.TempDir()
	writeObj := func(hash string) {
		shard := filepath.Join(dir, hash[:2])
		if err := os.MkdirAll(shard, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(shard, hash[2:]), []byte("data"), 0o444); err != nil {
			t.Fatal(err)
		}
	}

	writeObj("aabbccdd1122334455667788990011223344556677889900112233445566778899aa")
	writeObj("bbccddee2233445566778899001122334455667788990011223344556677889900bb")

	root, hashes, err := merkle.Compute(dir)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(hashes) != 2 {
		t.Errorf("expected 2 hashes, got %d", len(hashes))
	}
	if root == "" {
		t.Error("root should not be empty")
	}
}
