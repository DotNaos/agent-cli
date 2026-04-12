package agentcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// OutputFormat represents the output format for commands.
type OutputFormat string

const (
	OutputFormatText OutputFormat = "text"
	OutputFormatJSON OutputFormat = "json"
	OutputFormatYAML OutputFormat = "yaml"
)

// CommandError is the structured error payload used for machine-readable output.
type CommandError struct {
	Code    string `json:"code,omitempty" yaml:"code,omitempty"`
	Error   string `json:"error"          yaml:"error"`
	Command string `json:"command,omitempty" yaml:"command,omitempty"`
}

// CurrentFormat returns the active output format based on the flag values stored
// in the provided root options.
func (o *RootOptions) CurrentFormat() OutputFormat {
	switch {
	case o.OutputJSON:
		return OutputFormatJSON
	case o.OutputYAML || o.OutputYML:
		return OutputFormatYAML
	default:
		return OutputFormatText
	}
}

// IsMachineOutput returns true when a structured (non-text) format is requested.
func (o *RootOptions) IsMachineOutput() bool {
	return o.CurrentFormat() != OutputFormatText
}

// ValidateOutputFlags returns an error when more than one output flag is set.
func (o *RootOptions) ValidateOutputFlags() error {
	selected := 0
	if o.OutputJSON {
		selected++
	}
	if o.OutputYAML {
		selected++
	}
	if o.OutputYML {
		selected++
	}
	if selected <= 1 {
		return nil
	}
	return fmt.Errorf("choose exactly one of --json, --yaml, or --yml")
}

// WriteCommandOutput writes payload as structured output when a machine format
// is active, or falls back to renderText otherwise.
func (o *RootOptions) WriteCommandOutput(cmd *cobra.Command, payload any, renderText func(io.Writer) error) error {
	if o.IsMachineOutput() {
		return o.writeStructuredPayload(cmd.OutOrStdout(), payload)
	}
	return renderText(cmd.OutOrStdout())
}

// WriteStreamEvent writes a single structured event for streaming commands
// (e.g. the logs command). It appends a newline after each event.
func (o *RootOptions) WriteStreamEvent(w io.Writer, payload any) error {
	format := o.CurrentFormat()
	switch format {
	case OutputFormatJSON:
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "%s\n", data)
		return err
	case OutputFormatYAML:
		data, err := marshalYAML(payload)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "---\n%s\n", data)
		return err
	default:
		return fmt.Errorf("stream events require machine-readable output (--json / --yaml)")
	}
}

// WriteCommandError writes err to stdout/stderr in the appropriate format.
func (o *RootOptions) WriteCommandError(w io.Writer, cmd *cobra.Command, err error) {
	if err == nil {
		return
	}
	if o.IsMachineOutput() {
		payload := CommandError{
			Error:   err.Error(),
			Command: cmd.CommandPath(),
		}
		_ = o.writeStructuredPayload(w, payload)
		return
	}
	fmt.Fprintln(w, err.Error())
}

func (o *RootOptions) writeStructuredPayload(w io.Writer, payload any) error {
	data, err := o.marshalPayload(payload)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		_, err = w.Write([]byte("\n"))
	}
	return err
}

func (o *RootOptions) marshalPayload(payload any) ([]byte, error) {
	switch o.CurrentFormat() {
	case OutputFormatJSON:
		return json.MarshalIndent(payload, "", "  ")
	case OutputFormatYAML:
		return marshalYAML(payload)
	default:
		return nil, fmt.Errorf("unsupported structured output format: %s", o.CurrentFormat())
	}
}

func marshalYAML(payload any) ([]byte, error) {
	data, err := yaml.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return bytes.TrimRight(data, "\n"), nil
}
