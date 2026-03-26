package timeline

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/itsakash-real/rewinddb/internal/config"
	"github.com/rs/zerolog/log"
)

// Word lists for readable auto-generated branch names.
var branchAdjectives = []string{
	"swift", "quiet", "broken", "golden", "hidden", "frozen", "bright",
	"calm", "bold", "dark", "eager", "gentle", "keen", "rapid", "wild",
	"warm", "cool", "pale", "deep", "stark", "crisp", "faint", "sharp",
	"lost", "rare", "slow", "loud", "soft", "thin", "vast",
}

var branchNouns = []string{
	"river", "sunset", "falcon", "meadow", "canyon", "breeze", "summit",
	"forest", "tide", "flame", "spark", "stone", "ember", "ridge", "creek",
	"frost", "bloom", "cloud", "drift", "grove", "marsh", "brook", "cliff",
	"dawn", "dune", "leaf", "peak", "pine", "reef", "vale",
}

// generateBranchName creates a human-readable branch name like "swift-river-mar-9pm".
func generateBranchName() string {
	now := time.Now()
	adj := branchAdjectives[rand.Intn(len(branchAdjectives))]
	noun := branchNouns[rand.Intn(len(branchNouns))]
	ts := strings.ToLower(now.Format("Jan-3pm")) // e.g. "mar-9pm"
	return fmt.Sprintf("%s-%s-%s", adj, noun, ts)
}

// ─── Sentinel errors ──────────────────────────────────────────────────────────

var (
	ErrAlreadyInitialized = errors.New("timeline: repository already initialized")
	ErrCheckpointNotFound = errors.New("timeline: checkpoint not found")
	ErrBranchNotFound     = errors.New("timeline: branch not found")
	ErrNoCommonAncestor   = errors.New("timeline: no common ancestor found")
	ErrEmptyTimeline      = errors.New("timeline: no checkpoints on branch")
)

// ─── TimelineEngine ───────────────────────────────────────────────────────────

// TimelineEngine is the core DAG engine. It manages checkpoints, branches,
// and the persistent index. All mutating operations call persistIndex() before
// returning so the on-disk state is always consistent.
type TimelineEngine struct {
	Index     *Index
	IndexPath string
}

// New loads an existing index from indexPath and returns a ready engine.
func New(indexPath string) (*TimelineEngine, error) {
	idx, err := Load(indexPath)
	if err != nil {
		return nil, fmt.Errorf("timeline.New: %w", err)
	}
	return &TimelineEngine{Index: idx, IndexPath: indexPath}, nil
}

// ─── Init ─────────────────────────────────────────────────────────────────────

// Init bootstraps a brand-new RewindDB repository under projectRoot.
// It creates the .rewind/ directory layout, a "main" branch, a synthetic
// root checkpoint, and writes the initial index.json.
// Returns ErrAlreadyInitialized if the repository already exists.
func Init(projectRoot string) (*TimelineEngine, error) {
	cfg, err := config.Init()
	if err != nil {
		// config.Init returns an error when the directory already exists.
		if errors.Is(err, config.ErrNotInitialized) {
			return nil, err
		}
		// "already exists" error from config.Init
		return nil, fmt.Errorf("%w: %s", ErrAlreadyInitialized, projectRoot)
	}

	idx := NewIndex()

	// Create the main branch first so we have its ID.
	main := NewBranch("main", "") // root CP doesn't exist yet; set below

	// Create the synthetic root checkpoint (no snapshot yet, no parent).
	root := NewCheckpoint("(root)", "", main.ID, "")
	root.Tags = []string{"root"}

	// Now set the branch's root/head to the actual root checkpoint.
	main.RootCheckpointID = root.ID
	main.HeadCheckpointID = root.ID

	idx.AddBranch(main)
	idx.AddCheckpoint(root)
	// CurrentCheckpointID is set by AddCheckpoint; CurrentBranchID by AddBranch.

	engine := &TimelineEngine{Index: idx, IndexPath: cfg.IndexPath}
	if err := engine.persistIndex(); err != nil {
		return nil, fmt.Errorf("timeline.Init: persist: %w", err)
	}

	log.Info().
		Str("branch", main.ID).
		Str("root_checkpoint", root.ID).
		Msg("repository initialized")

	return engine, nil
}

