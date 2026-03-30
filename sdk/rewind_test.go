package sdk_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/itsakash-real/nimbi/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
	require.NoError(t, os.WriteFile(abs, []byte(content), 0o644))
}

func initClient(t *testing.T) (string, *sdk.Client) {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main")
	writeFile(t, root, "go.mod", "module example")
	c, err := sdk.Init(root)
	require.NoError(t, err)
	return root, c
}

// ─── Init / New ───────────────────────────────────────────────────────────────

func TestInit_CreatesRepository(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main")
	c, err := sdk.Init(root)
	require.NoError(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, root, c.ProjectRoot)
}

func TestNew_FailsOnUninitializedDirectory(t *testing.T) {
	root := t.TempDir()
	_, err := sdk.New(root)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a Nimbi repository")
}

func TestNew_LoadsExistingRepository(t *testing.T) {
	root, _ := initClient(t)
	c, err := sdk.New(root)
	require.NoError(t, err)
	assert.Equal(t, root, c.ProjectRoot)
}

func TestMustNew_PanicsOnBadPath(t *testing.T) {
	root := t.TempDir() // not initialised
	assert.Panics(t, func() { sdk.MustNew(root) })
}

// ─── Save ─────────────────────────────────────────────────────────────────────

func TestSave_ReturnsCheckpoint(t *testing.T) {
	_, c := initClient(t)
	cp, err := c.Save("first save")
	require.NoError(t, err)
	assert.NotEmpty(t, cp.ID)
	assert.Equal(t, "first save", cp.Message)
}

func TestSave_EmptyMessage_ReturnsError(t *testing.T) {
	_, c := initClient(t)
	_, err := c.Save("")
	assert.Error(t, err)
}

func TestSaveWithTags_AttachesTags(t *testing.T) {
	_, c := initClient(t)
	cp, err := c.SaveWithTags("tagged save", []string{"v1.0", "stable"})
	require.NoError(t, err)
	assert.Contains(t, cp.Tags, "v1.0")
	assert.Contains(t, cp.Tags, "stable")
}

func TestSave_ThreeCheckpoints_ParentChain(t *testing.T) {
	_, c := initClient(t)
	cp1, _ := c.Save("one")
	cp2, _ := c.Save("two")
	cp3, _ := c.Save("three")

	assert.Equal(t, cp1.ID, cp2.ParentID)
	assert.Equal(t, cp2.ID, cp3.ParentID)
}

// ─── Goto ─────────────────────────────────────────────────────────────────────

func TestGoto_ByPrefix_RestoresFiles(t *testing.T) {
	root, c := initClient(t)
	writeFile(t, root, "main.go", "package main // v1")
	cp1, err := c.Save("v1")
	require.NoError(t, err)

	writeFile(t, root, "main.go", "package main // v2")
	_, err = c.Save("v2")
	require.NoError(t, err)

	// Restore to v1 by prefix
	got, err := c.Goto(cp1.ID[:8])
	require.NoError(t, err)
	assert.Equal(t, cp1.ID, got.ID)

	// Verify file content
	data, err := os.ReadFile(filepath.Join(root, "main.go"))
	require.NoError(t, err)
	assert.Equal(t, "package main // v1", string(data))
}

func TestGoto_ByTagName(t *testing.T) {
	root, c := initClient(t)
	writeFile(t, root, "main.go", "v1 content")
	cp1, _ := c.Save("v1")
	require.NoError(t, c.Tag("release-1", cp1.ID))

	writeFile(t, root, "main.go", "v2 content")
	_, _ = c.Save("v2")

	_, err := c.Goto("release-1")
	require.NoError(t, err)

	data, _ := os.ReadFile(filepath.Join(root, "main.go"))
	assert.Equal(t, "v1 content", string(data))
}

func TestGoto_HeadTilde(t *testing.T) {
	root, c := initClient(t)
	writeFile(t, root, "a.go", "v1")
	_, _ = c.Save("v1")
	writeFile(t, root, "a.go", "v2")
	_, _ = c.Save("v2")
	writeFile(t, root, "a.go", "v3")
	_, _ = c.Save("v3")

	// Go back 2
	_, err := c.Goto("HEAD~2")
	require.NoError(t, err)

	data, _ := os.ReadFile(filepath.Join(root, "a.go"))
	assert.Equal(t, "v1", string(data))
}

// ─── List ─────────────────────────────────────────────────────────────────────

func TestList_DefaultBranch(t *testing.T) {
	_, c := initClient(t)
	c.Save("one")
	c.Save("two")
	c.Save("three")

	cps, err := c.List(sdk.ListOpts{})
	require.NoError(t, err)
	// 3 saves + 1 root = 4
	assert.Len(t, cps, 4)
}

