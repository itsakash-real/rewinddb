package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/itsakash-real/rewinddb/internal/diff"
	"github.com/itsakash-real/rewinddb/internal/storage"
	"github.com/spf13/cobra"
)

func watchCmd() *cobra.Command {
	var interval time.Duration
	var quiet bool

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch the project directory and auto-save checkpoints on changes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			projectRoot := parentDir(r.cfg.RewindDir)
			rewindDir := r.cfg.RewindDir

			if !quiet {
				fmt.Printf("Watching %s for changes. Auto-saving every %s. Ctrl+C to stop.\n",
					projectRoot, interval)
			}

			// Create fsnotify watcher.
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				return fmt.Errorf("watch: create watcher: %w", err)
			}
			defer watcher.Close()

			// Recursively add directories to the watcher.
			if err := watchDirRecursive(watcher, projectRoot, rewindDir); err != nil {
				return fmt.Errorf("watch: setup: %w", err)
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// Debounce timer — starts stopped; reset after detecting a change.
			timer := time.NewTimer(interval)
			if !timer.Stop() {
				// Drain in case it fired between creation and Stop.
				select {
				case <-timer.C:
				default:
				}
			}
			pendingChanges := false

			for {
				select {
				case <-ctx.Done():
					fmt.Println("\nStopping watch.")
					return nil

				case event, ok := <-watcher.Events:
					if !ok {
						// Watcher closed unexpectedly — keep running until Ctrl+C.
						continue
					}
					// Ignore changes inside .rewind/ directory.
					if strings.HasPrefix(filepath.ToSlash(event.Name),
						filepath.ToSlash(rewindDir)) {
						continue
					}
					if !pendingChanges {
						pendingChanges = true
						timer.Reset(interval)
					}

				case watchErr, ok := <-watcher.Errors:
					if !ok {
						// Watcher error channel closed — keep running until Ctrl+C.
						continue
					}
					fmt.Printf("watch: watcher error: %v\n", watchErr)

				case <-timer.C:
					if !pendingChanges {
						continue
					}
					pendingChanges = false
					// Auto-save checkpoint.
					cpID, changedFiles, saveErr := autoSaveWatch(r)
					if saveErr != nil {
						fmt.Printf("watch: save error: %v\n", saveErr)
					} else if changedFiles > 0 && !quiet {
						ts := time.Now().Local().Format("15:04:05")
						fmt.Printf("[%s] ✓ Auto-saved %s (%d file(s) changed)\n",
							ts, shortID(cpID), changedFiles)
					}
					// Refresh watcher in case new directories were added.
					_ = watchDirRecursive(watcher, projectRoot, rewindDir)
				}
			}
		},
	}

	cmd.Flags().DurationVar(&interval, "interval", 30*time.Second,
		"debounce interval between auto-saves (e.g. 30s, 5m)")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress per-save output")
	return cmd
}

// watchDirRecursive recursively adds all subdirectories under root to the watcher,
// excluding the .rewind/ directory.
func watchDirRecursive(w *fsnotify.Watcher, root, excludeDir string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // best-effort
		}
		if !d.IsDir() {
			return nil
		}
		// Skip the .rewind directory.
		absExclude, _ := filepath.Abs(excludeDir)
		absPath, _ := filepath.Abs(path)
		if absPath == absExclude || strings.HasPrefix(absPath+string(filepath.Separator), absExclude+string(filepath.Separator)) {
			return filepath.SkipDir
		}
		_ = w.Add(path) // ignore "already watched" errors
		return nil
	})
}

// autoSaveWatch performs a scan+save using the loaded repo, generating an auto-message.
// Returns the new checkpoint ID and the number of changed files (0 means nothing changed).
func autoSaveWatch(r *repo) (cpID string, changedFiles int, err error) {
	lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
	fl := storage.NewFileLock(lockPath)

	err = fl.WithLock(func() error {
		snap, scanErr := r.scanner.Scan()
		if scanErr != nil {
			return fmt.Errorf("scan: %w", scanErr)
		}

		// Compute diff for auto-message and changed-file count.
		var dr *diff.DiffResult
		if prevCP, ok := r.engine.Index.CurrentCheckpoint(); ok && prevCP.SnapshotRef != "" {
			if prevSnap, loadErr := r.scanner.Load(prevCP.SnapshotRef); loadErr == nil {
				diffEng := diff.New(r.store)
				if result, diffErr := diffEng.Compare(prevSnap, snap); diffErr == nil {
					dr = result
					changedFiles = result.TotalChanges()
				}
			}
		}

		message := autoMessage(r, dr)

		snapshotHash, saveErr := r.scanner.Save(snap)
		if saveErr != nil {
			return fmt.Errorf("save snapshot: %w", saveErr)
		}

		cp, saveErr := r.engine.SaveCheckpoint(message, snapshotHash)
		if saveErr != nil {
			return fmt.Errorf("save checkpoint: %w", saveErr)
		}
		cpID = cp.ID
		return nil
	})
	return cpID, changedFiles, err
}
