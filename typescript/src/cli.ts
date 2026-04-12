import { Command } from "commander";
import {
  OutputOptions,
  validateOutputFlags,
  isMachineOutput,
} from "./output";
import { createCompletionsCommand } from "./commands/completions";
import { createSkillCommand, SkillConfig } from "./commands/skill";
import { createLogsCommand, LogProvider } from "./commands/logs";

/** Names of the commands that every agent CLI must expose. */
export const REQUIRED_COMMANDS = ["completions", "skill", "logs"] as const;

/** Configuration for createAgentCLI. */
export interface AgentCLIConfig extends SkillConfig {
  /** The CLI name (e.g. "mycli"). */
  name: string;
  /** Short description shown in help output. */
  description: string;
  /** Version string (e.g. "1.2.3"). */
  version?: string;
  /**
   * Opens the log stream for the `logs` command.
   * If omitted the logs command will print an error asking the caller to
   * register a provider.
   */
  logProvider?: LogProvider;
}

/**
 * Creates a fully-configured commander root command for an agent CLI.
 *
 * The returned Command already has the required sub-commands attached:
 *   - completions  – shell tab-completion scripts
 *   - skill        – machine-readable CLI description for agents
 *   - logs         – streaming log output with agent-friendly options
 *
 * Global flags --json, --yaml, and --yml are added as root-level options.
 *
 * @example
 * ```ts
 * const program = createAgentCLI({
 *   name: "mycli",
 *   description: "My agent-friendly CLI",
 *   version: "1.0.0",
 *   logProvider: fileLogProvider("/var/log/mycli.log"),
 * });
 * program.addCommand(myCustomCommand);
 * program.parse();
 * ```
 */
export function createAgentCLI(cfg: AgentCLIConfig): Command {
  // Shared options object updated by the root-level flags.
  const opts: OutputOptions = { json: false, yaml: false, yml: false };

  const program = new Command()
    .name(cfg.name)
    .description(cfg.description)
    .version(cfg.version ?? "0.0.0");

  // Register the global output flags on the root command.
  program.option("--json", "Output machine-readable JSON", false);
  program.option("--yaml", "Output machine-readable YAML", false);
  program.option("--yml", "Alias for --yaml", false);

  // Sync the shared opts object after parsing.
  program.hook("preAction", (_thisCommand, actionCommand) => {
    const rootOpts = program.opts<{ json: boolean; yaml: boolean; yml: boolean }>();
    opts.json = rootOpts.json ?? false;
    opts.yaml = rootOpts.yaml ?? false;
    opts.yml = rootOpts.yml ?? false;
    validateOutputFlags(opts);
  });

  // Attach required commands.
  program.addCommand(createCompletionsCommand(program, opts));
  program.addCommand(createSkillCommand(program, opts, cfg));
  program.addCommand(createLogsCommand(opts, { logProvider: cfg.logProvider }));

  return program;
}

/**
 * Validates that all required commands are present in the given root command.
 * Returns an array of missing command names (empty if all present).
 */
export function validateRequiredCommands(root: Command): string[] {
  const present = new Set(root.commands.map((c) => c.name()));
  return REQUIRED_COMMANDS.filter((name) => !present.has(name));
}
