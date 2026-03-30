// Package sdk provides a clean programmatic API over Nimbi internals.
// It is the recommended entry point for embedding Nimbi into Go applications.
package sdk

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/itsakash-real/nimbi/internal/config"
	diffpkg "github.com/itsakash-real/nimbi/internal/diff"
	"github.com/itsakash-real/nimbi/internal/snapshot"
	"github.com/itsakash-real/nimbi/internal/storage"
	"github.com/itsakash-real/nimbi/internal/timeline"
	"github.com/rs/zerolog/log"
)

// ─── Option / result types ────────────────────────────────────────────────────

// ListOpts controls the output of Client.List.
type ListOpts struct {
	// BranchID filters by branch. Empty string = current branch.
	BranchID string
	// Limit caps the number of results. 0 = no limit.
	Limit int
	// AllBranches, when true, returns checkpoints from every branch.
	AllBranches bool
}

// StorageStats holds aggregate object store metrics.
type StorageStats struct {
	ObjectCount int
	TotalBytes  int64
}

// StatusResult is the value returned by Client.Status.
type StatusResult struct {
	CurrentBranch  *timeline.Branch
	HeadCheckpoint *timeline.Checkpoint
	ModifiedFiles  []string
	AddedFiles     []string
	RemovedFiles   []string
	StorageStats   StorageStats
	IsClean        bool
}

// GCResult describes what the garbage collector did (or would do).
type GCResult struct {
	RemovedObjects int
	FreedBytes     int64
	DryRun         bool
	// CandidatePaths lists the object paths that were (or would be) deleted.
	CandidatePaths []string
}

// ─── Client ───────────────────────────────────────────────────────────────────

// Client is the top-level SDK handle. All methods are safe to call from a
// single goroutine; concurrent access requires external synchronisation.
type Client struct {
	ProjectRoot string

	cfg        *config.Config
	engine     *timeline.TimelineEngine
	scanner    *snapshot.Scanner
	store      *storage.ObjectStore
	diffEngine *diffpkg.Engine
}

// New loads an existing Nimbi repository rooted at projectRoot.
// Returns an error if the project has not been initialised (rw init).
func New(projectRoot string) (*Client, error) {
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("sdk.New: resolve root: %w", err)
	}

	// When the caller provides an explicit projectRoot, look only in that
	// directory (no upward walk). Use config.LoadFrom if you want parent search.
	cfg, err := config.LoadStrict(absRoot)
	if err != nil {
		if errors.Is(err, config.ErrNotInitialized) {
			return nil, fmt.Errorf("sdk.New: %s is not a Nimbi repository — call sdk.Init first", absRoot)
		}
		return nil, fmt.Errorf("sdk.New: load config: %w", err)
	}

	return buildClient(absRoot, cfg)
}

// MustNew is like New but panics on error.
// Suitable for top-level main() or test helpers where failure is fatal.
func MustNew(projectRoot string) *Client {
	c, err := New(projectRoot)
	if err != nil {
		panic(fmt.Sprintf("sdk.MustNew: %v", err))
	}
	return c
}

// Init initialises a new Nimbi repository at projectRoot and returns a
// ready Client. Returns an error if already initialised.
func Init(projectRoot string) (*Client, error) {
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("sdk.Init: resolve root: %w", err)
	}

	orig, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("sdk.Init: getwd: %w", err)
	}
	if err := os.Chdir(absRoot); err != nil {
		return nil, fmt.Errorf("sdk.Init: chdir: %w", err)
	}
	defer os.Chdir(orig)

	engine, err := timeline.Init(absRoot)
	if err != nil {
		return nil, fmt.Errorf("sdk.Init: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("sdk.Init: reload config: %w", err)
	}

	c, err := buildClient(absRoot, cfg)
	if err != nil {
		return nil, err
	}
	c.engine = engine
	return c, nil
}

// buildClient wires together the internal stack from a resolved config.
func buildClient(projectRoot string, cfg *config.Config) (*Client, error) {
	eng, err := timeline.New(cfg.IndexPath)
	if err != nil {
		return nil, fmt.Errorf("sdk: load timeline engine: %w", err)
	}

	store := storage.New(cfg.ObjectsDir)
	sc := snapshot.New(projectRoot, store)
	de := diffpkg.New(store)

	return &Client{
		ProjectRoot: projectRoot,
		cfg:         cfg,
		engine:      eng,
		scanner:     sc,
		store:       store,
		diffEngine:  de,
	}, nil
}

