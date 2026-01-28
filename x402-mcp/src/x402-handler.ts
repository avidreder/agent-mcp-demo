/**
 * X402 Payment Handler for MCP Client
 * Uses official @x402/core types and utilities
 * Detects 402 payment requirements and handles payment flow
 */

import type {
  PaymentPayload,
  PaymentRequirements,
} from "@x402/core/types";
import { safeBase64Encode } from "@x402/core/utils";
import type {
  ResourceInfo,
  PaymentRequiredData,
  PaymentResponse,
  PaymentCheckResult,
} from "./x402-types.js";
import {
  X402_VERSION,
  META_KEY_PAYMENT,
  META_KEY_PAYMENT_REQUIRED,
  META_KEY_PAYMENT_RESPONSE,
  isPaymentRequired,
  isPaymentResponse,
} from "./x402-types.js";

/**
 * Configuration for the X402 payment handler
 */
export interface X402HandlerConfig {
  /** Wallet address for payment authorization */
  walletAddress: string;

  /** Callback to sign payment authorization (optional for simplified mode) */
  signPayment?: (requirements: PaymentRequiredData) => Promise<string>;

  /** Enable auto-retry with payment when 402 is detected */
  autoRetry?: boolean;

  /** Callback when payment is required */
  onPaymentRequired?: (requirements: PaymentRequiredData) => void;

  /** Callback when payment succeeds */
  onPaymentSuccess?: (response: PaymentResponse) => void;

  /** Callback when payment fails */
  onPaymentFailed?: (error: string) => void;
}

/**
 * X402 Payment Handler
 * Handles x402 payment detection and payload creation for MCP
 */
export class X402Handler {
  private config: X402HandlerConfig;

  constructor(config: X402HandlerConfig) {
    this.config = config;
  }

  /**
   * Check if a tool response indicates payment is required
   */
  checkPaymentRequired(content: unknown): PaymentCheckResult {
    // Check if content is an array (MCP content format)
    if (Array.isArray(content)) {
      for (const item of content) {
        if (item && typeof item === "object" && "type" in item && item.type === "text") {
          const textItem = item as { type: "text"; text: string };
          try {
            const parsed = JSON.parse(textItem.text);
            if (isPaymentRequired(parsed)) {
              return { required: true, requirements: parsed };
            }
          } catch {
            // Not JSON, continue
          }
        }
      }
    }

    // Check if content itself is payment required data
    if (isPaymentRequired(content)) {
      return { required: true, requirements: content };
    }

    return { required: false };
  }

  /**
   * Check if response meta contains payment required data
   */
  checkMetaForPaymentRequired(meta: Record<string, unknown> | undefined): PaymentCheckResult {
    if (!meta) {
      return { required: false };
    }

    const paymentRequired = meta[META_KEY_PAYMENT_REQUIRED];
    if (paymentRequired && isPaymentRequired(paymentRequired)) {
      return { required: true, requirements: paymentRequired };
    }

    return { required: false };
  }

  /**
   * Extract payment response from response meta
   */
  extractPaymentResponse(meta: Record<string, unknown> | undefined): PaymentResponse | null {
    if (!meta) {
      return null;
    }

    const paymentResponse = meta[META_KEY_PAYMENT_RESPONSE];
    if (paymentResponse && isPaymentResponse(paymentResponse)) {
      return paymentResponse;
    }

    return null;
  }

  /**
   * Create a payment payload using official x402 types
   * TODO: Integrate with x402Client for actual wallet signing
   * For now, base64 encodes the payment requirements as simplified "signature"
   */
  async createPaymentPayload(requirements: PaymentRequiredData): Promise<PaymentPayload> {
    // Select the first accepted payment scheme
    // TODO: Use x402Client.selectPaymentRequirements() for proper selection
    const selectedScheme = requirements.accepts[0];
    if (!selectedScheme) {
      throw new Error("No payment schemes available");
    }

    // Create resource info from requirements
    const resource: ResourceInfo = requirements.resource;

    // Create signature
    // TODO: Implement actual EIP-712 or EIP-3009 signing via SchemeNetworkClient
    // For simplified mode, we base64 encode the requirements as "signature"
    let signature: string;
    if (this.config.signPayment) {
      signature = await this.config.signPayment(requirements);
    } else {
      // Simplified mode: base64 encode the requirements as "signature"
      signature = safeBase64Encode(JSON.stringify(requirements));
    }

    // Build the accepted requirements (PaymentRequirements type from @x402/core)
    const accepted: PaymentRequirements = {
      scheme: selectedScheme.scheme,
      network: selectedScheme.network,
      asset: selectedScheme.asset,
      amount: selectedScheme.amount,
      payTo: selectedScheme.payTo,
      maxTimeoutSeconds: selectedScheme.maxTimeoutSeconds,
      extra: selectedScheme.extra ?? {},
    };

    // Create the payment payload using official types
    const payload: PaymentPayload = {
      x402Version: X402_VERSION,
      resource,
      accepted,
      payload: {
        signature,
        // Authorization is embedded in the signature for simplified mode
        // Real implementations would have separate authorization object
        authorization: {
          from: this.config.walletAddress,
          to: selectedScheme.payTo,
          value: selectedScheme.amount,
          validAfter: Math.floor(Date.now() / 1000).toString(),
          validBefore: (Math.floor(Date.now() / 1000) + selectedScheme.maxTimeoutSeconds).toString(),
          nonce: `0x${Math.random().toString(16).substring(2)}`,
        },
      },
    };

    return payload;
  }

  /**
   * Encode payment payload for MCP _meta field using base64
   */
  encodePaymentForMeta(paymentPayload: PaymentPayload): string {
    return safeBase64Encode(JSON.stringify(paymentPayload));
  }

  /**
   * Create the _meta object with payment payload for MCP requests
   */
  createPaymentMeta(paymentPayload: PaymentPayload): Record<string, unknown> {
    return {
      [META_KEY_PAYMENT]: paymentPayload,
    };
  }

  /**
   * Get the wallet address
   */
  getWalletAddress(): string {
    return this.config.walletAddress;
  }
}

/**
 * Create an X402 handler with default configuration
 */
export function createX402Handler(walletAddress: string): X402Handler {
  return new X402Handler({ walletAddress });
}
