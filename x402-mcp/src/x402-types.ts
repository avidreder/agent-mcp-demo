/**
 * X402 Payment Protocol Types for MCP
 * Uses official @x402/core types with MCP-specific extensions
 * Based on: https://github.com/coinbase/x402/blob/main/specs/transports-v2/mcp.md
 */

// Re-export official x402 types
export type {
  PaymentRequired,
  PaymentPayload,
  PaymentRequirements,
  SettleResponse,
  Network,
} from "@x402/core/types";

// Import for internal use
import type {
  PaymentRequired,
  PaymentPayload,
  PaymentRequirements,
  SettleResponse,
} from "@x402/core/types";

// MCP-specific constants (not in official x402 which is HTTP-focused)
export const X402_VERSION = 2;
export const META_KEY_PAYMENT = "x402/payment";
export const META_KEY_PAYMENT_RESPONSE = "x402/payment-response";
export const META_KEY_PAYMENT_REQUIRED = "x402/payment-required";
export const ERROR_CODE_PAYMENT_REQUIRED = 402;

/**
 * ResourceInfo describes the resource requiring payment
 * (Defined here since it's not directly exported from @x402/core/types)
 */
export interface ResourceInfo {
  url: string;
  description: string;
  mimeType: string;
}

/**
 * PaymentScheme represents a supported payment scheme
 * This is the element type for PaymentRequired.accepts array
 * (Alias for PaymentRequirements for backwards compatibility)
 */
export type PaymentScheme = PaymentRequirements;

/**
 * PaymentRequiredData is the MCP error.data payload for 402 responses
 * This extends the official PaymentRequired with MCP-specific error field
 */
export interface PaymentRequiredData {
  x402Version: number;
  error: string;
  resource: ResourceInfo;
  accepts: PaymentRequirements[];
  extensions?: Record<string, unknown>;
}

/**
 * PaymentResponse is the x402/payment-response meta payload (MCP-specific alias)
 * Maps to official SettleResponse
 */
export type PaymentResponse = SettleResponse;

/**
 * X402 error from tool call
 */
export interface X402Error {
  code: number;
  message: string;
  data?: PaymentRequiredData;
}

/**
 * Result of checking if a response requires payment
 */
export interface PaymentRequiredResult {
  required: true;
  requirements: PaymentRequiredData;
}

export interface PaymentNotRequiredResult {
  required: false;
}

export type PaymentCheckResult = PaymentRequiredResult | PaymentNotRequiredResult;

/**
 * Type guard for PaymentPayload
 */
export function isPaymentPayload(obj: unknown): obj is PaymentPayload {
  if (!obj || typeof obj !== "object") {
    return false;
  }
  const data = obj as Record<string, unknown>;
  return (
    typeof data.x402Version === "number" &&
    typeof data.payload === "object" &&
    typeof data.accepted === "object"
  );
}

/**
 * Type guard for PaymentRequired/PaymentRequiredData
 */
export function isPaymentRequired(obj: unknown): obj is PaymentRequiredData {
  if (!obj || typeof obj !== "object") {
    return false;
  }
  const data = obj as Record<string, unknown>;
  return (
    typeof data.x402Version === "number" &&
    Array.isArray(data.accepts) &&
    typeof data.resource === "object"
  );
}

/**
 * Type guard for SettleResponse/PaymentResponse
 */
export function isPaymentResponse(obj: unknown): obj is PaymentResponse {
  if (!obj || typeof obj !== "object") {
    return false;
  }
  const data = obj as Record<string, unknown>;
  return typeof data.success === "boolean";
}
