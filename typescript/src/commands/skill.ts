import { Command, Option } from "commander";
import * as fs from "fs";
import {
  OutputOptions,
  currentFormat,
  isMachineOutput,
  writeCommandOutput,
} from "../output";

/** Machine-readable description of a single flag. */
export interface FlagInfo {
  name: string;
  shorthand?: string;
  usage: string;
  default?: string;
}

/** Machine-readable description of a single command. */
export interface CommandInfo {
  name: string;
  description: string;
  arguments?: string;
  flags?: FlagInfo[];
  subCommands?: CommandInfo[];
}

/** Full structured output of the skill command. */
export interface SkillOutput {
  name: string;
  description: string;
  version?: string;
  commands: CommandInfo[];
  skillDoc?: string;
}

/** Configuration passed to createSkillCommand. */
export interface SkillConfig {
  /** Path to a SKILL.md file loaded verbatim for human-readable output. */
  skillSource?: string;
}

/**
 * Creates and returns a "skill" command that outputs a machine-readable
 * description of the CLI so an agent can learn how to use it.
 */
export function createSkillCommand(
  root: Command,
  opts: OutputOptions,
  cfg: SkillConfig = {}
): Command {
  return new Command("skill")
    .description(
      "Print the agent skill description for this CLI\n\n" +
        "Use --json or --yaml for machine-readable output.\n" +
        "Without a format flag the SKILL.md file (if present) is shown."
    )
    .action(() => {
      const payload = buildSkillOutput(root, cfg);
      writeCommandOutput(opts, payload, () => renderSkillText(payload, cfg));
    });
}

/** Assembles the SkillOutput from the root commander Command. */
export function buildSkillOutput(root: Command, cfg: SkillConfig): SkillOutput {
  const out: SkillOutput = {
    name: root.name(),
    description: root.description(),
    version: root.version() || undefined,
    commands: commandTree(root),
  };

  if (cfg.skillSource) {
    try {
      out.skillDoc = fs.readFileSync(cfg.skillSource, "utf8").trim();
    } catch {
      // skill source not found – continue without it
    }
  }

  return out;
}

/** Renders the human-readable version of the skill output. */
export function renderSkillText(payload: SkillOutput, _cfg: SkillConfig): string {
  if (payload.skillDoc) {
    return payload.skillDoc;
  }

  const lines: string[] = [];
  lines.push(`# ${payload.name}`);
  lines.push("");
  if (payload.description) {
    lines.push(payload.description);
    lines.push("");
  }
  if (payload.version) {
    lines.push(`Version: ${payload.version}`);
    lines.push("");
  }
  lines.push("## Commands");
  lines.push("");
  for (const cmd of payload.commands) {
    appendCommandInfo(lines, cmd, 0);
  }
  return lines.join("\n");
}

function appendCommandInfo(
  lines: string[],
  cmd: CommandInfo,
  depth: number
): void {
  const indent = "  ".repeat(depth);
  lines.push(`${indent}### ${cmd.name}`);
  lines.push("");
  if (cmd.description) {
    lines.push(`${indent}${cmd.description}`);
    lines.push("");
  }
  if (cmd.flags && cmd.flags.length > 0) {
    lines.push(`${indent}Flags:`);
    for (const f of cmd.flags) {
      let line = `${indent}  --${f.name}`;
      if (f.shorthand) line += `, -${f.shorthand}`;
      line += `  ${f.usage}`;
      if (f.default && f.default !== "false" && f.default !== "") {
        line += ` (default: ${f.default})`;
      }
      lines.push(line);
    }
    lines.push("");
  }
  if (cmd.subCommands) {
    for (const sub of cmd.subCommands) {
      appendCommandInfo(lines, sub, depth + 1);
    }
  }
}

/** Recursively converts a commander Command tree into CommandInfo structs. */
function commandTree(cmd: Command): CommandInfo[] {
  return cmd.commands
    .filter((sub) => !(sub as unknown as { hidden?: boolean }).hidden)
    .map((sub) => ({
      name: sub.name(),
      description: sub.description(),
      arguments: sub.usage() || undefined,
      flags: collectFlags(sub),
      subCommands: commandTree(sub),
    }));
}

function collectFlags(cmd: Command): FlagInfo[] {
  const flags: FlagInfo[] = [];
  // commander stores options internally; iterate via _optionValues keys is
  // unreliable, so we use the public options array.
  for (const opt of cmd.options) {
    if ((opt as { hidden?: boolean }).hidden) continue;
    const flag: FlagInfo = {
      name: opt.long?.replace(/^--/, "") ?? opt.name(),
      usage: opt.description,
    };
    if (opt.short) flag.shorthand = opt.short.replace(/^-/, "");
    if (opt.defaultValue !== undefined) {
      flag.default = String(opt.defaultValue);
    }
    flags.push(flag);
  }
  return flags;
}
