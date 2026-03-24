package timeline_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/itsakash-real/rewinddb/internal/timeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// initEngine creates a temp directory, calls timeline.Init, and returns the
// engine plus a cleanup function that restores the working directory.
func initEngine(t *testing.T) *timeline.TimelineEngine {
	t.Helper()
	tmp := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { os.Chdir(orig) })

	engine, err := timeline.Init(tmp)
	require.NoError(t, err)
	return engine
}

// saveN saves n checkpoints with sequential messages and fake snapshot refs.
func saveN(t *testing.T, e *timeline.TimelineEngine, n int) []*timeline.Checkpoint {
	t.Helper()
	var cps []*timeline.Checkpoint
	for i := 0; i < n; i++ {
		cp, err := e.SaveCheckpoint(fmt.Sprintf("checkpoint %d", i+1), fmt.Sprintf("snap-%d", i+1))
		require.NoError(t, err)
		cps = append(cps, cp)
	}
	return cps
}

// ─── Init ─────────────────────────────────────────────────────────────────────

func TestInit_CreatesMainBranchAndRootCheckpoint(t *testing.T) {
	e := initEngine(t)

	branch, ok := e.Index.CurrentBranch()
	require.True(t, ok)
	assert.Equal(t, "main", branch.Name)

	cp, ok := e.Index.CurrentCheckpoint()
	require.True(t, ok)
	assert.Contains(t, cp.Tags, "root")
	assert.Empty(t, cp.ParentID, "root checkpoint has no parent")
}

func TestInit_FailsIfAlreadyInitialized(t *testing.T) {
	initEngine(t) // first init

	_, err := timeline.Init(".")
	assert.Error(t, err)
	assert.ErrorIs(t, err, timeline.ErrAlreadyInitialized)
}

func TestInit_IndexPersistsAcrossLoad(t *testing.T) {
	e := initEngine(t)

	// Reload from disk
	loaded, err := timeline.New(e.IndexPath)
	require.NoError(t, err)

	assert.Equal(t, e.Index.CurrentBranchID, loaded.Index.CurrentBranchID)
	assert.Equal(t, e.Index.CurrentCheckpointID, loaded.Index.CurrentCheckpointID)
}

// ─── Linear timeline ──────────────────────────────────────────────────────────

func TestSaveCheckpoint_LinearTimeline_FiveCheckpoints(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 5)

	list, err := e.ListCheckpoints("")
	require.NoError(t, err)

	// ListCheckpoints returns newest-first; saved[4] is the head.
	// The root checkpoint is also in the list, so total = 6.
	require.Len(t, list, 6, "5 saved + 1 root checkpoint")
	assert.Equal(t, saved[4].ID, list[0].ID, "head is newest checkpoint")
	assert.Equal(t, saved[0].ID, list[4].ID, "fifth from top is first saved")
}

func TestSaveCheckpoint_ParentChainIsCorrect(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 3)

	assert.Equal(t, saved[0].ID, saved[1].ParentID)
	assert.Equal(t, saved[1].ID, saved[2].ParentID)
}

func TestSaveCheckpoint_AllOnMainBranch(t *testing.T) {
	e := initEngine(t)
	branch, _ := e.Index.CurrentBranch()
	saved := saveN(t, e, 3)

	for _, cp := range saved {
		assert.Equal(t, branch.ID, cp.BranchID)
	}
}

func TestSaveCheckpoint_PersistsToDisk(t *testing.T) {
	e := initEngine(t)
	cp, err := e.SaveCheckpoint("persisted", "snap-x")
	require.NoError(t, err)

	loaded, err := timeline.New(e.IndexPath)
	require.NoError(t, err)

	_, ok := loaded.Index.Checkpoints[cp.ID]
	assert.True(t, ok, "checkpoint must survive index reload")
}

// ─── GotoCheckpoint ───────────────────────────────────────────────────────────

func TestGotoCheckpoint_UpdatesIndexState(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 3)

	// Go back to the second checkpoint
	got, err := e.GotoCheckpoint(saved[1].ID)
	require.NoError(t, err)
	assert.Equal(t, saved[1].ID, got.ID)
	assert.Equal(t, saved[1].ID, e.Index.CurrentCheckpointID)
}

func TestGotoCheckpoint_UnknownID_ReturnsError(t *testing.T) {
	e := initEngine(t)
	_, err := e.GotoCheckpoint("does-not-exist")
	assert.ErrorIs(t, err, timeline.ErrCheckpointNotFound)
}

