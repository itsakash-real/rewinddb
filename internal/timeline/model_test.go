package timeline_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/itsakash-real/rewinddb/internal/timeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── NewCheckpoint ────────────────────────────────────────────────────────────

func TestNewCheckpoint_FieldsPopulated(t *testing.T) {
	before := time.Now().UTC()
	cp := timeline.NewCheckpoint("initial state", "", "branch-abc", "sha256-xyz")
	after := time.Now().UTC()

	assert.NotEmpty(t, cp.ID, "ID must be auto-generated")
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, cp.ID, "ID must be UUID v4")
	assert.Equal(t, "", cp.ParentID, "root checkpoint has empty ParentID")
	assert.Equal(t, "branch-abc", cp.BranchID)
	assert.Equal(t, "sha256-xyz", cp.SnapshotRef)
	assert.Equal(t, "initial state", cp.Message)
	assert.False(t, cp.CreatedAt.Before(before), "CreatedAt must not be before test start")
	assert.False(t, cp.CreatedAt.After(after), "CreatedAt must not be after test end")
	assert.NotNil(t, cp.Tags, "Tags slice must not be nil")
}

func TestNewCheckpoint_UniqueIDs(t *testing.T) {
	cp1 := timeline.NewCheckpoint("a", "", "b", "c")
	cp2 := timeline.NewCheckpoint("a", "", "b", "c")
	assert.NotEqual(t, cp1.ID, cp2.ID, "each checkpoint must have a unique UUID")
}

func TestNewCheckpoint_WithParent(t *testing.T) {
	root := timeline.NewCheckpoint("root", "", "branch-1", "hash-1")
	child := timeline.NewCheckpoint("child", root.ID, "branch-1", "hash-2")

	assert.Equal(t, root.ID, child.ParentID)
}

// ─── NewBranch ────────────────────────────────────────────────────────────────

func TestNewBranch_FieldsPopulated(t *testing.T) {
	before := time.Now().UTC()
	b := timeline.NewBranch("main", "cp-root-001")
	after := time.Now().UTC()

	assert.NotEmpty(t, b.ID)
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, b.ID)
	assert.Equal(t, "main", b.Name)
	assert.Equal(t, "cp-root-001", b.RootCheckpointID)
	assert.Equal(t, "cp-root-001", b.HeadCheckpointID, "Head starts at root")
	assert.False(t, b.CreatedAt.Before(before))
	assert.False(t, b.CreatedAt.After(after))
}

func TestNewBranch_UniqueIDs(t *testing.T) {
	b1 := timeline.NewBranch("main", "cp-1")
	b2 := timeline.NewBranch("main", "cp-1")
	assert.NotEqual(t, b1.ID, b2.ID)
}

// ─── Index: Save / Load roundtrip ─────────────────────────────────────────────

func TestIndex_SaveLoad_Roundtrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "index.json")

	// Build a populated index
	idx := timeline.NewIndex()
	cp := timeline.NewCheckpoint("first save", "", "branch-1", "hash-aaa")
	b := timeline.NewBranch("main", cp.ID)
	idx.AddBranch(b)
	idx.AddCheckpoint(cp)

	require.NoError(t, idx.Save(path))

	// Reload from disk
	loaded, err := timeline.Load(path)
	require.NoError(t, err)

	assert.Equal(t, idx.CurrentBranchID, loaded.CurrentBranchID)
	assert.Equal(t, idx.CurrentCheckpointID, loaded.CurrentCheckpointID)
	assert.Equal(t, len(idx.Branches), len(loaded.Branches))
	assert.Equal(t, len(idx.Checkpoints), len(loaded.Checkpoints))

	// Verify checkpoint integrity
	lcp, ok := loaded.Checkpoints[cp.ID]
	require.True(t, ok)
	assert.Equal(t, cp.Message, lcp.Message)
	assert.Equal(t, cp.BranchID, lcp.BranchID)
	assert.WithinDuration(t, cp.CreatedAt, lcp.CreatedAt, time.Second)
}

func TestIndex_Load_NilMapsAfterEmptyJSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "index.json")

	// Write a minimal JSON without the map keys
	require.NoError(t, os.WriteFile(path, []byte(`{}`), 0o644))

	loaded, err := timeline.Load(path)
	require.NoError(t, err)
	assert.NotNil(t, loaded.Branches, "Branches map must not be nil")
	assert.NotNil(t, loaded.Checkpoints, "Checkpoints map must not be nil")
}

func TestIndex_Save_IsValidJSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "index.json")

	idx := timeline.NewIndex()
	require.NoError(t, idx.Save(path))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var generic map[string]any
	assert.NoError(t, json.Unmarshal(raw, &generic), "saved file must be valid JSON")
}

// ─── Index: CurrentBranch / CurrentCheckpoint helpers ─────────────────────────

func TestIndex_CurrentBranch_Found(t *testing.T) {
	idx := timeline.NewIndex()
	b := timeline.NewBranch("feature", "cp-0")
	idx.AddBranch(b)

	got, ok := idx.CurrentBranch()
	require.True(t, ok)
	assert.Equal(t, "feature", got.Name)
}

func TestIndex_CurrentBranch_NotFound(t *testing.T) {
	idx := timeline.NewIndex()
	_, ok := idx.CurrentBranch()
	assert.False(t, ok, "empty index has no current branch")
}

func TestIndex_CurrentCheckpoint_NotFoundOnFreshIndex(t *testing.T) {
	idx := timeline.NewIndex()
	_, ok := idx.CurrentCheckpoint()
	assert.False(t, ok)
}

func TestIndex_CurrentCheckpoint_Found(t *testing.T) {
	idx := timeline.NewIndex()
	cp := timeline.NewCheckpoint("snap", "", "b1", "hash-x")
	b := timeline.NewBranch("main", cp.ID)
	idx.AddBranch(b)
	idx.AddCheckpoint(cp)

	got, ok := idx.CurrentCheckpoint()
	require.True(t, ok)
	assert.Equal(t, cp.ID, got.ID)
}

// ─── AddCheckpoint: Head advances ─────────────────────────────────────────────

func TestIndex_AddCheckpoint_AdvancesHead(t *testing.T) {
	idx := timeline.NewIndex()
	cp1 := timeline.NewCheckpoint("first", "", "branch-x", "h1")
	b := timeline.NewBranch("main", cp1.ID)
	idx.AddBranch(b)
	idx.AddCheckpoint(cp1)

	cp2 := timeline.NewCheckpoint("second", cp1.ID, b.ID, "h2")
	idx.AddCheckpoint(cp2)

	branch, ok := idx.CurrentBranch()
	require.True(t, ok)
	assert.Equal(t, cp2.ID, branch.HeadCheckpointID, "Head must advance to latest checkpoint")
}

// ─── FileEntry: JSON roundtrip with fs.FileMode ────────────────────────────────

func TestFileEntry_JSONRoundtrip(t *testing.T) {
	entry := timeline.FileEntry{
		Path: "cmd/rw/main.go",
		Hash: "abc123",
		Size: 4096,
		Mode: fs.FileMode(0o644),
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var decoded timeline.FileEntry
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, entry.Path, decoded.Path)
	assert.Equal(t, entry.Hash, decoded.Hash)
	assert.Equal(t, entry.Size, decoded.Size)
	assert.Equal(t, entry.Mode, decoded.Mode)
}
