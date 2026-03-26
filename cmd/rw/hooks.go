package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

const hooksDir = "hooks"

func hooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage lifecycle hooks (pre-save, post-save, pre-restore, post-restore)",
		Long: `Hooks are executable scripts in .rewind/hooks/ that run at key lifecycle points.

Available hooks:
  pre-save       runs before 'rw save' (non-zero exit cancels save)
  post-save      runs after 'rw save'
  pre-restore    runs before 'rw goto' / 'rw undo'
  post-restore   runs after 'rw goto' / 'rw undo'

Environment variables passed to hooks:
  RW_CHECKPOINT_ID    the checkpoint ID
  RW_MESSAGE          the checkpoint message

Use 'rw hooks install' to create example hook files.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			dir := filepath.Join(r.cfg.RewindDir, hooksDir)
			hookNames := []string{"pre-save", "post-save", "pre-restore", "post-restore"}

			sectionTitle("hooks")
			fmt.Println()

			for _, name := range hookNames {
				path := hookPath(dir, name)
				if _, err := os.Stat(path); err == nil {
					printOK(name)
				} else {
					printDim("%s  (not installed)", name)
				}
			}
			fmt.Println()
			return nil
		},
	}

	cmd.AddCommand(hooksInstallCmd())
	return cmd
}

func hooksInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Create example hook files in .rewind/hooks/",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			dir := filepath.Join(r.cfg.RewindDir, hooksDir)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("create hooks dir: %w", err)
			}

			hooks := map[string]string{
				"pre-save": `#!/bin/sh
# Pre-save hook: runs before 'rw save'.
# Exit non-zero to cancel the save.
# Example: run tests before saving
# go test ./... || exit 1
echo "pre-save hook running"
`,
				"post-save": `#!/bin/sh
# Post-save hook: runs after 'rw save'.
# RW_CHECKPOINT_ID and RW_MESSAGE are available.
echo "post-save: checkpoint $RW_CHECKPOINT_ID saved"
`,
				"pre-restore": `#!/bin/sh
# Pre-restore hook: runs before 'rw goto' / 'rw undo'.
echo "pre-restore hook running"
`,
				"post-restore": `#!/bin/sh
# Post-restore hook: runs after 'rw goto' / 'rw undo'.
echo "post-restore: restored to $RW_CHECKPOINT_ID"
`,
			}

			for name, content := range hooks {
				path := hookPath(dir, name)
				if _, err := os.Stat(path); err == nil {
					printDim("  %s already exists, skipping", name)
					continue
				}
				if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
					return fmt.Errorf("write hook %s: %w", name, err)
				}
				printSuccess("created %s", path)
			}
			fmt.Println()
			return nil
		},
	}
}

// RunHook executes a hook by name if it exists. Returns nil if the hook
// doesn't exist. Returns an error if the hook exits non-zero.
func RunHook(rewindDir, hookName, checkpointID, message string) error {
	dir := filepath.Join(rewindDir, hooksDir)
	path := hookPath(dir, hookName)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // hook not installed
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", path)
	} else {
		cmd = exec.Command("sh", path)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"RW_CHECKPOINT_ID="+checkpointID,
		"RW_MESSAGE="+message,
	)

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("hook %s failed (exit %d)", hookName, exitErr.ExitCode())
		}
		return fmt.Errorf("hook %s: %w", hookName, err)
	}
	return nil
}

func hookPath(dir, name string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(dir, name+".cmd")
	}
	return filepath.Join(dir, name)
}
