//go:build bench
// +build bench

// Run with: go test -tags bench -bench=. -benchtime=3s ./bench/
package bench_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/itsakash-real/nimbi/sdk"
)

// ─── Fixtures ─────────────────────────────────────────────────────────────────

// makeProject creates a temp project with n files of ~1 KB each and returns
// the root path + an initialised SDK Client.
func makeProject(b *testing.B, n int) (string, *sdk.Client) {
	b.Helper()
	root := b.TempDir()
	for i := 0; i < n; i++ {
		subdir := filepath.Join(root, fmt.Sprintf("pkg/mod%03d", i/50))
		os.MkdirAll(subdir, 0o755)
		content := make([]byte, 1024)
		for j := range content {
			content[j] = byte('a' + (i+j)%26)
		}
		os.WriteFile(filepath.Join(subdir, fmt.Sprintf("file_%04d.go", i)), content, 0o644)
	}
	c, err := sdk.Init(root)
	if err != nil {
		b.Fatalf("sdk.Init: %v", err)
	}
	return root, c
}

// ─── BenchmarkSave ────────────────────────────────────────────────────────────

// BenchmarkSave_1000Files measures time to scan + hash + store 1000 files.
// Target: < 500 ms on a modern laptop (NVMe SSD, 4+ cores) [file:122].
func BenchmarkSave_1000Files(b *testing.B) {
	root, c := makeProject(b, 1000)
	_ = root

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Save(fmt.Sprintf("bench checkpoint %d", i))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSave_1000Files_SingleWorker forces single-threaded hashing as
// a baseline to measure parallelism gain.
func BenchmarkSave_1000Files_SingleWorker(b *testing.B) {
	root, c := makeProject(b, 1000)
	_ = root
	c.SetScanWorkers(1) // exposed via SDK accessor

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Save(fmt.Sprintf("bench single %d", i))
	}
}

// BenchmarkSave_1000Files_MaxWorkers uses all CPUs.
func BenchmarkSave_1000Files_MaxWorkers(b *testing.B) {
	root, c := makeProject(b, 1000)
	_ = root
	c.SetScanWorkers(runtime.NumCPU())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Save(fmt.Sprintf("bench max %d", i))
	}
}

// ─── BenchmarkRestore ─────────────────────────────────────────────────────────

// BenchmarkRestore_1000Files measures full restore time.
// Target: < 200 ms for 1000 files (delta restore skips unchanged files) [file:122].
func BenchmarkRestore_1000Files(b *testing.B) {
	root, c := makeProject(b, 1000)

	// Save two checkpoints to have something to restore between.
	cp1, _ := c.Save("checkpoint 1")
	// Modify 100 files for cp2.
	for i := 0; i < 100; i++ {
		p := filepath.Join(root, fmt.Sprintf("pkg/mod%03d/file_%04d.go", i/50, i))
		os.WriteFile(p, []byte(fmt.Sprintf("modified content %d", i)), 0o644)
	}
	_, _ = c.Save("checkpoint 2")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Goto(cp1.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRestore_DeltaOnly measures how delta restore compares to full
// restore when most files are unchanged.
func BenchmarkRestore_DeltaOnly_10PercentChanged(b *testing.B) {
	root, c := makeProject(b, 1000)
	cp1, _ := c.Save("base")

	// Change only 10% of files.
	for i := 0; i < 100; i++ {
		p := filepath.Join(root, fmt.Sprintf("pkg/mod%03d/file_%04d.go", i/50, i))
		os.WriteFile(p, []byte("changed"), 0o644)
	}
	cp2, _ := c.Save("10pct changed")
	_ = cp2

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			c.Goto(cp2.ID)
		} else {
			c.Goto(cp1.ID)
		}
	}
}

// ─── BenchmarkStatus ──────────────────────────────────────────────────────────

// BenchmarkStatus_Cold measures status without the mtime cache (first run).
// Target: < 50 ms for 1000 files [file:122].
func BenchmarkStatus_Cold(b *testing.B) {
	_, c := makeProject(b, 1000)
	_, _ = c.Save("initial")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear the last-scan cache between iterations to simulate cold start.
		c.ClearScanCache()
		_, err := c.Status()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStatus_Warm measures status with the mtime cache populated
// (common case: no files changed since last scan).
func BenchmarkStatus_Warm(b *testing.B) {
	_, c := makeProject(b, 1000)
	_, _ = c.Save("initial")
	_, _ = c.Status() // warm the cache

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Status()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStatus_SnapshotCacheHit measures status when the snapshot is
// already in the LRU cache (multiple status calls in same process).
func BenchmarkStatus_SnapshotCacheHit(b *testing.B) {
	_, c := makeProject(b, 1000)
	_, _ = c.Save("initial")
	_, _ = c.Status() // populates both mtime and snapshot caches

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Status()
	}
}
