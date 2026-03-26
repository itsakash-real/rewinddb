package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/itsakash-real/rewinddb/internal/diff"
	"github.com/itsakash-real/rewinddb/internal/merkle"
	"github.com/itsakash-real/rewinddb/internal/storage"
	"github.com/itsakash-real/rewinddb/internal/wal"
	"github.com/spf13/cobra"
)

func saveCmd() *cobra.Command {
	var tag string
	var quiet bool
	var workers int

	cmd := &cobra.Command{
		Use:   "save [message]",
		Short: "Snapshot the current state and save a checkpoint",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}
			lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
			fl := storage.NewFileLock(lockPath)
			return fl.WithLock(func() error {
				// Extract optional message arg (may be empty/absent).
				var messageArg string
				if len(args) > 0 {
					messageArg = args[0]
				}

				if workers > 0 {
					r.scanner.Workers = workers
				}

				// ── Pre-save hook ────────────────────────────────────────────────
				if err := RunHook(r.cfg.RewindDir, "pre-save", "", messageArg); err != nil {
					return err
				}

				// ── Scan the working directory ────────────────────────────────────
				snap, err := r.scanner.Scan()
				if err != nil {
					return fmt.Errorf("scan failed: %w", err)
				}

				// ── Compute diff against previous snapshot (best-effort) ──────────
				var diffResult *diff.DiffResult
				changedCount := 0
				if prevCP, ok := r.engine.Index.CurrentCheckpoint(); ok && prevCP.SnapshotRef != "" {
					if prevSnap, err := r.scanner.Load(prevCP.SnapshotRef); err == nil {
						diffEng := diff.New(r.store)
						if result, err := diffEng.Compare(prevSnap, snap); err == nil {
							diffResult = result
							changedCount = result.TotalChanges()
						}
					}
				}

				// ── Auto-generate message if none provided ────────────────────────
				message := messageArg
				if message == "" {
					message = autoMessage(r, diffResult)
				}

				// ── WAL: record intent before any writes ──────────────────────────
				walPath := filepath.Join(r.cfg.RewindDir, wal.FileName)
				w, err := wal.Open(walPath)
				if err != nil {
					return fmt.Errorf("wal open: %w", err)
				}
				if err := w.WriteIntent(message); err != nil {
					w.Close()
					return fmt.Errorf("wal intent: %w", err)
				}

				// ── Persist snapshot objects ──────────────────────────────────────
				snapshotHash, err := r.scanner.Save(snap)
				if err != nil {
					w.Close()
					return fmt.Errorf("save snapshot: %w", err)
				}
				_ = w.WriteObject(snapshotHash)

				// ── Create checkpoint in the DAG ──────────────────────────────────
				cp, err := r.engine.SaveCheckpoint(message, snapshotHash)
				if err != nil {
					w.Close()
					return fmt.Errorf("save checkpoint: %w", err)
				}

				// ── WAL: commit (safe to remove on next startup) ──────────────────
				_ = w.WriteCommit(cp.ID)

				// ── Update Merkle root (best-effort — health can recompute) ───────
				if root, _, err := merkle.Compute(r.cfg.ObjectsDir); err == nil {
					_ = merkle.SaveRoot(r.cfg.RewindDir, root)
				}

				// Attach optional tag.
				if tag != "" {
					cp.Tags = append(cp.Tags, tag)
					r.engine.Index.Checkpoints[cp.ID] = *cp
					if saveErr := r.engine.Index.Save(r.cfg.IndexPath); saveErr != nil {
						return fmt.Errorf("persist tag: %w", saveErr)
					}
				}

				branch, _ := r.engine.Index.CurrentBranch()

				// ── Post-save hook ───────────────────────────────────────────────
				_ = RunHook(r.cfg.RewindDir, "post-save", cp.ID, message)

				// ── Background GC every 10 saves ─────────────────────────────────
				maybeBackgroundGC(r.cfg.RewindDir)

				// ── Output ────────────────────────────────────────────────────────
				if quiet {
					fmt.Println(cp.ID)
					return nil
				}

				sectionTitle("checkpoint saved")
				fmt.Println()
				kv("id", colorCyan+shortID(cp.ID)+colorReset)
				kv("message", fmt.Sprintf("%q", cp.Message))
				kv("branch", colorPurple+branch.Name+colorReset)
				kv("files", fmt.Sprintf("%s%d tracked%s  \u00b7  %s%d changed%s",
					colorDim, len(snap.Files), colorReset,
					colorBold, changedCount, colorReset,
				))
				kv("saved", dimP.Sprint("just now"))
				if tag != "" {
					kv("tag", colorCyan+tag+colorReset)
				}
				fmt.Println()

				return nil
			})
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "", "attach a tag label to the checkpoint")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "print only the checkpoint ID")
	cmd.Flags().IntVar(&workers, "workers", 0,
		"parallel file hashing workers (default: runtime.NumCPU())")
	return cmd
}

// autoMessage generates a descriptive message when none is provided by the user.
// If there is no previous checkpoint, returns "initial snapshot".
// Otherwise summarises the changed file names.
func autoMessage(r *repo, diffResult *diff.DiffResult) string {
	// No previous checkpoint at all.
	if diffResult == nil {
		return "initial snapshot"
	}

	// Collect changed file names (added + removed + modified).
	var names []string
	seen := make(map[string]struct{})
	add := func(path string) {
		// Use only the base name for brevity.
		base := filepath.Base(path)
		if _, ok := seen[base]; !ok {
			seen[base] = struct{}{}
			names = append(names, base)
		}
	}
	for _, f := range diffResult.Added {
		add(f.Path)
	}
	for _, f := range diffResult.Removed {
		add(f.Path)
	}
	for _, fd := range diffResult.Modified {
		add(fd.Path)
	}

	total := len(names)
	if total == 0 {
		return "no changes"
	}

	const maxFiles = 3
	var fileList string
	if total <= maxFiles {
		fileList = joinStrings(names, ", ")
	} else {
		fileList = joinStrings(names[:maxFiles], ", ") + fmt.Sprintf(" +%d more", total-maxFiles)
	}

	return fmt.Sprintf("auto: %s (%d file(s) changed)", fileList, total)
}

// joinStrings joins a slice of strings with sep.
func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

// maybeBackgroundGC increments the save counter and runs GC in the background
// every 10th save. It uses .rewind/save-count as a persistent counter file.
// Accepts rewindDir so it does not hold a reference to the caller's *repo.
func maybeBackgroundGC(rewindDir string) {
	countPath := filepath.Join(rewindDir, saveCountFile)

	// Read current count.
	var count int
	data, err := os.ReadFile(countPath)
	if err == nil {
		fmt.Sscanf(string(data), "%d", &count)
	}
	count++

	// Write updated count back.
	_ = os.WriteFile(countPath, []byte(fmt.Sprintf("%d", count)), 0o644)

	if count%10 == 0 {
		// Fire-and-forget GC in background goroutine with a fresh repo
		// so we don't race with the caller's state.
		go func() {
			r2, err := loadRepo()
			if err != nil {
				return
			}
			runGCBackground(r2)
		}()
	}
}

// parentDir returns the directory one level above path.
// e.g.  /home/user/project/.rewind  →  /home/user/project
func parentDir(path string) string {
	return filepath.Dir(path)
}
