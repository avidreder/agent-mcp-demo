package httpapi

import (
	"context"
	"fmt"
	"log"
	"os"

	x402local "github.com/andrewreder/agent-poc/go-api/x402"
	x402sdk "github.com/coinbase/x402/go"
	"github.com/coinbase/x402/go/extensions/bazaar"
	"github.com/coinbase/x402/go/extensions/types"
	x402http "github.com/coinbase/x402/go/http"
	ginmw "github.com/coinbase/x402/go/http/gin"
	evmexact "github.com/coinbase/x402/go/mechanisms/evm/exact/server"
	solanaexact "github.com/coinbase/x402/go/mechanisms/svm/exact/server"
	"github.com/gin-gonic/gin"
)

// getFacilitatorURL returns the facilitator URL from environment or default.
func getFacilitatorURL() string {
	if url := os.Getenv("FACILITATOR_URL"); url != "" {
		return url
	}
	return "http://localhost:8003/v2/x402"
}

// ConfigurePayments wires x402 payment enforcement for HTTP routes.
func ConfigurePayments(r *gin.Engine, baseURL string) error {
	unpaidJSON := func(message string) x402http.UnpaidResponseBodyFunc {
		return func(ctx context.Context, reqCtx x402http.HTTPRequestContext) (*x402http.UnpaidResponse, error) {
			return &x402http.UnpaidResponse{
				ContentType: "application/json",
				Body: map[string]string{
					"error": message,
					"hint":  "See PAYMENT-REQUIRED header for details",
				},
			}, nil
		}
	}

	discoveryExtension, err := bazaar.DeclareDiscoveryExtension(
		bazaar.MethodGET,
		map[string]interface{}{"city": "San Francisco"}, // Example query params
		types.JSONSchema{
			"properties": map[string]interface{}{
				"city": map[string]interface{}{
					"type":        "string",
					"description": "City name to get weather for",
				},
			},
			"required": []string{"city"},
		},
		"", // No body for GET request
		&types.OutputConfig{
			Example: map[string]interface{}{
				"city":        "San Francisco",
				"temperature": 71.2,
				"conditions":  "Partly cloudy",
				"unit":        "fahrenheit",
			},
			Schema: types.JSONSchema{
				"properties": map[string]interface{}{
					"city":        map[string]interface{}{"type": "string"},
					"temperature": map[string]interface{}{"type": "number"},
					"conditions":  map[string]interface{}{"type": "string"},
					"unit":        map[string]interface{}{"type": "string"},
				},
				"required": []string{"city", "temperature", "conditions", "unit"},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create bazaar extension: %w", err)
	}

	paymentRoutes := x402http.RoutesConfig{
		"GET /weather": {
			Accepts: []x402http.PaymentOption{
				// Base Sepolia USDC
				{
					Scheme: "exact",
					PayTo:  "0x8D170Db9aB247E7013d024566093E13dc7b0f181",
					Price: map[string]interface{}{
						"amount": "1000",                                       // 0.001 USDC (6 decimals)
						"asset":  "0x036CbD53842c5426634e7929541eC2318f3dCF7e", // Base Sepolia USDC
						"extra": map[string]interface{}{
							"name":    "USDC",
							"version": "2",
						},
					},
					Network:           x402sdk.Network("eip155:84532"),
					MaxTimeoutSeconds: 300,
				},
				// Base Sepolia random token
				{
					Scheme: "exact",
					PayTo:  "0x8D170Db9aB247E7013d024566093E13dc7b0f181",
					Price: map[string]interface{}{
						"amount": "1000",                                       // 0.001 USDC (6 decimals)
						"asset":  "0x046CbD53842c5426634e7929541eC2318f3dCF7e", // random token
						"extra": map[string]interface{}{
							"name":    "USDC",
							"version": "2",
						},
					},
					Network:           x402sdk.Network("eip155:84532"),
					MaxTimeoutSeconds: 300,
				},
				// Base mainnet USDC
				{
					Scheme: "exact",
					PayTo:  "0x8D170Db9aB247E7013d024566093E13dc7b0f181",
					Price: map[string]interface{}{
						"amount": "10000",
						"asset":  "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
						"extra": map[string]interface{}{
							"name":    "USDC",
							"version": "2",
						},
					},
					Network:           x402sdk.Network("eip155:8453"),
					MaxTimeoutSeconds: 300,
				},
				// Base mainnet random token
				{
					Scheme: "exact",
					PayTo:  "0x8D170Db9aB247E7013d024566093E13dc7b0f181",
					Price: map[string]interface{}{
						"amount": "10000",
						"asset":  "0x993589fcd6edb6e08f4c7c32d4f71b54bda02913",
						"extra": map[string]interface{}{
							"name":    "USDC",
							"version": "2",
						},
					},
					Network:           x402sdk.Network("eip155:8453"),
					MaxTimeoutSeconds: 300,
				},
				// Solana USDC
				{
					Scheme: "exact",
					PayTo:  "0x8D170Db9aB247E7013d024566093E13dc7b0f181",
					Price: map[string]interface{}{
						"amount": "10000",
						"asset":  "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
						"extra": map[string]interface{}{
							"name":    "USDC",
							"version": "2",
						},
					},
					Network:           x402sdk.Network("solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp"),
					MaxTimeoutSeconds: 300,
				},
				// Solana random token
				{
					Scheme: "exact",
					PayTo:  "0x8D170Db9aB247E7013d024566093E13dc7b0f181",
					Price: map[string]interface{}{
						"amount": "10000",
						"asset":  "FPjFFdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
						"extra": map[string]interface{}{
							"name":    "USDC",
							"version": "2",
						},
					},
					Network:           x402sdk.Network("solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp"),
					MaxTimeoutSeconds: 300,
				},
				// Solana Devnet USDC
				{
					Scheme: "exact",
					PayTo:  "0x8D170Db9aB247E7013d024566093E13dc7b0f181",
					Price: map[string]interface{}{
						"amount": "10000",
						"asset":  "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU",
						"extra": map[string]interface{}{
							"name":    "USDC",
							"version": "2",
						},
					},
					Network:           x402sdk.Network("solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp"),
					MaxTimeoutSeconds: 300,
				},
			},
			Resource:           fmt.Sprintf("%s/weather", baseURL),
			Description:        "Get synthetic weather data for a city",
			MimeType:           "application/json",
			UnpaidResponseBody: unpaidJSON("Payment required to access /weather"),
			Extensions: map[string]interface{}{
				types.BAZAAR: discoveryExtension,
			},
		},
	}

	facilitator := x402http.NewHTTPFacilitatorClient(
		x402local.FacilitatorConfigFromEnv(getFacilitatorURL()),
	)

	r.Use(ginmw.X402Payment(ginmw.Config{
		Routes:      paymentRoutes,
		Facilitator: facilitator,
		Schemes: []ginmw.SchemeConfig{
			{
				Network: x402sdk.Network("eip155:84532"),
				Server:  evmexact.NewExactEvmScheme(),
			},
			{
				Network: x402sdk.Network("eip155:8453"),
				Server:  evmexact.NewExactEvmScheme(),
			},
			{
				Network: x402sdk.Network("solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp"),
				Server:  solanaexact.NewExactSvmScheme(),
			},
			{
				Network: x402sdk.Network("solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp"),
				Server:  solanaexact.NewExactSvmScheme(),
			},
		},
		ErrorHandler: func(c *gin.Context, err error) {
			log.Printf("x402 payment error: %v (method=%s path=%s)", err, c.Request.Method, c.Request.URL.Path)
			paymentSignature := c.Request.Header.Get("PAYMENT-SIGNATURE")
			xPayment := c.Request.Header.Get("X-PAYMENT")
			log.Printf(
				"x402 payment headers present (PAYMENT-SIGNATURE=%t X-PAYMENT=%t)",
				paymentSignature != "",
				xPayment != "",
			)
			if paymentSignature == "" && xPayment != "" {
				log.Printf("x402 v2 expects PAYMENT-SIGNATURE; X-PAYMENT is treated as v1")
			}
		},
		SettlementHandler: func(c *gin.Context, settlement *x402sdk.SettleResponse) {
			log.Printf(
				"x402 payment settled (method=%s path=%s network=%s success=%t)",
				c.Request.Method,
				c.Request.URL.Path,
				settlement.Network,
				settlement.Success,
			)
		},
	}))

	return nil
}
