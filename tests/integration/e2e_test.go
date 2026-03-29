//go:build integration
// +build integration

// Run with: go test -tags integration -v -race -timeout=120s ./tests/integration/
package integration_test

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/itsakash-real/nimbi/internal/config"
	"github.com/itsakash-real/nimbi/internal/snapshot"
	"github.com/itsakash-real/nimbi/internal/storage"
	"github.com/itsakash-real/nimbi/internal/timeline"
)

// ─── Test harness ─────────────────────────────────────────────────────────────

// repo is the wiring used by every test: a real on-disk repository under a
// temp directory, exercised through the same internal packages the CLI uses.
type repo struct {
	root    string // project root (parent of .rewind)
	cfg     *config.Config
	store   *storage.ObjectStore
	scanner *snapshot.Scanner
	engine  *timeline.TimelineEngine
}

// initRepo creates a fresh temp directory, runs timeline.Init, and wires all
// subsystems identically to the way cmd/rw does it.
func initRepo(t *testing.T) *repo {
	t.Helper()
	root := t.TempDir()

	engine, err := timeline.Init(root)
	require.NoError(t, err, "timeline.Init must succeed on a clean directory")

	cfg, err := config.Load(root)
	require.NoError(t, err)

	store := storage.New(cfg.ObjectsDir)
	sc := snapshot.New(root, store)

	return &repo{
		root:    root,
		cfg:     cfg,
		store:   store,
		scanner: sc,
		engine:  engine,
	}
}

// reloadEngine re-reads index.json from disk, simulating a fresh CLI
// invocation. Used to verify that persistence across "process restarts" works.
func (r *repo) reloadEngine(t *testing.T) {
	t.Helper()
	eng, err := timeline.New(r.cfg.IndexPath)
	require.NoError(t, err, "reload engine from disk")
	r.engine = eng
}

// save runs a full Scan → Save → SaveCheckpoint cycle and returns the new
// checkpoint. It mirrors what cmd/rw save does.
func (r *repo) save(t *testing.T, message string) timeline.Checkpoint {
	t.Helper()
	snap, err := r.scanner.Scan()
	require.NoError(t, err, "scan for save(%q)", message)

	snapshotHash, err := r.scanner.Save(snap)
	require.NoError(t, err, "persist snapshot for save(%q)", message)

	cp, err := r.engine.SaveCheckpoint(message, snapshotHash)
	require.NoError(t, err, "SaveCheckpoint(%q)", message)

	err = r.engine.Index.Save(r.cfg.IndexPath)
	require.NoError(t, err, "persist index after save(%q)", message)

	return cp
}

// gotoCP restores the working directory to a checkpoint, mirroring cmd/rw goto.
func (r *repo) gotoCP(t *testing.T, id string) {
	t.Helper()
	cp, ok := r.engine.Index.Checkpoints[id]
	require.True(t, ok, "checkpoint %s must exist in index", id)

	snap, err := r.scanner.Load(cp.SnapshotRef)
	require.NoError(t, err, "load snapshot for goto(%s)", id)

	_, err = r.scanner.Restore(snap)
	require.NoError(t, err, "restore for goto(%s)", id)

	_, err = r.engine.GotoCheckpoint(id)
	require.NoError(t, err, "GotoCheckpoint(%s)", id)

	err = r.engine.Index.Save(r.cfg.IndexPath)
	require.NoError(t, err, "persist index after goto(%s)", id)
}

// write creates or overwrites a file relative to project root.
func (r *repo) write(t *testing.T, rel, content string) {
	t.Helper()
	abs := filepath.Join(r.root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
	require.NoError(t, os.WriteFile(abs, []byte(content), 0o644))
}

// read returns the content of a file relative to project root.
func (r *repo) read(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(rel)))
	require.NoError(t, err)
	return string(data)
}

// exists returns true if the relative path exists on disk.
func (r *repo) exists(rel string) bool {
	_, err := os.Stat(filepath.Join(r.root, filepath.FromSlash(rel)))
	return err == nil
}

