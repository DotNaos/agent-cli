import * as fs from "fs";
import * as readline from "readline";
import { Command } from "commander";
import { OutputOptions, isMachineOutput, writeStreamEvent } from "../output";

/** Log entry emitted per line when using machine-readable output. */
export interface LogEntry {
  timestamp: string;
  level?: string;
  line: string;
}

/** Flags accepted by the logs command. */
export interface LogFlags {
  /** Follow log output in real-time (like tail -f). */
  follow: boolean;
  /** Number of recent lines to show before following. */
  lines: number;
  /** Filter by log level keyword (e.g. "error", "warn"). */
  level?: string;
  /** Show entries from the last duration string (e.g. "1h", "30m"). */
  since?: string;
  /** Explicit path to the log file. */
  path?: string;
}

/**
 * A function that opens the log stream according to the given flags.
 * It should return a readable stream of log lines, or throw on failure.
 */
export type LogProvider = (flags: LogFlags) => NodeJS.ReadableStream;

/** Configuration for the logs command. */
export interface LogsConfig {
  logProvider?: LogProvider;
}

/**
 * Creates and returns a "logs" command that streams log output in an
 * agent-friendly way.
 */
export function createLogsCommand(
  opts: OutputOptions,
  cfg: LogsConfig = {}
): Command {
  const cmd = new Command("logs")
    .description(
      "Stream log output\n\n" +
        "Designed for agents: use --json or --yaml for structured per-line entries,\n" +
        "and use the filter flags to reduce noise.\n\n" +
        "Examples:\n" +
        "  # Show last 100 lines as JSON and follow\n" +
        "  <cli> logs --follow --lines 100 --json\n\n" +
        "  # Show only errors from the last hour\n" +
        "  <cli> logs --level error --since 1h\n\n" +
        "  # Stream all entries as YAML documents\n" +
        "  <cli> logs --follow --yaml"
    )
    .option("-f, --follow", "Follow log output in real-time (like tail -f)", false)
    .option("-n, --lines <number>", "Number of recent lines to show", "50")
    .option("--level <level>", "Filter by log level: debug, info, warn, error")
    .option(
      "--since <duration>",
      "Show entries from the last duration (e.g. 1h, 30m, 5s)"
    )
    .option("--path <path>", "Path to the log file (overrides the default)")
    .action((options: Record<string, string | boolean>) => {
      const flags: LogFlags = {
        follow: Boolean(options["follow"]),
        lines: parseInt(String(options["lines"] ?? "50"), 10),
        level: options["level"] ? String(options["level"]) : undefined,
        since: options["since"] ? String(options["since"]) : undefined,
        path: options["path"] ? String(options["path"]) : undefined,
      };

      if (!cfg.logProvider) {
        process.stderr.write(
          "No log provider registered. " +
            "Pass a logProvider in createAgentCLI({ logProvider: ... }).\n"
        );
        process.exit(1);
        return;
      }

      const sinceMs = flags.since ? parseDuration(flags.since) : null;
      const since = sinceMs !== null ? new Date(Date.now() - sinceMs) : null;

      const stream = cfg.logProvider(flags);
      const rl = readline.createInterface({ input: stream, crlfDelay: Infinity });

      rl.on("line", (line: string) => {
        if (flags.level && !lineMatchesLevel(line, flags.level)) return;
        if (since) {
          const ts = parseLineTimestamp(line);
          if (ts && ts < since) return;
        }

        if (isMachineOutput(opts)) {
          const entry: LogEntry = {
            timestamp: new Date().toISOString(),
            level: extractLevel(line) || undefined,
            line,
          };
          writeStreamEvent(opts, entry);
        } else {
          process.stdout.write(line + "\n");
        }
      });

      rl.on("error", (err: Error) => {
        process.stderr.write(`logs: ${err.message}\n`);
        process.exit(1);
      });
    });

  return cmd;
}

/** Returns true when line contains the given level keyword. */
function lineMatchesLevel(line: string, level: string): boolean {
  return line.toLowerCase().includes(level.toLowerCase());
}

/** Tries to detect the log level from a line. */
function extractLevel(line: string): string {
  const lower = line.toLowerCase();
  for (const lvl of ["error", "warn", "info", "debug"]) {
    if (lower.includes(lvl)) return lvl;
  }
  return "";
}

/** Tries to parse a timestamp at the start of a log line. */
function parseLineTimestamp(line: string): Date | null {
  const candidate = line.replace(/^[\s\[]+/, "").slice(0, 35);
  const formats = [
    /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/,
    /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/,
    /^\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}/,
  ];
  for (const re of formats) {
    const m = candidate.match(re);
    if (m) {
      const d = new Date(m[0].replace(" ", "T").replace("/", "-"));
      if (!isNaN(d.getTime())) return d;
    }
  }
  return null;
}

/**
 * Parses a human duration string into milliseconds.
 * Supports: s (seconds), m (minutes), h (hours), d (days).
 */
export function parseDuration(s: string): number {
  const match = s.match(/^(\d+)(s|m|h|d)?$/);
  if (!match) {
    throw new Error(`Cannot parse duration: "${s}"`);
  }
  const n = parseInt(match[1], 10);
  switch (match[2]) {
    case "s":
      return n * 1000;
    case "m":
    case undefined:
      return n * 60 * 1000;
    case "h":
      return n * 3600 * 1000;
    case "d":
      return n * 86400 * 1000;
    default:
      throw new Error(`Unknown duration unit: "${match[2]}"`);
  }
}

/**
 * Returns a LogProvider that reads from a file on disk.
 * When LogFlags.path is set it is used instead of defaultPath.
 */
export function fileLogProvider(defaultPath: string): LogProvider {
  return (flags: LogFlags): NodeJS.ReadableStream => {
    const p = flags.path ?? defaultPath;
    if (!p) {
      throw new Error("No log path specified – use --path");
    }
    return fs.createReadStream(p, { encoding: "utf8" });
  };
}
