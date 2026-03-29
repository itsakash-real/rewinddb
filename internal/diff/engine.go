package diff

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/aymanbagabas/go-udiff"
	"github.com/itsakash-real/nimbi/internal/snapshot"
	"github.com/itsakash-real/nimbi/internal/storage"
	"github.com/itsakash-real/nimbi/internal/timeline"
	"github.com/rs/zerolog/log"
)

// ─── ANSI colour constants (zero external dependencies) ───────────────────────
// Sourced from standard ANSI escape sequences [web:81].
const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiBold   = "\033[1m"
)

// ─── Result types ─────────────────────────────────────────────────────────────

// FileDiff holds the before/after metadata for a single modified file.
type FileDiff struct {
	Path    string
	OldHash string
	NewHash string
	OldSize int64
	NewSize int64
}

// SizeDelta returns the signed byte difference (positive = grew, negative = shrank).
func (fd FileDiff) SizeDelta() int64 { return fd.NewSize - fd.OldSize }

// DiffResult is the complete output of comparing two snapshots.
type DiffResult struct {
	Added     []timeline.FileEntry // present in B, absent in A
	Removed   []timeline.FileEntry // present in A, absent in B
	Modified  []FileDiff           // present in both, hash differs
	Unchanged []timeline.FileEntry // present in both, hash identical
}

// TotalChanges returns the count of non-unchanged entries.
func (r *DiffResult) TotalChanges() int {
	return len(r.Added) + len(r.Removed) + len(r.Modified)
}

// ─── Engine ───────────────────────────────────────────────────────────────────

// Engine computes deltas between Snapshot values. The embedded ObjectStore is
// used only for TextDiff and CompareCheckpoints — Compare itself is pure
// in-memory metadata work.
type Engine struct {
	Store *storage.ObjectStore
}

// New returns an Engine backed by store.
func New(store *storage.ObjectStore) *Engine {
	return &Engine{Store: store}
}

// ─── Compare ──────────────────────────────────────────────────────────────────

// Compare computes the structural diff between two snapshots.
// It builds a path-keyed map for each snapshot then classifies every file into
// Added, Removed, Modified, or Unchanged.
func (e *Engine) Compare(snapA, snapB *timeline.Snapshot) (*DiffResult, error) {
	if snapA == nil {
		return nil, fmt.Errorf("diff.Compare: snapA is nil")
	}
	if snapB == nil {
		return nil, fmt.Errorf("diff.Compare: snapB is nil")
	}

	mapA := fileMap(snapA.Files)
	mapB := fileMap(snapB.Files)

	result := &DiffResult{}

	// Files in A → check if they exist in B.
	for path, entA := range mapA {
		entB, inB := mapB[path]
		if !inB {
			result.Removed = append(result.Removed, entA)
			continue
		}
		if entA.Hash != entB.Hash {
			result.Modified = append(result.Modified, FileDiff{
				Path:    path,
				OldHash: entA.Hash,
				NewHash: entB.Hash,
				OldSize: entA.Size,
				NewSize: entB.Size,
			})
		} else {
			result.Unchanged = append(result.Unchanged, entA)
		}
	}

	// Files in B that are not in A → Added.
	for path, entB := range mapB {
		if _, inA := mapA[path]; !inA {
			result.Added = append(result.Added, entB)
		}
	}

	// Sort all slices for deterministic output.
	sort.Slice(result.Added, func(i, j int) bool { return result.Added[i].Path < result.Added[j].Path })
	sort.Slice(result.Removed, func(i, j int) bool { return result.Removed[i].Path < result.Removed[j].Path })
	sort.Slice(result.Modified, func(i, j int) bool { return result.Modified[i].Path < result.Modified[j].Path })
	sort.Slice(result.Unchanged, func(i, j int) bool { return result.Unchanged[i].Path < result.Unchanged[j].Path })

	log.Debug().
		Int("added", len(result.Added)).
		Int("removed", len(result.Removed)).
		Int("modified", len(result.Modified)).
		Int("unchanged", len(result.Unchanged)).
		Msg("diff computed")

	return result, nil
}

// ─── Summary ──────────────────────────────────────────────────────────────────

// Summary returns a compact one-line human-readable description.
// Example: "3 added, 1 removed, 5 modified, 42 unchanged"
func (e *Engine) Summary(result *DiffResult) string {
	return fmt.Sprintf("%d added, %d removed, %d modified, %d unchanged",
		len(result.Added),
		len(result.Removed),
		len(result.Modified),
		len(result.Unchanged),
	)
}

