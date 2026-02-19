import Anthropic from "@anthropic-ai/sdk";
import dotenv from "dotenv";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { createx402MCPClient, type Network } from "@x402/mcp";
import { ExactEvmScheme } from "@x402/evm/exact/client";
import { ExactEvmSchemeV1 } from "@x402/evm/exact/v1/client";
import { derivePaymentCapabilities, formatCapabilitiesForPrompt } from "./capabilities.js";
import { StreamableHTTPClientTransport } from "@modelcontextprotocol/sdk/client/streamableHttp.js";
import { createWallet } from "./wallet.js";

type MCPTool = {
  name: string;
  description?: string;
  inputSchema?: unknown;
};

type ToolCallHistoryItem = {
  toolName: string;
  args: Record<string, unknown>;
  result: unknown;
};

type AgentDecision =
  | {
      action: "call_tool";
      reason: string;
      toolName: string;
      args: Record<string, unknown>;
    }
  | {
      action: "done";
      reason: string;
      summary: string;
    };

const currentFile = fileURLToPath(import.meta.url);
const currentDir = path.dirname(currentFile);
const repoRoot = path.resolve(currentDir, "..", "..");
dotenv.config({ path: path.join(repoRoot, ".env") });
dotenv.config();

const MCP_SERVER_URL =
  process.env.MCP_SERVER_URL || "http://localhost:18081/discovery/mcp";
const PRIVATE_KEY = process.env.PRIVATE_KEY;
const LLM_API_KEY = process.env.LLM_API_KEY;
const LLM_BASE_URL = process.env.LLM_BASE_URL;
const SIMPLE_GOAL =
  process.env.SIMPLE_GOAL || "Find a tool to answer the weather in San Francisco.";
const MAX_STEPS = Number(process.env.SIMPLE_MAX_STEPS || "5");

function buildToolSummary(tools: MCPTool[]): string {
  return tools
    .map((t) => {
      const schema = t.inputSchema
        ? JSON.stringify(t.inputSchema, null, 2)
        : "No schema";
      return `- ${t.name}: ${t.description || "No description"}\n  inputSchema: ${schema}`;
    })
    .join("\n");
}

async function decideNextStep(
  client: Anthropic,
  goal: string,
  tools: MCPTool[],
  history: ToolCallHistoryItem[],
  paymentCapabilities: string
): Promise<AgentDecision> {
  const toolSummary = buildToolSummary(tools);
  const message = await client.messages.create({
    model: "claude-sonnet-4-20250514",
    max_tokens: 1024,
    messages: [
      {
        role: "user",
        content: `You are an AI agent working toward a goal and can call tools multiple times.
You only have access to the listed MCP tools and must decide which tool to call next.
${paymentCapabilities}

Available tools:
${toolSummary}

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
    throw new Error("Unexpected response type from model");
  }

  return JSON.parse(content.text) as AgentDecision;
}

async function main() {
  if (!PRIVATE_KEY) {
    throw new Error("PRIVATE_KEY environment variable is required");
  }
  if (!LLM_API_KEY) {
    throw new Error("LLM_API_KEY environment variable is required");
  }

  const transport = new StreamableHTTPClientTransport(new URL(MCP_SERVER_URL));

  const wallet = createWallet(PRIVATE_KEY);
  const schemes = [
    { network: "eip155:8453" as Network, client: new ExactEvmScheme(wallet.account) },
    { network: "eip155:84532" as Network, client: new ExactEvmScheme(wallet.account) },
    { network: "base" as Network, client: new ExactEvmSchemeV1(wallet.account), x402Version: 1 },
    { network: "base-sepolia" as Network, client: new ExactEvmSchemeV1(wallet.account), x402Version: 1 },
  ];
  const capabilitiesPrompt = formatCapabilitiesForPrompt(derivePaymentCapabilities(schemes));

  const mcpClient = createx402MCPClient({
    name: "simple-agent",
    version: "0.1.0",
    schemes,
    autoPayment: true,
  });
  await mcpClient.connect(transport);

  const toolsResponse = await mcpClient.listTools();
  const tools = Array.isArray((toolsResponse as { tools?: unknown }).tools)
    ? ((toolsResponse as { tools: MCPTool[] }).tools)
    : [];

  const anthropic = new Anthropic({
    apiKey: LLM_API_KEY,
    baseURL: LLM_BASE_URL,
  });
  const history: ToolCallHistoryItem[] = [];

  for (let step = 0; step < MAX_STEPS; step++) {
    const decision = await decideNextStep(anthropic, SIMPLE_GOAL, tools, history, capabilitiesPrompt);

    if (decision.action === "done") {
      // eslint-disable-next-line no-console
      console.log(`Summary: ${decision.summary}`);
      await mcpClient.close();
      return;
    }

    // eslint-disable-next-line no-console
    console.log(`Calling tool: ${decision.toolName}`);
    const result = await mcpClient.callTool(decision.toolName, decision.args);
    history.push({
      toolName: decision.toolName,
      args: decision.args,
      result: result.content,
    });
  }

  // eslint-disable-next-line no-console
  console.log("Reached max steps without completion.");
  await mcpClient.close();
}

main().catch((err) => {
  // eslint-disable-next-line no-console
  console.error(err instanceof Error ? err.message : err);
  process.exit(1);
});
