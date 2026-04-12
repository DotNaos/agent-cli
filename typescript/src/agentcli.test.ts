/**
 * Tests for the @dotnaos/agent-cli TypeScript framework.
 * Uses Node.js built-in test runner (node:test) available in Node 18+.
 */
import { test, describe } from "node:test";
import assert from "node:assert/strict";
import { Readable } from "stream";

import {
  createAgentCLI,
  validateRequiredCommands,
  REQUIRED_COMMANDS,
  currentFormat,
  isMachineOutput,
  validateOutputFlags,
  serialise,
  buildSkillOutput,
  renderSkillText,
  parseDuration,
  OutputOptions,
  LogFlags,
} from "./index";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Captures stdout output from executing a CLI with given args. */
function captureOutput(args: string[]): Promise<string> {
  return new Promise((resolve, reject) => {
    let out = "";
    const original = process.stdout.write.bind(process.stdout);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (process.stdout as any).write = (chunk: unknown) => {
      out += String(chunk);
      return true;
    };

    const program = createAgentCLI({
      name: "testcli",
      description: "A test CLI",
      version: "0.0.1",
      logProvider: (_flags: LogFlags): NodeJS.ReadableStream => {
        const content =
          "2024-01-01T00:00:00Z [info] started\n" +
          "2024-01-01T00:00:01Z [error] something failed\n" +
          "2024-01-01T00:00:02Z [debug] done\n";
        return Readable.from([content]);
      },
    });

    program.exitOverride();
    program.configureOutput({
      writeErr: () => {},
      writeOut: (str) => { out += str; },
    });

    try {
      program.parse(["node", "testcli", ...args]);
      // Restore stdout after a tick to capture async writes.
      setImmediate(() => {
        (process.stdout as any).write = original;
        resolve(out);
      });
    } catch (err) {
      (process.stdout as any).write = original;
      reject(err);
    }
  });
}

// ---------------------------------------------------------------------------
// Output helpers
// ---------------------------------------------------------------------------

describe("OutputOptions helpers", () => {
  test("currentFormat returns text by default", () => {
    const opts: OutputOptions = { json: false, yaml: false, yml: false };
    assert.equal(currentFormat(opts), "text");
  });

  test("currentFormat returns json when json=true", () => {
    const opts: OutputOptions = { json: true, yaml: false, yml: false };
    assert.equal(currentFormat(opts), "json");
  });

  test("currentFormat returns yaml when yaml=true", () => {
    const opts: OutputOptions = { json: false, yaml: true, yml: false };
    assert.equal(currentFormat(opts), "yaml");
  });

  test("currentFormat returns yaml when yml=true", () => {
    const opts: OutputOptions = { json: false, yaml: false, yml: true };
    assert.equal(currentFormat(opts), "yaml");
  });

  test("isMachineOutput is false for text format", () => {
    const opts: OutputOptions = { json: false, yaml: false, yml: false };
    assert.equal(isMachineOutput(opts), false);
  });

  test("isMachineOutput is true for json format", () => {
    const opts: OutputOptions = { json: true, yaml: false, yml: false };
    assert.equal(isMachineOutput(opts), true);
  });

  test("validateOutputFlags throws when multiple flags are set", () => {
    const opts: OutputOptions = { json: true, yaml: true, yml: false };
    assert.throws(() => validateOutputFlags(opts), /choose exactly one/);
  });

  test("validateOutputFlags does not throw for single flag", () => {
    const opts: OutputOptions = { json: true, yaml: false, yml: false };
    assert.doesNotThrow(() => validateOutputFlags(opts));
  });

  test("serialise produces valid JSON", () => {
    const data = { foo: "bar", n: 42 };
    const output = serialise(data, "json");
    assert.deepEqual(JSON.parse(output), data);
  });

  test("serialise produces valid YAML", () => {
    const data = { foo: "bar" };
    const output = serialise(data, "yaml");
    assert.ok(output.includes("foo:"));
    assert.ok(output.includes("bar"));
  });

  test("serialise throws for text format", () => {
    assert.throws(() => serialise({}, "text"), /renderText/);
  });
});

// ---------------------------------------------------------------------------
// parseDuration
// ---------------------------------------------------------------------------

describe("parseDuration", () => {
  test("parses seconds", () => assert.equal(parseDuration("30s"), 30_000));
  test("parses minutes", () => assert.equal(parseDuration("5m"), 300_000));
  test("parses hours", () => assert.equal(parseDuration("2h"), 7_200_000));
  test("parses days", () => assert.equal(parseDuration("1d"), 86_400_000));
  test("defaults bare number to minutes", () => assert.equal(parseDuration("10"), 600_000));
  test("throws on invalid input", () => {
    assert.throws(() => parseDuration("abc"), /Cannot parse/);
  });
});

// ---------------------------------------------------------------------------
// Framework validation
// ---------------------------------------------------------------------------

describe("validateRequiredCommands", () => {
  test("REQUIRED_COMMANDS contains completions, skill, logs", () => {
    const required = [...REQUIRED_COMMANDS];
    assert.ok(required.includes("completions"));
    assert.ok(required.includes("skill"));
    assert.ok(required.includes("logs"));
  });

  test("returns empty array when all required commands are present", () => {
    const program = createAgentCLI({
      name: "testcli",
      description: "test",
    });
    const missing = validateRequiredCommands(program);
    assert.deepEqual(missing, []);
  });

  test("returns missing command names when commands are absent", () => {
    const program = createAgentCLI({
      name: "testcli",
      description: "test",
    });
    // Simulate a CLI that's missing the skill command by checking against
    // a subset of commands – we test validateRequiredCommands directly.
    const { Command } = require("commander") as typeof import("commander");
    const bare = new Command().name("bare");
    bare.addCommand(new Command("completions"));
    bare.addCommand(new Command("logs"));
    // skill is intentionally absent
    const missing = validateRequiredCommands(bare);
    assert.ok(missing.includes("skill"));
  });
});

// ---------------------------------------------------------------------------
// buildSkillOutput
// ---------------------------------------------------------------------------

describe("buildSkillOutput", () => {
  test("returns correct name and description", () => {
    const program = createAgentCLI({
      name: "mycli",
      description: "My CLI",
      version: "1.2.3",
    });
    const output = buildSkillOutput(program, {});
    assert.equal(output.name, "mycli");
    assert.equal(output.description, "My CLI");
    assert.equal(output.version, "1.2.3");
  });

  test("includes the required commands in the command list", () => {
    const program = createAgentCLI({
      name: "mycli",
      description: "test",
    });
    const output = buildSkillOutput(program, {});
    const names = output.commands.map((c) => c.name);
    assert.ok(names.includes("completions"), "should include completions");
    assert.ok(names.includes("skill"), "should include skill");
    assert.ok(names.includes("logs"), "should include logs");
  });

  test("renderSkillText produces markdown with headers", () => {
    const program = createAgentCLI({ name: "mycli", description: "My CLI" });
    const output = buildSkillOutput(program, {});
    const text = renderSkillText(output, {});
    assert.ok(text.includes("# mycli"), "should have h1 header");
    assert.ok(text.includes("## Commands"), "should have Commands section");
  });
});
