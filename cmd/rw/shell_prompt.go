package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func shellInitCmd() *cobra.Command {
	var shell string

	cmd := &cobra.Command{
		Use:   "shell-init",
		Short: "Output shell script for prompt integration",
		Long: `Outputs a shell script that adds the current checkpoint ID to your prompt.

Add to your shell config:
  bash/zsh:   eval "$(rw shell-init --shell bash)"
  fish:       rw shell-init --shell fish | source
  powershell: Invoke-Expression (rw shell-init --shell powershell)

Shows nothing if not in a Nimbi repository.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch shell {
			case "bash", "zsh", "":
				fmt.Print(bashPromptScript)
			case "fish":
				fmt.Print(fishPromptScript)
			case "powershell", "pwsh":
				fmt.Print(powershellPromptScript)
			default:
				return fmt.Errorf("unsupported shell: %s (try bash, zsh, fish, or powershell)", shell)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&shell, "shell", "", "shell type (bash, zsh, fish, powershell)")
	return cmd
}

const bashPromptScript = `# Nimbi prompt integration
__rw_prompt() {
  local id
  id=$(rw _shell_hook 2>/dev/null)
  if [ -n "$id" ]; then
    echo " [$id]"
  fi
}
if [ -n "$ZSH_VERSION" ]; then
  setopt PROMPT_SUBST
  PROMPT='%~$(__rw_prompt) %# '
else
  PS1='\w$(__rw_prompt) \$ '
fi
`

const fishPromptScript = `# Nimbi prompt integration
function fish_right_prompt
  set -l id (rw _shell_hook 2>/dev/null)
  if test -n "$id"
    echo "[$id]"
  end
end
`

const powershellPromptScript = `# Nimbi prompt integration
function prompt {
  $id = & rw _shell_hook 2>$null
  $loc = Get-Location
  if ($id) {
    "$loc [$id]> "
  } else {
    "$loc> "
  }
}
`
