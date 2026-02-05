/**
 * X402 Payment Handler for MCP Client
 * Uses official @x402/core and @x402/evm for real EVM payment signing
 * Detects 402 payment requirements and handles payment flow
 */

import { x402Client } from "@x402/core/client";
import type {
  PaymentPayload,
  PaymentRequirements,
} from "@x402/core/types";
import { ExactEvmScheme, type ClientEvmSigner } from "@x402/evm";
import type {
  ResourceInfo,
  PaymentRequiredData,
  PaymentResponse,
  PaymentCheckResult,
} from "./x402-types.js";
import {
  META_KEY_PAYMENT,
  META_KEY_PAYMENT_RESPONSE,
  isPaymentRequired,
  isPaymentResponse,
} from "./x402-types.js";

/**
 * Configuration for the X402 payment handler
 */
export interface X402HandlerConfig {
  /** EVM signer for payment authorization (viem account) */
  signer: ClientEvmSigner;

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
 * Uses official @x402/evm ExactEvmScheme for real EIP-712 signing
 */
export class X402Handler {
  private config: X402HandlerConfig;
  private x402Client: x402Client;
  private evmScheme: ExactEvmScheme;

  constructor(config: X402HandlerConfig) {
    this.config = config;

    // Create the EVM scheme with the signer
    this.evmScheme = new ExactEvmScheme(config.signer);

    // Create x402 client and register the EVM scheme for EVM and legacy networks
    this.x402Client = new x402Client();
    this.x402Client.register("eip155:*" as `${string}:${string}`, this.evmScheme);
    this.x402Client.register("base-sepolia" as `${string}:${string}`, this.evmScheme);
    this.x402Client.register("base" as `${string}:${string}`, this.evmScheme);
  }

  /**
   * Check if a tool response indicates payment is required
   */
  checkPaymentRequired(result: {
    isError?: boolean;
    structuredContent?: unknown;
    content?: unknown;
  }): PaymentCheckResult {
    if (!result.isError) {
      return { required: false };
    }

    if (isPaymentRequired(result.structuredContent)) {
      return { required: true, requirements: result.structuredContent };
    }

    const content = result.content;
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
   * Create a payment payload using official x402 EVM signing
   * Uses EIP-712 typed data signing for real blockchain-verifiable payments
   */
  async createPaymentPayload(requirements: PaymentRequiredData): Promise<PaymentPayload> {
    // Convert PaymentRequiredData to PaymentRequired format expected by x402Client
    const paymentRequired = {
      x402Version: requirements.x402Version,
      error: requirements.error,
      resource: requirements.resource,
      accepts: requirements.accepts,
      extensions: requirements.extensions,
    };

    // Use x402Client to create the payment payload
    // This handles scheme selection and calls the ExactEvmScheme for EIP-712 signing
    const payload = await this.x402Client.createPaymentPayload(paymentRequired);

    return payload;
  }

  /**
   * Create the _meta object with payment payload for MCP requests
   */
  createPaymentMeta(paymentPayload: PaymentPayload): Record<string, unknown> {
    return {
      [META_KEY_PAYMENT]: {
        x402Version: paymentPayload.x402Version,
        resource: paymentPayload.resource,
        accepted: paymentPayload.accepted,
        payload: paymentPayload.payload,
      },
    };
  }

  /**
   * Get the wallet address
   */
  getWalletAddress(): string {
    return this.config.signer.address;
  }
}

/**
 * Create an X402 handler with an EVM signer
 */
export function createX402Handler(signer: ClientEvmSigner): X402Handler {
  return new X402Handler({ signer });
}