// ─── Save ─────────────────────────────────────────────────────────────────────

// Save scans the working directory and creates a new checkpoint with message.
func (c *Client) Save(message string) (*timeline.Checkpoint, error) {
	return c.SaveWithTags(message, nil)
}

// SaveWithTags is like Save but attaches one or more tag labels to the checkpoint.
func (c *Client) SaveWithTags(message string, tags []string) (*timeline.Checkpoint, error) {
	if strings.TrimSpace(message) == "" {
		return nil, fmt.Errorf("sdk.Save: message must not be empty")
	}

	snap, err := c.scanner.Scan()
	if err != nil {
		return nil, fmt.Errorf("sdk.Save: scan: %w", err)
	}

	snapshotHash, err := c.scanner.Save(snap)
	if err != nil {
		return nil, fmt.Errorf("sdk.Save: persist snapshot: %w", err)
	}

	cp, err := c.engine.SaveCheckpoint(message, snapshotHash)
	if err != nil {
		return nil, fmt.Errorf("sdk.Save: save checkpoint: %w", err)
	}

	if len(tags) > 0 {
		cp.Tags = append(cp.Tags, tags...)
		c.engine.Index.Checkpoints[cp.ID] = *cp
		if err := c.engine.Index.Save(c.cfg.IndexPath); err != nil {
			return nil, fmt.Errorf("sdk.Save: persist tags: %w", err)
		}
	}

	log.Debug().Str("id", cp.ID).Str("message", message).Msg("sdk: checkpoint saved")
	return cp, nil
}

// ─── Goto ─────────────────────────────────────────────────────────────────────

// Goto restores the working directory to the checkpoint identified by idOrAlias.
// idOrAlias accepts the same formats as the CLI: full UUID, 8-char prefix,
// tag name, "HEAD", or "HEAD~N".
func (c *Client) Goto(idOrAlias string) (*timeline.Checkpoint, error) {
	cp, err := c.resolveRef(idOrAlias)
	if err != nil {
		return nil, fmt.Errorf("sdk.Goto: %w", err)
	}

	if cp.SnapshotRef == "" {
		return nil, fmt.Errorf("sdk.Goto: checkpoint %s has no snapshot (root checkpoint)", cp.ID[:8])
	}

	snap, err := c.scanner.Load(cp.SnapshotRef)
	if err != nil {
		return nil, fmt.Errorf("sdk.Goto: load snapshot: %w", err)
	}

	if _, err := c.engine.GotoCheckpoint(cp.ID); err != nil {
		return nil, fmt.Errorf("sdk.Goto: update index: %w", err)
	}

	if err := c.scanner.Restore(snap); err != nil {
		return nil, fmt.Errorf("sdk.Goto: restore files: %w", err)
	}

	return &cp, nil
}

// ─── List ─────────────────────────────────────────────────────────────────────

// List returns checkpoints according to opts.
func (c *Client) List(opts ListOpts) ([]*timeline.Checkpoint, error) {
	if opts.AllBranches {
		var all []*timeline.Checkpoint
		for branchID := range c.engine.Index.Branches {
			cps, err := c.engine.ListCheckpoints(branchID)
			if err != nil {
				continue
			}
			all = append(all, cps...)
		}
		return applyLimit(all, opts.Limit), nil
	}

	cps, err := c.engine.ListCheckpoints(opts.BranchID)
	if err != nil {
		return nil, fmt.Errorf("sdk.List: %w", err)
	}
	return applyLimit(cps, opts.Limit), nil
}

// ─── Diff ─────────────────────────────────────────────────────────────────────

// Diff computes a file-level diff between two checkpoints. Either id may be
// any supported reference format. If id2 is empty, the current HEAD is used.
func (c *Client) Diff(id1, id2 string) (*diffpkg.DiffResult, error) {
	cp1, err := c.resolveRef(id1)
	if err != nil {
		return nil, fmt.Errorf("sdk.Diff: resolve id1: %w", err)
	}

	var cp2 timeline.Checkpoint
	if id2 == "" {
		cp2, err = c.resolveRef("HEAD")
	} else {
		cp2, err = c.resolveRef(id2)
	}
	if err != nil {
		return nil, fmt.Errorf("sdk.Diff: resolve id2: %w", err)
	}

	snap1, err := c.scanner.Load(cp1.SnapshotRef)
	if err != nil {
		return nil, fmt.Errorf("sdk.Diff: load snapshot 1: %w", err)
	}
	snap2, err := c.scanner.Load(cp2.SnapshotRef)
	if err != nil {
		return nil, fmt.Errorf("sdk.Diff: load snapshot 2: %w", err)
	}

	return c.diffEngine.Compare(snap1, snap2)
}

