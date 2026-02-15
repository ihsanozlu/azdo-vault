package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate completion scripts for azdo-vault.

Examples:
  azdo-vault completion bash
  azdo-vault completion zsh
  azdo-vault completion fish
  azdo-vault completion powershell
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)

		case "zsh":
			// Zsh likes the "compdef" header for proper integration
			if _, err := fmt.Fprintln(os.Stdout, "#compdef azdo-vault"); err != nil {
				return err
			}
			return rootCmd.GenZshCompletion(os.Stdout)

		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)

		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)

		default:
			return fmt.Errorf("unsupported shell: %s (use: bash|zsh|fish|powershell)", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
