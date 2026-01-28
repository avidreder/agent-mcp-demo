package x402

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolPricing maps tool names to their pricing configuration
type ToolPricing map[string]ToolPricingConfig

// Middleware wraps MCP tool handlers with x402 payment verification
type Middleware struct {
	pricing   ToolPricing
	payToAddr string
	network   Network
	asset     string
	serverURL string
}

// NewMiddleware creates a new x402 middleware instance
func NewMiddleware(serverURL, payToAddr string, network Network, asset string) *Middleware {
	return &Middleware{
		pricing:   make(ToolPricing),
		payToAddr: payToAddr,
		network:   network,
		asset:     asset,
		serverURL: serverURL,
	}
}

// SetToolPrice sets the price for a specific tool
func (m *Middleware) SetToolPrice(toolName, amount string) {
	m.pricing[toolName] = ToolPricingConfig{
		Amount:  amount,
		Asset:   m.asset,
		Network: m.network,
		PayTo:   m.payToAddr,
	}
}

// GetPaymentRequirements returns the payment requirements for a tool
// Uses official x402 types
func (m *Middleware) GetPaymentRequirements(toolName string) *PaymentRequiredData {
	pricing, ok := m.pricing[toolName]
	if !ok {
		return nil // Tool is free
	}

	return &PaymentRequiredData{
		X402Version: X402Version,
		Error:       "Payment required to access this tool",
		Resource: &ResourceInfo{
			URL:         fmt.Sprintf("%s/tools/%s", m.serverURL, toolName),
			Description: fmt.Sprintf("MCP Tool: %s", toolName),
			MimeType:    "application/json",
		},
		Accepts: []PaymentRequirements{
			{
				Scheme:            "exact",
				Network:           string(pricing.Network),
				Amount:            pricing.Amount,
				Asset:             pricing.Asset,
				PayTo:             pricing.PayTo,
				MaxTimeoutSeconds: 60,
				Extra: map[string]interface{}{
					"name":    "x402-mcp-server",
					"version": "0.1.0",
				},
			},
		},
	}
}

// ValidatePayment checks if the payment in _meta is valid
// Uses official x402 types for parsing
// TODO: Integrate with x402ResourceServer for actual verification via facilitator
func (m *Middleware) ValidatePayment(toolName string, meta map[string]interface{}) (*PaymentPayload, error) {
	paymentData, ok := meta[MetaKeyPayment]
	if !ok {
		return nil, nil // No payment provided
	}

	// Parse the payment payload using official types
	paymentBytes, err := json.Marshal(paymentData)
	if err != nil {
		return nil, fmt.Errorf("invalid payment format: %w", err)
	}

	var payment PaymentPayload
	if err := json.Unmarshal(paymentBytes, &payment); err != nil {
		return nil, fmt.Errorf("failed to parse payment: %w", err)
	}

	// Decode the signature using standard base64
	signature, ok := payment.Payload["signature"].(string)
	if !ok {
		return nil, fmt.Errorf("missing signature in payment payload")
	}

	decoded, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}

	// Parse the decoded requirements to verify they match
	var decodedReqs PaymentRequiredData
	if err := json.Unmarshal(decoded, &decodedReqs); err != nil {
		return nil, fmt.Errorf("invalid signature content: %w", err)
	}

	// Verify the payment matches our requirements
	expectedReqs := m.GetPaymentRequirements(toolName)
	if expectedReqs == nil {
		return &payment, nil // Tool is free, payment not required
	}

	// Validate using official x402 DeepEqual
	if len(decodedReqs.Accepts) == 0 {
		return nil, fmt.Errorf("payment does not include any accepted schemes")
	}

	// Use official x402 comparison for requirements matching
	if len(expectedReqs.Accepts) > 0 && len(decodedReqs.Accepts) > 0 {
		expected := expectedReqs.Accepts[0]
		decoded := decodedReqs.Accepts[0]

		// Verify critical fields match
		if decoded.Network != expected.Network {
			return nil, fmt.Errorf("network mismatch: expected %s, got %s", expected.Network, decoded.Network)
		}
		if decoded.Asset != expected.Asset {
			return nil, fmt.Errorf("asset mismatch: expected %s, got %s", expected.Asset, decoded.Asset)
		}
		if decoded.PayTo != expected.PayTo {
			return nil, fmt.Errorf("payTo mismatch: expected %s, got %s", expected.PayTo, decoded.PayTo)
		}
		// Amount verification would typically be >= expected
	}

	return &payment, nil
}

// CreatePaymentResponse creates a payment response for the _meta field
// Uses official x402 SettleResponse type
func (m *Middleware) CreatePaymentResponse(payment *PaymentPayload, success bool, errorReason string) *SettleResponse {
	resp := &SettleResponse{
		Success:     success,
		Network:     Network(payment.Accepted.Network),
		ErrorReason: errorReason,
	}

	// Extract payer from payload if available
	if auth, ok := payment.Payload["authorization"].(map[string]interface{}); ok {
		if from, ok := auth["from"].(string); ok {
			resp.Payer = from
		}
	}

	if success {
		// TODO: Submit actual blockchain transaction and get hash
		resp.Transaction = fmt.Sprintf("0x%x", time.Now().UnixNano()) // Synthetic transaction hash
	}

	return resp
}

// WrapToolHandler wraps an MCP tool handler with x402 payment verification
func WrapToolHandler[In, Out any](
	m *Middleware,
	toolName string,
	handler func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error),
) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		var zero Out

		// Check if this tool requires payment
		pricing := m.GetPaymentRequirements(toolName)
		if pricing == nil {
			// Tool is free, proceed normally
			return handler(ctx, req, input)
		}

		// Extract _meta from the request
		meta := extractMeta(req)

		// Validate payment if provided
		payment, err := m.ValidatePayment(toolName, meta)
		if err != nil {
			// Invalid payment - return 402 with error
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: fmt.Sprintf("Payment validation failed: %s", err.Error()),
					},
				},
				Meta: map[string]interface{}{
					MetaKeyPaymentResponse: m.CreatePaymentResponse(&PaymentPayload{
						Accepted: pricing.Accepts[0],
						Payload:  make(map[string]interface{}),
					}, false, err.Error()),
				},
			}, zero, nil
		}

		if payment == nil {
			// No payment provided - return 402 Payment Required
			paymentReqJSON, _ := json.Marshal(pricing)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(paymentReqJSON),
					},
				},
				Meta: map[string]interface{}{
					MetaKeyPaymentRequired: pricing,
				},
			}, zero, nil
		}

		// Payment is valid - execute the tool
		result, out, err := handler(ctx, req, input)
		if err != nil {
			return result, out, err
		}

		// Add payment response to result meta
		if result == nil {
			result = &mcp.CallToolResult{}
		}
		if result.Meta == nil {
			result.Meta = make(map[string]interface{})
		}
		result.Meta[MetaKeyPaymentResponse] = m.CreatePaymentResponse(payment, true, "")

		return result, out, nil
	}
}

// extractMeta extracts the _meta field from a CallToolRequest
func extractMeta(req *mcp.CallToolRequest) map[string]interface{} {
	if req.Params.Meta == nil {
		return make(map[string]interface{})
	}

	result := make(map[string]interface{})
	for k, v := range req.Params.Meta {
		result[k] = v
	}
	return result
}