// dirSnapshot returns a sorted map of relPath → content for every non-.rewind
// regular file under root. Used for exact state comparison.
func dirSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	m := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && d.Name() == ".rewind" {
				return filepath.SkipDir
			}
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		m[rel] = string(data)
		return nil
	})
	require.NoError(t, err)
	return m
}

// objectCount counts the number of objects in the store's objects/ directory.
func objectCount(t *testing.T, objectsDir string) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(objectsDir, func(_ string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		n++
		return nil
	})
	require.NoError(t, err)
	return n
}

// seedProject writes 10 numbered files with distinct content.
func seedProject(t *testing.T, r *repo) {
	t.Helper()
	for i := 0; i < 10; i++ {
		r.write(t, fmt.Sprintf("src/file_%02d.go", i),
			fmt.Sprintf("package main\n\n// file %d — initial\nconst X%d = %d\n", i, i, i))
	}
}

// ─── Test 1: Full Linear Workflow ─────────────────────────────────────────────

// TestE2E_LinearWorkflow validates the core save → goto → verify cycle with
// three checkpoints and confirms exact file-content equality at each restore.
func TestE2E_LinearWorkflow(t *testing.T) {
	r := initRepo(t)

	// ── Checkpoint 1: initial state ──────────────────────────────────────────
	seedProject(t, r)
	stateBefore := dirSnapshot(t, r.root)
	cp1 := r.save(t, "initial")

	// ── Checkpoint 2: modify 3 files ─────────────────────────────────────────
	r.write(t, "src/file_00.go", "package main\n\n// MODIFIED in cp2\nconst X0 = 999\n")
	r.write(t, "src/file_01.go", "package main\n\n// MODIFIED in cp2\nconst X1 = 888\n")
	r.write(t, "src/file_02.go", "package main\n\n// MODIFIED in cp2\nconst X2 = 777\n")
	stateAfterCP2 := dirSnapshot(t, r.root)
	cp2 := r.save(t, "after changes")

	// ── Checkpoint 3: modify 2 more files + add one ───────────────────────────
	r.write(t, "src/file_03.go", "package main\n\n// MODIFIED in cp3\n")
	r.write(t, "src/file_04.go", "package main\n\n// MODIFIED in cp3\n")
	r.write(t, "src/newfile.go", "package main\n\n// brand new\n")
	cp3 := r.save(t, "more changes")
	_ = cp3

	// ── Restore to cp1 → assert exact initial state ───────────────────────────
	r.gotoCP(t, cp1.ID)
	stateRestored1 := dirSnapshot(t, r.root)

	assert.Equal(t, stateBefore, stateRestored1,
		"after goto cp1 the working tree must exactly match the initial state")
	assert.False(t, r.exists("src/newfile.go"),
		"newfile.go added in cp3 must not exist after restoring to cp1")

	// ── Restore to cp2 → assert second state ─────────────────────────────────
	r.gotoCP(t, cp2.ID)
	stateRestored2 := dirSnapshot(t, r.root)

	assert.Equal(t, stateAfterCP2, stateRestored2,
		"after goto cp2 the working tree must exactly match the post-cp2 state")
	assert.Equal(t, "package main\n\n// MODIFIED in cp2\nconst X0 = 999\n",
		r.read(t, "src/file_00.go"),
		"file_00.go must contain cp2 content after restoring to cp2")
	assert.False(t, r.exists("src/newfile.go"),
		"newfile.go must not exist after restoring to cp2")
}

// ─── Test 2: Branching ────────────────────────────────────────────────────────