func TestList_WithLimit(t *testing.T) {
	_, c := initClient(t)
	for i := 0; i < 5; i++ {
		c.Save(fmt.Sprintf("save %d", i))
	}
	cps, err := c.List(sdk.ListOpts{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, cps, 3)
}

// ─── Status ───────────────────────────────────────────────────────────────────

func TestStatus_CleanAfterSave(t *testing.T) {
	_, c := initClient(t)
	_, err := c.Save("initial")
	require.NoError(t, err)

	status, err := c.Status()
	require.NoError(t, err)
	assert.True(t, status.IsClean)
	assert.Empty(t, status.ModifiedFiles)
	assert.Empty(t, status.AddedFiles)
}

func TestStatus_DetectsModifiedFile(t *testing.T) {
	root, c := initClient(t)
	_, _ = c.Save("initial")

	writeFile(t, root, "main.go", "package main // modified")

	status, err := c.Status()
	require.NoError(t, err)
	assert.False(t, status.IsClean)
	assert.Contains(t, status.ModifiedFiles, "main.go")
}

func TestStatus_DetectsAddedFile(t *testing.T) {
	root, c := initClient(t)
	_, _ = c.Save("initial")
	writeFile(t, root, "new_file.go", "package new")

	status, err := c.Status()
	require.NoError(t, err)
	assert.Contains(t, status.AddedFiles, "new_file.go")
}

// ─── Diff ─────────────────────────────────────────────────────────────────────

func TestDiff_TwoCheckpoints(t *testing.T) {
	root, c := initClient(t)
	writeFile(t, root, "main.go", "v1")
	cp1, _ := c.Save("v1")

	writeFile(t, root, "main.go", "v2 modified")
	writeFile(t, root, "added.go", "new file")
	cp2, _ := c.Save("v2")

	result, err := c.Diff(cp1.ID, cp2.ID)
	require.NoError(t, err)
	assert.Len(t, result.Modified, 1)
	assert.Len(t, result.Added, 1)
	assert.Equal(t, "added.go", result.Added[0].Path)
}

func TestDiff_OneIDUsesHead(t *testing.T) {
	root, c := initClient(t)
	writeFile(t, root, "main.go", "v1")
	cp1, _ := c.Save("v1")

	writeFile(t, root, "main.go", "v2")
	_, _ = c.Save("v2")

	// Diff cp1 against HEAD
	result, err := c.Diff(cp1.ID, "")
	require.NoError(t, err)
	assert.Len(t, result.Modified, 1)
}

// ─── Branches ─────────────────────────────────────────────────────────────────

func TestCreateBranch_And_Switch(t *testing.T) {
	root, c := initClient(t)
	writeFile(t, root, "main.go", "main branch content")
	_, _ = c.Save("main save")

	b, err := c.CreateBranch("feature-x")
	require.NoError(t, err)
	assert.Equal(t, "feature-x", b.Name)

	writeFile(t, root, "feature.go", "feature content")
	_, _ = c.Save("feature save")

	// Switch back to main
	require.NoError(t, c.SwitchBranch("main"))
	assert.NoFileExists(t, filepath.Join(root, "feature.go"),
		"feature file must not exist on main branch")
}

func TestBranches_ListAll(t *testing.T) {
	_, c := initClient(t)
	_, _ = c.Save("save 1")
	c.CreateBranch("branch-a")
	c.CreateBranch("branch-b")

	branches, err := c.Branches()
	require.NoError(t, err)
	// main + branch-a + branch-b = 3
	assert.Len(t, branches, 3)
}

// ─── Tag ──────────────────────────────────────────────────────────────────────

func TestTag_AttachesToHead(t *testing.T) {
	_, c := initClient(t)
	cp, _ := c.Save("release")
	require.NoError(t, c.Tag("v1.0", ""))

	// Resolve by tag
	resolved, err := c.Goto("v1.0")
	require.NoError(t, err)
	assert.Equal(t, cp.ID, resolved.ID)
}

func TestTag_DuplicateTagDifferentCheckpoint_ReturnsError(t *testing.T) {
	_, c := initClient(t)
	cp1, _ := c.Save("one")
	_, _ = c.Save("two")

	require.NoError(t, c.Tag("mytag", cp1.ID))
	err := c.Tag("mytag", "") // HEAD = cp2
	assert.Error(t, err)
}

// ─── GC ───────────────────────────────────────────────────────────────────────

func TestGC_DryRun_DoesNotDelete(t *testing.T) {
	_, c := initClient(t)
	_, _ = c.Save("save 1")

	statusBefore, err := c.Status()
	require.NoError(t, err)

	result, err := c.GC(true)
	require.NoError(t, err)
	assert.True(t, result.DryRun)

	statusAfter, err := c.Status()
	require.NoError(t, err)

	assert.Equal(t, statusBefore.StorageStats.ObjectCount, statusAfter.StorageStats.ObjectCount, "dry-run must not delete objects")
}
