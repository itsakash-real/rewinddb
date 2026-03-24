package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const rewindIgnoreFile = ".rewindignore"

func ignoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ignore <add|list|auto>",
		Short: "Manage .rewindignore patterns",
		Args:  cobra.RangeArgs(1, 2),
	}
	cmd.AddCommand(
		ignoreAddCmd(),
		ignoreListCmd(),
		ignoreAutoCmd(),
	)
	return cmd
}

func ignoreAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <pattern>",
		Short: "Add a pattern to .rewindignore",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern := args[0]

			r, err := loadRepo()
			if err != nil {
				return err
			}
			projectRoot := parentDir(r.cfg.RewindDir)

			added, err := addIgnorePattern(projectRoot, pattern)
			if err != nil {
				return err
			}
			if added {
				fmt.Printf("%s✓ Added %q to .rewindignore%s\n", colorGreen, pattern, colorReset)
			} else {
				fmt.Printf("  Pattern %q already in .rewindignore\n", pattern)
			}
			return nil
		},
	}
}

func ignoreListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show current .rewindignore contents",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}
			projectRoot := parentDir(r.cfg.RewindDir)
			ignorePath := filepath.Join(projectRoot, rewindIgnoreFile)

			data, err := os.ReadFile(ignorePath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No .rewindignore file found.")
					return nil
				}
				return fmt.Errorf("read .rewindignore: %w", err)
			}

			lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
			fmt.Printf(".rewindignore (%d patterns):\n", len(lines))
			for _, l := range lines {
				if l != "" {
					fmt.Printf("  %s\n", l)
				}
			}
			return nil
		},
	}
}

func ignoreAutoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "auto",
		Short: "Auto-detect and add common ignore patterns based on project type",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}
			projectRoot := parentDir(r.cfg.RewindDir)

			// Always-added patterns.
			always := []string{
				".env",
				".env.local",
				"*.log",
				"tmp/",
				"temp/",
			}

			candidates := make([]string, 0, 20)
			candidates = append(candidates, always...)

			// Language/framework detection.
			if fileExistsAt(projectRoot, "package.json") {
				candidates = append(candidates, "node_modules/")
			}
			if fileExistsAt(projectRoot, "go.mod") {
				candidates = append(candidates, "vendor/", "*.test", "*.exe")
			}
			if fileExistsAt(projectRoot, "requirements.txt") || fileExistsAt(projectRoot, "pyproject.toml") {
				candidates = append(candidates, "__pycache__/", ".venv/", "*.pyc")
			}
			if fileExistsAt(projectRoot, "Cargo.toml") {
				candidates = append(candidates, "target/")
			}
			if fileExistsAt(projectRoot, "pom.xml") {
				candidates = append(candidates, "target/")
			}

			added := 0
			for _, pat := range candidates {
				ok, addErr := addIgnorePattern(projectRoot, pat)
				if addErr != nil {
					return addErr
				}
				if ok {
					fmt.Printf("  Added: %s\n", pat)
					added++
				} else {
					fmt.Printf("  Skip (already present): %s\n", pat)
				}
			}

			fmt.Printf("\n%s✓ Auto-ignore complete: %d pattern(s) added.%s\n",
				colorGreen, added, colorReset)
			return nil
		},
	}
}

// addIgnorePattern appends a pattern to .rewindignore if not already present.
// Returns true if added, false if already present.
func addIgnorePattern(projectRoot, pattern string) (bool, error) {
	ignorePath := filepath.Join(projectRoot, rewindIgnoreFile)

	// Read existing patterns.
	existing := make(map[string]struct{})
	if data, err := os.ReadFile(ignorePath); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				existing[line] = struct{}{}
			}
		}
	}

	if _, ok := existing[pattern]; ok {
		return false, nil // already present
	}

	// Append to file (create if not exists).
	f, err := os.OpenFile(ignorePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return false, fmt.Errorf("open .rewindignore: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintln(f, pattern); err != nil {
		return false, fmt.Errorf("write .rewindignore: %w", err)
	}
	return true, nil
}

// fileExistsAt returns true if filename exists directly inside dir.
func fileExistsAt(dir, filename string) bool {
	_, err := os.Stat(filepath.Join(dir, filename))
	return err == nil
}
