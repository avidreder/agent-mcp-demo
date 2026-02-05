# MCP Utilities

This package exposes MCP tools over a JSON-RPC POST endpoint.

## List tools

```bash
curl -sS -X POST http://localhost:8080/discovery/mcp \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }' | jq .
```

## Search resources (first page)

```bash
curl -sS -X POST http://localhost:8080/discovery/mcp \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "search_resources",
      "arguments": {
        "limit": 10,
        "offset": 0
      }
    }
  }' | jq .
```

## Notes

- JSON-RPC notifications (requests without an `id`) return `204 No Content`.

## Example responses

### tools/list

Example request.

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list"
}
```

Example response.

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "search_resources",
        "description": "Discover additional x402 tools you can use. Use searchQuery to filter by text. After discovery, execute a returned tool via proxy_tool_call with a payment attached in meta x402/payment.",
        "inputSchema": {
          "type": "object",
          "properties": {
            "searchQuery": { "type": "string" },
            "limit": { "type": "integer" },
            "offset": { "type": "integer" }
          },
          "additionalProperties": false
        },
        "outputSchema": {
          "type": "object",
          "properties": {
            "pagination": {
              "type": "object",
              "properties": {
                "limit": { "type": "integer" },
                "offset": { "type": "integer" },
                "total": { "type": "integer" }
              },
              "additionalProperties": false
            },
            "x402Version": { "type": "integer" },
            "tools": {
              "type": "array",
              "items": {
                "type": "object",
                "properties": {
                  "_meta": { "type": "object", "additionalProperties": true },
                  "name": { "type": "string" },
                  "description": { "type": "string" },
                  "inputSchema": { "type": "object", "additionalProperties": true }
                },
                "additionalProperties": false
              }
            }
          },
          "additionalProperties": false
        }
      },
      {
        "name": "proxy_tool_call",
        "description": "Proxies a tool call to an HTTP x402 resource by tool name and parameters.",
        "inputSchema": {
          "type": "object",
          "properties": {
            "toolName": { "type": "string", "description": "Tool name to proxy" },
            "parameters": { "type": "object", "description": "Tool parameters for the proxied call" }
          },
          "required": ["toolName"]
        }
      }
    ]
  }
}
```

### tools/call (search_resources)

Example request to find resources to get weather information

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "search_resources",
    "arguments": {
      "searchQuery": "weather",
      "limit": 10,
      "offset": 0
    }
  }
}
```

Example response.

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [],
    "structuredContent": {
      "pagination": {
        "limit": 10,
        "offset": 0,
        "total": 1
      },
      "x402Version": 1,
      "tools": [
        {
          "name": "x402_get_http___localhost_8080_weather_9a0e7f76",
          "description": "Get synthetic weather data for a city Use proxy_tool_call with payment to execute.",
          "inputSchema": {
            "type": "object",
            "properties": {
              "parameters": {
                "type": "object",
                "properties": {
                  "query": {
                    "type": "object",
                    "properties": {
                      "city": { "type": "string" }
                    }
                  }
                }
              }
            }
          },
          "_meta": {
            "x402/payment-required": {
              "x402Version": 1,
              "resource": {
                "url": "mcp://tool/x402_get_http___localhost_8080_weather_9a0e7f76",
                "description": "Get synthetic weather data for a city Use proxy_tool_call with payment to execute.",
                "mimeType": "application/json"
              },
              "accepts": [
                {
                  "scheme": "exact",
                  "network": "base-sepolia",
                  "amount": "10000",
                  "asset": "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
                  "payTo": "0x8D170Db9aB247E7013d024566093E13dc7b0f181",
                  "maxTimeoutSeconds": 300,
                  "extra": {
                    "name": "USDC",
                    "version": "2"
                  }
                }
              ]
            },
            "x402/call-with": {
              "tool": "proxy_tool_call"
            }
          }
        }
      ]
    }
  }
}
```

### tools/call (proxy_tool_call)

