import { Router, type Router as RouterType } from "express";
import { getClient, getConnectionState, getLastPaymentEvent } from "../mcp-client.js";

const router: RouterType = Router();

router.get("/status", (_req, res) => {
  const state = getConnectionState();
  const lastPayment = getLastPaymentEvent();
  res.json({ ...state, lastPayment });
});

router.get("/tools", async (req, res) => {
  try {
    const client = getClient();
    const response = await client.listTools();
    const tools = Array.isArray((response as { tools?: unknown }).tools)
      ? (response as { tools: Array<{ name: string; description?: string; inputSchema?: unknown }> }).tools
      : [];

    const q = (req.query.q as string || "").toLowerCase();
    const filtered = q
      ? tools.filter(
          (t) =>
            t.name.toLowerCase().includes(q) ||
            (t.description || "").toLowerCase().includes(q),
        )
      : tools;

    res.json({ tools: filtered, total: tools.length });
  } catch (err) {
    res.status(500).json({ error: err instanceof Error ? err.message : String(err) });
  }
});

router.post("/tools/:name/call", async (req, res) => {
  try {
    const client = getClient();
    const { name } = req.params;
    const args = req.body || {};

    const response = await client.callTool(name, args);

    const paymentInfo = getLastPaymentEvent();

    res.json({
      content: response.content,
      isError: response.isError,
      payment: paymentInfo,
    });
  } catch (err) {
    res.status(500).json({ error: err instanceof Error ? err.message : String(err) });
  }
});

export default router;
