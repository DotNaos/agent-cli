/**
 * Example CLI built with the @dotnaos/agent-cli framework.
 * Run: node dist/example.js <command>
 */
import { createAgentCLI, fileLogProvider } from "./index";
import * as fs from "fs";
import * as path from "path";
import * as os from "os";

// Ensure a sample log file exists for demo purposes.
const logPath = path.join(os.tmpdir(), "example-agent-cli.log");
if (!fs.existsSync(logPath)) {
  fs.writeFileSync(
    logPath,
    [
      "2024-01-01T00:00:00Z [info] example TypeScript CLI started",
      "2024-01-01T00:00:01Z [debug] loading configuration",
      "2024-01-01T00:00:02Z [warn] configuration not found, using defaults",
      "2024-01-01T00:00:03Z [info] ready",
    ].join("\n") + "\n",
    "utf8"
  );
}

const program = createAgentCLI({
  name: "example-ts",
  description: "An example agent-friendly CLI built with the TypeScript agentcli framework",
  version: "0.1.0",
  logProvider: fileLogProvider(logPath),
});

// Add an application-specific command.
program
  .command("greet [name]")
  .description("Print a greeting")
  .action((name: string = "world") => {
    const message = `Hello, ${name}!`;
    const opts = program.opts<{ json: boolean; yaml: boolean; yml: boolean }>();
    if (opts.json) {
      process.stdout.write(JSON.stringify({ message }, null, 2) + "\n");
    } else if (opts.yaml || opts.yml) {
      process.stdout.write(`message: ${message}\n`);
    } else {
      process.stdout.write(message + "\n");
    }
  });

program.parse();
