package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/itsakash-real/nimbi/internal/storage"
	"github.com/spf13/cobra"
)

const stashDir = "stashes"
const stashIndexFile = "stash-index.json"

// stashEntry records one stash.
type stashEntry struct {
	Name        string    `json:"name"`
	SnapshotRef string    `json:"snapshot_ref"`
	Message     string    `json:"message"`
	CreatedAt   time.Time `json:"created_at"`
}

func stashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stash",
		Short: "Save current state to a temporary stash (not on timeline)",
		Long: `rw stash saves a snapshot of all files (including binaries and untracked)
into a temporary stash that lives outside the timeline DAG.

Use 'rw stash pop' to restore and delete the latest stash.
Use 'rw stash list' to see all stashes.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
			fl := storage.NewFileLock(lockPath)
			return fl.WithLock(func() error {
				snap, err := r.scanner.Scan()
				if err != nil {
					return fmt.Errorf("stash: scan: %w", err)
				}

				snapshotHash, err := r.scanner.Save(snap)
				if err != nil {
					return fmt.Errorf("stash: save snapshot: %w", err)
				}

				entries, err := loadStashIndex(r.cfg.RewindDir)
				if err != nil {
					return err
				}

				name := fmt.Sprintf("stash-%d", len(entries)+1)
				entries = append(entries, stashEntry{
					Name:        name,
					SnapshotRef: snapshotHash,
					Message:     fmt.Sprintf("stash at %s", time.Now().Format("15:04:05")),
					CreatedAt:   time.Now().UTC(),
				})

				if err := saveStashIndex(r.cfg.RewindDir, entries); err != nil {
					return err
				}

				printSuccess("stashed as %s (%d files)", name, len(snap.Files))
				return nil
			})
		},
	}

	cmd.AddCommand(stashPopCmd())
	cmd.AddCommand(stashListCmd())
	return cmd
}

func stashPopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pop",
		Short: "Restore the latest stash and delete it",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
			fl := storage.NewFileLock(lockPath)
			return fl.WithLock(func() error {
				entries, err := loadStashIndex(r.cfg.RewindDir)
				if err != nil {
					return err
				}
				if len(entries) == 0 {
					return fmt.Errorf("no stashes found")
				}

				// Pop the last stash.
				last := entries[len(entries)-1]
				entries = entries[:len(entries)-1]

				snap, err := r.scanner.Load(last.SnapshotRef)
				if err != nil {
					return fmt.Errorf("stash pop: load snapshot: %w", err)
				}

				if err := r.scanner.Restore(snap); err != nil {
					return fmt.Errorf("stash pop: restore: %w", err)
				}

				if err := saveStashIndex(r.cfg.RewindDir, entries); err != nil {
					return err
				}

				printSuccess("restored %s (%d files)", last.Name, len(snap.Files))
				return nil
			})
		},
	}
}

func stashListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show all stashes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			entries, err := loadStashIndex(r.cfg.RewindDir)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				printDim("no stashes")
				return nil
			}

			sectionTitle(fmt.Sprintf("stashes  ·  %d", len(entries)))
			fmt.Println()
			for _, e := range entries {
				elapsed := int64(time.Since(e.CreatedAt).Seconds())
				fmt.Printf("  %s%-10s%s  %s  %s\n",
					colorCyan, e.Name, colorReset,
					e.Message,
					dimP.Sprint(humanTime(elapsed)))
			}
			fmt.Println()
			return nil
		},
	}
}

func loadStashIndex(rewindDir string) ([]stashEntry, error) {
	path := filepath.Join(rewindDir, stashDir, stashIndexFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load stash index: %w", err)
	}
	var entries []stashEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse stash index: %w", err)
	}
	return entries, nil
}

func saveStashIndex(rewindDir string, entries []stashEntry) error {
	dir := filepath.Join(rewindDir, stashDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("save stash index: mkdir: %w", err)
	}

	path := filepath.Join(dir, stashIndexFile)
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal stash index: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".stash-idx-*.tmp")
	if err != nil {
		return fmt.Errorf("save stash index: create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}
