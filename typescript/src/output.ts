import { stringify as yamlStringify } from "yaml";

/** Supported output formats. */
export type OutputFormat = "text" | "json" | "yaml";

/** Global output options shared across all commands. */
export interface OutputOptions {
  json: boolean;
  yaml: boolean;
  yml: boolean;
}

/** Returns the active output format based on the provided options. */
export function currentFormat(opts: OutputOptions): OutputFormat {
  if (opts.json) return "json";
  if (opts.yaml || opts.yml) return "yaml";
  return "text";
}

/** Returns true when a machine-readable (non-text) format is active. */
export function isMachineOutput(opts: OutputOptions): boolean {
  return currentFormat(opts) !== "text";
}

/**
 * Validates that at most one output flag is active.
 * Throws if more than one of --json / --yaml / --yml is set.
 */
export function validateOutputFlags(opts: OutputOptions): void {
  const selected = [opts.json, opts.yaml, opts.yml].filter(Boolean).length;
  if (selected > 1) {
    throw new Error("choose exactly one of --json, --yaml, or --yml");
  }
}

/**
 * Serialises `payload` to the format indicated by `opts`.
 * Throws when the format is "text" and structured output was requested.
 */
export function serialise(payload: unknown, format: OutputFormat): string {
  switch (format) {
    case "json":
      return JSON.stringify(payload, null, 2);
    case "yaml":
      return yamlStringify(payload, { lineWidth: 0 }).trimEnd();
    default:
      throw new Error(
        `serialise() called with format "text" – use renderText() instead`
      );
  }
}

/**
 * Writes `payload` as structured output to `out`, or calls `renderText` when
 * plain text mode is active.
 */
export function writeCommandOutput(
  opts: OutputOptions,
  payload: unknown,
  renderText: () => string,
  out: NodeJS.WritableStream = process.stdout
): void {
  const format = currentFormat(opts);
  if (format !== "text") {
    out.write(serialise(payload, format) + "\n");
  } else {
    out.write(renderText() + "\n");
  }
}

/**
 * Writes a single structured event to `out` for streaming commands (e.g. logs).
 * In JSON mode each event is a single JSON line.
 * In YAML mode each event is preceded by a `---` document separator.
 */
export function writeStreamEvent(
  opts: OutputOptions,
  payload: unknown,
  out: NodeJS.WritableStream = process.stdout
): void {
  const format = currentFormat(opts);
  switch (format) {
    case "json":
      out.write(JSON.stringify(payload) + "\n");
      break;
    case "yaml":
      out.write("---\n" + yamlStringify(payload, { lineWidth: 0 }).trimEnd() + "\n");
      break;
    default:
      throw new Error("stream events require machine-readable output (--json / --yaml)");
  }
}
