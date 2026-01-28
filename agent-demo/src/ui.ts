import chalk from "chalk";
import boxen from "boxen";
import ora, { type Ora } from "ora";

export function header(title: string): void {
  console.log(
    boxen(chalk.bold.cyan(title), {
      padding: 1,
      margin: { top: 1, bottom: 0, left: 0, right: 0 },
      borderStyle: "round",
      borderColor: "cyan",
    })
  );
}

export function section(title: string, content: string | string[]): void {
  const lines = Array.isArray(content) ? content : [content];
  const body = lines.map((line) => `  ${line}`).join("\n");
  console.log(
    boxen(body, {
      title: chalk.bold(title),
      titleAlignment: "left",
      padding: { top: 0, bottom: 0, left: 1, right: 1 },
      margin: { top: 1, bottom: 0, left: 0, right: 0 },
      borderStyle: "single",
      borderColor: "gray",
    })
  );
}

export function step(icon: string, message: string): void {
  console.log(`\n${icon} ${message}`);
}

export function success(message: string): void {
  console.log(chalk.green(`✓ ${message}`));
}

export function error(message: string): void {
  console.log(chalk.red(`✗ ${message}`));
}

export function info(message: string): void {
  console.log(chalk.gray(`  ${message}`));
}

export function arrow(message: string): void {
  console.log(chalk.yellow(`→ ${message}`));
}

export function json(label: string, data: unknown): void {
  console.log(chalk.dim(`\n  ${label}:`));
  const formatted = JSON.stringify(data, null, 2)
    .split("\n")
    .map((line) => `    ${chalk.white(line)}`)
    .join("\n");
  console.log(formatted);
}

export function divider(): void {
  console.log(chalk.gray("\n" + "─".repeat(50)));
}

export function spinner(message: string): Ora {
  return ora({
    text: message,
    color: "cyan",
  }).start();
}

export function agentThought(thought: string): void {
  console.log(
    boxen(chalk.italic.magenta(thought), {
      title: chalk.bold("🤖 Agent Thinking"),
      titleAlignment: "left",
      padding: { top: 0, bottom: 0, left: 1, right: 1 },
      margin: { top: 1, bottom: 0, left: 0, right: 0 },
      borderStyle: "double",
      borderColor: "magenta",
    })
  );
}