// ─── Status ───────────────────────────────────────────────────────────────────

// Status returns the current repository state including working-directory changes.
func (c *Client) Status() (*StatusResult, error) {
	result := &StatusResult{}

	// Current branch.
	if b, ok := c.engine.Index.CurrentBranch(); ok {
		bCopy := b
		result.CurrentBranch = &bCopy
	}

	// Current checkpoint.
	if cp, ok := c.engine.Index.CurrentCheckpoint(); ok {
		cpCopy := cp
		result.HeadCheckpoint = &cpCopy
	}

	// Working directory diff.
	currentSnap, err := c.scanner.Scan()
	if err != nil {
		return nil, fmt.Errorf("sdk.Status: scan: %w", err)
	}

	if result.HeadCheckpoint != nil && result.HeadCheckpoint.SnapshotRef != "" {
		prevSnap, err := c.scanner.Load(result.HeadCheckpoint.SnapshotRef)
		if err == nil {
			dr, err := c.diffEngine.Compare(prevSnap, currentSnap)
			if err == nil {
				for _, f := range dr.Added {
					result.AddedFiles = append(result.AddedFiles, f.Path)
				}
				for _, f := range dr.Removed {
					result.RemovedFiles = append(result.RemovedFiles, f.Path)
				}
				for _, fd := range dr.Modified {
					result.ModifiedFiles = append(result.ModifiedFiles, fd.Path)
				}
				result.IsClean = dr.TotalChanges() == 0
			}
		}
	}

	// Storage stats.
	count, bytes, err := c.store.Stats()
	if err != nil {
		return nil, fmt.Errorf("sdk.Status: storage stats: %w", err)
	}
	result.StorageStats = StorageStats{ObjectCount: count, TotalBytes: bytes}

	return result, nil
}

// ─── Branches ─────────────────────────────────────────────────────────────────

// Branches returns all branches sorted by creation time (oldest first).
func (c *Client) Branches() ([]*timeline.Branch, error) {
	var branches []*timeline.Branch
	for _, b := range c.engine.Index.Branches {
		bCopy := b
		branches = append(branches, &bCopy)
	}
	return branches, nil
}

// CreateBranch creates a new named branch at the current HEAD checkpoint.
func (c *Client) CreateBranch(name string) (*timeline.Branch, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("sdk.CreateBranch: branch name must not be empty")
	}
	for _, b := range c.engine.Index.Branches {
		if b.Name == name {
			return nil, fmt.Errorf("sdk.CreateBranch: branch %q already exists", name)
		}
	}
	currentCPID := c.engine.Index.CurrentCheckpointID
	if currentCPID == "" {
		return nil, fmt.Errorf("sdk.CreateBranch: no HEAD checkpoint — save at least once first")
	}
	b := timeline.NewBranch(name, currentCPID)
	c.engine.Index.AddBranch(b)
	if err := c.engine.Index.Save(c.cfg.IndexPath); err != nil {
		return nil, fmt.Errorf("sdk.CreateBranch: persist: %w", err)
	}
	return &b, nil
}

// SwitchBranch moves HEAD to the named branch's head checkpoint and restores files.
func (c *Client) SwitchBranch(name string) error {
	var targetID string
	for id, b := range c.engine.Index.Branches {
		if b.Name == name {
			targetID = id
			break
		}
	}
	if targetID == "" {
		return fmt.Errorf("sdk.SwitchBranch: branch %q not found", name)
	}

	branch := c.engine.Index.Branches[targetID]
	headCP, ok := c.engine.Index.Checkpoints[branch.HeadCheckpointID]
	if !ok {
		return fmt.Errorf("sdk.SwitchBranch: branch head checkpoint not found")
	}
	if headCP.SnapshotRef == "" {
		return fmt.Errorf("sdk.SwitchBranch: branch head has no snapshot")
	}

	snap, err := c.scanner.Load(headCP.SnapshotRef)
	if err != nil {
		return fmt.Errorf("sdk.SwitchBranch: load snapshot: %w", err)
	}

	c.engine.Index.CurrentBranchID = targetID
	c.engine.Index.CurrentCheckpointID = branch.HeadCheckpointID
	if err := c.engine.Index.Save(c.cfg.IndexPath); err != nil {
		return fmt.Errorf("sdk.SwitchBranch: persist index: %w", err)
	}

	return c.scanner.Restore(snap)
}

