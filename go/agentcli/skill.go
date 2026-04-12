package agentcli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CommandInfo is a machine-readable description of a single CLI command.
type CommandInfo struct {
	Name        string        `json:"name"                  yaml:"name"`
	Use         string        `json:"use"                   yaml:"use"`
	Short       string        `json:"short"                 yaml:"short"`
	Long        string        `json:"long,omitempty"        yaml:"long,omitempty"`
	Flags       []FlagInfo    `json:"flags,omitempty"       yaml:"flags,omitempty"`
	SubCommands []CommandInfo `json:"subCommands,omitempty" yaml:"subCommands,omitempty"`
}

// FlagInfo is a machine-readable description of a single CLI flag.
type FlagInfo struct {
	Name      string `json:"name"                yaml:"name"`
	Shorthand string `json:"shorthand,omitempty" yaml:"shorthand,omitempty"`
	Usage     string `json:"usage"               yaml:"usage"`
	Default   string `json:"default,omitempty"   yaml:"default,omitempty"`
}

// SkillOutput is the full structured payload returned by the skill command.
type SkillOutput struct {
	Name        string        `json:"name"                yaml:"name"`
	Description string        `json:"description"         yaml:"description"`
	Version     string        `json:"version,omitempty"   yaml:"version,omitempty"`
	Commands    []CommandInfo `json:"commands"            yaml:"commands"`
	SkillDoc    string        `json:"skillDoc,omitempty"  yaml:"skillDoc,omitempty"`
}

// newSkillCmd returns a "skill" command that outputs a machine-readable
// description of the CLI so an agent can learn how to use it.
func newSkillCmd(root *cobra.Command, opts *RootOptions, cfg Config) *cobra.Command {
	return &cobra.Command{
		Use:   "skill",
		Short: "Print the agent skill description for this CLI",
		Long: `Print a description of this CLI that an agent can use to understand
all available commands and flags.

When --json or --yaml is specified the output is fully machine-readable.
Without a format flag the human-readable SKILL.md (if present) is shown.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			payload := buildSkillOutput(root, cfg)
			return opts.WriteCommandOutput(cmd, payload, func(w io.Writer) error {
				return renderSkillText(w, payload)
			})
		},
	}
}

// buildSkillOutput assembles the SkillOutput from the root cobra.Command.
func buildSkillOutput(root *cobra.Command, cfg Config) SkillOutput {
	out := SkillOutput{
		Name:        root.Name(),
		Description: root.Short,
		Version:     cfg.Version,
		Commands:    commandTree(root),
	}
	if cfg.SkillSource != "" {
		if data, err := os.ReadFile(cfg.SkillSource); err == nil {
			out.SkillDoc = strings.TrimSpace(string(data))
		}
	}
	return out
}

// renderSkillText writes the human-readable skill output.
func renderSkillText(w io.Writer, payload SkillOutput) error {
	if payload.SkillDoc != "" {
		_, err := fmt.Fprintln(w, payload.SkillDoc)
		return err
	}
	fmt.Fprintf(w, "# %s\n\n", payload.Name)
	if payload.Description != "" {
		fmt.Fprintf(w, "%s\n\n", payload.Description)
	}
	if payload.Version != "" {
		fmt.Fprintf(w, "Version: %s\n\n", payload.Version)
	}
	fmt.Fprintf(w, "## Commands\n\n")
	for _, c := range payload.Commands {
		printCommandInfo(w, c, 0)
	}
	return nil
}

func printCommandInfo(w io.Writer, c CommandInfo, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(w, "%s### %s\n\n", indent, c.Use)
	if c.Short != "" {
		fmt.Fprintf(w, "%s%s\n\n", indent, c.Short)
	}
	if len(c.Flags) > 0 {
		fmt.Fprintf(w, "%sFlags:\n", indent)
		for _, f := range c.Flags {
			line := fmt.Sprintf("%s  --%s", indent, f.Name)
			if f.Shorthand != "" {
				line += fmt.Sprintf(", -%s", f.Shorthand)
			}
			line += fmt.Sprintf("  %s", f.Usage)
			if f.Default != "" && f.Default != "false" && f.Default != "0" {
				line += fmt.Sprintf(" (default: %s)", f.Default)
			}
			fmt.Fprintln(w, line)
		}
		fmt.Fprintln(w)
	}
	for _, sub := range c.SubCommands {
		printCommandInfo(w, sub, depth+1)
	}
}

// commandTree recursively converts a cobra.Command into CommandInfo structs,
// skipping hidden and completion-helper commands.
func commandTree(cmd *cobra.Command) []CommandInfo {
	var out []CommandInfo
	for _, sub := range cmd.Commands() {
		if sub.Hidden {
			continue
		}
		info := CommandInfo{
			Name:        sub.Name(),
			Use:         sub.Use,
			Short:       sub.Short,
			Long:        strings.TrimSpace(sub.Long),
			Flags:       collectFlags(sub),
			SubCommands: commandTree(sub),
		}
		out = append(out, info)
	}
	return out
}

func collectFlags(cmd *cobra.Command) []FlagInfo {
	var flags []FlagInfo
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		info := FlagInfo{
			Name:      f.Name,
			Shorthand: f.Shorthand,
			Usage:     f.Usage,
			Default:   f.DefValue,
		}
		flags = append(flags, info)
	})
	return flags
}
