package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

**Bash:**

  $ source <(mycli completion bash)

**Zsh:**

  $ source <(mycli completion zsh)

**Fish:**

  $ mycli completion fish | source

**PowerShell:**

  PS> mycli completion powershell | Out-String | Invoke-Expression
`,
	Args: cobra.ExactValidArgs(1),
	ValidArgs: []string{
		"bash",
		"zsh",
		"fish",
		"powershell",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return generateZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell type: %s", args[0])
		}
	},
}

func generateZshCompletion(w io.Writer) error {
	// Generate the standard zsh completion script
	if err := rootCmd.GenZshCompletionNoDesc(w); err != nil {
		return err
	}

	// Append custom code for direct sourcing
	customScript := `
# Enable completion within quotes
compdef _mycli mycli
`
	_, err := io.WriteString(w, customScript)
	return err
}