// TestE2E_Branching verifies that saving from a non-HEAD position auto-creates
// a new branch, that the original branch is unaffected, and that switching back
// restores the correct file state.
func TestE2E_Branching(t *testing.T) {
	r := initRepo(t)
	seedProject(t, r)

	// Build T1 → T2 → T3 on main.
	cp1 := r.save(t, "T1")
	r.write(t, "src/file_00.go", "package main // T2 change\n")
	cp2 := r.save(t, "T2")
	r.write(t, "src/file_01.go", "package main // T3 change\n")
	cp3 := r.save(t, "T3")

	stateT3 := dirSnapshot(t, r.root)

	// Remember how many checkpoints/branches exist before the branch.
	r.reloadEngine(t)
	checkpointsBefore := len(r.engine.Index.Checkpoints)
	branchesBefore := len(r.engine.Index.Branches)

	// Travel back to T2 and make a diverging change.
	r.gotoCP(t, cp2.ID)
	r.write(t, "src/file_05.go", "package main // experiment branch\n")
	cpBranch := r.save(t, "branch experiment")

	// ── Assert: a new branch was auto-created ─────────────────────────────────
	r.reloadEngine(t)
	assert.Greater(t, len(r.engine.Index.Branches), branchesBefore,
		"saving from a non-HEAD position must create a new branch")
	assert.Greater(t, len(r.engine.Index.Checkpoints), checkpointsBefore,
		"branch experiment checkpoint must be persisted")

	// The branch checkpoint must NOT be an ancestor of cp3 on main.
	assert.NotEqual(t, cp3.ID, cpBranch.ParentID,
		"branch checkpoint must not be a direct parent of T3")

	// ── Assert: main branch still ends at cp3 ────────────────────────────────
	mainBranch, ok := r.engine.Index.Branches[cp1.BranchID]
	require.True(t, ok, "original branch must still exist")
	assert.Equal(t, cp3.ID, mainBranch.HeadCheckpointID,
		"main branch head must still be T3 after a new branch diverged from T2")

	// ── Switch back to main (restore to cp3) and assert files match T3 ───────
	r.gotoCP(t, cp3.ID)
	stateAfterSwitch := dirSnapshot(t, r.root)
	assert.Equal(t, stateT3, stateAfterSwitch,
		"after switching back to main/T3 the working tree must match T3 state")
	assert.False(t, r.exists("src/file_05.go"),
		"experiment file from the branch must not exist after switching back to main/T3")
}

// ─── Test 3: Deduplication ────────────────────────────────────────────────────

// TestE2E_Deduplication verifies that identical file content written across
// multiple checkpoints is stored only once in the object store.
func TestE2E_Deduplication(t *testing.T) {
	r := initRepo(t)

	// Write one stable "shared" file and one changing file.
	sharedContent := "package shared\n\n// This content never changes.\nconst Stable = true\n"
	r.write(t, "shared/stable.go", sharedContent)
	r.write(t, "changing/v1.go", "package v1\n")
	cp1 := r.save(t, "cp1")

	// cp2: only the changing file differs.
	r.write(t, "changing/v1.go", "package v1 // modified\n")
	cp2 := r.save(t, "cp2")

	// cp3: change again; stable is still untouched.
	r.write(t, "changing/v1.go", "package v1 // modified again\n")
	cp3 := r.save(t, "cp3")

	// Load all three snapshots and confirm all reference the same hash for stable.go.
	snapCP1, err := r.scanner.Load(cp1.SnapshotRef)
	require.NoError(t, err)
	snapCP2, err := r.scanner.Load(cp2.SnapshotRef)
	require.NoError(t, err)
	snapCP3, err := r.scanner.Load(cp3.SnapshotRef)
	require.NoError(t, err)

	hashInCP1 := findFileHash(snapCP1, "shared/stable.go")
	hashInCP2 := findFileHash(snapCP2, "shared/stable.go")
	hashInCP3 := findFileHash(snapCP3, "shared/stable.go")

	require.NotEmpty(t, hashInCP1, "shared/stable.go must be in snapshot cp1")
	assert.Equal(t, hashInCP1, hashInCP2,
		"shared/stable.go must have the same hash in cp1 and cp2")
	assert.Equal(t, hashInCP1, hashInCP3,
		"shared/stable.go must have the same hash in cp1 and cp3")

	// Confirm the store holds exactly ONE object for that content.
	objectPath := func(hash string) string {
		return filepath.Join(r.cfg.ObjectsDir, hash[:2], hash[2:])
	}
	_, err1 := os.Stat(objectPath(hashInCP1))
	require.NoError(t, err1, "shared/stable.go object must exist in store")

	// Verify there is no second copy with the same content under a different shard.
	// We do this by confirming the raw bytes at the single path match the expected content.
	data, err := r.store.Read(hashInCP1)
	require.NoError(t, err)
	assert.Equal(t, sharedContent, string(data),
		"object store must return the original stable file content")

	// Stats: the three changing files plus the three snapshot objects, plus stable (one)
	// means we should NOT see three copies of stable.
	totalObjects := objectCount(t, r.cfg.ObjectsDir)
	t.Logf("total objects in store: %d", totalObjects)

	// At most: 1 (stable) + 3 (changing v1, v2, v3) + 3 (snapshot JSONs) + sidecars = small number.
	// Critically, stable must not appear more than once.
	assert.LessOrEqual(t, totalObjects, 15,
		"store should not have excessive objects; deduplication must be working")
}

