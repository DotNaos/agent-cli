package agentcli_test

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/DotNaos/agent-cli/go/agentcli"
	"gopkg.in/yaml.v3"
)

// newTestCLI builds a root command wired up with a simple in-memory log source.
func newTestCLI() *agentcli.Config {
	return &agentcli.Config{
		Use:     "testcli",
		Short:   "A test CLI",
		Version: "0.0.1",
		LogProvider: func(_ agentcli.LogFlags) (io.ReadCloser, error) {
			content := "2024-01-01T00:00:00Z [info] started\n2024-01-01T00:00:01Z [error] something failed\n2024-01-01T00:00:02Z [debug] done\n"
			return io.NopCloser(strings.NewReader(content)), nil
		},
	}
}

// executeCmd runs a root command with the given args and returns stdout output.
func executeCmd(cfg agentcli.Config, args []string) (string, error) {
	root := agentcli.New(cfg)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// TestRequiredCommandsPresent verifies that New() always registers the three
// required commands.
func TestRequiredCommandsPresent(t *testing.T) {
	t.Parallel()
	root := agentcli.New(*newTestCLI())
	if err := agentcli.ValidateRequiredCommands(root); err != nil {
		t.Fatalf("required commands missing: %v", err)
	}
}

// TestRequiredCommandsList verifies the canonical list of required commands.
func TestRequiredCommandsList(t *testing.T) {
	t.Parallel()
	required := agentcli.RequiredCommands()
	want := map[string]bool{"completions": true, "skill": true, "logs": true}
	for _, name := range required {
		if !want[name] {
			t.Errorf("unexpected required command: %s", name)
		}
		delete(want, name)
	}
	for name := range want {
		t.Errorf("expected required command not listed: %s", name)
	}
}

// TestValidateRequiredCommandsMissing checks that ValidateRequiredCommands
// reports missing commands correctly.
func TestValidateRequiredCommandsMissing(t *testing.T) {
	t.Parallel()
	root := agentcli.New(*newTestCLI())
	// Remove the "skill" command by finding and removing it.
	for _, cmd := range root.Commands() {
		if cmd.Name() == "skill" {
			root.RemoveCommand(cmd)
			break
		}
	}
	err := agentcli.ValidateRequiredCommands(root)
	if err == nil {
		t.Fatal("expected error for missing 'skill' command, got nil")
	}
	if !strings.Contains(err.Error(), "skill") {
		t.Errorf("error message should mention 'skill', got: %v", err)
	}
}

// TestSkillOutputJSON verifies that `skill --json` produces valid JSON.
func TestSkillOutputJSON(t *testing.T) {
	t.Parallel()
	out, err := executeCmd(*newTestCLI(), []string{"skill", "--json"})
	if err != nil {
		t.Fatalf("skill --json failed: %v\noutput: %s", err, out)
	}
	var result agentcli.SkillOutput
	if jsonErr := json.Unmarshal([]byte(out), &result); jsonErr != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", jsonErr, out)
	}
	if result.Name != "testcli" {
		t.Errorf("expected name=testcli, got %q", result.Name)
	}
	if result.Version != "0.0.1" {
		t.Errorf("expected version=0.0.1, got %q", result.Version)
	}
	if len(result.Commands) == 0 {
		t.Error("expected at least one command in skill output")
	}
}

// TestSkillOutputYAML verifies that `skill --yaml` produces valid YAML.
func TestSkillOutputYAML(t *testing.T) {
	t.Parallel()
	out, err := executeCmd(*newTestCLI(), []string{"skill", "--yaml"})
	if err != nil {
		t.Fatalf("skill --yaml failed: %v\noutput: %s", err, out)
	}
	var result agentcli.SkillOutput
	if yamlErr := yaml.Unmarshal([]byte(out), &result); yamlErr != nil {
		t.Fatalf("invalid YAML output: %v\noutput: %s", yamlErr, out)
	}
	if result.Name != "testcli" {
		t.Errorf("expected name=testcli, got %q", result.Name)
	}
}

// TestSkillOutputYML verifies that --yml is an alias for --yaml.
func TestSkillOutputYML(t *testing.T) {
	t.Parallel()
	outYAML, err := executeCmd(*newTestCLI(), []string{"skill", "--yaml"})
	if err != nil {
		t.Fatalf("skill --yaml failed: %v", err)
	}
	outYML, err := executeCmd(*newTestCLI(), []string{"skill", "--yml"})
	if err != nil {
		t.Fatalf("skill --yml failed: %v", err)
	}
	if outYAML != outYML {
		t.Error("--yaml and --yml should produce identical output")
	}
}

