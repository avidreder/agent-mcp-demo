import { createx402MCPClient, type Network, type x402MCPClient } from "@x402/mcp";
import { ExactEvmScheme } from "@x402/evm/exact/client";
import { ExactEvmSchemeV1 } from "@x402/evm/exact/v1/client";
import { StreamableHTTPClientTransport } from "@modelcontextprotocol/sdk/client/streamableHttp.js";
import { privateKeyToAccount } from "viem/accounts";
import type { Hex } from "viem";

type ConnectionState = "disconnected" | "connecting" | "connected" | "error";

interface PaymentEvent {
  timestamp: number;
  endpoint: string;
  amount: string;
  currency: string;
  network: string;
}

let client: x402MCPClient | null = null;
let connectionState: ConnectionState = "disconnected";
let connectionError: string | null = null;
let lastPaymentEvent: PaymentEvent | null = null;

function createWallet(privateKey: string) {
  const formattedKey = privateKey.startsWith("0x")
    ? (privateKey as Hex)
    : (`0x${privateKey}` as Hex);
  const account = privateKeyToAccount(formattedKey);
  return { address: account.address, account };
}

export async function initializeMCPClient(
  serverUrl: string,
  privateKey: string,
): Promise<void> {
  connectionState = "connecting";
  connectionError = null;

  try {
    const wallet = createWallet(privateKey);
    console.log(`Wallet address: ${wallet.address}`);

    const schemes = [
      { network: "eip155:8453" as Network, client: new ExactEvmScheme(wallet.account) },
      { network: "eip155:84532" as Network, client: new ExactEvmScheme(wallet.account) },
      { network: "base" as Network, client: new ExactEvmSchemeV1(wallet.account), x402Version: 1 },
      { network: "base-sepolia" as Network, client: new ExactEvmSchemeV1(wallet.account), x402Version: 1 },
    ];

    const mcpClient = createx402MCPClient({
      name: "x402-bazaar-web",
      version: "0.1.0",
      schemes,
      autoPayment: true,
      onPaymentRequested: async ({ paymentRequired }) => {
        const accepts = paymentRequired.accepts ?? [];
        const preferredNetworks = new Set(schemes.map((s) => s.network));
        const selected =
          accepts.find((a) => preferredNetworks.has(a.network as Network)) ??
          accepts[0];

        const currency =
          (selected?.extra as { name?: string } | undefined)?.name ??
          selected?.asset ??
          "unknown";
        const amount =
          (selected as { amount?: string; maxAmountRequired?: string } | undefined)?.amount ??
          (selected as { maxAmountRequired?: string } | undefined)?.maxAmountRequired ??
          "unknown";
        const network = selected?.network ?? "unknown";
        const endpoint = (selected as { resource?: string } | undefined)?.resource ?? "unknown";

        lastPaymentEvent = {
          timestamp: Date.now(),
          endpoint,
          amount,
          currency,
          network,
        };

        console.log(`Payment requested: ${amount} ${currency} on ${network}`);
        return true;
      },
    });

    const transport = new StreamableHTTPClientTransport(new URL(serverUrl));
    await mcpClient.connect(transport);

    client = mcpClient;
    connectionState = "connected";
    console.log("MCP client connected");
  } catch (err) {
    connectionState = "error";
    connectionError = err instanceof Error ? err.message : String(err);
    console.error("MCP connection failed:", connectionError);
    throw err;
  }
}

export function getClient(): x402MCPClient {
  if (!client) {
    throw new Error("MCP client not initialized");
  }
  return client;
}

export function getConnectionState(): {
  state: ConnectionState;
  error: string | null;
} {
  return { state: connectionState, error: connectionError };
}

export function getLastPaymentEvent(): PaymentEvent | null {
  return lastPaymentEvent;
}
