package main

import (
	"fmt"
	"path/filepath"

	"github.com/itsakash-real/rewinddb/internal/storage"
	"github.com/spf13/cobra"
)

func tagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag <name> [checkpoint-id]",
		Short: "Attach a human-readable tag to a checkpoint",
		Long: `Attach a tag to a checkpoint so it can be referenced by name in
rw goto, rw diff, and other commands.

  rw tag v1.0            # tag HEAD
  rw tag v1.0 a3f2b1c8  # tag a specific checkpoint`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			tagName := args[0]

			r, err := loadRepo()
			if err != nil {
				return err
			}

			lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
			fl := storage.NewFileLock(lockPath)
			return fl.WithLock(func() error {
				// ── Resolve the target checkpoint ─────────────────────────────────
				var targetID string
				if len(args) == 2 {
					cp, err := resolveCheckpoint(r.engine, args[1])
					if err != nil {
						return fmt.Errorf("cannot resolve checkpoint: %w", err)
					}
					targetID = cp.ID
				} else {
					if r.engine.Index.CurrentCheckpointID == "" {
						return fmt.Errorf("no HEAD checkpoint — run 'rw save' first")
					}
					targetID = r.engine.Index.CurrentCheckpointID
				}

				// ── Guard: duplicate tag on same checkpoint is a no-op ────────────
				cp := r.engine.Index.Checkpoints[targetID]
				for _, t := range cp.Tags {
					if t == tagName {
						fmt.Printf("Tag %q already exists on checkpoint %s\n", tagName, shortID(targetID))
						return nil
					}
				}

				// ── Guard: tag name must be unique across all checkpoints ─────────
				for id, other := range r.engine.Index.Checkpoints {
					if id == targetID {
						continue
					}
					for _, t := range other.Tags {
						if t == tagName {
							return fmt.Errorf("tag %q already exists on checkpoint %s (%q)",
								tagName, shortID(id), other.Message)
						}
					}
				}

				// ── Apply tag ─────────────────────────────────────────────────────
				cp.Tags = append(cp.Tags, tagName)
				r.engine.Index.Checkpoints[targetID] = cp

				if err := r.engine.Index.Save(r.cfg.IndexPath); err != nil {
					return fmt.Errorf("persist index: %w", err)
				}

				fmt.Printf("✓ Tagged checkpoint %s%s%s as %s%q%s  (%q)\n",
					colorCyan, shortID(targetID), colorReset,
					colorBold, tagName, colorReset,
					cp.Message,
				)
				return nil
			})
		},
	}

	return cmd
}

