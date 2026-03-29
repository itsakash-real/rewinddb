package main

import (
	"fmt"
	"path/filepath"

	diffpkg "github.com/itsakash-real/nimbi/internal/diff"
	"github.com/itsakash-real/nimbi/internal/storage"
	"github.com/spf13/cobra"
)

func undoCmd() *cobra.Command {
	var n int
	var force bool
	var preview bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "undo",
		Short: "Go back N checkpoints (default 1)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if n < 1 {
				return fmt.Errorf("--n must be >= 1")
			}

			r, err := loadRepo()
			if err != nil {
				return err
			}

			lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
			fl := storage.NewFileLock(lockPath)
			return fl.WithLock(func() error {
				// Resolve HEAD~N using the existing resolveHeadTilde helper.
				target, err := resolveHeadTilde(r.engine, n)
				if err != nil {
					return fmt.Errorf("undo: %w", err)
				}

				if target.SnapshotRef == "" {
					return fmt.Errorf("Root checkpoint has no files to compare. Choose a later checkpoint.")
				}

				sectionTitle(fmt.Sprintf("undo  \u00b7  %d step(s) back", n))
				fmt.Println()
				kv("restoring to", colorCyan+shortID(target.ID)+colorReset)
				kv("message", fmt.Sprintf("%q", target.Message))
				fmt.Println()

				// ── Preview mode: show what will change ──────────────────
				if preview {
					return showUndoPreview(r, target.SnapshotRef)
				}

				// Ask confirmation unless --force or --yes.
				if !force && !yes {
					if !askConfirm("This will overwrite your working directory. Continue? [y/N]") {
						printDim("aborted")
						return nil
					}
				}

				// Load target snapshot.
				targetSnap, err := r.scanner.Load(target.SnapshotRef)
				if err != nil {
					return fmt.Errorf("undo: load snapshot: %w", err)
				}

				// Snapshot current state for dependency comparison.
				currentSnap, _ := currentSnapshot(r.engine, r.scanner)

				// Move DAG HEAD.
				if _, err := r.engine.GotoCheckpoint(target.ID); err != nil {
					return fmt.Errorf("undo: goto checkpoint: %w", err)
				}

				// Restore files on disk.
				if err := r.scanner.Restore(targetSnap); err != nil {
					return fmt.Errorf("undo: restore files: %w", err)
				}

				printSuccess("restored to %s", shortID(target.ID))
				fmt.Println()

				// Dependency change detection.
				if currentSnap != nil {
					checkDependencyChanges(currentSnap, targetSnap)
				}

				return nil
			})
		},
	}

	cmd.Flags().IntVar(&n, "n", 1, "number of checkpoints to go back")
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")
	cmd.Flags().BoolVar(&preview, "preview", false, "show what will change without restoring")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt")
	return cmd
}

// showUndoPreview loads the target snapshot and the current snapshot, computes
// a diff, and prints a summary of what would change without actually restoring.
func showUndoPreview(r *repo, targetSnapshotRef string) error {
	targetSnap, err := r.scanner.Load(targetSnapshotRef)
	if err != nil {
		return fmt.Errorf("load target snapshot: %w", err)
	}

	// Get current state by scanning.
	currentSnap, err := r.scanner.Scan()
	if err != nil {
		return fmt.Errorf("scan current state: %w", err)
	}

	diffEng := diffpkg.New(r.store)
	result, err := diffEng.Compare(currentSnap, targetSnap)
	if err != nil {
		return fmt.Errorf("compute diff: %w", err)
	}

	total := result.TotalChanges()
	if total == 0 {
		printInfo("No files would change.")
		return nil
	}

	fmt.Printf("  Will restore %d file(s):\n", total)
	for _, f := range result.Added {
		fmt.Printf("    %s%s%s     %s+ new file%s\n", colorGreen, f.Path, colorReset, colorDim, colorReset)
	}
	for _, f := range result.Removed {
		fmt.Printf("    %s%s%s     %s- will be removed%s\n", colorRed, f.Path, colorReset, colorDim, colorReset)
	}
	for _, f := range result.Modified {
		fmt.Printf("    %s%s%s     %s~ will change%s\n", colorYellow, f.Path, colorReset, colorDim, colorReset)
	}
	fmt.Println()
	printDim("Run without --preview to apply. Use --yes to skip confirmation.")
	return nil
}