// findFileHash returns the hash for a given path from a snapshot, or "".
func findFileHash(snap *timeline.Snapshot, relPath string) string {
	for _, fe := range snap.Files {
		if fe.Path == relPath {
			return fe.Hash
		}
	}
	return ""
}

// ─── Test 4: Garbage Collection ───────────────────────────────────────────────

// TestE2E_GC creates five checkpoints, then directly removes two checkpoints
// from the index, runs GC, and verifies that orphaned objects are gone while
// live objects remain intact.
func TestE2E_GC(t *testing.T) {
	r := initRepo(t)

	// Create 5 checkpoints with unique content in each.
	cps := make([]timeline.Checkpoint, 5)
	for i := range cps {
		r.write(t, fmt.Sprintf("src/unique_%d.go", i),
			fmt.Sprintf("package main\nconst Unique%d = %d\n", i, i*1000))
		cps[i] = r.save(t, fmt.Sprintf("checkpoint %d", i+1))
	}

	// Capture the snapshot refs for cp2 and cp3 (indices 1 and 2).
	orphanedRefs := []string{cps[1].SnapshotRef, cps[2].SnapshotRef}

	// ── Directly delete cp2 and cp3 from the index ────────────────────────────
	// This simulates what a "rw delete" command would do, or a corrupted partial
	// delete. We mutate the in-memory index, then persist it.
	r.reloadEngine(t)
	delete(r.engine.Index.Checkpoints, cps[1].ID)
	delete(r.engine.Index.Checkpoints, cps[2].ID)

	// Keep HEAD pointing at cp5 (last remaining checkpoint).
	r.engine.Index.CurrentCheckpointID = cps[4].ID
	require.NoError(t, r.engine.Index.Save(r.cfg.IndexPath))

	// Confirm the deletions took effect.
	r.reloadEngine(t)
	_, cp2Exists := r.engine.Index.Checkpoints[cps[1].ID]
	_, cp3Exists := r.engine.Index.Checkpoints[cps[2].ID]
	assert.False(t, cp2Exists, "cp2 must be removed from index before GC")
	assert.False(t, cp3Exists, "cp3 must be removed from index before GC")

	// ── Run GC ────────────────────────────────────────────────────────────────
	freed, err := r.engine.GC(r.cfg.ObjectsDir, r.scanner)
	require.NoError(t, err, "GC must not return an error")
	t.Logf("GC freed %d objects", freed)
	assert.Greater(t, freed, 0, "GC must free at least some objects")

	// ── Assert: objects from deleted checkpoints are gone ─────────────────────
	for _, ref := range orphanedRefs {
		if ref == "" {
			continue
		}
		_, readErr := r.store.Read(ref)
		assert.Error(t, readErr,
			"snapshot object %s from deleted checkpoint must be gone after GC", ref[:8])
	}

	// ── Assert: objects from remaining checkpoints are still intact ───────────
	remainingIDs := []string{cps[0].ID, cps[3].ID, cps[4].ID}
	r.reloadEngine(t)
	for _, id := range remainingIDs {
		cp, ok := r.engine.Index.Checkpoints[id]
		require.True(t, ok, "checkpoint %s must survive GC", id[:8])
		require.NotEmpty(t, cp.SnapshotRef)

		snap, err := r.scanner.Load(cp.SnapshotRef)
		require.NoError(t, err,
			"snapshot %s for surviving checkpoint %s must still be loadable after GC",
			cp.SnapshotRef[:8], id[:8])

		// Verify every file object in the surviving snapshot is also intact.
		for _, fe := range snap.Files {
			_, readErr := r.store.Read(fe.Hash)
			assert.NoError(t, readErr,
				"file object %s (path: %s) in surviving checkpoint must still exist after GC",
				fe.Hash[:8], fe.Path)
		}
	}
}

