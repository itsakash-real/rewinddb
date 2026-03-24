package main

import (
	"fmt"
	"path/filepath"

	"github.com/itsakash-real/rewinddb/internal/storage"
	"github.com/itsakash-real/rewinddb/internal/timeline"
	"github.com/spf13/cobra"
)

// branchesRootCmd groups: rw branches, rw branch <name>, rw switch <name>
func branchesCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "branches",
		Short: "Branch management (list, create, switch)",
		// Running `rw branches` with no subcommand lists all branches.
		RunE: listBranchesRunE,
	}

	root.AddCommand(branchCreateCmd())
	root.AddCommand(branchSwitchCmd())
	return root
}

// ─── rw branches (list) ───────────────────────────────────────────────────────

func listBranchesRunE(cmd *cobra.Command, _ []string) error {
	r, err := loadRepo()
	if err != nil {
		return err
	}

	fmt.Printf("%-30s  %-10s  %-8s  %s\n",
		colorBold+"Branch"+colorReset,
		colorBold+"Head"+colorReset,
		colorBold+"Commits"+colorReset,
		colorBold+"Latest message"+colorReset,
	)
	fmt.Println(repeatStr("─", 80))

	currentBranchID := r.engine.Index.CurrentBranchID

	for id, branch := range r.engine.Index.Branches {
		isCurrent := id == currentBranchID

		// Count checkpoints on this branch.
		cps, _ := r.engine.ListCheckpoints(id)
		count := len(cps)

		// Head checkpoint short ID + message.
		headShort := "—"
		headMsg := ""
		if branch.HeadCheckpointID != "" {
			headShort = shortID(branch.HeadCheckpointID)
			if cp, ok := r.engine.Index.Checkpoints[branch.HeadCheckpointID]; ok {
				headMsg = truncate(cp.Message, 40)
			}
		}

		currentMark := "  "
		nameColor := ""
		if isCurrent {
			currentMark = colorGreen + "* " + colorReset
			nameColor = colorBold
		}

		fmt.Printf("%s%s%-28s%s  %-10s  %-8d  %s\n",
			currentMark,
			nameColor, branch.Name, colorReset,
			headShort,
			count,
			headMsg,
		)
	}
	return nil
}

// ─── rw branch <name> ─────────────────────────────────────────────────────────

func branchCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "branch <name>",
		Short: "Create a new named branch at the current checkpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			r, err := loadRepo()
			if err != nil {
				return err
			}

			lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
			fl := storage.NewFileLock(lockPath)
			return fl.WithLock(func() error {
				// Prevent duplicate branch names.
				for _, b := range r.engine.Index.Branches {
					if b.Name == name {
						return fmt.Errorf("branch %q already exists", name)
					}
				}

				currentCPID := r.engine.Index.CurrentCheckpointID
				if currentCPID == "" {
					return fmt.Errorf("no checkpoint to branch from — run 'rw save' first")
				}

				newBranch := timeline.NewBranch(name, currentCPID)
				r.engine.Index.AddBranch(newBranch)

				if err := r.engine.Index.Save(r.cfg.IndexPath); err != nil {
					return fmt.Errorf("persist index: %w", err)
				}

				fmt.Printf("✓ Created branch %s%q%s at %s\n",
					colorBold, name, colorReset,
					shortID(currentCPID),
				)
				return nil
			})
		},
	}
}

// ─── rw switch <branch-name-or-id> ───────────────────────────────────────────

func branchSwitchCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "switch <branch-name-or-id>",
		Short: "Switch to a branch and restore its HEAD state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrID := args[0]

			r, err := loadRepo()
			if err != nil {
				return err
			}

			// Resolve branch by name, then by ID prefix.
			branchID, err := resolveBranchByName(r.engine, nameOrID)
			if err != nil {
				// Try by ID prefix.
				branchID, err = resolveBranchByIDPrefix(r.engine, nameOrID)
				if err != nil {
					return fmt.Errorf("branch %q not found", nameOrID)
				}
			}

			branch := r.engine.Index.Branches[branchID]
			headCPID := branch.HeadCheckpointID
			headCP, ok := r.engine.Index.Checkpoints[headCPID]
			if !ok {
				return fmt.Errorf("branch head checkpoint not found")
			}

			if !force {
				prompt := fmt.Sprintf("Switch to branch %q (HEAD: %s %q)? [y/N]",
					branch.Name, shortID(headCPID), headCP.Message)
				if !askConfirm(prompt) {
					fmt.Println("Aborted.")
					return nil
				}
			}

			// Load snapshot for branch HEAD.
			snap, err := r.scanner.Load(headCP.SnapshotRef)
			if err != nil {
				return fmt.Errorf("load snapshot for branch head: %w", err)
			}

			// Move index HEAD.
			r.engine.Index.CurrentBranchID = branchID
			r.engine.Index.CurrentCheckpointID = headCPID
			if err := r.engine.Index.Save(r.cfg.IndexPath); err != nil {
				return fmt.Errorf("persist index: %w", err)
			}

			// Restore files.
			if err := r.scanner.Restore(snap); err != nil {
				return fmt.Errorf("restore files: %w", err)
			}

			fmt.Printf("✓ Switched to branch %s%q%s\n", colorBold, branch.Name, colorReset)
			fmt.Printf("  HEAD: %s  %q\n", shortID(headCPID), headCP.Message)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")
	return cmd
}

// resolveBranchByIDPrefix finds a branch whose ID starts with prefix.
func resolveBranchByIDPrefix(engine *timeline.TimelineEngine, prefix string) (string, error) {
	var matches []string
	for id := range engine.Index.Branches {
		if len(id) >= len(prefix) && id[:len(prefix)] == prefix {
			matches = append(matches, id)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("no branch with ID prefix %q", prefix)
	default:
		return "", fmt.Errorf("ambiguous branch prefix %q", prefix)
	}
}

func repeatStr(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
