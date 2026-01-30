import type { ClientEvmSigner } from "@x402/evm";

export interface MCPTool {
  name: string;
  description?: string;
  inputSchema?: Record<string, unknown>;
}

export interface ToolCallRequest {
  name: string;
  args: Record<string, unknown>;
  /** Optional _meta field for x402 payment payloads */
  meta?: Record<string, unknown>;
}

export interface ToolCallResponse {
  name: string;
  content: unknown;
  duration: number;
  /** Whether payment was required for this call */
  paymentRequired?: boolean;
  /** Payment response if payment was made */
  paymentResponse?: import("./x402-types.js").PaymentResponse;
}

export interface InterceptorContext {
  timestamp: Date;
  requestId: string;
}

export type BeforeToolCallInterceptor = (
  request: ToolCallRequest,
  context: InterceptorContext
) => ToolCallRequest | Promise<ToolCallRequest>;

export type AfterToolCallInterceptor = (
  request: ToolCallRequest,
  response: ToolCallResponse,
  context: InterceptorContext
) => ToolCallResponse | Promise<ToolCallResponse>;

export type OnConnectInterceptor = (
  serverUrl: string,
  context: InterceptorContext
) => void | Promise<void>;

export type OnDisconnectInterceptor = (
  serverUrl: string,
  context: InterceptorContext
) => void | Promise<void>;

export type OnErrorInterceptor = (
  error: Error,
  context: InterceptorContext
) => void | Promise<void>;

export interface Interceptors {
  beforeToolCall?: BeforeToolCallInterceptor[];
  afterToolCall?: AfterToolCallInterceptor[];
  onConnect?: OnConnectInterceptor[];
  onDisconnect?: OnDisconnectInterceptor[];
  onError?: OnErrorInterceptor[];
}

export interface DebugEvent {
  timestamp: Date;
  requestId: string;
  type:
    | "connect"
    | "disconnect"
    | "list_tools"
    | "tool_call_start"
    | "tool_call_end"
    | "interceptor_before"
    | "interceptor_after"
    | "error"
    | "x402_payment_required"
    | "x402_payment_creating"
    | "x402_payment_sending"
    | "x402_payment_success"
    | "x402_payment_failed";
  data: Record<string, unknown>;
}

export type DebugHandler = (event: DebugEvent) => void;

export type MCPTransport = "sse" | "streamable-http";

export interface X402MCPClientOptions {
  serverUrl: string;
  /** Transport to use when connecting to MCP server */
  transport?: MCPTransport;
  debug?: boolean;
  debugHandler?: DebugHandler;
  interceptors?: Interceptors;
  /** EVM signer for x402 payments (viem account) */
  signer?: ClientEvmSigner;
  /** Enable automatic payment retry when 402 is received */
  autoPayment?: boolean;
}

// Re-export ClientEvmSigner for convenience
export type { ClientEvmSigner };
