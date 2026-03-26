package main

import (
	"fmt"

	diffpkg "github.com/itsakash-real/rewinddb/internal/diff"
	"github.com/itsakash-real/rewinddb/internal/timeline"
	"github.com/spf13/cobra"
)

func diffCmd() *cobra.Command {
	var stat bool

	cmd := &cobra.Command{
		Use:   "diff <id1> [id2]",
		Short: "Show file-level diff between two checkpoints",
		Long: `Compare two checkpoints. If only one ID is given, it is compared
against the current HEAD. Supports 8-char prefix matching.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			// ── Resolve checkpoint A ──────────────────────────────────────────
			cpA, err := resolveCheckpoint(r.engine, args[0])
			if err != nil {
				return fmt.Errorf("cannot resolve first checkpoint: %w", err)
			}

			// ── Resolve checkpoint B (default = current HEAD) ─────────────────
			var cpB timeline.Checkpoint
			if len(args) == 2 {
				cpB, err = resolveCheckpoint(r.engine, args[1])
				if err != nil {
					return fmt.Errorf("cannot resolve second checkpoint: %w", err)
				}
			} else {
				headID := r.engine.Index.CurrentCheckpointID
				if headID == "" {
					return fmt.Errorf("no HEAD checkpoint — save at least one checkpoint first")
				}
				cpB, err = resolveCheckpoint(r.engine, headID)
				if err != nil {
					return fmt.Errorf("cannot resolve HEAD checkpoint: %w", err)
				}
			}

			// ── Guard: root checkpoints have no snapshot ──────────────────────
			if cpA.SnapshotRef == "" {
				return fmt.Errorf("Root checkpoint has no files to compare. Choose a later checkpoint.")
			}
			if cpB.SnapshotRef == "" {
				return fmt.Errorf("Root checkpoint has no files to compare. Choose a later checkpoint.")
			}

			// ── Load snapshots ────────────────────────────────────────────────
			snapA, err := r.scanner.Load(cpA.SnapshotRef)
			if err != nil {
				return fmt.Errorf("load snapshot for %s: %w", shortID(cpA.ID), err)
			}
			snapB, err := r.scanner.Load(cpB.SnapshotRef)
			if err != nil {
				return fmt.Errorf("load snapshot for %s: %w", shortID(cpB.ID), err)
			}

			// ── Run diff ──────────────────────────────────────────────────────
			diffEng := diffpkg.New(r.store)
			result, err := diffEng.Compare(snapA, snapB)
			if err != nil {
				return fmt.Errorf("compare: %w", err)
			}

			// ── Print header ──────────────────────────────────────────────────
			fmt.Printf("%sDiff%s  %s%s%s → %s%s%s\n",
				colorBold, colorReset,
				colorCyan, shortID(cpA.ID), colorReset,
				colorCyan, shortID(cpB.ID), colorReset,
			)
			fmt.Printf("      %s%q%s → %s%q%s\n\n",
				colorDim, cpA.Message, colorReset,
				colorDim, cpB.Message, colorReset,
			)

			// ── --stat: summary only ──────────────────────────────────────────
			if stat {
				fmt.Println(diffEng.Summary(result))
				return nil
			}

			// ── Full pretty-print ─────────────────────────────────────────────
			fmt.Print(diffEng.PrettyPrint(result))

			// ── Per-file text diff for modified text files ────────────────────
			if len(result.Modified) > 0 {
				fmt.Printf("\n%s── Text diffs ──%s\n", colorBold, colorReset)
				for _, fd := range result.Modified {
					textDiff, err := diffEng.TextDiff(fd.OldHash, fd.NewHash)
					if err != nil || textDiff == "binary files differ" || textDiff == "(files are identical)" {
						fmt.Printf("\n%s%s%s: %s\n", colorBold, fd.Path, colorReset,
							firstLine(textDiff, "binary files differ"))
						continue
					}
					fmt.Printf("\n%s%s%s\n", colorBold, fd.Path, colorReset)
					// Colour unified diff lines: + green, - red, @@ cyan.
					for _, line := range splitLines(textDiff) {
						switch {
						case len(line) > 0 && line[0] == '+':
							fmt.Printf("%s%s%s\n", colorGreen, line, colorReset)
						case len(line) > 0 && line[0] == '-':
							fmt.Printf("%s%s%s\n", colorRed, line, colorReset)
						case len(line) > 1 && line[:2] == "@@":
							fmt.Printf("%s%s%s\n", colorCyan, line, colorReset)
						default:
							fmt.Println(line)
						}
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&stat, "stat", false, "print only the summary line")
	return cmd
}

// splitLines splits a string on "\n", preserving empty lines.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func firstLine(s, fallback string) string {
	if s == "" {
		return fallback
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}
