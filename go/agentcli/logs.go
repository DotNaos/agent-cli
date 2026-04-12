package agentcli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// LogEntry is the structured payload emitted per line by the logs command.
type LogEntry struct {
	Timestamp string `json:"timestamp"        yaml:"timestamp"`
	Level     string `json:"level,omitempty"  yaml:"level,omitempty"`
	Line      string `json:"line"             yaml:"line"`
}

// LogProviderFunc opens a log stream according to the given LogFlags.
// It returns a ReadCloser that the logs command will read from line-by-line.
// The caller is responsible for closing the reader when done.
type LogProviderFunc func(flags LogFlags) (io.ReadCloser, error)

// LogFlags holds the parsed flag values for the logs command.
type LogFlags struct {
	// Follow streams new log entries in real-time (like tail -f).
	Follow bool
	// Lines is the number of historical lines to show before following.
	Lines int
	// Level filters entries to the given log level (debug/info/warn/error).
	// An empty string means no filtering.
	Level string
	// Since filters entries to those written within the given duration
	// (e.g. "1h", "30m", "5s"). An empty string means no time filter.
	Since string
	// Path is an optional explicit path to the log file.
	Path string
}

// newLogsCmd returns a "logs" command that streams log output in an
// agent-friendly way.
func newLogsCmd(opts *RootOptions, cfg Config) *cobra.Command {
	var lf LogFlags

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Stream log output",
		Long: `Stream log output in real-time or show recent entries.

This command is designed to be agent-friendly: use --json or --yaml to receive
structured log entries that are easy to parse programmatically, and use the
filter flags to reduce noise.

Examples:
  # Show the last 100 lines and follow new entries as JSON
  <cli> logs --follow --lines 100 --json

  # Show only error-level entries from the last hour
  <cli> logs --level error --since 1h

  # Stream all entries as YAML documents
  <cli> logs --follow --yaml`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cfg.LogProvider == nil {
				return fmt.Errorf(
					"no log provider registered – call agentcli.New with a Config.LogProvider",
				)
			}

			rc, err := cfg.LogProvider(lf)
			if err != nil {
				return fmt.Errorf("opening log stream: %w", err)
			}
			defer rc.Close()

			return streamLogs(cmd, opts, rc, lf)
		},
	}

	cmd.Flags().BoolVarP(&lf.Follow, "follow", "f", false, "Follow log output in real-time (like tail -f)")
	cmd.Flags().IntVarP(&lf.Lines, "lines", "n", 50, "Number of recent lines to show")
	cmd.Flags().StringVar(&lf.Level, "level", "", "Filter by log level: debug, info, warn, error")
	cmd.Flags().StringVar(&lf.Since, "since", "", "Show entries from the last duration (e.g. 1h, 30m, 5s)")
	cmd.Flags().StringVar(&lf.Path, "path", "", "Path to the log file (overrides the default)")

	return cmd
}

// streamLogs reads from rc line-by-line, applies filters, and emits entries.
func streamLogs(cmd *cobra.Command, opts *RootOptions, rc io.Reader, lf LogFlags) error {
	w := cmd.OutOrStdout()

	var sinceTime *time.Time
	if lf.Since != "" {
		d, err := parseDuration(lf.Since)
		if err != nil {
			return fmt.Errorf("invalid --since value %q: %w", lf.Since, err)
		}
		t := time.Now().Add(-d)
		sinceTime = &t
	}

	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		line := scanner.Text()

		if lf.Level != "" && !lineMatchesLevel(line, lf.Level) {
			continue
		}

		if opts.IsMachineOutput() {
			entry := LogEntry{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Level:     extractLevel(line),
				Line:      line,
			}
			if sinceTime != nil {
				ts := parseLineTimestamp(line)
				if ts != nil && ts.Before(*sinceTime) {
					continue
				}
			}
			if err := opts.WriteStreamEvent(w, entry); err != nil {
				return err
			}
		} else {
			if sinceTime != nil {
				ts := parseLineTimestamp(line)
				if ts != nil && ts.Before(*sinceTime) {
					continue
				}
			}
			fmt.Fprintln(w, line)
		}
	}
	return scanner.Err()
}

// lineMatchesLevel returns true when line appears to contain the given level.
func lineMatchesLevel(line, level string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, strings.ToLower(level))
}

// extractLevel tries to detect the log level from a line.
func extractLevel(line string) string {
	lower := strings.ToLower(line)
	for _, lvl := range []string{"error", "warn", "info", "debug"} {
		if strings.Contains(lower, lvl) {
			return lvl
		}
	}
	return ""
}

// parseLineTimestamp tries to extract a timestamp from the beginning of a log
// line. It understands RFC3339 and a handful of common formats.
func parseLineTimestamp(line string) *time.Time {
	if len(line) == 0 {
		return nil
	}
	// Try up to the first 35 characters as a timestamp prefix.
	candidate := line
	if len(candidate) > 35 {
		candidate = candidate[:35]
	}
	// Strip leading non-digit characters (e.g. "[", spaces).
	candidate = strings.TrimLeft(candidate, "[ \t")

	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
	} {
		if len(candidate) < len(layout) {
			continue
		}
		if t, err := time.Parse(layout, candidate[:len(layout)]); err == nil {
			return &t
		}
	}
	return nil
}

// parseDuration extends time.ParseDuration to support bare hour/minute/day
// values such as "1h", "30m", "2d".
func parseDuration(s string) (time.Duration, error) {
	// Handle days manually before delegating to time.ParseDuration.
	if strings.HasSuffix(s, "d") {
		trimmed := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(trimmed, "%d", &days); err != nil {
			return 0, fmt.Errorf("invalid duration: %s", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// Try reading a plain number and defaulting to minutes.
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// If there's no unit suffix at all (e.g. "30") treat as minutes.
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err == nil && fmt.Sprintf("%d", n) == s {
		return time.Duration(n) * time.Minute, nil
	}

	return 0, fmt.Errorf("cannot parse duration: %q", s)
}

// DefaultFileLogProvider returns a LogProviderFunc that opens the file at
// path for reading. When LogFlags.Path is set it is used instead of path.
// This is a convenience helper for CLIs whose logs live in a plain file.
func DefaultFileLogProvider(defaultPath string) LogProviderFunc {
	return func(flags LogFlags) (io.ReadCloser, error) {
		p := defaultPath
		if flags.Path != "" {
			p = flags.Path
		}
		if p == "" {
			return nil, fmt.Errorf("no log path specified – use --path")
		}
		f, err := os.Open(p) // #nosec G304 – path is explicitly supplied by user
		if err != nil {
			return nil, err
		}
		if flags.Lines > 0 && !flags.Follow {
			lines, err := tailFile(f, flags.Lines)
			if err != nil {
				_ = f.Close()
				return nil, err
			}
			_ = f.Close()
			return io.NopCloser(strings.NewReader(strings.Join(lines, "\n"))), nil
		}
		return f, nil
	}
}

// tailFile returns the last n lines of f.
func tailFile(f *os.File, n int) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	return lines, scanner.Err()
}
