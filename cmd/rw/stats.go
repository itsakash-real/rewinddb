package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func statsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show repository statistics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			projectRoot := parentDir(r.cfg.RewindDir)
			branch, _ := r.engine.Index.CurrentBranch()
			headCP, hasHead := r.engine.Index.CurrentCheckpoint()

			objectCount, totalBytes, _ := r.store.Stats()

			// Count sessions (unique branches minus "main").
			sessionCount := len(r.engine.Index.Branches)

			// First and latest checkpoint times.
			var firstTime, latestTime time.Time
			for _, cp := range r.engine.Index.Checkpoints {
				if firstTime.IsZero() || cp.CreatedAt.Before(firstTime) {
					firstTime = cp.CreatedAt
				}
				if latestTime.IsZero() || cp.CreatedAt.After(latestTime) {
					latestTime = cp.CreatedAt
				}
			}

			bold := color.New(color.Bold)
			header := color.New(color.Bold, color.FgCyan)

			// ── Header ──────────────────────────────────────────────────────
			bold.Println("RewindDB Statistics")
			fmt.Println(strings.Repeat("═", 42))
			fmt.Println()

			// ── Repository ──────────────────────────────────────────────────
			fmt.Printf("  %-18s %s\n", "Repository:", projectRoot)
			fmt.Printf("  %-18s %s\n", "Branch:", branch.Name)
			if hasHead {
				fmt.Printf("  %-18s %s  %q\n", "Head:", shortID(headCP.ID), headCP.Message)
			} else {
				fmt.Printf("  %-18s (none)\n", "Head:")
			}

			fmt.Println()
			header.Println("  Timeline")
			fmt.Println("  " + strings.Repeat("─", 38))
			fmt.Printf("  %-28s %d\n", "Total checkpoints:", len(r.engine.Index.Checkpoints))
			fmt.Printf("  %-28s %d\n", "Total branches:", len(r.engine.Index.Branches))
			fmt.Printf("  %-28s %d\n", "Sessions:", sessionCount)

			fmt.Println()
			header.Println("  Storage")
			fmt.Println("  " + strings.Repeat("─", 38))
			fmt.Printf("  %-28s %d\n", "Objects stored:", objectCount)
			fmt.Printf("  %-28s %s\n", "Total size:", formatBytes(totalBytes))

			// Compression ratio: estimate raw size vs stored size.
			// (We report stored vs estimated raw; stored is compressed.)
			fmt.Printf("  %-28s %s\n", "Compression:", "stored (gzip compressed)")

			fmt.Println()
			header.Println("  Activity")
			fmt.Println("  " + strings.Repeat("─", 38))
			if !firstTime.IsZero() {
				elapsed := int64(time.Since(firstTime.Local()).Seconds())
				fmt.Printf("  %-28s %s\n", "First checkpoint:", humanTime(elapsed))
			} else {
				fmt.Printf("  %-28s %s\n", "First checkpoint:", "(none)")
			}
			if !latestTime.IsZero() {
				elapsed := int64(time.Since(latestTime.Local()).Seconds())
				fmt.Printf("  %-28s %s\n", "Latest checkpoint:", humanTime(elapsed))
			} else {
				fmt.Printf("  %-28s %s\n", "Latest checkpoint:", "(none)")
			}

			fmt.Println()
			return nil
		},
	}
}
