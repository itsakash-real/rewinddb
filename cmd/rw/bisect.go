package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/itsakash-real/rewinddb/internal/storage"
	"github.com/itsakash-real/rewinddb/internal/timeline"
	"github.com/spf13/cobra"
)

const bisectFileName = "bisect.json"

// bisectState is persisted to .rewind/bisect.json.
type bisectState struct {
	Active       bool     `json:"active"`
	GoodID       string   `json:"good_id,omitempty"`
	BadID        string   `json:"bad_id,omitempty"`
	OriginalHead string   `json:"original_head,omitempty"`
	Candidates   []string `json:"candidates,omitempty"`
	CurrentMid   string   `json:"current_mid,omitempty"`
}

func bisectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bisect <start|good|bad|reset> [checkpoint-id]",
		Short: "Binary-search for a bad checkpoint",
		Args:  cobra.RangeArgs(1, 2),
	}

	cmd.AddCommand(
		bisectStartCmd(),
		bisectGoodCmd(),
		bisectBadCmd(),
		bisectResetCmd(),
	)
	return cmd
}

func bisectStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Begin a bisect session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			state := bisectState{
				Active:       true,
				OriginalHead: r.engine.Index.CurrentCheckpointID,
			}
			if err := saveBisectState(r.cfg.RewindDir, &state); err != nil {
				return err
			}
			fmt.Println("Bisect started.")
			fmt.Println("Mark checkpoints with 'rw bisect good [id]' and 'rw bisect bad [id]'.")
			return nil
		},
	}
}

func bisectGoodCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "good [checkpoint-id]",
		Short: "Mark a checkpoint as good",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bisectMark(args, "good")
		},
	}
}

func bisectBadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bad [checkpoint-id]",
		Short: "Mark a checkpoint as bad",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bisectMark(args, "bad")
		},
	}
}

func bisectResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "End bisect session and restore original HEAD",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			state, err := loadBisectState(r.cfg.RewindDir)
			if err != nil {
				return fmt.Errorf("bisect not active: %w", err)
			}

			// Restore original head if it exists and has a snapshot.
			if state.OriginalHead != "" {
				lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
				fl := storage.NewFileLock(lockPath)
				_ = fl.WithLock(func() error {
					origCP, ok := r.engine.Index.Checkpoints[state.OriginalHead]
					if !ok || origCP.SnapshotRef == "" {
						return nil
					}
					targetSnap, loadErr := r.scanner.Load(origCP.SnapshotRef)
					if loadErr != nil {
						return loadErr
					}
					if _, gotoErr := r.engine.GotoCheckpoint(origCP.ID); gotoErr != nil {
						return gotoErr
					}
					return r.scanner.Restore(targetSnap)
				})
			}

			// Remove bisect.json.
			bisectPath := filepath.Join(r.cfg.RewindDir, bisectFileName)
			_ = os.Remove(bisectPath)

			fmt.Printf("%s✓ Bisect reset. Restored to %s%s\n",
				colorGreen, shortID(state.OriginalHead), colorReset)
			return nil
		},
	}
}

// bisectMark handles both "good" and "bad" subcommands.
func bisectMark(args []string, kind string) error {
	r, err := loadRepo()
	if err != nil {
		return err
	}

	state, err := loadBisectState(r.cfg.RewindDir)
	if err != nil {
		return fmt.Errorf("bisect not active (run 'rw bisect start' first): %w", err)
	}
	if !state.Active {
		return fmt.Errorf("bisect not active (run 'rw bisect start' first)")
	}

	// Resolve the target checkpoint.
	var ref string
	if len(args) > 0 {
		ref = args[0]
	} else {
		ref = "HEAD"
	}
	cp, err := resolveCheckpoint(r.engine, ref)
	if err != nil {
		return err
	}

	switch kind {
	case "good":
		state.GoodID = cp.ID
	case "bad":
		state.BadID = cp.ID
	}

	// If both good and bad are set, compute the midpoint and jump there.
	if state.GoodID != "" && state.BadID != "" {
		if err := bisectComputeAndJump(r, state); err != nil {
			return err
		}
	} else {
		// Just save the updated state.
		if err := saveBisectState(r.cfg.RewindDir, state); err != nil {
			return err
		}
		fmt.Printf("Checkpoint %s marked as %s.\n", shortID(cp.ID), kind)
		if state.GoodID == "" {
			fmt.Println("Now mark the bad checkpoint with 'rw bisect bad [id]'.")
		} else {
			fmt.Println("Now mark the good checkpoint with 'rw bisect good [id]'.")
		}
	}
	return nil
}

