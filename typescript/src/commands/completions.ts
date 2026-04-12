import { Command } from "commander";
import { OutputOptions, serialise, currentFormat } from "../output";

/** Structured result returned by the completions command. */
export interface CompletionResult {
  shell: string;
  script: string;
}

/**
 * Creates and returns a "completions" command that generates shell
 * autocompletion scripts.
 *
 * Because commander does not have built-in completion generation, this command
 * emits a minimal script for each shell that can be extended by the integrating
 * CLI. The shell-specific sub-commands are provided as a starting point.
 */
export function createCompletionsCommand(
  root: Command,
  opts: OutputOptions
): Command {
  const completions = new Command("completions")
    .description(
      "Generate shell autocompletion scripts\n\n" +
        "Run a sub-command to generate the script for your shell:\n" +
        "  bash, zsh, fish, powershell"
    )
    .action(() => {
      completions.help();
    });

  for (const shell of ["bash", "zsh", "fish", "powershell"] as const) {
    completions.addCommand(createShellCompletionCommand(root, opts, shell));
  }

  return completions;
}

function createShellCompletionCommand(
  root: Command,
  opts: OutputOptions,
  shell: "bash" | "zsh" | "fish" | "powershell"
): Command {
  const name = root.name() || "mycli";

  return new Command(shell)
    .description(`Generate the autocompletion script for ${shell}`)
    .addHelpText("after", setupInstructions(name, shell))
    .action(() => {
      const script = generateScript(name, shell);
      const result: CompletionResult = { shell, script };
      const format = currentFormat(opts);

      if (format !== "text") {
        process.stdout.write(serialise(result, format) + "\n");
      } else {
        process.stdout.write(script);
      }
    });
}

function setupInstructions(
  name: string,
  shell: "bash" | "zsh" | "fish" | "powershell"
): string {
  switch (shell) {
    case "bash":
      return (
        `\nTo load completions in your current shell session:\n` +
        `\n    source <(${name} completions bash)\n` +
        `\nTo load completions for every new session, execute once:\n` +
        `\n  Linux:\n    ${name} completions bash > /etc/bash_completion.d/${name}\n` +
        `\n  macOS:\n    ${name} completions bash > $(brew --prefix)/etc/bash_completion.d/${name}\n`
      );
    case "zsh":
      return (
        `\nTo load completions in your current shell session:\n` +
        `\n    source <(${name} completions zsh)\n` +
        `\nTo load completions for every new session, execute once:\n` +
        `\n    ${name} completions zsh > "\${fpath[1]}/_${name}"\n`
      );
    case "fish":
      return (
        `\nTo load completions in your current shell session:\n` +
        `\n    ${name} completions fish | source\n` +
        `\nTo load completions for every new session, execute once:\n` +
        `\n    ${name} completions fish > ~/.config/fish/completions/${name}.fish\n`
      );
    case "powershell":
      return (
        `\nTo load completions in your current shell session:\n` +
        `\n    ${name} completions powershell | Out-String | Invoke-Expression\n` +
        `\nTo load completions for every new session, add the above to your PowerShell profile.\n`
      );
  }
}

/**
 * Generates a minimal shell completion script for the given CLI name and shell.
 * Integrating CLIs may replace this with a richer script.
 */
function generateScript(
  name: string,
  shell: "bash" | "zsh" | "fish" | "powershell"
): string {
  const commands = ["completions", "skill", "logs", "help"];
  switch (shell) {
    case "bash":
      return (
        `# bash completion for ${name}\n` +
        `_${name}_completions() {\n` +
        `  local cur="\${COMP_WORDS[COMP_CWORD]}"\n` +
        `  local commands="${commands.join(" ")}"\n` +
        `  COMPREPLY=($(compgen -W "$commands" -- "$cur"))\n` +
        `}\n` +
        `complete -F _${name}_completions ${name}\n`
      );
    case "zsh":
      return (
        `#compdef ${name}\n` +
        `_${name}() {\n` +
        `  local -a commands\n` +
        `  commands=(${commands.map((c) => `'${c}'`).join(" ")})\n` +
        `  _describe 'command' commands\n` +
        `}\n` +
        `_${name}\n`
      );
    case "fish":
      return (
        commands
          .map(
            (c) =>
              `complete -c ${name} -f -n '__fish_use_subcommand' -a ${c} -d '${c} command'`
          )
          .join("\n") + "\n"
      );
    case "powershell":
      return (
        `Register-ArgumentCompleter -Native -CommandName '${name}' -ScriptBlock {\n` +
        `  param($wordToComplete, $commandAst, $cursorPosition)\n` +
        `  $commands = @(${commands.map((c) => `'${c}'`).join(", ")})\n` +
        `  $commands | Where-Object { $_ -like "$wordToComplete*" } |\n` +
        `    ForEach-Object { [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_) }\n` +
        `}\n`
      );
  }
}
