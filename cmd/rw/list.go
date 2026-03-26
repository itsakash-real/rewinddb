package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/itsakash-real/rewinddb/internal/timeline"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	var branchFlag string
	var all bool
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List checkpoints on the current (or specified) branch",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			if all {
				return printAllBranches(r)
			}

			// Resolve the target branch ID.
			branchID := r.engine.Index.CurrentBranchID
			if branchFlag != "" {
				branchID, err = resolveBranchByName(r.engine, branchFlag)
				if err != nil {
					return err
				}
			}

			checkpoints, err := r.engine.ListCheckpoints(branchID)
			if err != nil {
				return fmt.Errorf("list: %w", err)
			}

			if limit > 0 && len(checkpoints) > limit {
				checkpoints = checkpoints[:limit]
			}

			branch := r.engine.Index.Branches[branchID]
			headID := r.engine.Index.CurrentCheckpointID

			sectionTitle(fmt.Sprintf("%s  \u00b7  %d checkpoints", branch.Name, len(checkpoints)))
			fmt.Println()
			for _, cp := range checkpoints {
				printCheckpointLine(cp, headID, r.engine.Index)
			}
			if len(checkpoints) == 0 {
				fmt.Println("  (no checkpoints)")
			}

			// Note when HEAD is not at the branch tip (user has restored to an older checkpoint).
			if len(checkpoints) > 0 && headID != branch.HeadCheckpointID {
				newerCount := 0
				for _, cp := range checkpoints {
					if cp.ID == headID {
						break
					}
					newerCount++
				}
				if newerCount > 0 {
					fmt.Printf("\nNote: HEAD is not at the branch tip. You have %d newer checkpoint(s) above.\n",
						newerCount)
					fmt.Println("      Use 'rw goto HEAD' or 'rw list' to see your position.")
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&branchFlag, "branch", "b", "", "list checkpoints on a specific branch")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "show all branches in tree format")
	cmd.Flags().IntVarP(&limit, "limit", "n", 0, "max number of checkpoints to show (0 = all)")
	return cmd
}

// printCheckpointLine renders one checkpoint entry in the new purple style.
func printCheckpointLine(cp *timeline.Checkpoint, headID string, idx *timeline.Index) {
	elapsed := int64(time.Now().Sub(cp.CreatedAt.Local()).Seconds())
	isHead := cp.ID == headID

	sNum := idx.SNumberFor(cp.ID)
	idStr := shortID(cp.ID)
	// Show "S3  d0d2536c" format when S-number is available.
	if sNum != "" {
		idStr = fmt.Sprintf("%-4s %s", sNum, shortID(cp.ID))
	}
	timeStr := humanTime(elapsed)
	msg := truncate(cp.Message, 42)

	tags := ""
	if len(cp.Tags) > 0 && !(len(cp.Tags) == 1 && cp.Tags[0] == "root") {
		tags = "  " + cyanP.Sprint("["+strings.Join(cp.Tags, ", ")+"]")
	}

	if isHead {
		marker := purpleBoldP.Sprint("\u25c6")
		id := cyanP.Sprint(idStr)
		headTag := purpleBoldP.Sprint(" HEAD ")
		t := dimP.Sprint(fmt.Sprintf("%-18s", timeStr))
		fmt.Printf("  %s  %s %s  %s  %s%s\n", marker, id, headTag, t, msg, tags)
	} else {
		marker := dimP.Sprint("\u25cb")
		id := dimP.Sprint(idStr)
		t := dimP.Sprint(fmt.Sprintf("%-18s", timeStr))
		msgDim := dimP.Sprint(msg)
		fmt.Printf("  %s  %s            %s  %s%s\n", marker, id, t, msgDim, tags)
	}
}

// printAllBranches renders the --all tree view.
func printAllBranches(r *repo) error {
	headCheckpointID := r.engine.Index.CurrentCheckpointID

	for branchID, branch := range r.engine.Index.Branches {
		isCurrent := branchID == r.engine.Index.CurrentBranchID

		// current branch: purple ◆, others: dim ○
		if isCurrent {
			purpleBoldP.Printf("\n  \u25c6  %s\n", branch.Name)
		} else {
			fmt.Printf("\n  %s\u25cb  %s%s\n", colorDim, branch.Name, colorReset)
		}
		hrule(46)

		checkpoints, err := r.engine.ListCheckpoints(branchID)
		if err != nil || len(checkpoints) == 0 {
			fmt.Printf("   \u2514\u2500\u2500 (empty)\n")
			continue
		}

		for _, cp := range checkpoints {
			printCheckpointLine(cp, headCheckpointID, r.engine.Index)
		}
	}
	fmt.Println()
	return nil
}

func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "\u2026"
}

func resolveBranchByName(engine *timeline.TimelineEngine, name string) (string, error) {
	for id, b := range engine.Index.Branches {
		if b.Name == name {
			return id, nil
		}
	}
	return "", fmt.Errorf("branch %q not found", name)
}
