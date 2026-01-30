import "dotenv/config";
import { X402MCPClient, type ToolCallRequest, type ToolCallResponse } from "x402-mcp";
import { createWallet } from "./wallet.js";
import { Agent, type ToolCallHistoryItem } from "./agent.js";
import * as ui from "./ui.js";

const MCP_SERVER_URL =
  process.env.MCP_SERVER_URL || "http://localhost:8003/v2/x402/mcp";
const MCP_TRANSPORT = process.env.MCP_TRANSPORT || "streamable-http";
const PRIVATE_KEY = process.env.PRIVATE_KEY;
const ANTHROPIC_API_KEY = process.env.ANTHROPIC_API_KEY;
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
  if (!ANTHROPIC_API_KEY) {
    ui.error("ANTHROPIC_API_KEY environment variable is required");
    process.exit(1);
  }

  // Step 1: Create wallet
  ui.step("🔐", "Creating EVM Wallet");
  const wallet = createWallet(PRIVATE_KEY);
  ui.success(`Wallet created`);
  ui.info(`Address: ${wallet.address}`);

  // Step 2: Connect to MCP server with interceptors and debug mode
  ui.step("🔌", "Connecting to MCP Server");

  const mcpClient = new X402MCPClient({
    serverUrl: MCP_SERVER_URL,
    transport: MCP_TRANSPORT === "sse" ? "sse" : "streamable-http",
    debug: DEBUG_MODE,
    // Enable x402 payment support with EVM signer (real EIP-712 signing)
    signer: wallet.account,
    autoPayment: true, // Automatically retry with payment on 402
    interceptors: {
      // Example: Log outgoing requests with wallet context
      beforeToolCall: [
        async (request: ToolCallRequest) => {
          if (DEBUG_MODE) {
            ui.info(`Wallet ${wallet.address} calling ${request.name}`);
          }
          return request;
        },
      ],
      // Example: Log response timing and payment info
      afterToolCall: [
        async (
          request: ToolCallRequest,
          response: ToolCallResponse
        ) => {
          if (DEBUG_MODE) {
            ui.info(`Tool ${request.name} completed in ${response.duration}ms`);
          }
          // Log x402 payment info if present
          if (response.paymentRequired) {
            ui.info(`💰 Payment was required for ${request.name}`);
            if (response.paymentResponse?.success) {
              ui.success(`Payment successful: ${response.paymentResponse.transaction || 'completed'}`);
            }
          }
          return response;
        },
      ],
    },
  });

  const connectSpinner = ui.spinner(`Connecting to ${MCP_SERVER_URL}`);
  try {
    await withTimeout(mcpClient.connect(), TIMEOUT_MS, "MCP connection");
    connectSpinner.succeed("Connected to MCP server");
  } catch (err) {
    connectSpinner.fail("Failed to connect to MCP server");
    ui.error(err instanceof Error ? err.message : String(err));
    process.exit(1);
  }

  // Step 3: List available tools
  ui.step("🔧", "Fetching Available Tools");
  const tools = await mcpClient.listTools();
  ui.section(
    "Available Tools",
    tools.map((t) => `• ${t.name} - ${t.description || "No description"}`)
  );

  // Step 4+: Agent executes multiple steps until done
  ui.step("🤖", "Agent Analyzing Goal");
  ui.section("Goal", AGENT_GOAL);

  const agent = new Agent(ANTHROPIC_API_KEY);
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
  await mcpClient.disconnect();
  ui.divider();
  ui.success("Demo complete");
}

main().catch((err) => {
  ui.error(err instanceof Error ? err.message : String(err));
  process.exit(1);
});
