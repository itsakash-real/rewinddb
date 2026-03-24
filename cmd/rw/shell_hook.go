package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func shellHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "_shell_hook",
		Short:  "Output a short prompt fragment for PS1 embedding",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				// Silently fail — don't pollute the prompt.
				return nil
			}

			branch, ok := r.engine.Index.CurrentBranch()
			if !ok {
				return nil
			}

			headCP, hasHead := r.engine.Index.CurrentCheckpoint()
			shortHead := "-----"
			relTime := ""
			if hasHead {
				shortHead = shortID(headCP.ID)
				elapsed := int64(time.Since(headCP.CreatedAt.Local()).Seconds())
				relTime = "·" + compactDuration(elapsed)
			}

			// Output: [rw: main·a3f2b1·2m]
			fmt.Printf("[rw: %s·%s%s]", branch.Name, shortHead, relTime)
			return nil
		},
	}
	return cmd
}

func shellSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "shell-setup",
		Short: "Print instructions to add RewindDB to your shell prompt",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(`To add RewindDB to your shell prompt, add this to your ~/.bashrc or ~/.zshrc:

  # RewindDB prompt
  export PROMPT_COMMAND='__rw_ps1() { RW_STATUS=$(rw _shell_hook 2>/dev/null); }; __rw_ps1'
  # Then add $RW_STATUS to your PS1, for example:
  # PS1='\u@\h:\w $RW_STATUS\$ '

For zsh, add to ~/.zshrc:
  precmd() { RW_STATUS=$(rw _shell_hook 2>/dev/null) }
  PROMPT='%n@%m %~ $RW_STATUS %% '`)
		},
	}
}

// compactDuration returns a compact time-since string: "2m", "3h", "5d".
func compactDuration(sec int64) string {
	switch {
	case sec < 60:
		return fmt.Sprintf("%ds", sec)
	case sec < 3600:
		return fmt.Sprintf("%dm", sec/60)
	case sec < 86400:
		return fmt.Sprintf("%dh", sec/3600)
	default:
		return fmt.Sprintf("%dd", sec/86400)
	}
}
