package x402

// X402 Payment Protocol Types for MCP
// Uses official github.com/coinbase/x402/go types with MCP-specific extensions
// Based on: https://github.com/coinbase/x402/blob/main/specs/transports-v2/mcp.md

import (
	x402sdk "github.com/coinbase/x402/go"
	"github.com/coinbase/x402/go/types"
)

// MCP-specific constants (not in official x402 which is HTTP-focused)
const (
	X402Version            = 2
	MetaKeyPayment         = "x402/payment"
	MetaKeyPaymentResponse = "x402/payment-response"
	MetaKeyPaymentRequired = "x402/payment-required"
	ErrorCodePaymentRequired = 402
)

// Re-export official types for convenience
type (
	// PaymentRequirements is the official x402 payment requirements type
	PaymentRequirements = types.PaymentRequirements

	// PaymentPayload is the official x402 payment payload type
	PaymentPayload = types.PaymentPayload

	// PaymentRequired is the official x402 payment required response type
	PaymentRequired = types.PaymentRequired

	// ResourceInfo describes the resource requiring payment
	ResourceInfo = types.ResourceInfo

	// VerifyResponse is the official x402 verify response type
	VerifyResponse = x402sdk.VerifyResponse

	// SettleResponse is the official x402 settle response type
	SettleResponse = x402sdk.SettleResponse

	// Network is the official x402 network type (CAIP-2 format)
	Network = x402sdk.Network
)

// PaymentRequiredData extends PaymentRequired with MCP-specific error field
// This is the error.data payload for 402 responses in MCP
type PaymentRequiredData struct {
	X402Version int                   `json:"x402Version"`
	Error       string                `json:"error"`
	Resource    *ResourceInfo         `json:"resource"`
	Accepts     []PaymentRequirements `json:"accepts"`
	Extensions  map[string]interface{} `json:"extensions,omitempty"`
}

// ToPaymentRequired converts PaymentRequiredData to official PaymentRequired
func (p *PaymentRequiredData) ToPaymentRequired() *PaymentRequired {
	return &PaymentRequired{
		X402Version: p.X402Version,
		Error:       p.Error,
		Resource:    p.Resource,
		Accepts:     p.Accepts,
		Extensions:  p.Extensions,
	}
}

// FromPaymentRequired creates PaymentRequiredData from official PaymentRequired
func FromPaymentRequired(pr *PaymentRequired, errorMsg string) *PaymentRequiredData {
	return &PaymentRequiredData{
		X402Version: pr.X402Version,
		Error:       errorMsg,
		Resource:    pr.Resource,
		Accepts:     pr.Accepts,
		Extensions:  pr.Extensions,
	}
}

// PaymentResponse is the x402/payment-response meta payload
// This is an alias for SettleResponse for MCP compatibility
type PaymentResponse = SettleResponse

// ToolPricingConfig defines pricing for a tool
type ToolPricingConfig struct {
	Amount  string  // Amount in smallest unit (e.g., wei, satoshi)
	Asset   string  // Asset contract address or identifier
	Network Network // Network identifier (e.g., "eip155:84532")
	PayTo   string  // Recipient address
}
