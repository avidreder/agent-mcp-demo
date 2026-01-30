import Anthropic from "@anthropic-ai/sdk";
import type { MCPTool } from "x402-mcp";

export interface ToolSelection {
  toolName: string;
  reason: string;
  args: Record<string, unknown>;
}

export interface AgentDecision {
  action: "call_tool" | "done";
  reason: string;
  toolName?: string;
  args?: Record<string, unknown>;
  summary?: string;
}

export interface ToolCallHistoryItem {
  toolName: string;
  args: Record<string, unknown>;
  result: unknown;
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
      .map((t) => {
        const schema = t.inputSchema
          ? JSON.stringify(t.inputSchema, null, 2)
          : "No schema";
        return `- ${t.name}: ${t.description || "No description"}\n  inputSchema: ${schema}`;
      })
      .join("\n");

    const message = await this.client.messages.create({
      model: "claude-sonnet-4-20250514",
      max_tokens: 1024,
      messages: [
        {
          role: "user",
          content: `You are an AI agent that selects the best tool to accomplish a goal.
Payments may be required for some tools; proceed anyway and assume the client will handle payment.

Available tools:
${toolDescriptions}

Goal: ${goal}

Use the tool's inputSchema to choose exact argument names.
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

  async decideNextStep(
    goal: string,
    tools: MCPTool[],
    history: ToolCallHistoryItem[]
  ): Promise<AgentDecision> {
    const toolDescriptions = tools
      .map((t) => {
        const schema = t.inputSchema
          ? JSON.stringify(t.inputSchema, null, 2)
          : "No schema";
        return `- ${t.name}: ${t.description || "No description"}\n  inputSchema: ${schema}`;
      })
      .join("\n");

    const message = await this.client.messages.create({
      model: "claude-sonnet-4-20250514",
      max_tokens: 1024,
      messages: [
        {
          role: "user",
          content: `You are an AI agent working toward a goal and can call tools multiple times.
Payments may be required for some tools; proceed anyway and assume the client will handle payment.

Available tools:
${toolDescriptions}

Goal: ${goal}
History: ${JSON.stringify(history, null, 2)}

Use the tool's inputSchema to choose exact argument names.
If the goal is already satisfied, respond with action "done" and provide a short summary.
Otherwise respond with action "call_tool" and select the next best tool.
Respond with JSON only (no markdown):
{
  "action": "call_tool | done",
  "reason": "brief explanation of why",
  "toolName": "name of the selected tool (required when action is call_tool)",
  "args": {},
  "summary": "short summary (required when action is done)"
}`,
        },
      ],
    });

    const content = message.content[0];
    if (content.type !== "text") {
      throw new Error("Unexpected response type");
    }

    return JSON.parse(content.text) as AgentDecision;
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
