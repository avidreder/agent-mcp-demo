import type { SchemeNetworkClient, Network } from "@x402/mcp";

export interface SchemeEntry {
  network: Network;
  client: SchemeNetworkClient;
  x402Version?: number;
}

export interface PaymentCapability {
  network: string;
  scheme: string;
  x402Version: number;
}

const NETWORK_LABELS: Record<string, string> = {
  "eip155:8453": "Base (mainnet)",
  "eip155:84532": "Base Sepolia (testnet)",
  base: "Base (mainnet, v1 legacy)",
  "base-sepolia": "Base Sepolia (testnet, v1 legacy)",
};

export function derivePaymentCapabilities(
  schemes: SchemeEntry[],
): PaymentCapability[] {
  return schemes.map((s) => ({
    network: s.network,
    scheme: s.client.scheme,
    x402Version: s.x402Version ?? 2,
  }));
}

export function formatCapabilitiesForPrompt(
  capabilities: PaymentCapability[],
): string {
  const lines = capabilities.map((c) => {
    const label = NETWORK_LABELS[c.network] ?? c.network;
    return `  - ${label} (network: ${c.network}, scheme: ${c.scheme}, x402 v${c.x402Version})`;
  });

  return [
    "You have an EVM wallet that can make payments via the x402 protocol.",
    "Payments may be required for some tools; proceed anyway -- the client handles payment automatically.",
    "Supported payment networks:",
    ...lines,
    "You are authorized to pay up to 1000 USDC on any of the above networks.",
  ].join("\n");
}
