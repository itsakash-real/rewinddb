package snapshot

import (
	"container/list"
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"

	"github.com/itsakash-real/nimbi/internal/timeline"
	"github.com/rs/zerolog/log"
)

const defaultCacheCapacity = 10

// snapshotCache is a thread-safe LRU cache for *timeline.Snapshot values.
// It uses container/list for O(1) eviction [web:130] and singleflight to
// deduplicate concurrent loads for the same hash [web:132][web:135].
type snapshotCache struct {
	mu      sync.Mutex
	cap     int
	list    *list.List
	items   map[string]*list.Element
	sfGroup singleflight.Group
}

// cacheEntry is the value stored in each list.Element.
type cacheEntry struct {
	hash string
	snap *timeline.Snapshot
}

// newSnapshotCache returns a cache holding up to capacity snapshots.
func newSnapshotCache(capacity int) *snapshotCache {
	if capacity <= 0 {
		capacity = defaultCacheCapacity
	}
	return &snapshotCache{
		cap:   capacity,
		list:  list.New(),
		items: make(map[string]*list.Element, capacity),
	}
}

// get returns the cached snapshot and true, or nil and false on miss.
func (c *snapshotCache) get(hash string) (*timeline.Snapshot, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[hash]
	if !ok {
		return nil, false
	}
	// Move to front = mark as recently used.
	c.list.MoveToFront(el)
	return el.Value.(*cacheEntry).snap, true
}

// put stores snap under hash, evicting the LRU entry if at capacity.
func (c *snapshotCache) put(hash string, snap *timeline.Snapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Already cached — just refresh position.
	if el, ok := c.items[hash]; ok {
		c.list.MoveToFront(el)
		return
	}

	// Evict LRU entry if at capacity.
	if c.list.Len() >= c.cap {
		lru := c.list.Back()
		if lru != nil {
			entry := lru.Value.(*cacheEntry)
			log.Debug().Str("evicted", entry.hash[:8]).Msg("snapshot cache: evict LRU")
			c.list.Remove(lru)
			delete(c.items, entry.hash)
		}
	}

	el := c.list.PushFront(&cacheEntry{hash: hash, snap: snap})
	c.items[hash] = el
}

// invalidate removes a hash from the cache if present.
func (c *snapshotCache) invalidate(hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[hash]; ok {
		c.list.Remove(el)
		delete(c.items, hash)
	}
}

// Len returns the current number of cached snapshots.
func (c *snapshotCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.list.Len()
}

// globalSnapshotCache is the process-wide LRU used by Scanner.Load.
// Keeping it package-level means all Scanner instances share cache hits.
var globalSnapshotCache = newSnapshotCache(defaultCacheCapacity)

// LoadCached loads a snapshot by hash, returning the cached copy if available.
// Concurrent loads for the same hash are collapsed via singleflight [web:132].
func (s *Scanner) LoadCached(hash string, loader func(string) (*timeline.Snapshot, error)) (*timeline.Snapshot, error) {
	// Fast path: already in cache.
	if snap, ok := globalSnapshotCache.get(hash); ok {
		log.Debug().Str("hash", hash[:8]).Msg("snapshot cache: hit")
		return snap, nil
	}

	// Slow path: collapse concurrent loads for the same hash [web:135].
	val, err, shared := globalSnapshotCache.sfGroup.Do(hash, func() (interface{}, error) {
		snap, err := loader(hash)
		if err != nil {
			return nil, err
		}
		globalSnapshotCache.put(hash, snap)
		return snap, nil
	})

	if err != nil {
		return nil, err
	}
	_ = shared // log or metric: shared=true means another goroutine's result was reused
	snap, ok := val.(*timeline.Snapshot)
	if !ok {
		return nil, fmt.Errorf("snapshot cache: unexpected type %T from singleflight", val)
	}
	return snap, nil
}

// CacheStats returns a snapshot of the current cache state for observability.
type CacheStats struct {
	Size     int
	Capacity int
}

func GetCacheStats() CacheStats {
	return CacheStats{
		Size:     globalSnapshotCache.Len(),
		Capacity: globalSnapshotCache.cap,
	}
}