// bisectComputeAndJump computes the midpoint between bad and good checkpoints,
// navigates there, and prints bisect status.
func bisectComputeAndJump(r *repo, state *bisectState) error {
	// Collect candidates: walk from bad toward good along parent chain.
	candidates := collectCandidates(r.engine, state.BadID, state.GoodID)
	state.Candidates = candidates

	if len(candidates) == 0 {
		fmt.Printf("%sBisect complete: no candidates between good and bad.%s\n",
			colorYellow, colorReset)
		if err := saveBisectState(r.cfg.RewindDir, state); err != nil {
			return err
		}
		return nil
	}

	if len(candidates) == 1 {
		cp := r.engine.Index.Checkpoints[candidates[0]]
		fmt.Printf("%sBisect complete: bug introduced at %s: %q%s\n",
			colorGreen, shortID(cp.ID), cp.Message, colorReset)
		if err := saveBisectState(r.cfg.RewindDir, state); err != nil {
			return err
		}
		return nil
	}

	// Jump to midpoint.
	midIdx := len(candidates) / 2
	midID := candidates[midIdx]
	state.CurrentMid = midID

	if err := saveBisectState(r.cfg.RewindDir, state); err != nil {
		return err
	}

	// Restore to midpoint.
	lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
	fl := storage.NewFileLock(lockPath)
	err := fl.WithLock(func() error {
		midCP, ok := r.engine.Index.Checkpoints[midID]
		if !ok {
			return fmt.Errorf("mid checkpoint %s not found", shortID(midID))
		}
		if midCP.SnapshotRef == "" {
			return fmt.Errorf("mid checkpoint has no snapshot")
		}
		targetSnap, loadErr := r.scanner.Load(midCP.SnapshotRef)
		if loadErr != nil {
			return loadErr
		}
		if _, gotoErr := r.engine.GotoCheckpoint(midCP.ID); gotoErr != nil {
			return gotoErr
		}
		return r.scanner.Restore(targetSnap)
	})
	if err != nil {
		return err
	}

	midCP := r.engine.Index.Checkpoints[midID]
	remaining := len(candidates) / 2
	fmt.Printf("Bisecting: ~%d steps remaining. Currently at: %s %q\n",
		remaining, shortID(midCP.ID), midCP.Message)
	fmt.Println("Test your code. Run 'rw bisect good' or 'rw bisect bad'.")
	return nil
}

// collectCandidates walks from badID toward goodID along the parent chain,
// returning the IDs of checkpoints strictly between bad and good (exclusive).
func collectCandidates(engine *timeline.TimelineEngine, badID, goodID string) []string {
	var candidates []string
	visited := make(map[string]struct{})

	// Walk backward from bad, stop at good (exclusive).
	cur := badID
	for cur != "" && cur != goodID {
		if _, seen := visited[cur]; seen {
			break
		}
		visited[cur] = struct{}{}

		cp, ok := engine.Index.Checkpoints[cur]
		if !ok {
			break
		}

		// Include this node only if it's strictly between bad and good.
		if cur != badID {
			candidates = append(candidates, cur)
		}
		cur = cp.ParentID
	}

	return candidates
}

// loadBisectState reads and parses .rewind/bisect.json.
func loadBisectState(rewindDir string) (*bisectState, error) {
	path := filepath.Join(rewindDir, bisectFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bisect state: %w", err)
	}
	var state bisectState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse bisect state: %w", err)
	}
	return &state, nil
}

// saveBisectState writes the bisect state to .rewind/bisect.json.
func saveBisectState(rewindDir string, state *bisectState) error {
	path := filepath.Join(rewindDir, bisectFileName)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal bisect state: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}
