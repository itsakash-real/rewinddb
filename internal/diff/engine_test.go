package diff_test

import (
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/itsakash-real/nimbi/internal/diff"
	"github.com/itsakash-real/nimbi/internal/storage"
	"github.com/itsakash-real/nimbi/internal/timeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func newEngine(t *testing.T) *diff.Engine {
	t.Helper()
	store := storage.New(t.TempDir())
	return diff.New(store)
}

// entry builds a FileEntry for test snapshots.
func entry(path, hash string, size int64) timeline.FileEntry {
	return timeline.FileEntry{Path: path, Hash: hash, Size: size, Mode: fs.FileMode(0o644)}
}

// snap builds a minimal Snapshot from a list of entries.
func snap(entries ...timeline.FileEntry) *timeline.Snapshot {
	return &timeline.Snapshot{
		Hash:      "fake-hash",
		Files:     entries,
		CreatedAt: time.Now().UTC(),
	}
}

// ─── Compare: basic classification ───────────────────────────────────────────

func TestCompare_AllUnchanged(t *testing.T) {
	e := newEngine(t)
	files := []timeline.FileEntry{
		entry("main.go", "h1", 100),
		entry("go.mod", "h2", 50),
	}
	result, err := e.Compare(snap(files...), snap(files...))
	require.NoError(t, err)

	assert.Len(t, result.Unchanged, 2)
	assert.Empty(t, result.Added)
	assert.Empty(t, result.Removed)
	assert.Empty(t, result.Modified)
}

func TestCompare_AddedFiles(t *testing.T) {
	e := newEngine(t)
	snapA := snap(entry("main.go", "h1", 100))
	snapB := snap(
		entry("main.go", "h1", 100),
		entry("new_file.go", "h2", 200),
	)

	result, err := e.Compare(snapA, snapB)
	require.NoError(t, err)

	require.Len(t, result.Added, 1)
	assert.Equal(t, "new_file.go", result.Added[0].Path)
	assert.Len(t, result.Unchanged, 1)
	assert.Empty(t, result.Removed)
}

func TestCompare_RemovedFiles(t *testing.T) {
	e := newEngine(t)
	snapA := snap(
		entry("main.go", "h1", 100),
		entry("old_file.go", "h2", 80),
	)
	snapB := snap(entry("main.go", "h1", 100))

	result, err := e.Compare(snapA, snapB)
	require.NoError(t, err)

	require.Len(t, result.Removed, 1)
	assert.Equal(t, "old_file.go", result.Removed[0].Path)
	assert.Len(t, result.Unchanged, 1)
	assert.Empty(t, result.Added)
}

func TestCompare_ModifiedFiles(t *testing.T) {
	e := newEngine(t)
	snapA := snap(entry("main.go", "hash-old", 100))
	snapB := snap(entry("main.go", "hash-new", 180))

	result, err := e.Compare(snapA, snapB)
	require.NoError(t, err)

	require.Len(t, result.Modified, 1)
	fd := result.Modified[0]
	assert.Equal(t, "main.go", fd.Path)
	assert.Equal(t, "hash-old", fd.OldHash)
	assert.Equal(t, "hash-new", fd.NewHash)
	assert.Equal(t, int64(100), fd.OldSize)
	assert.Equal(t, int64(180), fd.NewSize)
	assert.Equal(t, int64(80), fd.SizeDelta())
}

func TestCompare_ShrunkFile_NegativeDelta(t *testing.T) {
	e := newEngine(t)
	snapA := snap(entry("file.go", "ha", 500))
	snapB := snap(entry("file.go", "hb", 200))

	result, err := e.Compare(snapA, snapB)
	require.NoError(t, err)
	require.Len(t, result.Modified, 1)
	assert.Equal(t, int64(-300), result.Modified[0].SizeDelta())
}

func TestCompare_MixedChanges(t *testing.T) {
	e := newEngine(t)
	snapA := snap(
		entry("unchanged.go", "u1", 10),
		entry("modified.go", "m-old", 100),
		entry("removed.go", "r1", 50),
	)
	snapB := snap(
		entry("unchanged.go", "u1", 10),
		entry("modified.go", "m-new", 150),
		entry("added.go", "a1", 75),
	)

	result, err := e.Compare(snapA, snapB)
	require.NoError(t, err)

	assert.Len(t, result.Unchanged, 1)
	assert.Len(t, result.Modified, 1)
	assert.Len(t, result.Removed, 1)
	assert.Len(t, result.Added, 1)
	assert.Equal(t, 3, result.TotalChanges())
}

