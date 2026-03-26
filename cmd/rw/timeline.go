package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/itsakash-real/rewinddb/internal/timeline"
	"github.com/spf13/cobra"
)

func timelineCmd() *cobra.Command {
	var visual bool

	cmd := &cobra.Command{
		Use:   "timeline",
		Short: "Show the checkpoint timeline",
		Long:  `Displays the checkpoint history. Use --visual for an ASCII art DAG view.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			if visual {
				return printVisualTimeline(r)
			}

			// Default: delegate to list.
			checkpoints, err := r.engine.ListCheckpoints("")
			if err != nil {
				return err
			}
			branch, _ := r.engine.Index.CurrentBranch()
			headID := r.engine.Index.CurrentCheckpointID

			sectionTitle(fmt.Sprintf("timeline  ·  %s", branch.Name))
			fmt.Println()
			for _, cp := range checkpoints {
				printCheckpointLine(cp, headID, r.engine.Index)
			}
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().BoolVar(&visual, "visual", false, "show ASCII art DAG timeline")
	return cmd
}

// printVisualTimeline renders an ASCII art view of the DAG.
//
//	main:  S1──S2──S3──S4
//	           └──B1──B2  (HEAD)
func printVisualTimeline(r *repo) error {
	headID := r.engine.Index.CurrentCheckpointID
	currentBranchID := r.engine.Index.CurrentBranchID

	// Collect branches sorted by creation time (main first).
	type branchInfo struct {
		id     string
		branch timeline.Branch
	}
	var branches []branchInfo
	for id, b := range r.engine.Index.Branches {
		branches = append(branches, branchInfo{id, b})
	}
	sort.Slice(branches, func(i, j int) bool {
		if branches[i].branch.Name == "main" {
			return true
		}
		if branches[j].branch.Name == "main" {
			return false
		}
		return branches[i].branch.CreatedAt.Before(branches[j].branch.CreatedAt)
	})

	sectionTitle("timeline")
	fmt.Println()

	for _, bi := range branches {
		cps, err := r.engine.ListCheckpoints(bi.id)
		if err != nil || len(cps) == 0 {
			continue
		}

		// Reverse to get oldest-first.
		for i, j := 0, len(cps)-1; i < j; i, j = i+1, j-1 {
			cps[i], cps[j] = cps[j], cps[i]
		}

		isCurrent := bi.id == currentBranchID
		branchLabel := bi.branch.Name
		if isCurrent {
			branchLabel = purpleBoldP.Sprint(branchLabel)
		} else {
			branchLabel = dimP.Sprint(branchLabel)
		}

		// Build the node line.
		var nodes []string
		for _, cp := range cps {
			sNum := r.engine.Index.SNumberFor(cp.ID)
			label := shortID(cp.ID)[:4]
			if sNum != "" {
				label = sNum
			}
			if cp.ID == headID {
				label = cyanP.Sprint(label) + purpleBoldP.Sprint("*")
			} else {
				label = dimP.Sprint(label)
			}
			nodes = append(nodes, label)
		}

		line := strings.Join(nodes, dimP.Sprint("──"))
		suffix := ""
		if bi.id == currentBranchID {
			suffix = purpleBoldP.Sprint("  (HEAD)")
		}

		// For non-main branches, show a fork marker.
		if bi.branch.Name != "main" {
			fmt.Printf("  %s  %s└──%s%s%s\n",
				fmt.Sprintf("%-12s", branchLabel),
				colorDim, colorReset,
				line, suffix)
		} else {
			fmt.Printf("  %s  %s%s\n",
				fmt.Sprintf("%-12s", branchLabel),
				line, suffix)
		}
	}

	fmt.Println()
	return nil
}