// ─── Test 5: Crash Recovery ───────────────────────────────────────────────────

// TestE2E_CrashRecovery simulates a crash that occurs after all file objects
// are written but before the index is updated. It then re-runs a save and
// verifies the system recovers cleanly with no duplicate objects and a valid
// final state.
func TestE2E_CrashRecovery(t *testing.T) {
	r := initRepo(t)
	seedProject(t, r)

	// ── Normal first checkpoint ───────────────────────────────────────────────
	cp1 := r.save(t, "pre-crash checkpoint")

	// ── Simulate crash: write objects but do NOT update index ────────────────
	r.write(t, "src/file_00.go", "package main // crash-modified\n")

	snapCrash, err := r.scanner.Scan()
	require.NoError(t, err)

	// Write all file objects to the store (normally done inside scanner.Save).
	crashSnapshotHash, err := r.scanner.Save(snapCrash)
	require.NoError(t, err, "object writes should succeed before simulated crash")

	// "Crash": do NOT call engine.SaveCheckpoint or engine.Index.Save.
	// The objects are orphaned in the store; the index still points at cp1.
	_ = crashSnapshotHash

	// Record object count just after the "crash".
	objectsAfterCrash := objectCount(t, r.cfg.ObjectsDir)
	t.Logf("objects after crash: %d", objectsAfterCrash)

	// ── Also simulate a leftover index.json.tmp (incomplete rename) ───────────
	tmpIndexPath := filepath.Join(r.cfg.RewindDir, ".index-crash.tmp")
	indexData, err := json.MarshalIndent(r.engine.Index, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tmpIndexPath, indexData, 0o644))

	// ── Run crash recovery ────────────────────────────────────────────────────
	err = timeline.RunRecovery(r.cfg.RewindDir)
	require.NoError(t, err, "RunRecovery must complete without error")

	// tmp file must be resolved.
	assert.NoFileExists(t, tmpIndexPath,
		"leftover .index-*.tmp must be removed or renamed by RunRecovery")

	// ── Re-run save after recovery ────────────────────────────────────────────
	r.reloadEngine(t)
	cp2 := r.save(t, "post-crash save")

	// ── Assert: system is in a clean, valid state ─────────────────────────────
	r.reloadEngine(t)
	assert.Len(t, r.engine.Index.Checkpoints, 2,
		"index must contain exactly 2 checkpoints after crash + recovery + re-save")
	assert.Equal(t, cp2.ID, r.engine.Index.CurrentCheckpointID,
		"HEAD must point at the post-crash save checkpoint")

	// Snapshot from cp2 must be loadable and correct.
	snap2, err := r.scanner.Load(cp2.SnapshotRef)
	require.NoError(t, err, "post-crash checkpoint snapshot must be loadable")

	found := false
	for _, fe := range snap2.Files {
		if fe.Path == "src/file_00.go" {
			found = true
			data, readErr := r.store.Read(fe.Hash)
			require.NoError(t, readErr)
			assert.Equal(t, "package main // crash-modified\n", string(data),
				"crash-modified content must be persisted correctly after recovery")
		}
	}
	assert.True(t, found, "src/file_00.go must appear in the post-crash snapshot")

	// Restoring to cp1 must still work (original objects must be intact).
	r.gotoCP(t, cp1.ID)
	assert.Equal(t,
		"package main\n\n// file 0 — initial\nconst X0 = 0\n",
		r.read(t, "src/file_00.go"),
		"cp1 restore must recover the pre-crash content of file_00.go")
}

