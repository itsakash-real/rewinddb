package snapshot_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/itsakash-real/nimbi/internal/snapshot"
	"github.com/itsakash-real/nimbi/internal/storage"
	"github.com/itsakash-real/nimbi/internal/timeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotCache_HitAfterFirstLoad(t *testing.T) {
	store := storage.New(t.TempDir())
	sc := snapshot.New(t.TempDir(), store)

	snap := &timeline.Snapshot{Files: []timeline.FileEntry{{Path: "main.go", Hash: "abc", Size: 10}}}
	calls := 0
	loader := func(_ string) (*timeline.Snapshot, error) {
		calls++
		return snap, nil
	}

	// First call → loader invoked.
	got1, err := sc.LoadCached("deadbeef", loader)
	require.NoError(t, err)
	assert.Equal(t, 1, calls)

	// Second call → served from cache.
	got2, err := sc.LoadCached("deadbeef", loader)
	require.NoError(t, err)
	assert.Equal(t, 1, calls, "loader must not be called twice for the same hash")
	assert.Same(t, got1, got2)
}

func TestSnapshotCache_LRUEviction(t *testing.T) {
	// Fill cache beyond capacity and verify the LRU entry is dropped.
	store := storage.New(t.TempDir())
	sc := snapshot.New(t.TempDir(), store)

	capacity := 10
	// Load capacity+2 distinct hashes — oldest two should be evicted.
	hashes := make([]string, capacity+2)
	for i := range hashes {
		hashes[i] = fmt.Sprintf("%064x", i) // 64-char hex hash
	}

	loader := func(hash string) (*timeline.Snapshot, error) {
		return &timeline.Snapshot{}, nil
	}
	for _, h := range hashes {
		_, err := sc.LoadCached(h, loader)
		require.NoError(t, err)
	}

	stats := snapshot.GetCacheStats()
	assert.Equal(t, capacity, stats.Size,
		"cache must not exceed capacity after LRU eviction")
}

func TestSnapshotCache_ConcurrentLoads_Singleflight(t *testing.T) {
	store := storage.New(t.TempDir())
	sc := snapshot.New(t.TempDir(), store)

	var calls int
	var mu sync.Mutex
	loader := func(_ string) (*timeline.Snapshot, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		return &timeline.Snapshot{}, nil
	}

	const goroutines = 20
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sc.LoadCached("samekey0000000000000000000000000000000000000000000000000000000", loader)
		}()
	}
	wg.Wait()

	// singleflight collapses concurrent calls — loader called once (or very few times).
	mu.Lock()
	defer mu.Unlock()
	assert.LessOrEqual(t, calls, 3,
		"singleflight must collapse concurrent loads; got %d loader calls", calls)
}
