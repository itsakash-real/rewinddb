package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/itsakash-real/nimbi/internal/storage"
	"github.com/itsakash-real/nimbi/internal/timeline"
	"github.com/spf13/cobra"
)

func timeCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "time <expression>",
		Short: "Restore to a checkpoint matching a natural language time expression",
		Long: `Find and restore to a checkpoint near the described time.

Supported expressions:
  rw time "2 hours ago"
  rw time "yesterday 3pm"
  rw time "before I broke auth"

Time expressions are parsed as relative durations or fuzzy keywords.
Message-based search ("before I broke auth") searches checkpoint messages
for negative keywords (broke, broken, crash, error, bug, fail) near the
specified term, and picks the checkpoint just before the match.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			expr := args[0]

			// Try message-based search first ("before I broke auth").
			if strings.HasPrefix(strings.ToLower(expr), "before ") {
				return findBeforeMessage(r, expr[7:], yes)
			}

			// Try relative time parsing.
			targetTime, err := parseRelativeTime(expr)
			if err != nil {
				return fmt.Errorf("cannot parse time expression %q: %w\n"+
					"  Supported: \"2 hours ago\", \"yesterday 3pm\", \"before I broke auth\"", expr, err)
			}

			return findNearestCheckpoint(r, targetTime, yes)
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt")
	return cmd
}

// parseRelativeTime handles common relative time expressions.
func parseRelativeTime(expr string) (time.Time, error) {
	now := time.Now()
	lower := strings.ToLower(strings.TrimSpace(expr))

	// Handle "N hours/minutes/days ago"
	agoRe := regexp.MustCompile(`^(\d+)\s+(second|minute|hour|day|week)s?\s+ago$`)
	if m := agoRe.FindStringSubmatch(lower); m != nil {
		var n int
		fmt.Sscanf(m[1], "%d", &n)
		switch m[2] {
		case "second":
			return now.Add(-time.Duration(n) * time.Second), nil
		case "minute":
			return now.Add(-time.Duration(n) * time.Minute), nil
		case "hour":
			return now.Add(-time.Duration(n) * time.Hour), nil
		case "day":
			return now.AddDate(0, 0, -n), nil
		case "week":
			return now.AddDate(0, 0, -n*7), nil
		}
	}

	// Handle "yesterday" optionally with time.
	if strings.HasPrefix(lower, "yesterday") {
		yesterday := now.AddDate(0, 0, -1)
		rest := strings.TrimSpace(strings.TrimPrefix(lower, "yesterday"))
		if rest == "" {
			return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(),
				12, 0, 0, 0, now.Location()), nil
		}
		// Try parsing time part like "3pm", "15:00".
		t, err := parseTimeOfDay(rest)
		if err == nil {
			return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(),
				t.Hour(), t.Minute(), 0, 0, now.Location()), nil
		}
	}

	// Handle "today Xpm/Xam".
	if strings.HasPrefix(lower, "today") {
		rest := strings.TrimSpace(strings.TrimPrefix(lower, "today"))
		if t, err := parseTimeOfDay(rest); err == nil {
			return time.Date(now.Year(), now.Month(), now.Day(),
				t.Hour(), t.Minute(), 0, 0, now.Location()), nil
		}
	}

	// Try bare time of day for today.
	if t, err := parseTimeOfDay(lower); err == nil {
		return time.Date(now.Year(), now.Month(), now.Day(),
			t.Hour(), t.Minute(), 0, 0, now.Location()), nil
	}

	return time.Time{}, fmt.Errorf("unrecognized time expression")
}

// parseTimeOfDay parses "3pm", "3:30pm", "15:00" etc.
func parseTimeOfDay(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	formats := []string{"3pm", "3:04pm", "3PM", "3:04PM", "15:04", "15"}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %s", s)
}

// findNearestCheckpoint finds the checkpoint closest to targetTime and offers to restore.
func findNearestCheckpoint(r *repo, targetTime time.Time, skipConfirm bool) error {
	cps := allCheckpointsSortedByTime(r)
	if len(cps) == 0 {
		return fmt.Errorf("no checkpoints found")
	}

	// Find nearest checkpoint.
	best := cps[0]
	bestDist := absDuration(cps[0].CreatedAt.Sub(targetTime))
	for _, cp := range cps[1:] {
		dist := absDuration(cp.CreatedAt.Sub(targetTime))
		if dist < bestDist {
			best = cp
			bestDist = dist
		}
	}

	elapsed := int64(time.Since(best.CreatedAt).Seconds())
	sNum := r.engine.Index.SNumberFor(best.ID)
	label := shortID(best.ID)
	if sNum != "" {
		label = sNum + " " + label
	}

	fmt.Printf("\n  Found: %s%s%s %q saved %s\n\n",
		colorCyan, label, colorReset,
		best.Message, humanTime(elapsed))

	if best.SnapshotRef == "" {
		return fmt.Errorf("Root checkpoint has no files to compare. Choose a later checkpoint.")
	}

	if !skipConfirm {
		if !askConfirm("Restore? [y/N]") {
			printDim("aborted")
			return nil
		}
	}

	return restoreToCheckpoint(r, best)
}

// findBeforeMessage searches for a checkpoint message containing negative keywords
// near the user's search term, then picks the checkpoint just before that.
func findBeforeMessage(r *repo, query string, skipConfirm bool) error {
	negativeWords := []string{"broke", "broken", "crash", "error", "bug", "fail", "failed", "failure"}
	queryLower := strings.ToLower(query)

	cps := allCheckpointsSortedByTime(r)
	if len(cps) == 0 {
		return fmt.Errorf("no checkpoints found")
	}

	// Sort oldest-first for "just before" logic.
	sort.Slice(cps, func(i, j int) bool {
		return cps[i].CreatedAt.Before(cps[j].CreatedAt)
	})

	// Look for the "bad" checkpoint.
	badIdx := -1
	for i, cp := range cps {
		msgLower := strings.ToLower(cp.Message)
		// Check if message contains any word from the query.
		queryWords := strings.Fields(queryLower)
		matchesQuery := false
		for _, qw := range queryWords {
			if qw == "i" || qw == "the" || qw == "a" {
				continue
			}
			if strings.Contains(msgLower, qw) {
				matchesQuery = true
				break
			}
		}
		if !matchesQuery {
			continue
		}
		// Check for negative words.
		hasNegative := false
		for _, neg := range negativeWords {
			if strings.Contains(msgLower, neg) || strings.Contains(queryLower, neg) {
				hasNegative = true
				break
			}
		}
		if matchesQuery && hasNegative {
			// Found a match — use the checkpoint just before this one.
			badIdx = i
			break
		}
	}

	if badIdx < 0 {
		// No negative-keyword match; fall back to simple message search.
		for i, cp := range cps {
			msgLower := strings.ToLower(cp.Message)
			queryWords := strings.Fields(queryLower)
			for _, qw := range queryWords {
				if qw == "i" || qw == "the" || qw == "a" {
					continue
				}
				if strings.Contains(msgLower, qw) {
					badIdx = i
					break
				}
			}
			if badIdx >= 0 {
				break
			}
		}
	}

	if badIdx < 0 {
		return fmt.Errorf("no checkpoint found matching %q", query)
	}

	// Pick the one just before the match.
	targetIdx := badIdx - 1
	if targetIdx < 0 {
		return fmt.Errorf("the matching checkpoint is the very first one — nothing before it")
	}

	target := cps[targetIdx]
	elapsed := int64(time.Since(target.CreatedAt).Seconds())
	sNum := r.engine.Index.SNumberFor(target.ID)
	label := shortID(target.ID)
	if sNum != "" {
		label = sNum + " " + label
	}

	fmt.Printf("\n  Found: %s%s%s %q saved %s (just before match)\n\n",
		colorCyan, label, colorReset,
		target.Message, humanTime(elapsed))

	if target.SnapshotRef == "" {
		return fmt.Errorf("Root checkpoint has no files to compare. Choose a later checkpoint.")
	}

	if !skipConfirm {
		if !askConfirm("Restore? [y/N]") {
			printDim("aborted")
			return nil
		}
	}

	return restoreToCheckpoint(r, target)
}

func restoreToCheckpoint(r *repo, cp *timeline.Checkpoint) error {
	lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
	fl := storage.NewFileLock(lockPath)
	return fl.WithLock(func() error {
		snap, err := r.scanner.Load(cp.SnapshotRef)
		if err != nil {
			return fmt.Errorf("load snapshot: %w", err)
		}
		if _, err := r.engine.GotoCheckpoint(cp.ID); err != nil {
			return fmt.Errorf("goto checkpoint: %w", err)
		}
		if err := r.scanner.Restore(snap); err != nil {
			return fmt.Errorf("restore: %w", err)
		}
		printSuccess("restored to %s", shortID(cp.ID))
		fmt.Println()
		return nil
	})
}

// allCheckpointsSortedByTime returns all checkpoints sorted newest-first.
func allCheckpointsSortedByTime(r *repo) []*timeline.Checkpoint {
	var all []*timeline.Checkpoint
	for i := range r.engine.Index.Checkpoints {
		cp := r.engine.Index.Checkpoints[i]
		all = append(all, &cp)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	return all
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
