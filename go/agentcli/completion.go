package agentcli

import (
	"bytes"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// completionResult is the structured payload returned by the completions command
// when --json or --yaml is active.
type completionResult struct {
	Shell  string `json:"shell"  yaml:"shell"`
	Script string `json:"script" yaml:"script"`
}

// newCompletionCmd returns a "completions" command that generates shell
// autocompletion scripts for the given root command.
func newCompletionCmd(root *cobra.Command, opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completions",
		Short: "Generate shell autocompletion scripts",
		Long: `Generate the autocompletion script for the specified shell so that you
get tab-completion when using this CLI interactively.

Supported shells: bash, zsh, fish, powershell

Run 'completions --help' for setup instructions specific to your shell.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newShellCompletionCmd(root, opts, "bash", func(w io.Writer) error {
			return root.GenBashCompletionV2(w, true)
		}),
		newShellCompletionCmd(root, opts, "zsh", func(w io.Writer) error {
			return root.GenZshCompletion(w)
		}),
		newShellCompletionCmd(root, opts, "fish", func(w io.Writer) error {
			return root.GenFishCompletion(w, true)
		}),
		newShellCompletionCmd(root, opts, "powershell", func(w io.Writer) error {
			return root.GenPowerShellCompletionWithDesc(w)
		}),
	)
	return cmd
}

func newShellCompletionCmd(root *cobra.Command, opts *RootOptions, shell string, generate func(io.Writer) error) *cobra.Command {
	return &cobra.Command{
		Use:   shell,
		Short: fmt.Sprintf("Generate the autocompletion script for %s", shell),
		Long:  shellCompletionLong(root, shell),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var buf bytes.Buffer
			if err := generate(&buf); err != nil {
				return err
			}
			result := completionResult{
				Shell:  shell,
				Script: buf.String(),
			}
			return opts.WriteCommandOutput(cmd, result, func(w io.Writer) error {
				_, err := io.Copy(w, &buf)
				return err
			})
		},
	}
}

func shellCompletionLong(root *cobra.Command, shell string) string {
	name := root.Name()
	switch shell {
	case "bash":
		return fmt.Sprintf(`Generate the autocompletion script for bash.

To load completions in your current shell session:

    source <(%s completions bash)

To load completions for every new session, execute once:

  Linux:
    %s completions bash > /etc/bash_completion.d/%s

  macOS:
    %s completions bash > $(brew --prefix)/etc/bash_completion.d/%s
`, name, name, name, name, name)
	case "zsh":
		return fmt.Sprintf(`Generate the autocompletion script for zsh.

To load completions in your current shell session:

    source <(%s completions zsh)

To load completions for every new session, execute once:

    %s completions zsh > "${fpath[1]}/_%s"
`, name, name, name)
	case "fish":
		return fmt.Sprintf(`Generate the autocompletion script for fish.

To load completions in your current shell session:

    %s completions fish | source

To load completions for every new session, execute once:

    %s completions fish > ~/.config/fish/completions/%s.fish
`, name, name, name)
	case "powershell":
		return fmt.Sprintf(`Generate the autocompletion script for PowerShell.

To load completions in your current shell session:

    %s completions powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your PowerShell profile.
`, name)
	default:
		return fmt.Sprintf("Generate the autocompletion script for %s.", shell)
	}
}
