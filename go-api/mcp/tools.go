package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTools registers all MCP tools for x402 discovery.
func (s *Server) registerTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "search_resources",
		Title:       "Search x402 Tools",
		Description: "Discover additional x402 tools you can use. Use searchQuery to filter by text. After discovery, execute a returned tool via proxy_tool_call with a payment attached in meta x402/payment.",
		Meta: map[string]any{
			"x402/usage": map[string]any{
				"step": "discover",
				"next": "proxy_tool_call",
			},
		},
		OutputSchema: searchResourcesOutputSchema(),
	}, s.SearchResources)

	// Register proxy_tool_call tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "proxy_tool_call",
		Title:       "Execute x402 Tool",
		Description: "Executes a discovered x402 tool. Provide toolName and parameters. Use search_resources to discover available tools.",
		Meta: map[string]any{
			"x402/usage": map[string]any{
				"step": "execute",
				"via":  "proxy_tool_call",
			},
		},
	}, s.ProxyToolCall)
}

// SearchResourcesParams defines parameters for the search_resources tool.
type SearchResourcesParams struct {
	// SearchQuery free-form search string to filter available resources.
	SearchQuery string `json:"searchQuery,omitempty" jsonschema:"Search string for filtering resources"`
	// Limit optional pagination limit.
	Limit *int `json:"limit,omitempty"       jsonschema:"Optional pagination limit"`
	// Offset optional pagination offset.
	Offset *int `json:"offset,omitempty"      jsonschema:"Optional pagination offset"`
}

// SearchResourcesPagination defines pagination for the search_resources tool output.
type SearchResourcesPagination struct {
	Limit  *int `json:"limit,omitempty"`
	Offset *int `json:"offset,omitempty"`
	Total  *int `json:"total,omitempty"`
}

// SearchResourcesOutput defines the structured output for the search_resources tool.
type SearchResourcesOutput struct {
	Pagination  SearchResourcesPagination `json:"pagination"`
	X402Version int                       `json:"x402Version"`
	Tools       []*mcp.Tool               `json:"tools,omitempty"`
}

// ProxyToolCallParams defines parameters for the proxy_tool_call tool.
type ProxyToolCallParams struct {
	// ToolName is the name of the tool to proxy.
	ToolName string `json:"toolName"             jsonschema:"Tool name to proxy,required"`
	// Parameters is the input for the proxied tool call.
	Parameters map[string]any `json:"parameters,omitempty" jsonschema:"Tool parameters for the proxied call"`
}

// SearchResources returns a static list of resources matching the search query.
// This method is exported for testing purposes.
func (s *Server) SearchResources(
	ctx context.Context,
	req *mcp.CallToolRequest,
	params *SearchResourcesParams,
) (*mcp.CallToolResult, SearchResourcesOutput, error) {
	query := params.SearchQuery
	resources := filterWeatherResources(s.resources)
	filtered := filterDiscoveryResources(resources, query)
	paged, pagination := paginateResources(filtered, params.Limit, params.Offset)
	tools := make([]*mcp.Tool, 0, len(paged))
	for _, resource := range paged {
		if tool := resourceToTool(resource); tool != nil {
			tools = append(tools, tool)
		}
	}
	x402Version := 1
	if len(filtered) > 0 {
		x402Version = filtered[0].X402Version
	}

	return nil, SearchResourcesOutput{
		Pagination:  pagination,
		X402Version: x402Version,
		Tools:       tools,
	}, nil
}

// ProxyToolCall proxies a call to an HTTP x402 resource and returns an MCP response.
func (s *Server) ProxyToolCall(
	ctx context.Context,
	req *mcp.CallToolRequest,
	params *ProxyToolCallParams,
) (*mcp.CallToolResult, any, error) {
	if params.ToolName == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "Error: 'toolName' parameter is required.",
				},
			},
			IsError: true,
		}, nil, nil
	}

	parameters := params.Parameters
	if req != nil && req.Params != nil {
		if meta := req.Params.GetMeta(); meta != nil {
			if payment, ok := meta["x402/payment"]; ok && payment != nil {
				var err error
				parameters, err = injectPaymentSignature(parameters, payment)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							&mcp.TextContent{
								Text: fmt.Sprintf("Error: invalid x402 payment metadata: %v", err),
							},
						},
						IsError: true,
					}, nil, nil
				}
			}
		}
	}

	resource, err := findResourceForToolName(s.resources, params.ToolName)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: err.Error(),
				},
			},
			IsError: true,
		}, nil, nil
	}

	httpReq, err := proxyToolCallToHTTPRequest(ctx, *resource, parameters)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build proxy request: %w", err)
	}

	httpResp, err := defaultHTTPClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("proxy request failed: %w", err)
	}
	defer httpResp.Body.Close()

	result, err := httpResponseToMCPResult(httpResp)
	if err != nil {
		return nil, nil, err
	}
	return result, nil, nil
}

func injectPaymentSignature(params map[string]any, payment any) (map[string]any, error) {
	paymentMap, ok := payment.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("x402/payment metadata must be an object")
	}

	payload, ok := paymentMap["payload"]
	if !ok || payload == nil {
		return nil, fmt.Errorf("x402/payment metadata missing payload")
	}

	header, err := encodePaymentHeader(paymentMap)
	if err != nil {
		return nil, fmt.Errorf("unable to encode x402 payment payload: %w", err)
	}

	if header.Version >= 2 {
		if _, ok := paymentMap["resource"].(map[string]any); !ok {
			return nil, fmt.Errorf("x402/payment metadata missing resource for v2 payment")
		}
		if _, ok := paymentMap["accepted"].(map[string]any); !ok {
			return nil, fmt.Errorf("x402/payment metadata missing accepted for v2 payment")
		}
	}

	if params == nil {
		params = map[string]any{}
	}

	if rawHeaders, ok := params["headers"]; ok {
		headers, ok := rawHeaders.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("headers must be an object to set %s", header.Name)
		}
		if _, exists := headers[header.Name]; !exists {
			headers[header.Name] = header.Value
		}
		params["headers"] = headers
		return params, nil
	}

	params["headers"] = map[string]any{
		header.Name: header.Value,
	}
	return params, nil
}

func normalizeX402Version(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		parsed, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	default:
		return 0, false
	}
}

func searchResourcesOutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pagination": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit":  map[string]any{"type": "integer"},
					"offset": map[string]any{"type": "integer"},
					"total":  map[string]any{"type": "integer"},
				},
				"additionalProperties": false,
			},
			"x402Version": map[string]any{"type": "integer"},
			"tools": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"_meta":       map[string]any{"type": "object", "additionalProperties": true},
						"name":        map[string]any{"type": "string"},
						"description": map[string]any{"type": "string"},
						"inputSchema": map[string]any{"type": "object", "additionalProperties": true},
						"outputSchema": map[string]any{
							"type":                 "object",
							"additionalProperties": true,
						},
						"title": map[string]any{"type": "string"},
						"annotations": map[string]any{
							"type":                 "object",
							"additionalProperties": true,
						},
					},
					"additionalProperties": false,
				},
			},
		},
		"additionalProperties": false,
	}
}

func filterDiscoveryResources(items []X402DiscoveryResource, query string) []X402DiscoveryResource {
	if query == "" {
		return items
	}
	query = strings.ToLower(query)
	filtered := make([]X402DiscoveryResource, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Resource), query) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