// ─── Branch fork (detached HEAD) ──────────────────────────────────────────────

func TestSaveCheckpoint_ForksBranchWhenHeadIsDetached(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 4) // T1 T2 T3 T4 on main

	mainBranchID := saved[0].BranchID

	// Rewind to T2
	_, err := e.GotoCheckpoint(saved[1].ID)
	require.NoError(t, err)

	// Save a new checkpoint — HEAD ≠ branch head → fork
	forked, err := e.SaveCheckpoint("fork from T2", "snap-fork")
	require.NoError(t, err)

	assert.NotEqual(t, mainBranchID, forked.BranchID, "fork must live on a new branch")
	assert.Equal(t, saved[1].ID, forked.ParentID, "fork's parent must be T2")

	// The new branch must exist in the index
	newBranch, ok := e.Index.Branches[forked.BranchID]
	require.True(t, ok)
	assert.Equal(t, forked.ID, newBranch.HeadCheckpointID)
	assert.Equal(t, forked.ID, newBranch.RootCheckpointID)
}

func TestSaveCheckpoint_MainBranchUnchangedAfterFork(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 3)
	mainBranchID := saved[0].BranchID
	originalHead := saved[2].ID

	_, err := e.GotoCheckpoint(saved[0].ID)
	require.NoError(t, err)
	_, err = e.SaveCheckpoint("fork", "snap-fork")
	require.NoError(t, err)

	main := e.Index.Branches[mainBranchID]
	assert.Equal(t, originalHead, main.HeadCheckpointID,
		"fork must not advance main branch head")
}

func TestSaveCheckpoint_MultipleForks(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 3)

	// Fork 1 from T1
	_, err := e.GotoCheckpoint(saved[0].ID)
	require.NoError(t, err)
	fork1, err := e.SaveCheckpoint("fork-1", "snap-f1")
	require.NoError(t, err)

	// Fork 2 from T2
	_, err = e.GotoCheckpoint(saved[1].ID)
	require.NoError(t, err)
	fork2, err := e.SaveCheckpoint("fork-2", "snap-f2")
	require.NoError(t, err)

	assert.NotEqual(t, fork1.BranchID, fork2.BranchID, "each fork must create a distinct branch")

	// Total branches: main + 2 forks
	assert.Len(t, e.Index.Branches, 3)
}

// ─── ListCheckpoints ─────────────────────────────────────────────────────────

func TestListCheckpoints_CurrentBranch_WhenBranchIDEmpty(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 3)

	list, err := e.ListCheckpoints("")
	require.NoError(t, err)

	// 3 saved + root = 4
	assert.Len(t, list, 4)
	assert.Equal(t, saved[2].ID, list[0].ID)
}

func TestListCheckpoints_ForkedBranch_ContainsOnlyItsCheckpoints(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 3)

	_, err := e.GotoCheckpoint(saved[1].ID)
	require.NoError(t, err)
	fork, err := e.SaveCheckpoint("fork cp", "snap-fork")
	require.NoError(t, err)

	// Extend the fork with two more checkpoints
	ext1, err := e.SaveCheckpoint("fork ext 1", "snap-fe1")
	require.NoError(t, err)
	_, err = e.SaveCheckpoint("fork ext 2", "snap-fe2")
	require.NoError(t, err)

	forkList, err := e.ListCheckpoints(fork.BranchID)
	require.NoError(t, err)

	ids := make([]string, len(forkList))
	for i, c := range forkList {
		ids[i] = c.ID
	}

	assert.Equal(t, e.Index.Branches[fork.BranchID].HeadCheckpointID, ids[0],
		"head of fork branch must be first in list")
	assert.Contains(t, ids, ext1.ID)
	assert.Contains(t, ids, fork.ID)
}

func TestListCheckpoints_UnknownBranch_ReturnsError(t *testing.T) {
	e := initEngine(t)
	_, err := e.ListCheckpoints("no-such-branch")
	assert.ErrorIs(t, err, timeline.ErrBranchNotFound)
}

// ─── GetAncestors ─────────────────────────────────────────────────────────────

func TestGetAncestors_LinearChain(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 4)

	ancestors, err := e.GetAncestors(saved[3].ID)
	require.NoError(t, err)

	// T4's ancestors: T3, T2, T1, root
	require.Len(t, ancestors, 4)
	assert.Equal(t, saved[2].ID, ancestors[0].ID, "first ancestor is direct parent")
	assert.Equal(t, saved[0].ID, ancestors[2].ID)
}

