import Anthropic from "@anthropic-ai/sdk";
import type { MCPTool } from "x402-mcp";

export interface ToolSelection {
  toolName: string;
  reason: string;
  args: Record<string, unknown>;
}

export class Agent {
  private client: Anthropic;

  constructor(apiKey: string) {
    this.client = new Anthropic({ apiKey });
  }

  async selectTool(
    goal: string,
    tools: MCPTool[]
  ): Promise<ToolSelection> {
    const toolDescriptions = tools
      .map((t) => `- ${t.name}: ${t.description || "No description"}`)
      .join("\n");

    const message = await this.client.messages.create({
      model: "claude-sonnet-4-20250514",
      max_tokens: 1024,
      messages: [
        {
          role: "user",
          content: `You are an AI agent that selects the best tool to accomplish a goal.

Available tools:
${toolDescriptions}

Goal: ${goal}

Respond with JSON only (no markdown):
{
  "toolName": "name of the selected tool",
  "reason": "brief explanation of why this tool was selected",
  "args": {}
}`,
        },
      ],
    });

    const content = message.content[0];
    if (content.type !== "text") {
      throw new Error("Unexpected response type");
    }

    return JSON.parse(content.text) as ToolSelection;
  }

  async summarizeResult(
    goal: string,
    toolName: string,
    result: unknown
  ): Promise<string> {
    const message = await this.client.messages.create({
      model: "claude-sonnet-4-20250514",
      max_tokens: 512,
      messages: [
        {
          role: "user",
          content: `You are an AI agent. You just called a tool to accomplish a goal.

Goal: ${goal}
Tool used: ${toolName}
Result: ${JSON.stringify(result, null, 2)}

Provide a brief, friendly summary of what was accomplished. Be concise (1-2 sentences).`,
        },
      ],
    });

    const content = message.content[0];
    if (content.type !== "text") {
      throw new Error("Unexpected response type");
    }

    return content.text;
  }
}
