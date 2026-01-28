export interface MCPTool {
  name: string;
  description?: string;
  inputSchema?: Record<string, unknown>;
}

export interface ToolCallRequest {
  name: string;
  args: Record<string, unknown>;
}

export interface ToolCallResponse {
  name: string;
  content: unknown;
  duration: number;
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
    | "error";
  data: Record<string, unknown>;
}

export type DebugHandler = (event: DebugEvent) => void;

export interface X402MCPClientOptions {
  serverUrl: string;
  debug?: boolean;
  debugHandler?: DebugHandler;
  interceptors?: Interceptors;
}
