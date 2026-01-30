package x402

import (
	"context"
	"encoding/json"
	"fmt"

	x402http "github.com/coinbase/x402/go/http"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolPricing maps tool names to their pricing configuration
type ToolPricing map[string]ToolPricingConfig

// Middleware wraps MCP tool handlers with x402 payment verification
type Middleware struct {
	pricing          ToolPricing
	payToAddr        string
	network          Network
	asset            string
	serverURL        string
	facilitatorURL   string
	facilitator      *x402http.HTTPFacilitatorClient
}

// NewMiddleware creates a new x402 middleware instance
func NewMiddleware(serverURL, payToAddr string, network Network, asset, facilitatorURL string) *Middleware {
	// Create facilitator client
	facilitator := x402http.NewHTTPFacilitatorClient(&x402http.FacilitatorConfig{
		URL: facilitatorURL,
	})

	return &Middleware{
		pricing:        make(ToolPricing),
		payToAddr:      payToAddr,
		network:        network,
		asset:          asset,
		serverURL:      serverURL,
		facilitatorURL: facilitatorURL,
		facilitator:    facilitator,
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
					"name":    "USDC",
					"version": "2",
				},
			},
		},
	}
}

// VerifyPayment validates a payment using the facilitator
func (m *Middleware) VerifyPayment(ctx context.Context, toolName string, meta map[string]interface{}) (*PaymentPayload, error) {
	paymentData, ok := meta[MetaKeyPayment]
	if !ok {
		return nil, nil // No payment provided
	}

	// Parse the payment payload
	paymentBytes, err := json.Marshal(paymentData)
	if err != nil {
		return nil, fmt.Errorf("invalid payment format: %w", err)
	}

	var payment PaymentPayload
	if err := json.Unmarshal(paymentBytes, &payment); err != nil {
		return nil, fmt.Errorf("failed to parse payment: %w", err)
	}

	// Get expected requirements
	expectedReqs := m.GetPaymentRequirements(toolName)
	if expectedReqs == nil {
		return &payment, nil // Tool is free, payment not required
	}

	// Marshal requirements for facilitator
	requirementsBytes, err := json.Marshal(expectedReqs.Accepts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal requirements: %w", err)
	}

	// Verify payment using facilitator
	verifyResp, err := m.facilitator.Verify(ctx, paymentBytes, requirementsBytes)
	if err != nil {
		return nil, fmt.Errorf("payment verification failed: %w", err)
	}

	if !verifyResp.IsValid {
		return nil, fmt.Errorf("payment invalid: %s", verifyResp.InvalidReason)
	}

	return &payment, nil
}

// SettlePayment settles a payment using the facilitator
func (m *Middleware) SettlePayment(ctx context.Context, payment *PaymentPayload, requirements *PaymentRequirements) (*SettleResponse, error) {
	// Marshal payment and requirements
	payloadBytes, err := json.Marshal(payment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment: %w", err)
	}

	requirementsBytes, err := json.Marshal(requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal requirements: %w", err)
	}

	// Settle payment using facilitator
	settleResp, err := m.facilitator.Settle(ctx, payloadBytes, requirementsBytes)
	if err != nil {
		return nil, fmt.Errorf("payment settlement failed: %w", err)
	}

	return settleResp, nil
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

		// Verify payment using facilitator
		payment, err := m.VerifyPayment(ctx, toolName, meta)
		if err != nil {
			// Invalid payment - return 402 with error
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: fmt.Sprintf("Payment verification failed: %s", err.Error()),
					},
				},
				Meta: map[string]interface{}{
					MetaKeyPaymentResponse: &SettleResponse{
						Success:     false,
						Network:     m.network,
						ErrorReason: err.Error(),
					},
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

		// Payment verified - settle it
		settleResp, err := m.SettlePayment(ctx, payment, &pricing.Accepts[0])
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: fmt.Sprintf("Payment settlement failed: %s", err.Error()),
					},
				},
				Meta: map[string]interface{}{
					MetaKeyPaymentResponse: &SettleResponse{
						Success:     false,
						Network:     m.network,
						ErrorReason: err.Error(),
					},
				},
			}, zero, nil
		}

		if !settleResp.Success {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: fmt.Sprintf("Payment settlement failed: %s", settleResp.ErrorReason),
					},
				},
				Meta: map[string]interface{}{
					MetaKeyPaymentResponse: settleResp,
				},
			}, zero, nil
		}

		// Payment settled - execute the tool
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
		result.Meta[MetaKeyPaymentResponse] = settleResp

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