// ─── PrettyPrint ──────────────────────────────────────────────────────────────

// PrettyPrint renders a colour-coded terminal report of the diff:
//
//	[+] added/file.go              (green)
//	[-] removed/file.go            (red)
//	[~] modified/file.go  +120 B   (yellow)
//
// Each section is preceded by a bold header. ANSI codes are always emitted;
// callers that write to a non-TTY should strip them with a writer wrapper.
func (e *Engine) PrettyPrint(result *DiffResult) string {
	var sb strings.Builder

	if len(result.Added) > 0 {
		fmt.Fprintf(&sb, "%s%sAdded (%d)%s\n", ansiBold, ansiGreen, len(result.Added), ansiReset)
		for _, f := range result.Added {
			fmt.Fprintf(&sb, "%s[+] %s%s\n", ansiGreen, f.Path, ansiReset)
		}
	}

	if len(result.Removed) > 0 {
		fmt.Fprintf(&sb, "%s%sRemoved (%d)%s\n", ansiBold, ansiRed, len(result.Removed), ansiReset)
		for _, f := range result.Removed {
			fmt.Fprintf(&sb, "%s[-] %s%s\n", ansiRed, f.Path, ansiReset)
		}
	}

	if len(result.Modified) > 0 {
		fmt.Fprintf(&sb, "%s%sModified (%d)%s\n", ansiBold, ansiYellow, len(result.Modified), ansiReset)
		for _, fd := range result.Modified {
			delta := fd.SizeDelta()
			sign := "+"
			if delta < 0 {
				sign = ""
			}
			fmt.Fprintf(&sb, "%s[~] %s  (%s%d B)%s\n",
				ansiYellow, fd.Path, sign, delta, ansiReset)
		}
	}

	if result.TotalChanges() == 0 {
		fmt.Fprintf(&sb, "No changes between snapshots.\n")
	}

	return sb.String()
}

// ─── TextDiff ─────────────────────────────────────────────────────────────────

// TextDiff reads two objects from the store by hash and returns a unified diff
// string if both are valid UTF-8 text, or "binary files differ" otherwise [web:76].
func (e *Engine) TextDiff(hashA, hashB string) (string, error) {
	rawA, err := e.Store.Read(hashA)
	if err != nil {
		return "", fmt.Errorf("diff.TextDiff: read A (%s): %w", hashA, err)
	}
	rawB, err := e.Store.Read(hashB)
	if err != nil {
		return "", fmt.Errorf("diff.TextDiff: read B (%s): %w", hashB, err)
	}

	if !utf8.Valid(rawA) || !utf8.Valid(rawB) {
		return "binary files differ", nil
	}

	textA := string(rawA)
	textB := string(rawB)

	// go-udiff implements Myers' algorithm and emits standard unified diff [web:76].
	unified := udiff.Unified(
		fmt.Sprintf("a/%s", hashA[:8]),
		fmt.Sprintf("b/%s", hashB[:8]),
		textA,
		textB,
	)

	if unified == "" {
		return "(files are identical)", nil
	}
	return unified, nil
}

// ─── CompareCheckpoints ───────────────────────────────────────────────────────

// CompareCheckpoints loads the two snapshots referenced by the given
// checkpoints via scanner, then delegates to Compare.
func (e *Engine) CompareCheckpoints(
	cpA, cpB *timeline.Checkpoint,
	sc *snapshot.Scanner,
) (*DiffResult, error) {
	snapA, err := sc.Load(cpA.SnapshotRef)
	if err != nil {
		return nil, fmt.Errorf("diff.CompareCheckpoints: load snapshot A (%s): %w", cpA.SnapshotRef, err)
	}
	snapB, err := sc.Load(cpB.SnapshotRef)
	if err != nil {
		return nil, fmt.Errorf("diff.CompareCheckpoints: load snapshot B (%s): %w", cpB.SnapshotRef, err)
	}
	return e.Compare(snapA, snapB)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// fileMap converts a []FileEntry slice into a map keyed by path for O(1) lookup.
func fileMap(files []timeline.FileEntry) map[string]timeline.FileEntry {
	m := make(map[string]timeline.FileEntry, len(files))
	for _, f := range files {
		m[f.Path] = f
	}
	return m
}
