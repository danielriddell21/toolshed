package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func completionCmd(name string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: fmt.Sprintf(`Generate shell completion scripts for %[1]s.

Bash:
  %[1]s completion bash > /etc/bash_completion.d/%[1]s
  # or for the current user:
  %[1]s completion bash > ~/.local/share/bash-completion/completions/%[1]s

Zsh:
  %[1]s completion zsh > "${fpath[1]}/_%[1]s"
  # then restart your shell or run: autoload -U compinit && compinit

Fish:
  %[1]s completion fish > ~/.config/fish/completions/%[1]s.fish

PowerShell:
  %[1]s completion powershell | Out-String | Invoke-Expression
  # to persist, add that line to your $PROFILE`, name),
		ValidArgs:    []string{"bash", "zsh", "fish", "powershell"},
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return root.GenZshCompletion(os.Stdout)
			case "fish":
				return root.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return cmd.Help()
			}
		},
	}
	return cmd
}
