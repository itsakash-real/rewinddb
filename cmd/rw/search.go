package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func searchCmd() *cobra.Command {
	var branchFlag string

	cmd := &cobra.Command{
		Use:   "search <keyword>",
		Short: "Search checkpoints by message or tag (case-insensitive)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyword := args[0]
			lower := strings.ToLower(keyword)

			r, err := loadRepo()
			if err != nil {
				return err
			}

			headID := r.engine.Index.CurrentCheckpointID

			// If a branch filter is set, restrict to that branch.
			var branchIDs map[string]struct{}
			if branchFlag != "" {
				id, resolveErr := resolveBranchByName(r.engine, branchFlag)
				if resolveErr != nil {
					return resolveErr
				}
				branchIDs = map[string]struct{}{id: {}}
			}

			// Build a reverse map: checkpointID -> branchName(s).
			cpToBranch := make(map[string]string)
			for _, b := range r.engine.Index.Branches {
				cps, _ := r.engine.ListCheckpoints(b.ID)
				for _, cp := range cps {
					cpToBranch[cp.ID] = b.Name
				}
			}

			// Walk all checkpoints.
			type match struct {
				id         string
				message    string
				tags       []string
				branchName string
				createdAt  time.Time
			}
			var results []match

			for id, cp := range r.engine.Index.Checkpoints {
				// Branch filter.
				if branchIDs != nil {
					if _, ok := branchIDs[cp.BranchID]; !ok {
						continue
					}
				}

				// Match against message and tags.
				msgMatch := strings.Contains(strings.ToLower(cp.Message), lower)
				tagMatch := false
				for _, t := range cp.Tags {
					if strings.Contains(strings.ToLower(t), lower) {
						tagMatch = true
						break
					}
				}

				if msgMatch || tagMatch {
					results = append(results, match{
						id:         id,
						message:    cp.Message,
						tags:       cp.Tags,
						branchName: cpToBranch[id],
						createdAt:  cp.CreatedAt,
					})
				}
			}

			if len(results) == 0 {
				fmt.Printf("No checkpoints found matching %q\n", keyword)
				return nil
			}

			// Sort results by creation time (newest first).
			for i := 0; i < len(results)-1; i++ {
				for j := i + 1; j < len(results); j++ {
					if results[j].createdAt.After(results[i].createdAt) {
						results[i], results[j] = results[j], results[i]
					}
				}
			}

			yellowHL := color.New(color.FgYellow, color.Bold)
			boldPrint := color.New(color.Bold)
			dimPrint := color.New(color.Faint)
			cyanPrint := color.New(color.FgCyan)

			fmt.Printf("Found %d checkpoint(s) matching %q:\n\n", len(results), keyword)

			for _, m := range results {
				short := shortID(m.id)
				isHead := m.id == headID

				// Highlight keyword in message.
				highlighted := highlightKeyword(m.message, keyword, yellowHL)

				headLabel := ""
				if isHead {
					headLabel = "  " + color.New(color.FgYellow, color.Bold).Sprint("[HEAD]")
				}

				elapsed := int64(time.Since(m.createdAt.Local()).Seconds())
				relTime := humanTime(elapsed)

				tags := ""
				if len(m.tags) > 0 && !(len(m.tags) == 1 && m.tags[0] == "root") {
					tags = "  " + cyanPrint.Sprint("["+strings.Join(m.tags, ", ")+"]")
				}

				branchLabel := ""
				if m.branchName != "" {
					branchLabel = " " + dimPrint.Sprintf("(%s)", m.branchName)
				}

				fmt.Printf("  %s%s  %s  %s  %q%s%s\n",
					boldPrint.Sprint(short),
					headLabel,
					dimPrint.Sprint(relTime),
					branchLabel,
					highlighted,
					tags,
					"",
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&branchFlag, "branch", "", "restrict search to a specific branch")
	return cmd
}

// highlightKeyword wraps every (case-insensitive) occurrence of keyword in text
// with the provided color printer, returning the marked-up string.
func highlightKeyword(text, keyword string, hl *color.Color) string {
	lower := strings.ToLower(text)
	lowerKW := strings.ToLower(keyword)

	var result strings.Builder
	remaining := text
	lowerRemaining := lower

	for {
		idx := strings.Index(lowerRemaining, lowerKW)
		if idx < 0 {
			result.WriteString(remaining)
			break
		}
		result.WriteString(remaining[:idx])
		result.WriteString(hl.Sprint(remaining[idx : idx+len(keyword)]))
		remaining = remaining[idx+len(keyword):]
		lowerRemaining = lowerRemaining[idx+len(keyword):]
	}
	return result.String()
}
