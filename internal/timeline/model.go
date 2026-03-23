package timeline

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/google/uuid"
)

// ─── Core Types ──────────────────────────────────────────────────────────────

// Checkpoint is a single point-in-time node in the snapshot DAG.
// It is analogous to a Git commit.
type Checkpoint struct {
	ID          string    `json:"id"`
	ParentID    string    `json:"parent_id,omitempty"`
	BranchID    string    `json:"branch_id"`
	SnapshotRef string    `json:"snapshot_ref"` // SHA-256 of the Snapshot
	Message     string    `json:"message"`
	CreatedAt   time.Time `json:"created_at"`
	Tags        []string  `json:"tags,omitempty"`
}

// Branch is a named pointer to a linear chain within the DAG.
// HeadCheckpointID advances with every new save on this branch.
type Branch struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	RootCheckpointID  string    `json:"root_checkpoint_id"`
	HeadCheckpointID  string    `json:"head_checkpoint_id"`
	CreatedAt         time.Time `json:"created_at"`
}

// Snapshot is a complete picture of the filesystem at a point in time.
// Its Hash is the SHA-256 of all sorted FileEntry hashes concatenated.
type Snapshot struct {
	Hash      string      `json:"hash"`
	Files     []FileEntry `json:"files"`
	CreatedAt time.Time   `json:"created_at"`
}

// FileEntry records the state of a single file within a Snapshot.
type FileEntry struct {
	Path string      `json:"path"` // relative to project root
	Hash string      `json:"hash"` // SHA-256 of file content
	Size int64       `json:"size"`
	Mode fs.FileMode `json:"mode"`
}

// Index is the live state persisted to .rewind/index.json.
// It tracks which branch and checkpoint are currently active.
type Index struct {
	CurrentBranchID     string                `json:"current_branch_id"`
	CurrentCheckpointID string                `json:"current_checkpoint_id,omitempty"`
	Branches            map[string]Branch     `json:"branches"`
	Checkpoints         map[string]Checkpoint `json:"checkpoints"`
}

// ─── Constructors ─────────────────────────────────────────────────────────────

// NewCheckpoint creates a Checkpoint with an auto-generated UUID v4 and a UTC
// timestamp. parentID may be empty for the root checkpoint of a branch.
func NewCheckpoint(message, parentID, branchID, snapshotRef string) Checkpoint {
	return Checkpoint{
		ID:          uuid.New().String(),
		ParentID:    parentID,
		BranchID:    branchID,
		SnapshotRef: snapshotRef,
		Message:     message,
		CreatedAt:   time.Now().UTC(),
		Tags:        []string{},
	}
}

// NewBranch creates a Branch with an auto-generated UUID v4. The HeadCheckpointID
// starts equal to rootCheckpointID and advances as new checkpoints are saved.
func NewBranch(name, rootCheckpointID string) Branch {
	return Branch{
		ID:               uuid.New().String(),
		Name:             name,
		RootCheckpointID: rootCheckpointID,
		HeadCheckpointID: rootCheckpointID,
		CreatedAt:        time.Now().UTC(),
	}
}

// NewIndex creates an empty, ready-to-use Index.
func NewIndex() *Index {
	return &Index{
		Branches:    make(map[string]Branch),
		Checkpoints: make(map[string]Checkpoint),
	}
}

// ─── Index: Persistence ───────────────────────────────────────────────────────

// Save serializes the Index to a JSON file at the given path.
// The file is written atomically via a temp file + rename to avoid corruption.
func (idx *Index) Save(path string) error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("index.Save: marshal failed: %w", err)
	}

	// Write to a temp file in the same directory, then atomically rename.
	dir := pathDir(path)
	tmp, err := os.CreateTemp(dir, ".index-*.tmp")
	if err != nil {
		return fmt.Errorf("index.Save: create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("index.Save: write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("index.Save: close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("index.Save: rename to %s: %w", path, err)
	}

	return nil
}

// Load deserializes an Index from a JSON file at the given path.
func Load(path string) (*Index, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("index.Load: read file: %w", err)
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("index.Load: unmarshal failed: %w", err)
	}

	// Ensure maps are never nil after loading from an empty/minimal JSON.
	if idx.Branches == nil {
		idx.Branches = make(map[string]Branch)
	}
	if idx.Checkpoints == nil {
		idx.Checkpoints = make(map[string]Checkpoint)
	}

	return &idx, nil
}

// ─── Index: Helpers ───────────────────────────────────────────────────────────

// CurrentBranch returns the active Branch and whether it was found.
func (idx *Index) CurrentBranch() (Branch, bool) {
	b, ok := idx.Branches[idx.CurrentBranchID]
	return b, ok
}

// CurrentCheckpoint returns the active Checkpoint and whether it was found.
// Returns false when no checkpoint has been saved yet (fresh branch).
func (idx *Index) CurrentCheckpoint() (Checkpoint, bool) {
	if idx.CurrentCheckpointID == "" {
		return Checkpoint{}, false
	}
	c, ok := idx.Checkpoints[idx.CurrentCheckpointID]
	return c, ok
}

// AddBranch inserts a Branch into the Index and sets it as the current branch.
func (idx *Index) AddBranch(b Branch) {
	idx.Branches[b.ID] = b
	idx.CurrentBranchID = b.ID
}

// AddCheckpoint inserts a Checkpoint into the Index, advances the parent
// branch's HeadCheckpointID, and sets it as the current checkpoint.
func (idx *Index) AddCheckpoint(c Checkpoint) {
	idx.Checkpoints[c.ID] = c
	idx.CurrentCheckpointID = c.ID

	if b, ok := idx.Branches[c.BranchID]; ok {
		b.HeadCheckpointID = c.ID
		idx.Branches[c.BranchID] = b
	}
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// pathDir returns the directory of a file path without importing path/filepath
// at top level just for this one call.
func pathDir(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[:i]
		}
	}
	return "."
}