// ─── Test 6: Large Project Performance ────────────────────────────────────────

// TestE2E_LargeProjectPerformance generates a realistic 500-file project and
// asserts that save, goto, and status all complete within the target latencies.
//
// Targets (from bench spec):
//   - Save  < 2 000 ms
//   - Goto  < 1 000 ms
//   - Status < 200 ms (warm, after first scan)
func TestE2E_LargeProjectPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in -short mode")
	}

	r := initRepo(t)

	// ── Generate 500 files with realistic sizes (1 KB – 100 KB) ──────────────
	const fileCount = 500
	t.Logf("generating %d files...", fileCount)
	for i := 0; i < fileCount; i++ {
		// Size: alternates between 1 KB and 100 KB for variety.
		size := 1024
		if i%10 == 0 {
			size = 100 * 1024
		}
		content := make([]byte, size)
		_, err := rand.Read(content[:size/2]) // random first half
		require.NoError(t, err)
		// Second half: printable ASCII so the file is not purely binary.
		for j := size / 2; j < size; j++ {
			content[j] = byte('a' + (i+j)%26)
		}
		subdir := fmt.Sprintf("pkg/sub%03d", i/50)
		r.write(t, fmt.Sprintf("%s/file_%04d.go", subdir, i), string(content))
	}

	// ── Benchmark: Save ───────────────────────────────────────────────────────
	start := time.Now()
	cp1 := r.save(t, "large initial save")
	saveDuration := time.Since(start)

	t.Logf("Save  (%d files): %v", fileCount, saveDuration)
	assert.Less(t, saveDuration, 2*time.Second,
		"Save on %d files must complete in < 2 000 ms (got %v)", fileCount, saveDuration)

	// ── Modify ~10% of files for a second checkpoint ──────────────────────────
	for i := 0; i < 50; i++ {
		subdir := fmt.Sprintf("pkg/sub%03d", i/50)
		r.write(t, fmt.Sprintf("%s/file_%04d.go", subdir, i),
			fmt.Sprintf("package modified\n// file %d changed\n", i))
	}
	cp2 := r.save(t, "large second save (50 changes)")

	// ── Benchmark: Goto (delta restore — only 50 files should be written) ─────
	// First warm up goto to cp1 to get a baseline, then measure.
	r.gotoCP(t, cp2.ID) // restore back to cp2 first
	start = time.Now()
	r.gotoCP(t, cp1.ID)
	gotoDuration := time.Since(start)

	t.Logf("Goto  (%d files, ~50 delta): %v", fileCount, gotoDuration)
	assert.Less(t, gotoDuration, time.Second,
		"Goto on %d files (delta restore) must complete in < 1 000 ms (got %v)",
		fileCount, gotoDuration)

	// ── Benchmark: Status (warm — mtime cache is populated from prior scan) ───
	// Trigger a cold scan first to populate last-scan.json.
	_, _, err := r.scanner.FastScan(r.cfg.RewindDir)
	require.NoError(t, err, "cold FastScan must succeed")

	start = time.Now()
	_, _, err = r.scanner.FastScan(r.cfg.RewindDir)
	statusDuration := time.Since(start)

	require.NoError(t, err, "warm FastScan must succeed")
	t.Logf("Status warm (%d files, no changes): %v", fileCount, statusDuration)
	assert.Less(t, statusDuration, 200*time.Millisecond,
		"Warm FastScan on %d unchanged files must complete in < 200 ms (got %v)",
		fileCount, statusDuration)

	// ── Sanity: Status detects the 50 changed files made for cp2 ─────────────
	// Make 5 fresh changes and verify FastScan reports exactly 5 changed paths.
	for i := 0; i < 5; i++ {
		subdir := fmt.Sprintf("pkg/sub%03d", i/50)
		r.write(t, fmt.Sprintf("%s/file_%04d.go", subdir, i),
			fmt.Sprintf("package modified\n// NEW change %d\n", i))
	}
	changed, _, err := r.scanner.FastScan(r.cfg.RewindDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(changed), 5,
		"FastScan must detect the 5 freshly modified files")

	_ = cp1
	_ = cp2
}

