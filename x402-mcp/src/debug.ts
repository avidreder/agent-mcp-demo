import type { DebugEvent, DebugHandler } from "./types.js";

export function createDefaultDebugHandler(): DebugHandler {
  return (event: DebugEvent) => {
    const time = event.timestamp.toISOString();
    const prefix = `[x402-mcp] [${event.requestId.slice(0, 8)}]`;

    switch (event.type) {
      case "connect":
        console.log(`${prefix} ${time} CONNECT → ${event.data.serverUrl}`);
        break;
      case "disconnect":
        console.log(`${prefix} ${time} DISCONNECT ← ${event.data.serverUrl}`);
        break;
      case "list_tools":
        console.log(
          `${prefix} ${time} LIST_TOOLS → ${(event.data.tools as unknown[]).length} tools`
        );
        break;
      case "tool_call_start":
        console.log(
          `${prefix} ${time} CALL → ${event.data.name}`,
          JSON.stringify(event.data.args)
        );
        break;
      case "tool_call_end":
        console.log(
          `${prefix} ${time} RESPONSE ← ${event.data.name} (${event.data.duration}ms)`
        );
        break;
      case "interceptor_before":
        console.log(
          `${prefix} ${time} INTERCEPT_BEFORE [${event.data.index}] ${event.data.name}`
        );
        if (event.data.modified) {
          console.log(`${prefix}   └─ args modified:`, event.data.args);
        }
        break;
      case "interceptor_after":
        console.log(
          `${prefix} ${time} INTERCEPT_AFTER [${event.data.index}] ${event.data.name}`
        );
        if (event.data.modified) {
          console.log(`${prefix}   └─ response modified`);
        }
        break;
      case "error":
        console.error(`${prefix} ${time} ERROR:`, event.data.message);
        break;
    }
  };
}