func TestGetAncestors_Root_HasNoAncestors(t *testing.T) {
	e := initEngine(t)
	rootID := e.Index.CurrentCheckpointID

	ancestors, err := e.GetAncestors(rootID)
	require.NoError(t, err)
	assert.Empty(t, ancestors)
}

func TestGetAncestors_CrossesBranchBoundary(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 2)

	_, err := e.GotoCheckpoint(saved[0].ID)
	require.NoError(t, err)
	fork, err := e.SaveCheckpoint("fork", "snap-f")
	require.NoError(t, err)

	// fork's ancestry: saved[0], root
	ancestors, err := e.GetAncestors(fork.ID)
	require.NoError(t, err)

	ids := make(map[string]bool)
	for _, a := range ancestors {
		ids[a.ID] = true
	}
	assert.True(t, ids[saved[0].ID], "fork ancestor must include T1 from main branch")
}

func TestGetAncestors_UnknownID_ReturnsError(t *testing.T) {
	e := initEngine(t)
	_, err := e.GetAncestors("unknown-id")
	assert.ErrorIs(t, err, timeline.ErrCheckpointNotFound)
}

// ─── FindCommonAncestor ───────────────────────────────────────────────────────

func TestFindCommonAncestor_ForkedBranches(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 3) // T1 T2 T3 on main

	// Fork from T2
	_, err := e.GotoCheckpoint(saved[1].ID)
	require.NoError(t, err)
	fork, err := e.SaveCheckpoint("fork", "snap-f")
	require.NoError(t, err)
	forkExt, err := e.SaveCheckpoint("fork ext", "snap-fe")
	require.NoError(t, err)

	// LCA of T3 (main) and forkExt should be T2 (the fork point)
	lca, err := e.FindCommonAncestor(saved[2].ID, forkExt.ID)
	require.NoError(t, err)
	assert.Equal(t, saved[1].ID, lca.ID,
		"LCA must be T2, the checkpoint where branches diverged")
	_ = fork
}

func TestFindCommonAncestor_SameNode_ReturnsSelf(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 2)

	lca, err := e.FindCommonAncestor(saved[1].ID, saved[1].ID)
	require.NoError(t, err)
	assert.Equal(t, saved[1].ID, lca.ID)
}

func TestFindCommonAncestor_DirectAncestor(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 3)

	// LCA(T3, T1) should be T1 because T1 is an ancestor of T3
	lca, err := e.FindCommonAncestor(saved[2].ID, saved[0].ID)
	require.NoError(t, err)
	assert.Equal(t, saved[0].ID, lca.ID)
}

func TestFindCommonAncestor_UnknownID_ReturnsError(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 1)
	_, err := e.FindCommonAncestor(saved[0].ID, "ghost-id")
	assert.ErrorIs(t, err, timeline.ErrCheckpointNotFound)
}

// ─── GetDAG ───────────────────────────────────────────────────────────────────

func TestGetDAG_ReturnsAllBranches(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 3)

	_, err := e.GotoCheckpoint(saved[0].ID)
	require.NoError(t, err)
	_, err = e.SaveCheckpoint("fork A", "snap-a")
	require.NoError(t, err)

	_, err = e.GotoCheckpoint(saved[1].ID)
	require.NoError(t, err)
	_, err = e.SaveCheckpoint("fork B", "snap-b")
	require.NoError(t, err)

	dag := e.GetDAG()
	// main + 2 forks = 3 branches
	assert.Len(t, dag, 3)
}

func TestGetDAG_EachBranchHasCorrectCheckpoints(t *testing.T) {
	e := initEngine(t)
	saved := saveN(t, e, 2)

	mainBranchID := saved[0].BranchID

	_, err := e.GotoCheckpoint(saved[0].ID)
	require.NoError(t, err)
	fork, err := e.SaveCheckpoint("fork", "snap-f")
	require.NoError(t, err)

	dag := e.GetDAG()

	mainCPs := dag[mainBranchID]
	mainIDs := make(map[string]bool)
	for _, c := range mainCPs {
		mainIDs[c.ID] = true
	}
	assert.True(t, mainIDs[saved[0].ID])
	assert.True(t, mainIDs[saved[1].ID])
	assert.False(t, mainIDs[fork.ID], "fork checkpoint must not appear on main branch")
}