func TestCompare_EmptyToPopulated(t *testing.T) {
	e := newEngine(t)
	snapA := snap()
	snapB := snap(
		entry("a.go", "h1", 10),
		entry("b.go", "h2", 20),
	)

	result, err := e.Compare(snapA, snapB)
	require.NoError(t, err)
	assert.Len(t, result.Added, 2)
	assert.Empty(t, result.Removed)
	assert.Empty(t, result.Modified)
}

func TestCompare_PopulatedToEmpty(t *testing.T) {
	e := newEngine(t)
	snapA := snap(entry("a.go", "h1", 10), entry("b.go", "h2", 20))
	snapB := snap()

	result, err := e.Compare(snapA, snapB)
	require.NoError(t, err)
	assert.Len(t, result.Removed, 2)
	assert.Empty(t, result.Added)
}

func TestCompare_BothEmpty(t *testing.T) {
	e := newEngine(t)
	result, err := e.Compare(snap(), snap())
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalChanges())
}

// ─── Compare: ordering ────────────────────────────────────────────────────────

func TestCompare_ResultsAreSortedByPath(t *testing.T) {
	e := newEngine(t)
	snapA := snap(
		entry("z.go", "h1", 10),
		entry("a.go", "h2", 20),
		entry("m.go", "h3", 30),
	)
	snapB := snap(
		entry("z.go", "h1-new", 10),
		entry("a.go", "h2-new", 20),
		entry("m.go", "h3-new", 30),
	)

	result, err := e.Compare(snapA, snapB)
	require.NoError(t, err)
	require.Len(t, result.Modified, 3)

	assert.Equal(t, "a.go", result.Modified[0].Path)
	assert.Equal(t, "m.go", result.Modified[1].Path)
	assert.Equal(t, "z.go", result.Modified[2].Path)
}

// ─── Compare: nil guards ──────────────────────────────────────────────────────

func TestCompare_NilSnapshotA_ReturnsError(t *testing.T) {
	e := newEngine(t)
	_, err := e.Compare(nil, snap())
	assert.Error(t, err)
}

func TestCompare_NilSnapshotB_ReturnsError(t *testing.T) {
	e := newEngine(t)
	_, err := e.Compare(snap(), nil)
	assert.Error(t, err)
}

// ─── Summary ──────────────────────────────────────────────────────────────────

func TestSummary_Format(t *testing.T) {
	e := newEngine(t)
	snapA := snap(
		entry("unchanged.go", "u1", 10),
		entry("modified.go", "m-old", 100),
		entry("removed.go", "r1", 50),
	)
	snapB := snap(
		entry("unchanged.go", "u1", 10),
		entry("modified.go", "m-new", 150),
		entry("added.go", "a1", 75),
	)
	result, err := e.Compare(snapA, snapB)
	require.NoError(t, err)

	summary := e.Summary(result)
	assert.Equal(t, "1 added, 1 removed, 1 modified, 1 unchanged", summary)
}

func TestSummary_NoChanges(t *testing.T) {
	e := newEngine(t)
	files := []timeline.FileEntry{entry("main.go", "h1", 100)}
	result, err := e.Compare(snap(files...), snap(files...))
	require.NoError(t, err)
	assert.Equal(t, "0 added, 0 removed, 0 modified, 1 unchanged", e.Summary(result))
}

// ─── PrettyPrint ──────────────────────────────────────────────────────────────

func TestPrettyPrint_ContainsSymbols(t *testing.T) {
	e := newEngine(t)
	snapA := snap(
		entry("existing.go", "h1", 100),
		entry("to_remove.go", "h2", 50),
	)
	snapB := snap(
		entry("existing.go", "h1-new", 120),
		entry("brand_new.go", "h3", 75),
	)
	result, err := e.Compare(snapA, snapB)
	require.NoError(t, err)

	output := e.PrettyPrint(result)
	assert.Contains(t, output, "[+]", "added files must show [+]")
	assert.Contains(t, output, "[-]", "removed files must show [-]")
	assert.Contains(t, output, "[~]", "modified files must show [~]")
	assert.Contains(t, output, "brand_new.go")
	assert.Contains(t, output, "to_remove.go")
	assert.Contains(t, output, "existing.go")
}

func TestPrettyPrint_NoChanges_PrintsMessage(t *testing.T) {
	e := newEngine(t)
	files := []timeline.FileEntry{entry("main.go", "h1", 100)}
	result, err := e.Compare(snap(files...), snap(files...))
	require.NoError(t, err)
	assert.Contains(t, e.PrettyPrint(result), "No changes")
}