// ─── SaveCheckpoint ───────────────────────────────────────────────────────────

// SaveCheckpoint creates a new Checkpoint on the current branch (or forks a
// new branch if the HEAD has been rewound via GotoCheckpoint).
//
// Branching logic:
//   - Normal case (HEAD == branch head): append to current branch.
//   - Detached HEAD (HEAD != branch head): fork a new "branch-<ts>" branch
//     whose root is the new checkpoint.
func (e *TimelineEngine) SaveCheckpoint(message, snapshotRef string) (*Checkpoint, error) {
	branch, ok := e.Index.CurrentBranch()
	if !ok {
		return nil, ErrBranchNotFound
	}

	currentCPID := e.Index.CurrentCheckpointID

	cp := NewCheckpoint(message, currentCPID, branch.ID, snapshotRef)

	if currentCPID == branch.HeadCheckpointID {
		// ── Normal append ────────────────────────────────────────────────────
		cp.BranchID = branch.ID
		e.Index.AddCheckpoint(cp) // advances HeadCheckpointID
		log.Debug().
			Str("branch", branch.Name).
			Str("checkpoint", cp.ID).
			Msg("appended checkpoint to branch head")
	} else {
		// ── Detached HEAD: fork a new branch ─────────────────────────────────
		forkName := generateBranchName()
		newBranch := NewBranch(forkName, cp.ID)
		cp.BranchID = newBranch.ID
		cp.ParentID = currentCPID // preserve lineage across the fork

		e.Index.AddBranch(newBranch) // sets CurrentBranchID = newBranch.ID
		e.Index.AddCheckpoint(cp)    // sets CurrentCheckpointID = cp.ID

		log.Info().
			Str("new_branch", newBranch.Name).
			Str("forked_from", currentCPID).
			Str("checkpoint", cp.ID).
			Msg("forked new branch (detached HEAD)")
	}

	if err := e.persistIndex(); err != nil {
		return nil, fmt.Errorf("timeline.SaveCheckpoint: persist: %w", err)
	}
	return &cp, nil
}

// ─── GotoCheckpoint ───────────────────────────────────────────────────────────

// GotoCheckpoint updates the index so the engine points at checkpointID.
// It locates the owning branch (the one whose ancestry chain contains the
// checkpoint). The caller is responsible for physically restoring files.
func (e *TimelineEngine) GotoCheckpoint(checkpointID string) (*Checkpoint, error) {
	cp, ok := e.Index.Checkpoints[checkpointID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrCheckpointNotFound, checkpointID)
	}

	// Update index state: HEAD moves to this checkpoint + its owning branch.
	e.Index.CurrentCheckpointID = cp.ID
	e.Index.CurrentBranchID = cp.BranchID

	if err := e.persistIndex(); err != nil {
		return nil, fmt.Errorf("timeline.GotoCheckpoint: persist: %w", err)
	}

	log.Info().
		Str("checkpoint", cp.ID).
		Str("branch", cp.BranchID).
		Msg("HEAD moved to checkpoint")

	return &cp, nil
}

// ─── ListCheckpoints ──────────────────────────────────────────────────────────

// ListCheckpoints walks the ParentID chain from the branch head back to the
// root, returning checkpoints newest-first. If branchID is empty the current
// branch is used.
//
// The walk stops when ParentID is empty (root) or the parent is not found
// (cross-branch reference, treated as the end of this branch's history).
func (e *TimelineEngine) ListCheckpoints(branchID string) ([]*Checkpoint, error) {
	if branchID == "" {
		branchID = e.Index.CurrentBranchID
	}

	branch, ok := e.Index.Branches[branchID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrBranchNotFound, branchID)
	}

	if branch.HeadCheckpointID == "" {
		return nil, ErrEmptyTimeline
	}

	var result []*Checkpoint
	visited := make(map[string]struct{})
	curID := branch.HeadCheckpointID

	for curID != "" {
		if _, seen := visited[curID]; seen {
			break // cycle guard (should never happen in a well-formed DAG)
		}
		visited[curID] = struct{}{}

		cp, ok := e.Index.Checkpoints[curID]
		if !ok {
			break
		}
		cpCopy := cp
		result = append(result, &cpCopy)
		curID = cp.ParentID
	}

	return result, nil
}

