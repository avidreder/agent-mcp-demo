package mcp

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxProxyResponseBytes = 1 << 20 // 1MB

var defaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

func resourceToTool(resource X402DiscoveryResource) *mcp.Tool {
	if strings.ToLower(resource.Type) != "http" {
		return nil
	}

	description := fmt.Sprintf("Proxy call to %s", resource.Resource)
	acceptsDesc, input := extractAcceptsMetadata(resource)
	if acceptsDesc != "" {
		description = acceptsDesc
	} else if resource.Metadata != nil {
		if rawDesc, ok := (*resource.Metadata)["description"]; ok {
			if desc, ok := rawDesc.(string); ok && desc != "" {
				description = desc
			}
		}
	}

	method := methodFromInput(input)
	if method == "" {
		if metaInput, ok := extractMetadataInput(resource); ok {
			method = methodFromInput(metaInput)
		}
	}

	description = fmt.Sprintf("%s Use proxy_tool_call with payment to execute.", strings.TrimSpace(description))

	toolName := toolNameFromResource(resource.Resource, method)
	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: defaultProxyToolSchema(resource, input),
	}
	if meta := buildPricingMeta(resource, description, toolName); meta != nil {
		tool.Meta = meta
	}
	if tool.Meta == nil {
		tool.Meta = map[string]any{}
	}
	tool.Meta["x402/call-with"] = map[string]any{
		"tool": "proxy_tool_call",
	}
	return tool
}

func toolNameFromResource(resource, method string) string {
	sanitized := sanitizeToolName(resource)
	methodPrefix := ""
	if method != "" {
		methodPrefix = sanitizeToolName(strings.ToLower(method)) + "_"
	}
	hash := sha1.Sum([]byte(method + ":" + resource))
	return fmt.Sprintf("x402_%s%s_%s", methodPrefix, sanitized, hex.EncodeToString(hash[:4]))
}

func sanitizeToolName(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	s := strings.Trim(b.String(), "_")
	if s == "" {
		return "resource"
	}
	return s
}

func defaultProxyToolSchema(resource X402DiscoveryResource, input map[string]any) map[string]any {
	parametersProps := map[string]any{}
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"parameters": map[string]any{
				"type":       "object",
				"properties": parametersProps,
			},
		},
	}

	if method := methodFromInput(input); method != "" {
		schema["description"] = fmt.Sprintf("HTTP %s to %s", strings.ToUpper(method), resource.Resource)
	}

	if input != nil {
		if rawQueryParams, ok := input["queryParams"].(map[string]any); ok && len(rawQueryParams) > 0 {
			queryProps := map[string]any{}
			for key, value := range rawQueryParams {
				prop := map[string]any{
					"type": "string",
				}
				if value != nil {
					prop["description"] = fmt.Sprint(value)
				}
				queryProps[key] = prop
			}
			parametersProps["query"] = map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"description":          "Query parameters to include on the request.",
				"properties":           queryProps,
			}
		}

		if rawHeaders, ok := input["headers"].(map[string]any); ok && len(rawHeaders) > 0 {
			headerProps := map[string]any{}
			for key, value := range rawHeaders {
				prop := map[string]any{
					"type": "string",
				}
				if value != nil {
					prop["description"] = fmt.Sprint(value)
				}
				headerProps[key] = prop
			}
			parametersProps["headers"] = map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"description":          "Additional headers to include on the request.",
				"properties":           headerProps,
			}
		}

		if _, ok := input["body"]; ok {
			parametersProps["body"] = map[string]any{
				"description": "JSON body to include on the request.",
			}
		}
	}

	if len(parametersProps) > 0 {
		schema["required"] = []string{"parameters"}
	}

	return schema
}

func extractMetadataInput(resource X402DiscoveryResource) (map[string]any, bool) {
	if resource.Metadata == nil {
		return nil, false
	}
	rawInput, ok := (*resource.Metadata)["input"]
	if !ok {
		return nil, false
	}
	input, ok := rawInput.(map[string]any)
	return input, ok
}

func extractAcceptsMetadata(resource X402DiscoveryResource) (string, map[string]any) {
	if resource.Accepts == nil {
		return "", nil
	}
	for _, requirement := range *resource.Accepts {
		raw, err := json.Marshal(requirement)
		if err != nil {
			continue
		}
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			continue
		}
		desc, _ := decoded["description"].(string)
		var input map[string]any
		if outputSchema, ok := decoded["outputSchema"].(map[string]any); ok {
			if rawInput, ok := outputSchema["input"].(map[string]any); ok {
				input = rawInput
			}
		}
		if desc != "" || input != nil {
			return desc, input
		}
	}
	return "", nil
}

func methodFromInput(input map[string]any) string {
	if input == nil {
		return ""
	}
	if method, ok := input["method"].(string); ok && method != "" {
		return strings.ToUpper(method)
	}
	return ""
}

