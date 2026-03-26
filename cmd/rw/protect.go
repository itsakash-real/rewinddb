package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const protectedFile = "protected.json"

func protectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "protect <file>...",
		Short: "Mark files as protected — they are never overwritten by goto or undo",
		Long: `Protected files are tracked in checkpoints (for diffing) but never
overwritten when restoring to a previous checkpoint.

Examples:
  rw protect .env
  rw protect secrets.json config/local.yml`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			protected, err := loadProtected(r.cfg.RewindDir)
			if err != nil {
				return err
			}

			for _, file := range args {
				// Normalize and sanitize path to prevent traversal attacks.
				file = filepath.ToSlash(filepath.Clean(file))
				if strings.HasPrefix(file, "../") || strings.HasPrefix(file, "/") {
					return fmt.Errorf("invalid path %q: must be relative to project root", file)
				}
				if !contains(protected, file) {
					protected = append(protected, file)
					printSuccess("protected: %s", file)
				} else {
					printDim("%s is already protected", file)
				}
			}

			return saveProtected(r.cfg.RewindDir, protected)
		},
	}
}

func unprotectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unprotect <file>...",
		Short: "Remove protection from files",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			protected, err := loadProtected(r.cfg.RewindDir)
			if err != nil {
				return err
			}

			for _, file := range args {
				file = filepath.ToSlash(file)
				if idx := indexOf(protected, file); idx >= 0 {
					protected = append(protected[:idx], protected[idx+1:]...)
					printSuccess("unprotected: %s", file)
				} else {
					printDim("%s was not protected", file)
				}
			}

			return saveProtected(r.cfg.RewindDir, protected)
		},
	}
}

func listProtectedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-protected",
		Short: "Show all protected files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			protected, err := loadProtected(r.cfg.RewindDir)
			if err != nil {
				return err
			}

			if len(protected) == 0 {
				printDim("no protected files")
				return nil
			}

			sectionTitle(fmt.Sprintf("protected files  ·  %d", len(protected)))
			fmt.Println()
			for _, f := range protected {
				fmt.Printf("  %s🔒 %s%s\n", colorCyan, f, colorReset)
			}
			fmt.Println()
			return nil
		},
	}
}

// loadProtected reads .rewind/protected.json.
func loadProtected(rewindDir string) ([]string, error) {
	path := filepath.Join(rewindDir, protectedFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load protected: %w", err)
	}
	var files []string
	if err := json.Unmarshal(data, &files); err != nil {
		return nil, fmt.Errorf("parse protected: %w", err)
	}
	return files, nil
}

// saveProtected atomically writes .rewind/protected.json.
func saveProtected(rewindDir string, files []string) error {
	path := filepath.Join(rewindDir, protectedFile)
	data, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal protected: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".protected-*.tmp")
	if err != nil {
		return fmt.Errorf("save protected: create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// IsProtected returns true if the file path is in the protected list.
func IsProtected(rewindDir, filePath string) bool {
	protected, err := loadProtected(rewindDir)
	if err != nil || len(protected) == 0 {
		return false
	}
	normalized := filepath.ToSlash(filePath)
	return contains(protected, normalized)
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func indexOf(ss []string, s string) int {
	for i, v := range ss {
		if v == s {
			return i
		}
	}
	return -1
}