// ─── GetAncestors ─────────────────────────────────────────────────────────────

// GetAncestors returns the full ancestor chain of checkpointID, including
// cross-branch parents, ordered from immediate parent to root. The checkpoint
// itself is not included in the result.
func (e *TimelineEngine) GetAncestors(checkpointID string) ([]*Checkpoint, error) {
	if _, ok := e.Index.Checkpoints[checkpointID]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrCheckpointNotFound, checkpointID)
	}

	var ancestors []*Checkpoint
	visited := make(map[string]struct{})

	curID := e.Index.Checkpoints[checkpointID].ParentID
	for curID != "" {
		if _, seen := visited[curID]; seen {
			break
		}
		visited[curID] = struct{}{}

		cp, ok := e.Index.Checkpoints[curID]
		if !ok {
			break
		}
		cpCopy := cp
		ancestors = append(ancestors, &cpCopy)
		curID = cp.ParentID
	}

	return ancestors, nil
}

// ─── FindCommonAncestor ───────────────────────────────────────────────────────

// FindCommonAncestor locates the Lowest Common Ancestor (LCA) of id1 and id2
// using a two-pointer ancestry set intersection [web:75].
//
// Algorithm:
//  1. Collect the full ancestor set of id1 (including id1 itself).
//  2. Walk the ancestor chain of id2 (including id2 itself) and return the
//     first node that appears in the ancestor set of id1.
//
// This is O(d1 + d2) where d1, d2 are the depths of the two nodes.
func (e *TimelineEngine) FindCommonAncestor(id1, id2 string) (*Checkpoint, error) {
	if _, ok := e.Index.Checkpoints[id1]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrCheckpointNotFound, id1)
	}
	if _, ok := e.Index.Checkpoints[id2]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrCheckpointNotFound, id2)
	}

	// Build ancestor set for id1 (inclusive of id1 itself).
	ancestors1 := make(map[string]struct{})
	for cur := id1; cur != ""; {
		ancestors1[cur] = struct{}{}
		cp, ok := e.Index.Checkpoints[cur]
		if !ok {
			break
		}
		cur = cp.ParentID
	}

	// Walk id2's ancestry and find first match in ancestors1 [web:62].
	for cur := id2; cur != ""; {
		if _, found := ancestors1[cur]; found {
			cp := e.Index.Checkpoints[cur]
			return &cp, nil
		}
		cp, ok := e.Index.Checkpoints[cur]
		if !ok {
			break
		}
		cur = cp.ParentID
	}

	return nil, ErrNoCommonAncestor
}

// ─── GetDAG ───────────────────────────────────────────────────────────────────

// GetDAG returns all checkpoints in the index grouped by branch ID.
// Each slice is ordered newest-first (via ListCheckpoints).
// Orphaned checkpoints (no matching branch) are collected under the key "".
func (e *TimelineEngine) GetDAG() map[string][]*Checkpoint {
	dag := make(map[string][]*Checkpoint)

	for branchID := range e.Index.Branches {
		cps, err := e.ListCheckpoints(branchID)
		if err != nil {
			continue
		}
		dag[branchID] = cps
	}

	return dag
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (e *TimelineEngine) persistIndex() error {
	return e.Index.Save(e.IndexPath)
}

// ForceFlush writes the current in-memory index to disk immediately.
// Use this after directly mutating Index fields (e.g. CurrentBranchID) that
// are not covered by a dedicated mutating method.
func (e *TimelineEngine) ForceFlush() error {
	return e.persistIndex()
}
