package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/itsakash-real/nimbi/internal/diff"
	"github.com/itsakash-real/nimbi/internal/storage"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	var noRollback bool
	var quiet bool

	cmd := &cobra.Command{
		Use:   "run <command>",
		Short: "Run a shell command with automatic before/after checkpoints and rollback on failure",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Join all remaining args as the command string.
			userCmd := joinStrings(args, " ")

			r, err := loadRepo()
			if err != nil {
				return err
			}

			lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
			fl := storage.NewFileLock(lockPath)

			// ── Pre-run checkpoint ────────────────────────────────────────────
			var preRunID string
			// Record the original branch ID before any save (which may auto-fork).
			originalBranchID := r.engine.Index.CurrentBranchID
			err = fl.WithLock(func() error {
				preMsg := "pre-run: " + userCmd
				cp, err := saveCheckpointNow(r, preMsg)
				if err != nil {
					return fmt.Errorf("pre-run save: %w", err)
				}
				preRunID = cp.ID
				if !quiet {
					fmt.Printf("Checkpoint saved before run: %s\n", shortID(preRunID))
				}
				return nil
			})
			if err != nil {
				return err
			}

			// ── Execute the user command ──────────────────────────────────────
			if !quiet {
				fmt.Printf("Running: %s\n", userCmd)
				fmt.Println(colorDim + "─────────────────────────────────────" + colorReset)
			}

			exitCode := execCommand(userCmd)

			if !quiet {
				fmt.Println(colorDim + "─────────────────────────────────────" + colorReset)
			}

			// ── Post-run handling ─────────────────────────────────────────────
			if exitCode == 0 {
				// Command succeeded — save a post-run checkpoint.
				var postID string
				err = fl.WithLock(func() error {
					postMsg := "post-run: " + userCmd + " \u2713"
					cp, err := saveCheckpointNow(r, postMsg)
					if err != nil {
						return fmt.Errorf("post-run save: %w", err)
					}
					postID = cp.ID
					return nil
				})
				if err != nil {
					return err
				}
				fmt.Printf("%s✓ Command succeeded. Checkpoint saved: %s%s\n",
					colorGreen, shortID(postID), colorReset)
				return nil
			}

			// Command failed.
			fmt.Printf("%s✗ Command failed (exit %d).%s\n", colorRed, exitCode, colorReset)

			if noRollback {
				fmt.Println("  --no-rollback set; not restoring.")
				return nil
			}

			fmt.Printf("  Rolling back to pre-run checkpoint %s...\n", shortID(preRunID))

			err = fl.WithLock(func() error {
				// Reload engine state (may have changed).
				r2, err := loadRepo()
				if err != nil {
					return err
				}
				preCP, ok := r2.engine.Index.Checkpoints[preRunID]
				if !ok {
					return fmt.Errorf("pre-run checkpoint %s not found", shortID(preRunID))
				}
				if preCP.SnapshotRef == "" {
					return fmt.Errorf("pre-run checkpoint has no snapshot")
				}
				targetSnap, err := r2.scanner.Load(preCP.SnapshotRef)
				if err != nil {
					return fmt.Errorf("load pre-run snapshot: %w", err)
				}
				if _, err := r2.engine.GotoCheckpoint(preCP.ID); err != nil {
					return fmt.Errorf("goto pre-run checkpoint: %w", err)
				}
				// Restore the original branch the user was on before the pre-run save,
				// in case the pre-run checkpoint auto-forked a new branch.
				if originalBranchID != "" {
					if _, exists := r2.engine.Index.Branches[originalBranchID]; exists {
						r2.engine.Index.CurrentBranchID = originalBranchID
						if persistErr := r2.engine.ForceFlush(); persistErr != nil {
							return fmt.Errorf("restore original branch: %w", persistErr)
						}
					}
				}
				if err := r2.scanner.Restore(targetSnap); err != nil {
					return fmt.Errorf("restore pre-run state: %w", err)
				}
				return nil
			})
			if err != nil {
				return err
			}

			fmt.Printf("%s✓ Rolled back to %s%s\n", colorGreen, shortID(preRunID), colorReset)
			return nil
		},
	}

	cmd.Flags().BoolVar(&noRollback, "no-rollback", false, "do not roll back on failure")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress checkpoint output")
	return cmd
}

// execCommand runs a shell command and returns its exit code.
// stdout and stderr are streamed directly to the terminal.
func execCommand(command string) int {
	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.Command("cmd", "/c", command)
	} else {
		c = exec.Command("sh", "-c", command)
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	err := c.Run()
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}

// saveCheckpointNow scans and saves a checkpoint with the given message.
// It uses the already-loaded repo (r) directly.
func saveCheckpointNow(r *repo, message string) (*checkpointResult, error) {
	snap, err := r.scanner.Scan()
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	// Compute diff for auto-message only (not used here but part of the pattern).
	var dr *diff.DiffResult
	if prevCP, ok := r.engine.Index.CurrentCheckpoint(); ok && prevCP.SnapshotRef != "" {
		if prevSnap, loadErr := r.scanner.Load(prevCP.SnapshotRef); loadErr == nil {
			diffEng := diff.New(r.store)
			if result, diffErr := diffEng.Compare(prevSnap, snap); diffErr == nil {
				dr = result
			}
		}
	}
	_ = dr // suppress unused warning

	snapshotHash, err := r.scanner.Save(snap)
	if err != nil {
		return nil, fmt.Errorf("save snapshot: %w", err)
	}

	cp, err := r.engine.SaveCheckpoint(message, snapshotHash)
	if err != nil {
		return nil, fmt.Errorf("save checkpoint: %w", err)
	}

	return &checkpointResult{ID: cp.ID, Message: cp.Message}, nil
}

// checkpointResult holds the minimal result of a programmatic save.
type checkpointResult struct {
	ID      string
	Message string
}
