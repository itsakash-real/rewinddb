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

const sessionsFileName = "sessions.json"

// sessionEntry represents a named work session.
type sessionEntry struct {
	Name            string    `json:"name"`
	StartCheckpoint string    `json:"start_checkpoint"`
	EndCheckpoint   string    `json:"end_checkpoint,omitempty"`
	StartedAt       time.Time `json:"started_at"`
	EndedAt         time.Time `json:"ended_at,omitempty"`
}

func sessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session <start|end|list|restore>",
		Short: "Manage named work sessions",
		Args:  cobra.RangeArgs(1, 2),
	}
	cmd.AddCommand(
		sessionStartCmd(),
		sessionEndCmd(),
		sessionListCmd(),
		sessionRestoreCmd(),
	)
	return cmd
}

func sessionStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <name>",
		Short: "Start a new named session at the current HEAD",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			r, err := loadRepo()
			if err != nil {
				return err
			}

			cp, ok := r.engine.Index.CurrentCheckpoint()
			if !ok {
				return fmt.Errorf("no HEAD checkpoint — run 'rw save' first")
			}

			sessions, err := loadSessions(r.cfg.RewindDir)
			if err != nil {
				sessions = []sessionEntry{}
			}

			// Check for duplicate name.
			for _, s := range sessions {
				if s.Name == name && s.EndCheckpoint == "" {
					return fmt.Errorf("session %q is already active", name)
				}
			}

			sessions = append(sessions, sessionEntry{
				Name:            name,
				StartCheckpoint: cp.ID,
				StartedAt:       time.Now().UTC(),
			})

			if err := saveSessions(r.cfg.RewindDir, sessions); err != nil {
				return err
			}

			fmt.Printf("%s✓ Session %q started at checkpoint %s%s\n",
				colorGreen, name, shortID(cp.ID), colorReset)
			return nil
		},
	}
}

func sessionEndCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "end",
		Short: "End the most recent active session at the current HEAD",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			cp, ok := r.engine.Index.CurrentCheckpoint()
			if !ok {
				return fmt.Errorf("no HEAD checkpoint")
			}

			sessions, err := loadSessions(r.cfg.RewindDir)
			if err != nil {
				return fmt.Errorf("no sessions found")
			}

			// Find the last active session (no EndCheckpoint).
			found := -1
			for i := len(sessions) - 1; i >= 0; i-- {
				if sessions[i].EndCheckpoint == "" {
					found = i
					break
				}
			}
			if found < 0 {
				return fmt.Errorf("no active session (use 'rw session start <name>' to begin one)")
			}

			sessions[found].EndCheckpoint = cp.ID
			sessions[found].EndedAt = time.Now().UTC()

			if err := saveSessions(r.cfg.RewindDir, sessions); err != nil {
				return err
			}

			duration := sessions[found].EndedAt.Sub(sessions[found].StartedAt).Round(time.Second)
			fmt.Printf("%s✓ Session %q ended. Duration: %s%s\n",
				colorGreen, sessions[found].Name, duration, colorReset)
			return nil
		},
	}
}

func sessionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all sessions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			sessions, err := loadSessions(r.cfg.RewindDir)
			if err != nil || len(sessions) == 0 {
				fmt.Println("No sessions recorded.")
				return nil
			}

			fmt.Printf("%-30s  %-10s  %-10s  %-10s  %s\n",
				"Name", "Start", "End", "Duration", "Started")
			fmt.Println(colorDim + "─────────────────────────────────────────────────────────────" + colorReset)

			for _, s := range sessions {
				endStr := shortID(s.EndCheckpoint)
				if endStr == "" {
					endStr = colorYellow + "(active)" + colorReset
				}
				duration := "(ongoing)"
				if !s.EndedAt.IsZero() {
					duration = s.EndedAt.Sub(s.StartedAt).Round(time.Second).String()
				}
				started := humanTime(int64(time.Since(s.StartedAt.Local()).Seconds()))
				fmt.Printf("%-30s  %-10s  %-10s  %-10s  %s\n",
					truncate(s.Name, 30),
					shortID(s.StartCheckpoint),
					endStr,
					duration,
					started,
				)
			}
			return nil
		},
	}
}

func sessionRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <name>",
		Short: "Restore to the start checkpoint of the named session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			r, err := loadRepo()
			if err != nil {
				return err
			}

			sessions, err := loadSessions(r.cfg.RewindDir)
			if err != nil {
				return fmt.Errorf("no sessions found")
			}

			var target *sessionEntry
			for i := range sessions {
				if sessions[i].Name == name {
					target = &sessions[i]
					break
				}
			}
			if target == nil {
				return fmt.Errorf("session %q not found", name)
			}

			// Use the lock and restore.
			lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
			fl := storage.NewFileLock(lockPath)
			return fl.WithLock(func() error {
				cp, ok := r.engine.Index.Checkpoints[target.StartCheckpoint]
				if !ok {
					return fmt.Errorf("start checkpoint %s not found", shortID(target.StartCheckpoint))
				}
				if cp.SnapshotRef == "" {
					return fmt.Errorf("start checkpoint has no snapshot")
				}
				targetSnap, loadErr := r.scanner.Load(cp.SnapshotRef)
				if loadErr != nil {
					return loadErr
				}
				if _, gotoErr := r.engine.GotoCheckpoint(cp.ID); gotoErr != nil {
					return gotoErr
				}
				if restoreErr := r.scanner.Restore(targetSnap); restoreErr != nil {
					return restoreErr
				}
				fmt.Printf("%s✓ Restored to session %q start: %s%s\n",
					colorGreen, name, shortID(cp.ID), colorReset)
				return nil
			})
		},
	}
}

// loadSessions reads sessions.json from the .rewind directory.
func loadSessions(rewindDir string) ([]sessionEntry, error) {
	path := filepath.Join(rewindDir, sessionsFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sessions []sessionEntry
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// saveSessions writes sessions.json to the .rewind directory.
func saveSessions(rewindDir string, sessions []sessionEntry) error {
	path := filepath.Join(rewindDir, sessionsFileName)
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sessions: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}
