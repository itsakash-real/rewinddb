package timeline_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/itsakash-real/rewinddb/internal/timeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunRecovery_RemovesOrphanedTempObjects simulates an interrupted object
// write by leaving a .obj-*.tmp file in the objects shard directory.
func TestRunRecovery_RemovesOrphanedTempObjects(t *testing.T) {
	rewindDir := t.TempDir()
	objectsDir := filepath.Join(rewindDir, "objects", "ab")
	require.NoError(t, os.MkdirAll(objectsDir, 0o755))

	orphan := filepath.Join(objectsDir, ".obj-12345678.tmp")
	require.NoError(t, os.WriteFile(orphan, []byte("partial"), 0o644))

	require.NoError(t, timeline.RunRecovery(rewindDir))
	assert.NoFileExists(t, orphan, "orphaned tmp object must be removed by recovery")
}

// TestRunRecovery_RemovesCorruptObject writes an object whose filename hash
// does not match its content, simulating a crash after rename but with
// corrupted data (or a manually tampered file).
func TestRunRecovery_RemovesCorruptObject(t *testing.T) {
	rewindDir := t.TempDir()

	// Manufacture a correct-looking shard path with wrong content.
	// Use a known hash of "good content" but store "bad content".
	goodContent := []byte("good content")
	h := sha256.Sum256(goodContent)
	hash := hex.EncodeToString(h[:])

	shardDir := filepath.Join(rewindDir, "objects", hash[:2])
	require.NoError(t, os.MkdirAll(shardDir, 0o755))

	// Write corrupt bytes at the path that should contain "good content".
	objPath := filepath.Join(shardDir, hash[2:])
	require.NoError(t, os.WriteFile(objPath, []byte("bad content"), 0o444))

	require.NoError(t, timeline.RunRecovery(rewindDir))
	assert.NoFileExists(t, objPath, "corrupt object must be removed by recovery")
}

// TestRunRecovery_CompletesInterruptedIndexRename simulates a crash after
// temp file write but before os.Rename into index.json.
func TestRunRecovery_CompletesInterruptedIndexRename(t *testing.T) {
	rewindDir := t.TempDir()

	// No index.json exists yet — only a tmp.
	tmpIndex := filepath.Join(rewindDir, ".index-abcdef.tmp")
	validJSON := []byte(`{"current_branch_id":"b1","current_checkpoint_id":"c1","branches":{},"checkpoints":{}}`)
	require.NoError(t, os.WriteFile(tmpIndex, validJSON, 0o644))

	require.NoError(t, timeline.RunRecovery(rewindDir))

	indexPath := filepath.Join(rewindDir, "index.json")
	assert.FileExists(t, indexPath, "recovery must complete interrupted rename → index.json")
	assert.NoFileExists(t, tmpIndex, "tmp file must be gone after recovery")
}

// TestRunRecovery_DiscardsLeftoverTmpWhenIndexExists simulates a crash after
// os.Rename completed — both index.json and a leftover tmp exist.
func TestRunRecovery_DiscardsLeftoverTmpWhenIndexExists(t *testing.T) {
	rewindDir := t.TempDir()

	// index.json already present (rename succeeded before crash).
	indexPath := filepath.Join(rewindDir, "index.json")
	require.NoError(t, os.WriteFile(indexPath,
		[]byte(`{"current_branch_id":"","current_checkpoint_id":"","branches":{},"checkpoints":{}}`), 0o644))

	// Leftover tmp from the same operation.
	tmpIndex := filepath.Join(rewindDir, ".index-stale.tmp")
	require.NoError(t, os.WriteFile(tmpIndex, []byte("old"), 0o644))

	require.NoError(t, timeline.RunRecovery(rewindDir))

	assert.FileExists(t, indexPath, "index.json must be untouched")
	assert.NoFileExists(t, tmpIndex, "stale tmp must be discarded")
}

// TestRunRecovery_NoRewindDir_ReturnsNil ensures recovery is a no-op when
// called on a non-existent directory (fresh install).
func TestRunRecovery_NoObjectsDir_IsNoOp(t *testing.T) {
	rewindDir := t.TempDir()
	// No objects/ subdirectory at all.
	assert.NoError(t, timeline.RunRecovery(rewindDir))
}
