package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func heatmapCmd() *cobra.Command {
	var top int
	var since string

	cmd := &cobra.Command{
		Use:   "heatmap",
		Short: "Show which files changed most across checkpoints",
		Long: `Analyzes all checkpoints and counts how many times each file changed.
Use --top N to limit output. Use --since S2 to start from a specific checkpoint.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			// Determine the starting checkpoint for analysis.
			var sinceID string
			if since != "" {
				cp, err := resolveCheckpoint(r.engine, since)
				if err != nil {
					return err
				}
				sinceID = cp.ID
			}

			// Walk all checkpoints on the current branch.
			checkpoints, err := r.engine.ListCheckpoints("")
			if err != nil {
				return err
			}

			// Reverse to oldest-first.
			for i, j := 0, len(checkpoints)-1; i < j; i, j = i+1, j-1 {
				checkpoints[i], checkpoints[j] = checkpoints[j], checkpoints[i]
			}

			// If --since is set, trim to only checkpoints after sinceID.
			if sinceID != "" {
				start := -1
				for i, cp := range checkpoints {
					if cp.ID == sinceID {
						start = i
						break
					}
				}
				if start >= 0 {
					checkpoints = checkpoints[start:]
				}
			}

			// Count file changes between consecutive checkpoints.
			changeCounts := make(map[string]int)
			var prevFiles map[string]string // path -> hash

			for _, cp := range checkpoints {
				if cp.SnapshotRef == "" {
					continue
				}
				snap, err := r.scanner.Load(cp.SnapshotRef)
				if err != nil {
					continue
				}

				currentFiles := make(map[string]string, len(snap.Files))
				for _, f := range snap.Files {
					currentFiles[f.Path] = f.Hash
				}

				if prevFiles != nil {
					// Count added/modified files.
					for path, hash := range currentFiles {
						if prevHash, ok := prevFiles[path]; !ok || prevHash != hash {
							changeCounts[path]++
						}
					}
					// Count removed files.
					for path := range prevFiles {
						if _, ok := currentFiles[path]; !ok {
							changeCounts[path]++
						}
					}
				}

				prevFiles = currentFiles
			}

			if len(changeCounts) == 0 {
				printDim("no file changes found")
				return nil
			}

			// Sort by change count descending.
			type fileCount struct {
				path  string
				count int
			}
			var sorted []fileCount
			for path, count := range changeCounts {
				sorted = append(sorted, fileCount{path, count})
			}
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].count > sorted[j].count
			})

			if top > 0 && len(sorted) > top {
				sorted = sorted[:top]
			}

			// Find max count for bar scaling.
			maxCount := sorted[0].count
			maxBarWidth := 20

			sectionTitle(fmt.Sprintf("heatmap  ·  %d files", len(sorted)))
			fmt.Println()

			for _, fc := range sorted {
				barLen := (fc.count * maxBarWidth) / maxCount
				if barLen < 1 {
					barLen = 1
				}
				bar := strings.Repeat("█", barLen)
				fmt.Printf("  %-40s %s%s%s  %d changes\n",
					fc.path, colorYellow, bar, colorReset, fc.count)
			}
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().IntVar(&top, "top", 0, "show only top N files (0 = all)")
	cmd.Flags().StringVar(&since, "since", "", "analyze from this checkpoint (e.g. S2)")
	return cmd
}
