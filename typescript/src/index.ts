/**
 * @dotnaos/agent-cli
 *
 * Framework for building agent-friendly CLIs.
 *
 * Every CLI built with this framework exposes three required commands:
 *   - completions  – shell tab-completion scripts for humans
 *   - skill        – machine-readable CLI description for agents
 *   - logs         – streaming log output with agent-friendly options
 *
 * Global flags --json, --yaml, and --yml are always available.
 *
 * @example
 * ```ts
 * import { createAgentCLI, fileLogProvider } from "@dotnaos/agent-cli";
 *
 * const program = createAgentCLI({
 *   name: "mycli",
 *   description: "My agent-friendly CLI",
 *   version: "1.0.0",
 *   logProvider: fileLogProvider("/var/log/mycli.log"),
 * });
 *
 * // Add your custom commands
 * program.command("greet [name]")
 *   .description("Print a greeting")
 *   .action((name = "world") => console.log(`Hello, ${name}!`));
 *
 * program.parse();
 * ```
 */

export {
  createAgentCLI,
  validateRequiredCommands,
  REQUIRED_COMMANDS,
  AgentCLIConfig,
} from "./cli";

export {
  OutputOptions,
  OutputFormat,
  currentFormat,
  isMachineOutput,
  validateOutputFlags,
  serialise,
  writeCommandOutput,
  writeStreamEvent,
} from "./output";

export {
  createCompletionsCommand,
  CompletionResult,
} from "./commands/completions";

export {
  createSkillCommand,
  buildSkillOutput,
  renderSkillText,
  SkillOutput,
  CommandInfo,
  FlagInfo,
  SkillConfig,
} from "./commands/skill";

export {
  createLogsCommand,
  fileLogProvider,
  parseDuration,
  LogEntry,
  LogFlags,
  LogProvider,
  LogsConfig,
} from "./commands/logs";
