import dotenv from "dotenv";
import path from "node:path";
import { fileURLToPath } from "node:url";
import express from "express";
import cors from "cors";
import { initializeMCPClient } from "./mcp-client.js";
import toolsRouter from "./routes/tools.js";

const currentFile = fileURLToPath(import.meta.url);
const currentDir = path.dirname(currentFile);
const repoRoot = path.resolve(currentDir, "..", "..");
dotenv.config({ path: path.join(repoRoot, ".env") });
dotenv.config();

const PORT = Number(process.env.WEB_PORT || 3402);
const MCP_SERVER_URL =
  process.env.MCP_SERVER_URL || "http://localhost:18081/discovery/mcp";
const PRIVATE_KEY = process.env.PRIVATE_KEY;

async function main() {
  if (!PRIVATE_KEY) {
    console.error("PRIVATE_KEY environment variable is required");
    process.exit(1);
  }

  console.log("Initializing MCP client...");
  await initializeMCPClient(MCP_SERVER_URL, PRIVATE_KEY);

  const app = express();
  app.use(cors());
  app.use(express.json());

  // Serve static files
  const publicDir = path.resolve(currentDir, "..", "public");
  app.use(express.static(publicDir));

  // API routes
  app.use("/api", toolsRouter);

  app.listen(PORT, () => {
    console.log(`x402 MCP Bazaar running at http://localhost:${PORT}`);
  });
}

main().catch((err) => {
  console.error("Fatal:", err);
  process.exit(1);
});
