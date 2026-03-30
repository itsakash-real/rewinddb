package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	diffpkg "github.com/itsakash-real/nimbi/internal/diff"
	"github.com/itsakash-real/nimbi/internal/timeline"
	"github.com/spf13/cobra"
)

func diffCmd() *cobra.Command {
	var stat bool
	var categorize bool

	cmd := &cobra.Command{
		Use:   "diff <id1> [id2]",
		Short: "Show file-level diff between two checkpoints",
		Long: `Compare two checkpoints. If only one ID is given, it is compared
against the current HEAD. Supports 8-char prefix matching.

Use --categorize to see which files git would track vs files only nimbi tracks.`,
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

			// ── --categorize: git-tracked vs nimbi-only ───────────────────────
			if categorize {
				projectRoot := parentDir(r.cfg.RewindDir)
				printCategorizedDiff(result, projectRoot)

				// Text diffs for modified
				if len(result.Modified) > 0 {
					fmt.Printf("\n%s── Text diffs ──%s\n", colorBold, colorReset)
					printTextDiffs(diffEng, result)
				}
				return nil
			}

			// ── Full pretty-print ─────────────────────────────────────────────
			fmt.Print(diffEng.PrettyPrint(result))

			// ── Per-file text diff for modified text files ────────────────────
			if len(result.Modified) > 0 {
				fmt.Printf("\n%s── Text diffs ──%s\n", colorBold, colorReset)
				printTextDiffs(diffEng, result)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&stat, "stat", false, "print only the summary line")
	cmd.Flags().BoolVar(&categorize, "categorize", false, "separate files git would show vs files only nimbi tracks")
	return cmd
}

// printCategorizedDiff separates diff output into git-tracked vs nimbi-only files.
func printCategorizedDiff(result *diffpkg.DiffResult, projectRoot string) {
	gitIgnored := loadGitIgnorePatterns(projectRoot)

	// Classify changes.
	type changeEntry struct {
		symbol string // "+", "~", "-"
		path   string
		reason string // for nimbi-only files
		color  string
	}

	var gitChanges []changeEntry
	var nimbiOnly []changeEntry

	// Added files.
	for _, f := range result.Added {
		if isPathGitIgnored(f.Path, gitIgnored) {
			nimbiOnly = append(nimbiOnly, changeEntry{"+", f.Path, reasonForFile(f.Path), colorGreen})
		} else {
			gitChanges = append(gitChanges, changeEntry{"+", f.Path, "", colorGreen})
		}
	}

	// Removed files.
	for _, f := range result.Removed {
		if isPathGitIgnored(f.Path, gitIgnored) {
			nimbiOnly = append(nimbiOnly, changeEntry{"-", f.Path, reasonForFile(f.Path), colorRed})
		} else {
			gitChanges = append(gitChanges, changeEntry{"-", f.Path, "", colorRed})
		}
	}

	// Modified files.
	for _, fd := range result.Modified {
		if isPathGitIgnored(fd.Path, gitIgnored) {
			nimbiOnly = append(nimbiOnly, changeEntry{"~", fd.Path, reasonForFile(fd.Path), colorYellow})
		} else {
			gitChanges = append(gitChanges, changeEntry{"~", fd.Path, "", colorYellow})
		}
	}

	// Print git-tracked changes.
	if len(gitChanges) > 0 {
		fmt.Printf("%s%sfiles git would show:%s\n", colorBold, colorDim, colorReset)
		for _, c := range gitChanges {
			fmt.Printf("  %s%s%s  %s\n", c.color, c.symbol, colorReset, c.path)
		}
	}

	// Print nimbi-only changes.
	if len(nimbiOnly) > 0 {
		if len(gitChanges) > 0 {
			fmt.Println()
		}
		fmt.Printf("%s%sfiles ONLY nimbi tracks:%s\n", colorBold, colorCyan, colorReset)
		for _, c := range nimbiOnly {
			fmt.Printf("  %s%s%s  %-30s %s← %s%s\n",
				c.color, c.symbol, colorReset,
				c.path,
				colorDim, c.reason, colorReset)
		}
	}

	if len(gitChanges) == 0 && len(nimbiOnly) == 0 {
		fmt.Println("No changes between snapshots.")
	}
}

// loadGitIgnorePatterns reads .gitignore and returns the raw patterns.
func loadGitIgnorePatterns(projectRoot string) []string {
	path := filepath.Join(projectRoot, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// isPathGitIgnored heuristically checks if a path matches .gitignore patterns.
func isPathGitIgnored(relPath string, patterns []string) bool {
	for _, pattern := range patterns {
		clean := strings.TrimSuffix(strings.TrimPrefix(pattern, "/"), "/")

		// Direct match.
		if clean == relPath || clean == filepath.Dir(relPath) {
			return true
		}

		// Prefix match for directories.
		if strings.HasSuffix(pattern, "/") {
			dir := strings.TrimSuffix(pattern, "/")
			if strings.HasPrefix(relPath, dir+"/") || relPath == dir {
				return true
			}
		}

		// Wildcard match.
		if strings.HasPrefix(pattern, "*") {
			suffix := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(relPath, suffix) {
				return true
			}
		}
	}
	return false
}

// reasonForFile returns a human-readable reason why a file is nimbi-only.
func reasonForFile(path string) string {
	lower := strings.ToLower(path)

	switch {
	case strings.HasPrefix(lower, ".env"):
		return "environment config"
	case strings.Contains(lower, "config/local"):
		return "local config"
	case strings.Contains(lower, "secret"):
		return "secrets file"
	case strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".dll") || strings.HasSuffix(lower, ".so"):
		return "compiled binary"
	case strings.HasPrefix(lower, "build/") || strings.HasPrefix(lower, "dist/") || strings.HasPrefix(lower, "out/"):
		return "build output"
	case strings.HasSuffix(lower, ".log"):
		return "log file"
	case strings.HasSuffix(lower, ".db") || strings.HasSuffix(lower, ".sqlite"):
		return "database file"
	default:
		return "not in git"
	}
}

// printTextDiffs prints per-file text diffs for modified files.
func printTextDiffs(diffEng *diffpkg.Engine, result *diffpkg.DiffResult) {
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
