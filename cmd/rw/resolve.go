package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/itsakash-real/nimbi/internal/timeline"
)

// resolveCheckpoint resolves a checkpoint reference to a Checkpoint value.
//
// Supported reference formats (evaluated in order):
//
//  1. "HEAD"         → current checkpoint
//  2. "HEAD~N"       → N checkpoints back along ParentID chain
//  3. "S3", "s12"    → sequential snapshot number
//  4. Tag name       → checkpoint whose Tags slice contains the name
//  5. Full UUID      → exact match in index
//  6. 8-char prefix  → unique prefix match
func resolveCheckpoint(engine *timeline.TimelineEngine, ref string) (timeline.Checkpoint, error) {
	upper := strings.ToUpper(ref)

	// ── 1. HEAD alias ─────────────────────────────────────────────────────────
	if upper == "HEAD" {
		id := engine.Index.CurrentCheckpointID
		if id == "" {
			return timeline.Checkpoint{}, fmt.Errorf("HEAD is not set (no checkpoint saved yet)")
		}
		cp, ok := engine.Index.Checkpoints[id]
		if !ok {
			return timeline.Checkpoint{}, fmt.Errorf("HEAD points to missing checkpoint %q", id)
		}
		return cp, nil
	}

	// ── 2. HEAD~N ─────────────────────────────────────────────────────────────
	if strings.HasPrefix(upper, "HEAD~") {
		nStr := ref[5:] // preserve original case for the number
		n, err := strconv.Atoi(nStr)
		if err != nil || n < 0 {
			return timeline.Checkpoint{}, fmt.Errorf("invalid HEAD~N reference: %q", ref)
		}
		return resolveHeadTilde(engine, n)
	}

	// ── 3. S-number (e.g. "S3", "s12") ───────────────────────────────────────
	if id := engine.Index.ResolveSNumber(ref); id != "" {
		if cp, ok := engine.Index.Checkpoints[id]; ok {
			return cp, nil
		}
	}

	// ── 4. Tag name ───────────────────────────────────────────────────────────
	for _, cp := range engine.Index.Checkpoints {
		for _, tag := range cp.Tags {
			if tag == ref {
				return cp, nil
			}
		}
	}

	// ── 5. Exact full ID ──────────────────────────────────────────────────────
	if cp, ok := engine.Index.Checkpoints[ref]; ok {
		return cp, nil
	}

	// ── 6. Prefix match ───────────────────────────────────────────────────────
	var matches []timeline.Checkpoint
	for id, cp := range engine.Index.Checkpoints {
		if strings.HasPrefix(id, ref) {
			matches = append(matches, cp)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return timeline.Checkpoint{}, fmt.Errorf(
			"no checkpoint found for reference %q\n"+
				"  Hint: use 'rw list' to see valid IDs, tags, or try HEAD~N", ref)
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = shortID(m.ID)
		}
		return timeline.Checkpoint{}, fmt.Errorf(
			"ambiguous prefix %q — matches: %s\n  Use more characters to disambiguate",
			ref, strings.Join(ids, ", "))
	}
}

// resolveHeadTilde walks n steps back from HEAD along the ParentID chain.
func resolveHeadTilde(engine *timeline.TimelineEngine, n int) (timeline.Checkpoint, error) {
	curID := engine.Index.CurrentCheckpointID
	if curID == "" {
		return timeline.Checkpoint{}, fmt.Errorf("HEAD is not set")
	}

	visited := make(map[string]struct{})
	for i := 0; i < n; i++ {
		cp, ok := engine.Index.Checkpoints[curID]
		if !ok {
			return timeline.Checkpoint{}, fmt.Errorf(
				"HEAD~%d: reached missing checkpoint at step %d", n, i)
		}
		if _, seen := visited[curID]; seen {
			return timeline.Checkpoint{}, fmt.Errorf(
				"HEAD~%d: cycle detected at step %d", n, i)
		}
		visited[curID] = struct{}{}

		if cp.ParentID == "" {
			return timeline.Checkpoint{}, fmt.Errorf(
				"HEAD~%d: only %d ancestor(s) exist (reached root at step %d)", n, i, i)
		}
		curID = cp.ParentID
	}

	cp, ok := engine.Index.Checkpoints[curID]
	if !ok {
		return timeline.Checkpoint{}, fmt.Errorf("HEAD~%d: checkpoint %q not found", n, curID)
	}
	return cp, nil
}