func buildPricingMeta(
	resource X402DiscoveryResource,
	description string,
	toolName string,
) map[string]any {
	if resource.Accepts == nil || len(*resource.Accepts) == 0 {
		return nil
	}

	acceptsList := make([]map[string]any, 0, len(*resource.Accepts))
	for _, requirement := range *resource.Accepts {
		payload, err := json.Marshal(requirement)
		if err != nil {
			continue
		}
		var decoded map[string]any
		if err := json.Unmarshal(payload, &decoded); err != nil {
			continue
		}
		accepts := map[string]any{
			"scheme":            decoded["scheme"],
			"network":           decoded["network"],
			"amount":            decoded["maxAmountRequired"],
			"asset":             decoded["asset"],
			"payTo":             decoded["payTo"],
			"maxTimeoutSeconds": decoded["maxTimeoutSeconds"],
			"extra":             decoded["extra"],
		}
		acceptsList = append(acceptsList, accepts)
	}
	if len(acceptsList) == 0 {
		return nil
	}

	resourceMeta := map[string]any{
		"url":         fmt.Sprintf("mcp://tool/%s", toolName),
		"description": description,
	}
	if mimeType := findMimeType(*resource.Accepts); mimeType != "" {
		resourceMeta["mimeType"] = mimeType
	}

	return map[string]any{
		"x402/payment-required": map[string]any{
			"x402Version": int(resource.X402Version),
			"resource":    resourceMeta,
			"accepts":     acceptsList,
		},
	}
}

func findMimeType(accepts []X402PaymentRequirements) string {
	for _, requirement := range accepts {
		payload, err := json.Marshal(requirement)
		if err != nil {
			continue
		}
		var decoded map[string]any
		if err := json.Unmarshal(payload, &decoded); err != nil {
			continue
		}
		if mimeType, ok := decoded["mimeType"].(string); ok && mimeType != "" {
			return mimeType
		}
	}
	return ""
}

func findResourceForToolName(
	items []X402DiscoveryResource,
	toolName string,
) (*X402DiscoveryResource, error) {
	for idx := range items {
		resource := items[idx]
		if resourceToTool(resource) == nil {
			continue
		}
		method := ""
		if _, input := extractAcceptsMetadata(resource); input != nil {
			method = methodFromInput(input)
		}
		if method == "" {
			if metaInput, ok := extractMetadataInput(resource); ok {
				method = methodFromInput(metaInput)
			}
		}
		if toolNameFromResource(resource.Resource, method) == toolName {
			return &resource, nil
		}
	}
	return nil, fmt.Errorf("tool %q not found", toolName)
}

func proxyToolCallToHTTPRequest(
	ctx context.Context,
	resource X402DiscoveryResource,
	params map[string]any,
) (*http.Request, error) {
	method := http.MethodGet
	if _, input := extractAcceptsMetadata(resource); input != nil {
		if derived := methodFromInput(input); derived != "" {
			method = derived
		}
	}
	if method == http.MethodGet {
		if metaInput, ok := extractMetadataInput(resource); ok {
			if derived := methodFromInput(metaInput); derived != "" {
				method = derived
			}
		}
	}

	endpoint, err := url.Parse(resource.Resource)
	if err != nil {
		return nil, fmt.Errorf("invalid resource url: %w", err)
	}

	query := endpoint.Query()
	if params != nil {
		if rawQuery, ok := params["query"].(map[string]any); ok {
			for key, value := range rawQuery {
				query.Set(key, fmt.Sprint(value))
			}
		}
	}
	endpoint.RawQuery = query.Encode()

	var body io.Reader
	if params != nil {
		if rawBody, ok := params["body"]; ok && rawBody != nil {
			payload, err := json.Marshal(rawBody)
			if err != nil {
				return nil, fmt.Errorf("invalid body payload: %w", err)
			}
			body = bytes.NewReader(payload)
			if method == http.MethodGet {
				method = http.MethodPost
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if params != nil {
		if rawHeaders, ok := params["headers"].(map[string]any); ok {
			for key, value := range rawHeaders {
				req.Header.Set(key, fmt.Sprint(value))
			}
		}
	}

	return req, nil
}

func httpResponseToMCPResult(resp *http.Response) (*mcp.CallToolResult, error) {
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxProxyResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read proxy response: %w", err)
	}

	if paymentRequired := decodePaymentRequired(resp, bodyBytes); paymentRequired != nil {
		contentJSON, err := json.Marshal(paymentRequired)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payment-required payload: %w", err)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: string(contentJSON),
				},
			},
			StructuredContent: paymentRequired,
			IsError:           true,
		}, nil
	}

	payload := map[string]any{
		"status":  resp.StatusCode,
		"headers": resp.Header,
		"body":    string(bodyBytes),
	}

	contentJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proxy response: %w", err)
	}

	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(contentJSON),
			},
		},
		IsError: resp.StatusCode >= http.StatusBadRequest,
	}

	if paymentResponse := decodePaymentResponse(resp); paymentResponse != nil {
		result.Meta = map[string]any{
			"x402/payment-response": paymentResponse,
		}
	}

	return result, nil
}

func decodePaymentHeader(raw string) map[string]any {
	if raw == "" {
		return nil
	}
	payload, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		payload, err = base64.RawStdEncoding.DecodeString(raw)
		if err != nil {
			return nil
		}
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil
	}
	return decoded
}
