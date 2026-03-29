package main

import (
	"os"

	"github.com/spf13/cobra"
)

// completionCmd delegates entirely to Cobra's built-in completion engine.
// Cobra generates correct completions for bash, zsh, fish, and PowerShell
// automatically from command/flag metadata [web:98].
func completionCmd(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for Nimbi.

Bash:
  # Add to ~/.bashrc:
  source <(rw completion bash)

  # Or install system-wide:
  rw completion bash > /etc/bash_completion.d/rw

Zsh:
  # If shell completion is not already enabled, enable it:
  echo "autoload -U compinit; compinit" >> ~/.zshrc

  # Add to ~/.zshrc:
  source <(rw completion zsh)

  # Or install to your $fpath:
  rw completion zsh > "${fpath[1]}/_rw"

Fish:
  rw completion fish | source

  # Or persist:
  rw completion fish > ~/.config/fish/completions/rw.fish

PowerShell:
  rw completion powershell | Out-String | Invoke-Expression
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
	return cmd
}
