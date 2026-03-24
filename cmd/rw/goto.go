package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/itsakash-real/rewinddb/internal/diff"
	"github.com/itsakash-real/rewinddb/internal/snapshot"
	"github.com/itsakash-real/rewinddb/internal/storage"
	"github.com/itsakash-real/rewinddb/internal/timeline"
	"github.com/spf13/cobra"
)

func gotoCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "goto <checkpoint-id>",
		Short: "Restore the working directory to a previous checkpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}
			lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
			fl := storage.NewFileLock(lockPath)
			return fl.WithLock(func() error {
				prefix := args[0]

				// ── Resolve checkpoint by ID prefix ───────────────────────────────
				cp, err := resolveCheckpoint(r.engine, prefix)
				if err != nil {
					return err
				}

				branch, _ := r.engine.Index.Branches[cp.BranchID]

				// ── Compute distance from current HEAD ────────────────────────────
				distance := computeDistance(r.engine, r.engine.Index.CurrentCheckpointID, cp.ID)

				// ── Auto-stash check: detect unsaved changes ──────────────────────
				if err := autoStashIfDirty(r, cp.ID, force); err != nil {
					return err
				}

				// ── User confirmation (unless --force) ────────────────────────────
				if !force {
					distStr := ""
					if distance > 0 {
						distStr = fmt.Sprintf(" (%d checkpoint(s) ago)", distance)
					}
					prompt := fmt.Sprintf("Restore to: %q%s? [y/N]", cp.Message, distStr)
					if !askConfirm(prompt) {
						printDim("aborted")
						return nil
					}
				}

				// ── Load target snapshot ──────────────────────────────────────────
				targetSnap, err := r.scanner.Load(cp.SnapshotRef)
				if err != nil {
					return fmt.Errorf("load snapshot %s: %w", cp.SnapshotRef, err)
				}

				// ── Diff current state vs target for reporting ────────────────────
				currentSnap, _ := currentSnapshot(r.engine, r.scanner)
				var restoredCount, removedCount int
				if currentSnap != nil {
					diffEng := diff.New(r.store)
					if result, err := diffEng.Compare(currentSnap, targetSnap); err == nil {
						restoredCount = len(result.Modified) + len(result.Added)
						removedCount = len(result.Removed)
					}
				}

				// ── Move the DAG HEAD ─────────────────────────────────────────────
				if _, err := r.engine.GotoCheckpoint(cp.ID); err != nil {
					return fmt.Errorf("goto checkpoint: %w", err)
				}

				// ── Restore files on disk ─────────────────────────────────────────
				if err := r.scanner.Restore(targetSnap); err != nil {
					return fmt.Errorf("restore files: %w", err)
				}

				// ── Output ────────────────────────────────────────────────────────
				sectionTitle("restored")
				fmt.Println()
				kv("checkpoint", colorCyan+shortID(cp.ID)+colorReset)
				kv("message",    fmt.Sprintf("%q", cp.Message))
				kv("branch",     colorPurple+branch.Name+colorReset)
				kv("written",    fmt.Sprintf("%s%d file(s)%s", colorBold, restoredCount, colorReset))
				kv("removed",    fmt.Sprintf("%s%d file(s)%s", colorDim, removedCount, colorReset))
				fmt.Println()
				return nil
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")
	return cmd
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// computeDistance walks from currentID backwards counting hops to targetID.
// Returns 0 if target is not an ancestor (e.g. it's in the future or a fork).
func computeDistance(engine *timeline.TimelineEngine, currentID, targetID string) int {
	dist := 0
	cur := currentID
	visited := make(map[string]struct{})
	for cur != "" && cur != targetID {
		if _, seen := visited[cur]; seen {
			break
		}
		visited[cur] = struct{}{}
		cp, ok := engine.Index.Checkpoints[cur]
		if !ok {
			break
		}
		cur = cp.ParentID
		dist++
	}
	if cur == targetID {
		return dist
	}
	return 0
}

// currentSnapshot loads the snapshot for the current HEAD checkpoint.
// Returns nil, nil if no snapshot is attached yet.
func currentSnapshot(engine *timeline.TimelineEngine, sc *snapshot.Scanner) (*timeline.Snapshot, error) {
	cp, ok := engine.Index.CurrentCheckpoint()
	if !ok || cp.SnapshotRef == "" {
		return nil, nil
	}
	return sc.Load(cp.SnapshotRef)
}

// autoStashIfDirty checks whether the working directory has unsaved changes
// compared to the current HEAD snapshot. If dirty:
//   - If force is true: silently auto-saves a stash checkpoint.
//   - If force is false: asks the user whether to auto-save; warns if no.
func autoStashIfDirty(r *repo, targetID string, force bool) error {
	headCP, ok := r.engine.Index.CurrentCheckpoint()
	if !ok || headCP.SnapshotRef == "" {
		return nil // no baseline to compare against
	}

	// Scan the current working directory.
	currentSnap, err := r.scanner.Scan()
	if err != nil {
		return nil // best-effort: don't block goto on scan errors
	}

	prevSnap, err := r.scanner.Load(headCP.SnapshotRef)
	if err != nil {
		return nil
	}

	diffEng := diff.New(r.store)
	result, err := diffEng.Compare(prevSnap, currentSnap)
	if err != nil || result.TotalChanges() == 0 {
		return nil // clean — nothing to stash
	}

	changedCount := result.TotalChanges()
	fmt.Printf("You have unsaved changes in %d file(s).\n", changedCount)

	doStash := force
	if force {
		fmt.Printf("%sWarning: %d file(s) with unsaved changes will be auto-stashed (--force).%s\n",
			colorYellow, changedCount, colorReset)
	} else {
		doStash = askConfirmDefault("Auto-save before restoring? [Y/n]: ", true)
		if !doStash {
			fmt.Printf("%sWarning: unsaved changes will be overwritten.%s\n",
				colorYellow, colorReset)
			return nil
		}
	}

	if doStash {
		stashMsg := fmt.Sprintf("auto-stash before goto %s", shortID(targetID))
		snapshotHash, saveErr := r.scanner.Save(currentSnap)
		if saveErr != nil {
			return fmt.Errorf("auto-stash: save snapshot: %w", saveErr)
		}
		cp, saveErr := r.engine.SaveCheckpoint(stashMsg, snapshotHash)
		if saveErr != nil {
			return fmt.Errorf("auto-stash: save checkpoint: %w", saveErr)
		}
		kv("auto-stashed", colorCyan+shortID(cp.ID)+colorReset)
	}
	return nil
}

// askConfirmDefault reads a y/n response from stdin, defaulting to def if empty.
func askConfirmDefault(prompt string, def bool) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	response, err := reader.ReadString('\n')
	if err != nil {
		return def
	}
	response = strings.ToLower(strings.TrimSpace(response))
	if response == "" {
		return def
	}
	return response == "y" || response == "yes"
}

// askConfirm prints prompt and reads a y/yes response from stdin.
func askConfirm(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt + " ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
