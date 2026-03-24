package main

import (
	"fmt"
	"path/filepath"

	"github.com/itsakash-real/rewinddb/internal/storage"
	"github.com/spf13/cobra"
)

func undoCmd() *cobra.Command {
	var n int
	var force bool

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
					return fmt.Errorf("undo: target checkpoint %s has no snapshot (it may be the root checkpoint)", shortID(target.ID))
				}

				sectionTitle(fmt.Sprintf("undo  \u00b7  %d step(s) back", n))
				fmt.Println()
				kv("restoring to", colorCyan+shortID(target.ID)+colorReset)
				kv("message",      fmt.Sprintf("%q", target.Message))
				fmt.Println()

				// Ask confirmation unless --force.
				if !force {
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
				return nil
			})
		},
	}

	cmd.Flags().IntVar(&n, "n", 1, "number of checkpoints to go back")
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")
	return cmd
}