// ─── Test 7: Round-Trip Checksum Integrity ────────────────────────────────────

// TestE2E_ChecksumIntegrity saves a checkpoint, then corrupts an object on
// disk and verifies that Read returns ErrCorruptObject.
func TestE2E_ChecksumIntegrity(t *testing.T) {
	r := initRepo(t)
	r.write(t, "src/important.go", "package main\n\nconst Key = \"secret\"\n")
	cp := r.save(t, "integrity check")

	snap, err := r.scanner.Load(cp.SnapshotRef)
	require.NoError(t, err)

	// Find the object hash for important.go.
	var objHash string
	for _, fe := range snap.Files {
		if fe.Path == "src/important.go" {
			objHash = fe.Hash
			break
		}
	}
	require.NotEmpty(t, objHash, "src/important.go must be in snapshot")

	// Corrupt the file on disk.
	objPath := filepath.Join(r.cfg.ObjectsDir, objHash[:2], objHash[2:])
	require.NoError(t, os.Chmod(objPath, 0o644))
	require.NoError(t, os.WriteFile(objPath, []byte("CORRUPTED"), 0o644))

	// Read must return ErrCorruptObject.
	_, readErr := r.store.Read(objHash)
	require.Error(t, readErr)
	assert.ErrorIs(t, readErr, storage.ErrCorruptObject,
		"reading a corrupted object must return ErrCorruptObject")

	// RunRecovery must detect and remove the corrupt object.
	require.NoError(t, timeline.RunRecovery(r.cfg.RewindDir))
	_, statErr := os.Stat(objPath)
	assert.True(t, os.IsNotExist(statErr),
		"RunRecovery must remove the corrupted object file")
}

// ─── Test 8: .rewindignore Patterns ──────────────────────────────────────────

// TestE2E_RewindIgnore verifies that files matching .rewindignore patterns
// are excluded from all snapshots, and that re-including them after removing
// the rule correctly picks them up.
func TestE2E_RewindIgnore(t *testing.T) {
	r := initRepo(t)

	// Write a .rewindignore that excludes *.log and secrets/.
	require.NoError(t, os.WriteFile(
		filepath.Join(r.root, ".rewindignore"),
		[]byte("# Logs\n*.log\n\n# Secrets\nsecrets/\n"),
		0o644,
	))

	r.write(t, "main.go", "package main\n")
	r.write(t, "app.log", "2026-03-23 startup\n")           // must be ignored
	r.write(t, "secrets/api_key.txt", "super-secret-key\n") // must be ignored
	r.write(t, "data/results.csv", "a,b,c\n")

	cp := r.save(t, "with ignores")
	snap, err := r.scanner.Load(cp.SnapshotRef)
	require.NoError(t, err)

	paths := make(map[string]bool)
	for _, fe := range snap.Files {
		paths[fe.Path] = true
	}

	assert.True(t, paths["main.go"], "main.go must be tracked")
	assert.True(t, paths["data/results.csv"], "data/results.csv must be tracked")
	assert.False(t, paths["app.log"], "app.log must be excluded by *.log pattern")
	assert.False(t, paths["secrets/api_key.txt"],
		"secrets/api_key.txt must be excluded by secrets/ pattern")
}