Example request and response for `http://localhost:8080/weather?city=San%20Francisco`.

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "proxy_tool_call",
    "arguments": {
      "toolName": "x402_get_http___localhost_8080_weather_9a0e7f76",
      "parameters": {
        "query": {
          "city": "San Francisco"
        }
      }
    },
    "_meta": {
      "x402/payment": {
        "x402Version": 2,
        "resource": {
          "url": "http://localhost:8080/weather",
          "description": "Get synthetic weather data for a city",
          "mimeType": "application/json"
        },
        "accepted": {
          "scheme": "exact",
          "network": "base-sepolia",
          "amount": "10000",
          "asset": "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
          "payTo": "0x8D170Db9aB247E7013d024566093E13dc7b0f181",
          "maxTimeoutSeconds": 60,
          "extra": {
            "name": "USDC",
            "version": "2"
          }
        },
        "payload": {
          "signature": "0xdeadbeef",
          "authorization": {
            "from": "0x857b06519E91e3A54538791bDbb0E22373e36b66",
            "to": "0x209693Bc6afc0C5328bA36FaF03C514EF312287C",
            "value": "10000",
            "validAfter": "1740672089",
            "validBefore": "1740672154",
            "nonce": "0xf3746613c2d920b5fdabc0856f2aeb2d4f88ee6037b8cc5d04a71a4462f13480"
          }
        }
      }
    }
  }
}
```

Example response.

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"status\":200,\"headers\":{\"Content-Type\":[\"application/json\"]},\"body\":\"{\\\"temperature\\\":\\\"71.0\\\"}\"}"
      }
    ],
    "_meta": {
      "x402/payment-response": {
        "success": true,
        "transaction": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
        "network": "eip155:84532",
        "payer": "0x857b06519E91e3A54538791bDbb0E22373e36b66"
      }
    }
  }
}
```

### tools/call (proxy_tool_call) with a payment-required response

If the upstream resource responds with `PAYMENT-REQUIRED`, the MCP result surfaces that in `_meta.x402/payment-required`.

Example request.

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "proxy_tool_call",
    "arguments": {
      "toolName": "x402_get_http___localhost_8080_weather_9a0e7f76",
      "parameters": {
        "query": {
          "city": "San Francisco"
        }
      }
    },
    "_meta": {
      "x402/payment": {
        "x402Version": 2,
        "resource": {
          "url": "http://localhost:8080/weather",
          "description": "Get synthetic weather data for a city",
          "mimeType": "application/json"
        },
        "accepted": {
          "scheme": "exact",
          "network": "base-sepolia",
          "amount": "10000",
          "asset": "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
          "payTo": "0x8D170Db9aB247E7013d024566093E13dc7b0f181",
          "maxTimeoutSeconds": 60,
          "extra": {
            "name": "USDC",
            "version": "2"
          }
        },
        "payload": {
          "signature": "0xdeadbeef",
          "authorization": {
            "from": "0x857b06519E91e3A54538791bDbb0E22373e36b66",
            "to": "0x209693Bc6afc0C5328bA36FaF03C514EF312287C",
            "value": "10000",
            "validAfter": "1740672089",
            "validBefore": "1740672154",
            "nonce": "0xf3746613c2d920b5fdabc0856f2aeb2d4f88ee6037b8cc5d04a71a4462f13480"
          }
        }
      }
    }
  }
}
```

Example response.

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"status\":402,\"headers\":{\"Payment-Required\":[\"<base64-json>\"]},\"body\":\"Payment required\"}"
      }
    ],
    "_meta": {
      "x402/payment-required": {
        "x402Version": 2,
        "resource": {
          "url": "https://example-resource.com/weather",
          "description": "Get weather data for a city",
          "mimeType": "application/json"
        },
        "accepts": [
          {
            "scheme": "exact",
            "network": "eip155:84532",
            "amount": "10000",
            "asset": "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
            "payTo": "0x209693Bc6afc0C5328bA36FaF03C514EF312287C",
            "maxTimeoutSeconds": 60,
            "extra": {
              "name": "USDC",
              "version": "2"
            }
          }
        ]
      }
    }
  }
}
```