func TestPrettyPrint_SizeDeltaShownForModified(t *testing.T) {
	e := newEngine(t)
	result := &diff.DiffResult{
		Modified: []diff.FileDiff{
			{Path: "big.go", OldHash: "ha", NewHash: "hb", OldSize: 100, NewSize: 350},
		},
	}
	output := e.PrettyPrint(result)
	// +250 B delta
	assert.Contains(t, output, "+250")
}

// ─── TextDiff ─────────────────────────────────────────────────────────────────

func TestTextDiff_TextFiles_ProducesUnifiedDiff(t *testing.T) {
	store := storage.New(t.TempDir())
	e := diff.New(store)

	contentA := "line one\nline two\nline three\n"
	contentB := "line one\nline TWO modified\nline three\nline four\n"

	hashA, err := store.Write([]byte(contentA))
	require.NoError(t, err)
	hashB, err := store.Write([]byte(contentB))
	require.NoError(t, err)

	result, err := e.TextDiff(hashA, hashB)
	require.NoError(t, err)

	// Unified diff must contain the standard header markers
	assert.Contains(t, result, "---")
	assert.Contains(t, result, "+++")
	assert.Contains(t, result, "@@")
	assert.Contains(t, result, "-line two")
	assert.Contains(t, result, "+line TWO modified")
}

func TestTextDiff_IdenticalContent_ReturnsIdenticalMessage(t *testing.T) {
	store := storage.New(t.TempDir())
	e := diff.New(store)

	content := "same content\n"
	hash, err := store.Write([]byte(content))
	require.NoError(t, err)

	result, err := e.TextDiff(hash, hash)
	require.NoError(t, err)
	assert.Contains(t, result, "identical")
}

func TestTextDiff_BinaryContent_ReturnsBinaryMessage(t *testing.T) {
	store := storage.New(t.TempDir())
	e := diff.New(store)

	// Invalid UTF-8 bytes simulate binary content
	binaryA := []byte{0x00, 0xFF, 0xFE, 0x80, 0x90}
	binaryB := []byte{0x00, 0xFE, 0xFF, 0x81, 0x91}

	hashA, err := store.Write(binaryA)
	require.NoError(t, err)
	hashB, err := store.Write(binaryB)
	require.NoError(t, err)

	result, err := e.TextDiff(hashA, hashB)
	require.NoError(t, err)
	assert.Equal(t, "binary files differ", result)
}

func TestTextDiff_MissingHash_ReturnsError(t *testing.T) {
	store := storage.New(t.TempDir())
	e := diff.New(store)

	content := "hello"
	hash, err := store.Write([]byte(content))
	require.NoError(t, err)

	_, err = e.TextDiff(hash, "0000000000000000000000000000000000000000000000000000000000000000")
	assert.Error(t, err)
}

// ─── FileDiff helpers ─────────────────────────────────────────────────────────

func TestFileDiff_SizeDelta_Positive(t *testing.T) {
	fd := diff.FileDiff{OldSize: 100, NewSize: 400}
	assert.Equal(t, int64(300), fd.SizeDelta())
}

func TestFileDiff_SizeDelta_Negative(t *testing.T) {
	fd := diff.FileDiff{OldSize: 400, NewSize: 150}
	assert.Equal(t, int64(-250), fd.SizeDelta())
}

func TestFileDiff_SizeDelta_Zero(t *testing.T) {
	fd := diff.FileDiff{OldSize: 200, NewSize: 200}
	assert.Equal(t, int64(0), fd.SizeDelta())
}

// ─── TotalChanges ─────────────────────────────────────────────────────────────

func TestDiffResult_TotalChanges(t *testing.T) {
	result := &diff.DiffResult{
		Added:     make([]timeline.FileEntry, 3),
		Removed:   make([]timeline.FileEntry, 1),
		Modified:  make([]diff.FileDiff, 5),
		Unchanged: make([]timeline.FileEntry, 42),
	}
	assert.Equal(t, 9, result.TotalChanges())
}

// ─── PrettyPrint: ANSI codes present ─────────────────────────────────────────

func TestPrettyPrint_ContainsANSICodes(t *testing.T) {
	e := newEngine(t)
	snapA := snap(entry("x.go", "ha", 10))
	snapB := snap(entry("x.go", "hb", 20))
	result, err := e.Compare(snapA, snapB)
	require.NoError(t, err)

	output := e.PrettyPrint(result)
	// ANSI reset sequence must be present
	assert.True(t, strings.Contains(output, "\033[0m"), "output must contain ANSI reset code")
}
