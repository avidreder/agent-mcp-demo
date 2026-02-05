package mcp

import "time"

// X402DiscoveryResource represents a discoverable x402 HTTP resource.
type X402DiscoveryResource struct {
	Accepts     *[]X402PaymentRequirements `json:"accepts,omitempty"`
	LastUpdated time.Time                  `json:"lastUpdated"`
	Resource    string                     `json:"resource"`
	Type        string                     `json:"type"`
	X402Version int                        `json:"x402Version"`
	Metadata    *map[string]any            `json:"metadata,omitempty"`
}

// X402PaymentRequirements captures payment requirements for a resource.
type X402PaymentRequirements struct {
	Asset             string         `json:"asset,omitempty"`
	Description       string         `json:"description,omitempty"`
	Extra             map[string]any `json:"extra,omitempty"`
	MaxAmountRequired string         `json:"maxAmountRequired,omitempty"`
	MaxTimeoutSeconds int            `json:"maxTimeoutSeconds,omitempty"`
	MimeType          string         `json:"mimeType,omitempty"`
	Network           string         `json:"network,omitempty"`
	OutputSchema      map[string]any `json:"outputSchema,omitempty"`
	PayTo             string         `json:"payTo,omitempty"`
	Resource          string         `json:"resource,omitempty"`
	Scheme            string         `json:"scheme,omitempty"`
}
