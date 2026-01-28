import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { SSEClientTransport } from "@modelcontextprotocol/sdk/client/sse.js";
import { createDefaultDebugHandler } from "./debug.js";
import type {
  MCPTool,
  ToolCallRequest,
  ToolCallResponse,
  InterceptorContext,
  Interceptors,
  DebugEvent,
  DebugHandler,
  X402MCPClientOptions,
} from "./types.js";

function generateRequestId(): string {
  return Math.random().toString(36).substring(2) + Date.now().toString(36);
}

export class X402MCPClient {
  private client: Client;
  private transport: SSEClientTransport | null = null;
  private serverUrl: string;
  private connected = false;
  private debug: boolean;
  private debugHandler: DebugHandler;
  private interceptors: Interceptors;

  constructor(options: X402MCPClientOptions) {
    this.serverUrl = options.serverUrl;
    this.debug = options.debug ?? false;
    this.debugHandler = options.debugHandler ?? createDefaultDebugHandler();
    this.interceptors = options.interceptors ?? {};

    this.client = new Client(
      { name: "x402-agent", version: "0.1.0" },
      { capabilities: {} }
    );
  }

  private emit(
    type: DebugEvent["type"],
    data: Record<string, unknown>,
    requestId?: string
  ): void {
    if (!this.debug) return;
    this.debugHandler({
      timestamp: new Date(),
      requestId: requestId ?? generateRequestId(),
      type,
      data,
    });
  }

  private createContext(): InterceptorContext {
    return {
      timestamp: new Date(),
      requestId: generateRequestId(),
    };
  }

  async connect(): Promise<void> {
    const ctx = this.createContext();
    try {
      const url = new URL(this.serverUrl);
      this.transport = new SSEClientTransport(url);
      await this.client.connect(this.transport);
      this.connected = true;

      this.emit("connect", { serverUrl: this.serverUrl }, ctx.requestId);

      // Run onConnect interceptors
      if (this.interceptors.onConnect) {
        for (const interceptor of this.interceptors.onConnect) {
          await interceptor(this.serverUrl, ctx);
        }
      }
    } catch (err) {
      const error = err instanceof Error ? err : new Error(String(err));
      this.emit("error", { message: error.message }, ctx.requestId);

      if (this.interceptors.onError) {
        for (const interceptor of this.interceptors.onError) {
          await interceptor(error, ctx);
        }
      }
      throw error;
    }
  }

  async disconnect(): Promise<void> {
    const ctx = this.createContext();
    if (this.transport) {
      await this.transport.close();
      this.connected = false;

      this.emit("disconnect", { serverUrl: this.serverUrl }, ctx.requestId);

      if (this.interceptors.onDisconnect) {
        for (const interceptor of this.interceptors.onDisconnect) {
          await interceptor(this.serverUrl, ctx);
        }
      }
    }
  }

  async listTools(): Promise<MCPTool[]> {
    if (!this.connected) {
      throw new Error("Not connected to MCP server");
    }

    const ctx = this.createContext();
    const result = await this.client.listTools();

    const tools = result.tools.map((tool) => ({
      name: tool.name,
      description: tool.description,
      inputSchema: tool.inputSchema as Record<string, unknown>,
    }));

    this.emit("list_tools", { tools }, ctx.requestId);

    return tools;
  }

  async callTool(
    name: string,
    args: Record<string, unknown> = {}
  ): Promise<ToolCallResponse> {
    if (!this.connected) {
      throw new Error("Not connected to MCP server");
    }

    const ctx = this.createContext();
    let request: ToolCallRequest = { name, args };

    // Run beforeToolCall interceptors
    if (this.interceptors.beforeToolCall) {
      for (let i = 0; i < this.interceptors.beforeToolCall.length; i++) {
        const interceptor = this.interceptors.beforeToolCall[i];
        const originalArgs = JSON.stringify(request.args);
        request = await interceptor(request, ctx);
        const modified = originalArgs !== JSON.stringify(request.args);

        this.emit(
          "interceptor_before",
          {
            index: i,
            name: request.name,
            args: request.args,
            modified,
          },
          ctx.requestId
        );
      }
    }

    this.emit(
      "tool_call_start",
      { name: request.name, args: request.args },
      ctx.requestId
    );

    const startTime = Date.now();

    try {
      const result = await this.client.callTool({
        name: request.name,
        arguments: request.args,
      });

      const duration = Date.now() - startTime;

      let response: ToolCallResponse = {
        name: request.name,
        content: result.content,
        duration,
      };

      this.emit(
        "tool_call_end",
        { name: request.name, duration, content: response.content },
        ctx.requestId
      );

      // Run afterToolCall interceptors
      if (this.interceptors.afterToolCall) {
        for (let i = 0; i < this.interceptors.afterToolCall.length; i++) {
          const interceptor = this.interceptors.afterToolCall[i];
          const originalContent = JSON.stringify(response.content);
          response = await interceptor(request, response, ctx);
          const modified = originalContent !== JSON.stringify(response.content);

          this.emit(
            "interceptor_after",
            {
              index: i,
              name: request.name,
              modified,
            },
            ctx.requestId
          );
        }
      }

      return response;
    } catch (err) {
      const error = err instanceof Error ? err : new Error(String(err));
      this.emit("error", { message: error.message }, ctx.requestId);

      if (this.interceptors.onError) {
        for (const interceptor of this.interceptors.onError) {
          await interceptor(error, ctx);
        }
      }
      throw error;
    }
  }

  isConnected(): boolean {
    return this.connected;
  }

  // Interceptor management
  addBeforeToolCallInterceptor(
    interceptor: Interceptors["beforeToolCall"] extends (infer T)[] | undefined
      ? T
      : never
  ): void {
    if (!this.interceptors.beforeToolCall) {
      this.interceptors.beforeToolCall = [];
    }
    this.interceptors.beforeToolCall.push(interceptor);
  }

  addAfterToolCallInterceptor(
    interceptor: Interceptors["afterToolCall"] extends (infer T)[] | undefined
      ? T
      : never
  ): void {
    if (!this.interceptors.afterToolCall) {
      this.interceptors.afterToolCall = [];
    }
    this.interceptors.afterToolCall.push(interceptor);
  }

  clearInterceptors(): void {
    this.interceptors = {};
  }

  setDebug(enabled: boolean): void {
    this.debug = enabled;
  }

  setDebugHandler(handler: DebugHandler): void {
    this.debugHandler = handler;
  }
}
