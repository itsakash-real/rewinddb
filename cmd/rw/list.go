package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
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

			color.New(color.Bold).Printf("● %s", branch.Name)
		fmt.Printf("  (%d checkpoints)\n", len(checkpoints))
			for _, cp := range checkpoints {
				printCheckpointLine(cp, headID)
			}
			if len(checkpoints) == 0 {
				fmt.Println("  (no checkpoints)")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&branchFlag, "branch", "b", "", "list checkpoints on a specific branch")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "show all branches in tree format")
	cmd.Flags().IntVarP(&limit, "limit", "n", 0, "max number of checkpoints to show (0 = all)")
	return cmd
}

// printCheckpointLine renders one checkpoint entry in git-log style.
//
//	◉ a3f2b1c  [HEAD]  2 minutes ago   "working auth"    [v1.2]
func printCheckpointLine(cp *timeline.Checkpoint, headID string) {
	now := time.Now()
	elapsed := int64(now.Sub(cp.CreatedAt.Local()).Seconds())
	relTime := humanTime(elapsed)

	short := shortID(cp.ID)
	msg := truncate(cp.Message, 48)

	isHead := cp.ID == headID

	// Circle marker: ◉ for HEAD, ○ for others.
	marker := "  ○"
	if isHead {
		marker = "  " + color.New(color.FgGreen, color.Bold).Sprint("◉")
	}

	headLabel := ""
	if isHead {
		headLabel = "  " + color.New(color.FgYellow, color.Bold).Sprint("[HEAD]")
	}

	idStr := color.New(color.Bold).Sprint(short)
	timeStr := color.New(color.Faint).Sprint(relTime)
	msgStr := fmt.Sprintf("%q", msg)

	tags := ""
	if len(cp.Tags) > 0 && !(len(cp.Tags) == 1 && cp.Tags[0] == "root") {
		tags = "  " + color.New(color.FgCyan).Sprint("["+strings.Join(cp.Tags, ", ")+"]")
	}

	fmt.Printf("%s %s%s  %-18s  %-50s%s%s\n",
		marker,
		idStr,
		headLabel,
		timeStr,
		msgStr,
		tags,
		"",
	)
}

// printAllBranches renders the --all tree view.
func printAllBranches(r *repo) error {
	headCheckpointID := r.engine.Index.CurrentCheckpointID

	for branchID, branch := range r.engine.Index.Branches {
		isCurrent := branchID == r.engine.Index.CurrentBranchID
		marker := "  "
		if isCurrent {
			marker = colorGreen + "* " + colorReset
		}
		fmt.Printf("\n%s%s%s%s\n", marker, colorBold, branch.Name, colorReset)

		checkpoints, err := r.engine.ListCheckpoints(branchID)
		if err != nil || len(checkpoints) == 0 {
			fmt.Printf("   └── (empty)\n")
			continue
		}

		for i, cp := range checkpoints {
			connector := "├──"
			if i == len(checkpoints)-1 {
				connector = "└──"
			}
			ts := cp.CreatedAt.Local().Format("2006-01-02 15:04")
			isHead := cp.ID == headCheckpointID
			headMark := ""
			if isHead {
				headMark = colorYellow + " (HEAD)" + colorReset
			}
			fmt.Printf("   %s %s%s%s  %s  %q%s\n",
				connector,
				colorBold, shortID(cp.ID), colorReset,
				colorDim+ts+colorReset,
				truncate(cp.Message, 40),
				headMark,
			)
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
	return s[:max-1] + "…"
}

func resolveBranchByName(engine *timeline.TimelineEngine, name string) (string, error) {
	for id, b := range engine.Index.Branches {
		if b.Name == name {
			return id, nil
		}
	}
	return "", fmt.Errorf("branch %q not found", name)
}
