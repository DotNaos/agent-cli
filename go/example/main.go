// Package main is a minimal example CLI that uses the agentcli framework.
// It demonstrates how to create an agent-friendly CLI with all required
// commands (completions, skill, logs) plus custom application commands.
package main

import (
	"io"
	"os"
	"strings"

	"github.com/DotNaos/agent-cli/go/agentcli"
	"github.com/spf13/cobra"
)

func main() {
	root := agentcli.New(agentcli.Config{
		Use:     "example",
		Short:   "An example agent-friendly CLI built with agentcli",
		Long:    "example is a demonstration CLI that showcases the agentcli framework.",
		Version: "0.1.0",
		// SkillSource: "SKILL.md",  // Uncomment to load a SKILL.md file.
		LogProvider: agentcli.DefaultFileLogProvider("example.log"),
	})

	// Add application-specific commands here.
	root.AddCommand(greetCmd())

	agentcli.Execute(root)
}

func greetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "greet [name]",
		Short: "Print a greeting",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "world"
			if len(args) > 0 {
				name = args[0]
			}
			type greetResult struct {
				Message string `json:"message" yaml:"message"`
			}
			payload := greetResult{Message: "Hello, " + name + "!"}
			return agentcli.Opts(cmd).WriteCommandOutput(
				cmd,
				payload,
				func(w io.Writer) error {
					_, err := io.WriteString(w, payload.Message+"\n")
					return err
				},
			)
		},
	}
}

// Ensure example.log exists for the logs command.
func init() {
	if _, err := os.Stat("example.log"); os.IsNotExist(err) {
		_ = os.WriteFile("example.log", []byte(strings.Join([]string{
			"2024-01-01T00:00:00Z [info] example CLI started",
			"2024-01-01T00:00:01Z [debug] loading configuration",
			"2024-01-01T00:00:02Z [info] ready",
		}, "\n")+"\n"), 0o600)
	}
}
