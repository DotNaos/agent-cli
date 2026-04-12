# agent-cli

A predefined configuration and framework for CLIs that are mainly used by agents.

---

## Overview

`agent-cli` is a library/framework for building **agent-friendly CLIs** in both **Go** and **TypeScript**. It enforces that every CLI built on top of it exposes the three required commands that every agent needs:

| Command | Purpose |
|---|---|
| `completions` | Shell tab-completion scripts (bash, zsh, fish, powershell) |
| `skill` | Machine-readable description of the CLI so agents know how to use it |
| `logs` | Real-time log streaming with agent-friendly filter options |

All CLIs automatically gain `--json`, `--yaml` / `--yml` flags for machine-readable output.

---

## Go – `go/agentcli`

### Requirements
- Go 1.21+
- Uses [Cobra](https://github.com/spf13/cobra)

### Usage

```go
import "github.com/DotNaos/agent-cli/go/agentcli"

func main() {
    root := agentcli.New(agentcli.Config{
        Use:         "mycli",
        Short:       "My agent-friendly CLI",
        Version:     "1.0.0",
        SkillSource: "SKILL.md",                             // optional
        LogProvider: agentcli.DefaultFileLogProvider("/var/log/mycli.log"),
    })

    // Add your own commands
    root.AddCommand(myCustomCmd)

    agentcli.Execute(root)
}
```

Access global output options from any sub-command:

```go
func myCustomCmd() *cobra.Command {
    return &cobra.Command{
        Use: "greet [name]",
        RunE: func(cmd *cobra.Command, args []string) error {
            payload := map[string]string{"message": "Hello, " + args[0] + "!"}
            return agentcli.Opts(cmd).WriteCommandOutput(cmd, payload, func(w io.Writer) error {
                _, err := fmt.Fprintln(w, payload["message"])
                return err
            })
        },
    }
}
```

### Built-in Commands

#### `completions <shell>`
Generates shell completion scripts. Supported: `bash`, `zsh`, `fish`, `powershell`.

```bash
source <(mycli completions bash)
mycli completions zsh --json   # get the script as JSON
```

#### `skill`
Outputs a description of the CLI for agents.

```bash
mycli skill          # human-readable Markdown
mycli skill --json   # machine-readable JSON
mycli skill --yaml   # machine-readable YAML
```

#### `logs`
Streams log output with agent-friendly options.

```bash
mycli logs                           # show last 50 lines
mycli logs --follow --json           # stream new lines as JSON
mycli logs --level error --since 1h  # errors from the last hour
mycli logs -n 100 --yaml             # last 100 lines as YAML documents
```

Flags:
- `-f, --follow` – follow new log entries in real-time
- `-n, --lines <N>` – show last N lines (default: 50)
- `--level <level>` – filter by log level: `debug`, `info`, `warn`, `error`
- `--since <duration>` – show entries from the last duration (e.g. `1h`, `30m`)
- `--path <path>` – override the log file path

### Framework Validation

```go
if err := agentcli.ValidateRequiredCommands(root); err != nil {
    log.Fatal(err)
}
```

### Build & Test

```bash
cd go
go build ./agentcli/...
go test ./agentcli/...
go build -o /tmp/example-cli ./example/
```

---

## TypeScript – `typescript/`

### Requirements
- Node.js 18+
- Uses [commander](https://github.com/tj/commander.js)

### Usage

```ts
import { createAgentCLI, fileLogProvider } from "@dotnaos/agent-cli";

const program = createAgentCLI({
  name: "mycli",
  description: "My agent-friendly CLI",
  version: "1.0.0",
  logProvider: fileLogProvider("/var/log/mycli.log"),
});

// Add your own commands
program.command("greet [name]")
  .description("Print a greeting")
  .action((name = "world") => console.log(`Hello, ${name}!`));

program.parse();
```

### Build & Test

```bash
cd typescript
npm install
npm run build     # compile TypeScript → dist/
npm test          # run tests with node:test
```

### Framework Validation

```ts
import { validateRequiredCommands } from "@dotnaos/agent-cli";
const missing = validateRequiredCommands(program);
if (missing.length > 0) {
  console.error("Missing required commands:", missing);
  process.exit(1);
}
```

---

## Required Commands Reference

| Command | Subcommands | Key Flags |
|---|---|---|
| `completions` | `bash`, `zsh`, `fish`, `powershell` | `--json`, `--yaml` |
| `skill` | — | `--json`, `--yaml` |
| `logs` | — | `-f/--follow`, `-n/--lines`, `--level`, `--since`, `--path`, `--json`, `--yaml` |

---

## Reference

See [moodle-cli](https://github.com/DotNaos/moodle-cli) for a full-featured CLI built on top of this pattern.