// TestMutuallyExclusiveOutputFlags verifies that --json and --yaml cannot be
// used together.
func TestMutuallyExclusiveOutputFlags(t *testing.T) {
	t.Parallel()
	_, err := executeCmd(*newTestCLI(), []string{"skill", "--json", "--yaml"})
	if err == nil {
		t.Fatal("expected error when both --json and --yaml are specified")
	}
}

// TestLogsOutputJSON verifies that `logs --json` emits newline-delimited JSON.
func TestLogsOutputJSON(t *testing.T) {
	t.Parallel()
	out, err := executeCmd(*newTestCLI(), []string{"logs", "--json"})
	if err != nil {
		t.Fatalf("logs --json failed: %v\noutput: %s", err, out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 {
		t.Fatal("expected at least one log entry")
	}
	for i, line := range lines {
		var entry agentcli.LogEntry
		if jsonErr := json.Unmarshal([]byte(line), &entry); jsonErr != nil {
			t.Errorf("line %d is not valid JSON: %v\nline: %s", i, jsonErr, line)
		}
		if entry.Line == "" {
			t.Errorf("line %d has empty 'line' field", i)
		}
	}
}

// TestLogsLevelFilter verifies that --level filters log entries.
func TestLogsLevelFilter(t *testing.T) {
	t.Parallel()
	out, err := executeCmd(*newTestCLI(), []string{"logs", "--level", "error"})
	if err != nil {
		t.Fatalf("logs --level error failed: %v\noutput: %s", err, out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if !strings.Contains(strings.ToLower(line), "error") {
			t.Errorf("filtered line does not contain 'error': %q", line)
		}
	}
}

// TestCompletionsBash verifies that `completions bash` emits a non-empty script.
func TestCompletionsBash(t *testing.T) {
	t.Parallel()
	out, err := executeCmd(*newTestCLI(), []string{"completions", "bash"})
	if err != nil {
		t.Fatalf("completions bash failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "bash") {
		t.Error("bash completion script should mention 'bash'")
	}
	if len(out) < 100 {
		t.Errorf("bash completion script seems too short (%d bytes)", len(out))
	}
}

// TestCompletionsZsh verifies that `completions zsh` emits a non-empty script.
func TestCompletionsZsh(t *testing.T) {
	t.Parallel()
	out, err := executeCmd(*newTestCLI(), []string{"completions", "zsh"})
	if err != nil {
		t.Fatalf("completions zsh failed: %v\noutput: %s", err, out)
	}
	if len(out) < 100 {
		t.Errorf("zsh completion script seems too short (%d bytes)", len(out))
	}
}

// TestCompletionsJSON verifies structured output for completions.
func TestCompletionsJSON(t *testing.T) {
	t.Parallel()
	out, err := executeCmd(*newTestCLI(), []string{"completions", "bash", "--json"})
	if err != nil {
		t.Fatalf("completions bash --json failed: %v\noutput: %s", err, out)
	}
	var result struct {
		Shell  string `json:"shell"`
		Script string `json:"script"`
	}
	if jsonErr := json.Unmarshal([]byte(out), &result); jsonErr != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", jsonErr, out)
	}
	if result.Shell != "bash" {
		t.Errorf("expected shell=bash, got %q", result.Shell)
	}
	if len(result.Script) < 100 {
		t.Errorf("script field seems too short (%d bytes)", len(result.Script))
	}
}

// TestOptsReturnsRootOptions verifies that Opts() can retrieve the RootOptions
// from a command in the tree.
func TestOptsReturnsRootOptions(t *testing.T) {
	t.Parallel()
	root := agentcli.New(*newTestCLI())
	// Opts of root itself.
	opts := agentcli.Opts(root)
	if opts == nil {
		t.Fatal("Opts() returned nil for root command")
	}
	// Opts of a sub-command.
	for _, sub := range root.Commands() {
		subOpts := agentcli.Opts(sub)
		if subOpts == nil {
			t.Fatalf("Opts() returned nil for sub-command %q", sub.Name())
		}
	}
}