// ─── GC ───────────────────────────────────────────────────────────────────────

// GC runs garbage collection over the object store. When dryRun is true no
// files are deleted, but the result still reports what would be freed.
func (c *Client) GC(dryRun bool) (*GCResult, error) {
	// Collect reachable object hashes.
	reachable := make(map[string]struct{})
	for _, cp := range c.engine.Index.Checkpoints {
		if cp.SnapshotRef == "" {
			continue
		}
		reachable[cp.SnapshotRef] = struct{}{}

		// H2 fix: Mark the sidecar index object as reachable.
		sidecarKey := sdkSidecarKey(cp.SnapshotRef)
		reachable[sidecarKey] = struct{}{}

		snap, err := c.scanner.Load(cp.SnapshotRef)
		if err != nil {
			log.Warn().Str("snapshot", cp.SnapshotRef).Err(err).Msg("sdk.GC: skip unloadable snapshot")
			continue
		}
		for _, fe := range snap.Files {
			reachable[fe.Hash] = struct{}{}
		}
	}

	// Walk object store and identify unreferenced objects.
	type candidate struct {
		path string
		size int64
	}
	var dead []candidate

	err := filepath.WalkDir(c.cfg.ObjectsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		rel, err := filepath.Rel(c.cfg.ObjectsDir, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) != 2 {
			return nil
		}
		hash := parts[0] + parts[1]
		if _, ok := reachable[hash]; ok {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		dead = append(dead, candidate{path: path, size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("sdk.GC: walk objects: %w", err)
	}

	result := &GCResult{DryRun: dryRun}
	for _, obj := range dead {
		result.FreedBytes += obj.size
		result.CandidatePaths = append(result.CandidatePaths, obj.path)
	}

	if dryRun {
		result.RemovedObjects = len(dead)
		return result, nil
	}

	for _, obj := range dead {
		if err := os.Chmod(obj.path, 0o644); err != nil {
			log.Warn().Str("path", obj.path).Err(err).Msg("sdk.GC: chmod failed")
			continue
		}
		if err := os.Remove(obj.path); err != nil {
			log.Warn().Str("path", obj.path).Err(err).Msg("sdk.GC: remove failed")
			continue
		}
		result.RemovedObjects++
	}

	// Prune empty shard directories.
	_ = filepath.WalkDir(c.cfg.ObjectsDir, func(path string, d fs.DirEntry, _ error) error {
		if d != nil && d.IsDir() && path != c.cfg.ObjectsDir {
			if entries, err := os.ReadDir(path); err == nil && len(entries) == 0 {
				os.Remove(path)
			}
		}
		return nil
	})

	return result, nil
}

// ─── Tag ──────────────────────────────────────────────────────────────────────

// Tag attaches name to the checkpoint identified by checkpointID.
// If checkpointID is empty the current HEAD is tagged.
// The name must be unique across all checkpoints in the repository.
func (c *Client) Tag(name, checkpointID string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("sdk.Tag: tag name must not be empty")
	}

	targetID := checkpointID
	if targetID == "" {
		targetID = c.engine.Index.CurrentCheckpointID
	}
	if targetID == "" {
		return fmt.Errorf("sdk.Tag: no HEAD checkpoint — save at least once first")
	}

	cp, ok := c.engine.Index.Checkpoints[targetID]
	if !ok {
		return fmt.Errorf("sdk.Tag: checkpoint %q not found", targetID)
	}

	// Enforce global uniqueness.
	for id, other := range c.engine.Index.Checkpoints {
		for _, t := range other.Tags {
			if t == name {
				if id == targetID {
					return nil // idempotent — tag already present
				}
				return fmt.Errorf("sdk.Tag: tag %q already exists on checkpoint %s", name, id[:8])
			}
		}
	}

	cp.Tags = append(cp.Tags, name)
	c.engine.Index.Checkpoints[targetID] = cp

	if err := c.engine.Index.Save(c.cfg.IndexPath); err != nil {
		return fmt.Errorf("sdk.Tag: persist: %w", err)
	}
	return nil
}

// ─── Benchmarks & Utility ─────────────────────────────────────────────────────

// SetScanWorkers sets the number of parallel hashing workers for this client.
func (c *Client) SetScanWorkers(n int) {
	c.scanner.Workers = n
}

// ClearScanCache removes the last-scan.json cache file, forcing a full
// re-hash on the next Status or Save call.
func (c *Client) ClearScanCache() {
	scanPath := filepath.Join(c.cfg.RewindDir, "last-scan.json")
	os.Remove(scanPath)
}

// Restore restores the working directory to the checkpoint identified by idOrAlias.
func (c *Client) Restore(idOrAlias string) error {
	cp, err := c.resolveRef(idOrAlias)
	if err != nil {
		return fmt.Errorf("sdk.Restore: %w", err)
	}
	snap, err := c.scanner.Load(cp.SnapshotRef)
	if err != nil {
		return fmt.Errorf("sdk.Restore: load snapshot: %w", err)
	}
	return c.scanner.Restore(snap)
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// resolveRef translates any supported checkpoint reference to a Checkpoint.
// Supported: "HEAD", "HEAD~N", tag name, full UUID, 8-char prefix.
func (c *Client) resolveRef(ref string) (timeline.Checkpoint, error) {
	upper := strings.ToUpper(ref)

	// HEAD
	if upper == "HEAD" {
		id := c.engine.Index.CurrentCheckpointID
		if id == "" {
			return timeline.Checkpoint{}, fmt.Errorf("HEAD is not set")
		}
		cp, ok := c.engine.Index.Checkpoints[id]
		if !ok {
			return timeline.Checkpoint{}, fmt.Errorf("HEAD points to missing checkpoint %q", id)
		}
		return cp, nil
	}

	// HEAD~N
	if strings.HasPrefix(upper, "HEAD~") {
		nStr := ref[5:]
		n := 0
		if _, err := fmt.Sscanf(nStr, "%d", &n); err != nil || n < 0 {
			return timeline.Checkpoint{}, fmt.Errorf("invalid HEAD~N: %q", ref)
		}
		return c.walkTilde(n)
	}

	// Tag
	for _, cp := range c.engine.Index.Checkpoints {
		for _, t := range cp.Tags {
			if t == ref {
				return cp, nil
			}
		}
	}

	// Exact ID
	if cp, ok := c.engine.Index.Checkpoints[ref]; ok {
		return cp, nil
	}

	// Prefix
	var matches []timeline.Checkpoint
	for id, cp := range c.engine.Index.Checkpoints {
		if strings.HasPrefix(id, ref) {
			matches = append(matches, cp)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return timeline.Checkpoint{}, fmt.Errorf("no checkpoint for reference %q", ref)
	default:
		return timeline.Checkpoint{}, fmt.Errorf("ambiguous reference %q (%d matches)", ref, len(matches))
	}
}

func (c *Client) walkTilde(n int) (timeline.Checkpoint, error) {
	curID := c.engine.Index.CurrentCheckpointID
	for i := 0; i < n; i++ {
		cp, ok := c.engine.Index.Checkpoints[curID]
		if !ok {
			return timeline.Checkpoint{}, fmt.Errorf("HEAD~%d: checkpoint not found at step %d", n, i)
		}
		if cp.ParentID == "" {
			return timeline.Checkpoint{}, fmt.Errorf("HEAD~%d: reached root at step %d", n, i)
		}
		curID = cp.ParentID
	}
	cp, ok := c.engine.Index.Checkpoints[curID]
	if !ok {
		return timeline.Checkpoint{}, fmt.Errorf("HEAD~%d: final checkpoint not found", n)
	}
	return cp, nil
}

func applyLimit(cps []*timeline.Checkpoint, limit int) []*timeline.Checkpoint {
	if limit <= 0 || len(cps) <= limit {
		return cps
	}
	return cps[:limit]
}

// sdkSidecarKey mirrors the snapshot.sidecarObjectKey formula to avoid
// a circular import. Formula: SHA-256("snapshot-index:" + hash).
func sdkSidecarKey(snapshotHash string) string {
	h := sha256.Sum256([]byte("snapshot-index:" + snapshotHash))
	return hex.EncodeToString(h[:])
}
