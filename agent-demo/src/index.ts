import dotenv from "dotenv";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { createx402MCPClient, type Network } from "@x402/mcp";
import { ExactEvmScheme } from "@x402/evm/exact/client";
import { ExactEvmSchemeV1 } from "@x402/evm/exact/v1/client";
import { derivePaymentCapabilities, formatCapabilitiesForPrompt } from "./capabilities.js";
import { StreamableHTTPClientTransport } from "@modelcontextprotocol/sdk/client/streamableHttp.js";
import { createWallet } from "./wallet.js";
import { Agent, type ToolCallHistoryItem } from "./agent.js";
import * as ui from "./ui.js";

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
const AGENT_GOAL = process.env.AGENT_GOAL || "Find out the weather";
const DEBUG_MODE = process.env.DEBUG === "true";
const TIMEOUT_MS = 20000;
const MAX_TOOL_CALLS = Number(process.env.MAX_TOOL_CALLS || "5");

function withTimeout<T>(promise: Promise<T>, ms: number, operation: string): Promise<T> {
  return Promise.race([
    promise,
    new Promise<T>((_, reject) =>
      setTimeout(() => reject(new Error(`${operation} timed out after ${ms}ms`)), ms)
    ),
  ]);
}

async function main() {
  ui.header("x402 Agent Demo");

  // Validate environment
  if (!PRIVATE_KEY) {
    ui.error("PRIVATE_KEY environment variable is required");
    process.exit(1);
  }
  if (!LLM_API_KEY) {
    ui.error("LLM_API_KEY environment variable is required");
    process.exit(1);
  }

  // Step 1: Create wallet
  ui.step("🔐", "Creating EVM Wallet");
  const wallet = createWallet(PRIVATE_KEY);
  ui.success(`Wallet created`);
  ui.info(`Address: ${wallet.address}`);

  // Step 2: Connect to MCP server
  ui.step("🔌", "Connecting to MCP Server");

  const schemes = [
    { network: "eip155:8453" as Network, client: new ExactEvmScheme(wallet.account) },
    { network: "eip155:84532" as Network, client: new ExactEvmScheme(wallet.account) },
    { network: "base" as Network, client: new ExactEvmSchemeV1(wallet.account), x402Version: 1 },
    { network: "base-sepolia" as Network, client: new ExactEvmSchemeV1(wallet.account), x402Version: 1 },
  ];

  const mcpClient = createx402MCPClient({
    name: "x402-agent",
    version: "0.1.0",
    schemes,
    autoPayment: true,
    onPaymentRequested: async ({ paymentRequired }) => {
      if (DEBUG_MODE) {
        ui.info(`💰 Payment requested: ${paymentRequired.accepts[0]?.amount ?? "unknown"}`);
      }
      return true;
    },
  });

  const connectSpinner = ui.spinner(`Connecting to ${MCP_SERVER_URL}`);
  try {
    const transport = new StreamableHTTPClientTransport(new URL(MCP_SERVER_URL));
    await withTimeout(mcpClient.connect(transport), TIMEOUT_MS, "MCP connection");
    connectSpinner.succeed("Connected to MCP server");
  } catch (err) {
    connectSpinner.fail("Failed to connect to MCP server");
    ui.error(err instanceof Error ? err.message : String(err));
    process.exit(1);
  }

  // Step 3: List available tools
  ui.step("🔧", "Fetching Available Tools");
  const toolsResponse = await mcpClient.listTools();
  const tools = Array.isArray((toolsResponse as { tools?: unknown }).tools)
    ? ((toolsResponse as { tools: Array<{ name: string; description?: string; inputSchema?: unknown }> }).tools)
    : [];
  ui.section(
    "Available Tools",
    tools.map((t) => `• ${t.name} - ${t.description || "No description"}`)
  );

  // Step 4+: Agent executes multiple steps until done
  ui.step("🤖", "Agent Analyzing Goal");
  ui.section("Goal", AGENT_GOAL);

  const capabilitiesPrompt = formatCapabilitiesForPrompt(derivePaymentCapabilities(schemes));
  const agent = new Agent(LLM_API_KEY, LLM_BASE_URL, capabilitiesPrompt);
  const history: ToolCallHistoryItem[] = [];
  let done = false;
  let finalSummary = "";

  for (let step = 1; step <= MAX_TOOL_CALLS; step++) {
    const thinkingSpinner = ui.spinner(`Agent is thinking (step ${step}/${MAX_TOOL_CALLS})...`);
    const decision = await withTimeout(
      agent.decideNextStep(AGENT_GOAL, tools, history),
      TIMEOUT_MS,
      "Agent decision"
    );
    thinkingSpinner.stop();

    ui.agentThought(decision.reason);

    if (decision.action === "done") {
      done = true;
      finalSummary = decision.summary || "Goal completed.";
      break;
    }

    if (!decision.toolName) {
      ui.error("Agent did not provide a tool name.");
      break;
    }

    ui.section("Selected Tool", [
      `Name: ${decision.toolName}`,
      `Args: ${JSON.stringify(decision.args || {})}`,
    ]);

    ui.step("⚡", `Executing Tool (step ${step})`);
    const executeSpinner = ui.spinner(`Calling ${decision.toolName}...`);

    try {
      const response = await withTimeout(
        mcpClient.callTool(decision.toolName, decision.args || {}),
        TIMEOUT_MS,
        "Tool execution"
      );
      executeSpinner.succeed(`Tool executed successfully`);
      ui.json("Response", response.content);

      if (DEBUG_MODE && response.paymentMade) {
        ui.info(`💰 Payment submitted for ${decision.toolName}`);
      }
      if (response.paymentResponse?.success) {
        ui.success(
          `Payment successful: ${response.paymentResponse.transaction || "completed"}`
        );
      }

      history.push({
        toolName: decision.toolName,
        args: decision.args || {},
        result: response.content,
      });
    } catch (err) {
      executeSpinner.fail("Tool execution failed");
      ui.error(err instanceof Error ? err.message : String(err));
      break;
    }
  }

  ui.step("📝", "Agent Summary");
  if (done) {
    ui.section("Result", finalSummary);
  } else if (history.length > 0) {
    const summarySpinner = ui.spinner("Generating summary...");
    const last = history[history.length - 1];
    const summary = await withTimeout(
      agent.summarizeResult(AGENT_GOAL, last.toolName, last.result),
      TIMEOUT_MS,
      "Agent summary"
    );
    summarySpinner.stop();
    ui.section("Result", summary);
  } else {
    ui.section("Result", "No tool calls were completed.");
  }

  // Cleanup
  await mcpClient.close();
  ui.divider();
  ui.success("Demo complete");
}

main().catch((err) => {
  ui.error(err instanceof Error ? err.message : String(err));
  process.exit(1);
});
