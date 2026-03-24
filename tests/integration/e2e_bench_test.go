//go:build integration
// +build integration

package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/itsakash-real/rewinddb/internal/config"
	"github.com/itsakash-real/rewinddb/internal/snapshot"
	"github.com/itsakash-real/rewinddb/internal/storage"
	"github.com/itsakash-real/rewinddb/internal/timeline"
	"github.com/stretchr/testify/require"
)

// BenchmarkE2E_Save_500Files is a coarse integration-level benchmark that
// exercises the full stack (scan → hash → store → index) on 500 files.
func BenchmarkE2E_Save_500Files(b *testing.B) {
	r := initRepo(b)
	makeNFilesBench(b, r, 500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snap, err := r.scanner.Scan()
		if err != nil {
			b.Fatal(err)
		}
		h, err := r.scanner.Save(snap)
		if err != nil {
			b.Fatal(err)
		}
		_, err = r.engine.SaveCheckpoint(fmt.Sprintf("bench %d", i), h)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkE2E_Goto_DeltaRestore benchmarks goto when ~10% of files changed.
func BenchmarkE2E_Goto_DeltaRestore(b *testing.B) {
	r := initRepo(b)
	makeNFilesBench(b, r, 500)

	snapBase, _ := r.scanner.Scan()
	h1, _ := r.scanner.Save(snapBase)
	cp1, _ := r.engine.SaveCheckpoint("base", h1)
	r.engine.Index.Save(r.cfg.IndexPath)

	// Change 50 files.
	for i := 0; i < 50; i++ {
		r.write(b, fmt.Sprintf("gen/sub%03d/gen_%04d.go", i/50, i),
			fmt.Sprintf("package gen\n// changed %d\n", i))
	}
	snapV2, _ := r.scanner.Scan()
	h2, _ := r.scanner.Save(snapV2)
	cp2, _ := r.engine.SaveCheckpoint("v2", h2)
	r.engine.Index.Save(r.cfg.IndexPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			cpTarget := cp1
			snap, _ := r.scanner.Load(cpTarget.SnapshotRef)
			r.scanner.Restore(snap)
			r.engine.GotoCheckpoint(cpTarget.ID)
		} else {
			snap, _ := r.scanner.Load(cp2.SnapshotRef)
			r.scanner.Restore(snap)
			r.engine.GotoCheckpoint(cp2.ID)
		}
	}
}

// BenchmarkE2E_Status_Warm benchmarks fast mtime-based status after first scan.
func BenchmarkE2E_Status_Warm(b *testing.B) {
	r := initRepo(b)
	makeNFilesBench(b, r, 500)

	// Warm the last-scan cache.
	r.scanner.FastScan(r.cfg.RewindDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := r.scanner.FastScan(r.cfg.RewindDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ─── benchmark helpers ────────────────────────────────────────────────────────

type benchRepo struct {
	*repo
}

func initRepo(tb testing.TB) *repo {
	// Wrapper that works for both *testing.T and *testing.B.
	switch v := tb.(type) {
	case *testing.T:
		return initRepoT(v)
	case *testing.B:
		return initRepoBench(v)
	default:
		panic("unsupported testing.TB type")
	}
}

func initRepoBench(b *testing.B) *repo {
	b.Helper()
	// Reuse the same initRepo logic but with b.TempDir.
	root := b.TempDir()
	engine, err := timeline.Init(root)
	if err != nil {
		b.Fatalf("timeline.Init: %v", err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		b.Fatalf("config.Load: %v", err)
	}
	store := storage.New(cfg.ObjectsDir)
	sc := snapshot.New(root, store)
	return &repo{root: root, cfg: cfg, store: store, scanner: sc, engine: engine}
}

func makeNFilesBench(b *testing.B, r *repo, n int) {
	b.Helper()
	for i := 0; i < n; i++ {
		subdir := fmt.Sprintf("gen/sub%03d", i/50)
		content := fmt.Sprintf("package gen\n\nconst Gen%04d = %d\n", i, i)
		abs := filepath.Join(r.root, filepath.FromSlash(subdir),
			fmt.Sprintf("gen_%04d.go", i))
		os.MkdirAll(filepath.Dir(abs), 0o755)
		os.WriteFile(abs, []byte(content), 0o644)
	}
}

func initRepoT(t *testing.T) *repo {
	t.Helper()
	root := t.TempDir()
	engine, err := timeline.Init(root)
	require.NoError(t, err)
	cfg, err := config.Load(root)
	require.NoError(t, err)
	store := storage.New(cfg.ObjectsDir)
	sc := snapshot.New(root, store)
	return &repo{root: root, cfg: cfg, store: store, scanner: sc, engine: engine}
}

// ─── Performance target assertion helper ─────────────────────────────────────

type perfAssertion struct {
	label   string
	elapsed time.Duration
	limit   time.Duration
}

func assertPerf(t *testing.T, assertions ...perfAssertion) {
	t.Helper()
	for _, a := range assertions {
		if a.elapsed > a.limit {
			t.Errorf("PERF FAIL — %s: took %v, limit is %v", a.label, a.elapsed, a.limit)
		} else {
			t.Logf("PERF OK  — %s: %v (limit %v)", a.label, a.elapsed, a.limit)
		}
	}
}