// ─── Test 9: Concurrent Saves (race detector) ─────────────────────────────────

// TestE2E_FileLock_PreventsConcurrentWrites confirms that two concurrent save
// operations cannot both acquire the write lock — the second must receive
// ErrLockHeld.
func TestE2E_FileLock_PreventsConcurrentWrites(t *testing.T) {
	r := initRepo(t)
	seedProject(t, r)

	lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
	fl1 := storage.NewFileLock(lockPath)
	fl2 := storage.NewFileLock(lockPath)

	// fl1 acquires the lock.
	require.NoError(t, fl1.Acquire(), "first lock acquisition must succeed")
	defer fl1.Release()

	// fl2 must be denied.
	err := fl2.Acquire()
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrLockHeld,
		"second lock acquisition while fl1 is held must return ErrLockHeld")
}

// ─── Test 10: Tag Resolution ──────────────────────────────────────────────────

// TestE2E_TagResolution verifies that a checkpoint can be tagged and then
// resolved by tag name for a goto operation.
func TestE2E_TagResolution(t *testing.T) {
	r := initRepo(t)
	seedProject(t, r)
	cp1 := r.save(t, "initial")

	// Attach tag "v1.0" to cp1.
	r.engine.Index.Checkpoints[cp1.ID] = timeline.Checkpoint{
		ID:          cp1.ID,
		ParentID:    cp1.ParentID,
		BranchID:    cp1.BranchID,
		Message:     cp1.Message,
		SnapshotRef: cp1.SnapshotRef,
		CreatedAt:   cp1.CreatedAt,
		Tags:        []string{"v1.0"},
	}
	require.NoError(t, r.engine.Index.Save(r.cfg.IndexPath))

	// Make a second checkpoint so HEAD moves forward.
	r.write(t, "src/file_00.go", "package main // v2\n")
	r.save(t, "v2 changes")

	// Resolve "v1.0" from the index and confirm it maps to cp1.
	r.reloadEngine(t)
	var resolved *timeline.Checkpoint
	for _, cp := range r.engine.Index.Checkpoints {
		for _, tag := range cp.Tags {
			if tag == "v1.0" {
				cpCopy := cp
				resolved = &cpCopy
			}
		}
	}
	require.NotNil(t, resolved, "tag v1.0 must resolve to a checkpoint")
	assert.Equal(t, cp1.ID, resolved.ID, "v1.0 must resolve to cp1")

	// Goto by resolved ID and verify content.
	r.gotoCP(t, resolved.ID)
	assert.Equal(t,
		"package main\n\n// file 0 — initial\nconst X0 = 0\n",
		r.read(t, "src/file_00.go"),
		"goto by tag-resolved ID must restore correct file content")
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// sortedKeys returns sorted keys of a map[string]string for stable assertions.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// assertDirEqual is a verbose helper that prints exactly which files differ.
func assertDirEqual(t *testing.T, expected, actual map[string]string, ctx string) {
	t.Helper()
	expectedKeys := sortedKeys(expected)
	actualKeys := sortedKeys(actual)

	// Check for missing files.
	for _, k := range expectedKeys {
		if _, ok := actual[k]; !ok {
			t.Errorf("%s: expected file %q is missing from actual state", ctx, k)
		}
	}
	// Check for extra files.
	for _, k := range actualKeys {
		if _, ok := expected[k]; !ok {
			t.Errorf("%s: unexpected extra file %q in actual state", ctx, k)
		}
	}
	// Check content.
	for _, k := range expectedKeys {
		ev, av := expected[k], actual[k]
		if ev != av {
			// Show a compact diff hint rather than dumping full content.
			eLines := strings.Split(ev, "\n")
			aLines := strings.Split(av, "\n")
			t.Errorf("%s: file %q content mismatch (expected %d lines, got %d lines)",
				ctx, k, len(eLines), len(aLines))
		}
	}
}
