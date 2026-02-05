package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	x402mcp "github.com/andrewreder/agent-poc/go-api/x402"
	x402sdk "github.com/coinbase/x402/go"
	"github.com/coinbase/x402/go/extensions/bazaar"
	"github.com/coinbase/x402/go/extensions/types"
	x402http "github.com/coinbase/x402/go/http"
	ginmw "github.com/coinbase/x402/go/http/gin"
	evmexact "github.com/coinbase/x402/go/mechanisms/evm/exact/server"
	solanaexact "github.com/coinbase/x402/go/mechanisms/svm/exact/server"
	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverBaseURL = "http://localhost:8080"

// Resource represents a discoverable resource
type Resource struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URI         string `json:"uri"`
}

// Static resource data
var resources = []Resource{
	{
		ID:          "1",
		Name:        "Weather API",
		Description: "Real-time weather data for any location",
		URI:         "https://api.weather.example/v1",
	},
	{
		ID:          "2",
		Name:        "Stock Quotes",
		Description: "Live stock market quotes and historical data",
		URI:         "https://api.stocks.example/v1",
	},
	{
		ID:          "3",
		Name:        "News Feed",
		Description: "Aggregated news from multiple sources",
		URI:         "https://api.news.example/v1",
	},
}

type X402OutputSchema struct {
	Input X402InputSchema `json:"input"`
}

type X402InputSchema struct {
	Method      string            `json:"method"`
	QueryParams map[string]string `json:"queryParams,omitempty"`
	Body        map[string]string `json:"body,omitempty"`
	Type        string            `json:"type"`
}

type X402AcceptRequirement struct {
	Asset             string            `json:"asset"`
	Description       string            `json:"description"`
	Extra             map[string]string `json:"extra"`
	MaxAmountRequired string            `json:"maxAmountRequired"`
	MaxTimeoutSeconds int               `json:"maxTimeoutSeconds"`
	MimeType          string            `json:"mimeType"`
	Network           string            `json:"network"`
	OutputSchema      X402OutputSchema  `json:"outputSchema"`
	PayTo             string            `json:"payTo"`
	Resource          string            `json:"resource"`
	Scheme            string            `json:"scheme"`
}

type X402EndpointEntry struct {
	Accepts     []X402AcceptRequirement `json:"accepts"`
	LastUpdated string                  `json:"lastUpdated"`
	Resource    string                  `json:"resource"`
	Type        string                  `json:"type"`
	X402Version int                     `json:"x402Version"`
}

// Tool: Get weather for a location
type GetWeatherInput struct {
	Location string `json:"location" jsonschema:"the city or location to get weather for"`
}

type GetWeatherOutput struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`
	Conditions  string  `json:"conditions"`
	Unit        string  `json:"unit"`
}

func getWeather(ctx context.Context, req *mcp.CallToolRequest, input GetWeatherInput) (*mcp.CallToolResult, GetWeatherOutput, error) {
	// Simulated weather data
	return nil, GetWeatherOutput{
		Location:    input.Location,
		Temperature: 72.5,
		Conditions:  "Partly cloudy",
		Unit:        "fahrenheit",
	}, nil
}

// Tool: Get stock quote
type GetStockQuoteInput struct {
	Symbol string `json:"symbol" jsonschema:"the stock ticker symbol (e.g. AAPL, GOOGL)"`
}

type GetStockQuoteOutput struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
	Change float64 `json:"change"`
}

func getStockQuote(ctx context.Context, req *mcp.CallToolRequest, input GetStockQuoteInput) (*mcp.CallToolResult, GetStockQuoteOutput, error) {
	// Simulated stock data
	return nil, GetStockQuoteOutput{
		Symbol: input.Symbol,
		Price:  185.42,
		Change: 2.35,
	}, nil
}

// Tool: Get news headlines
type GetNewsInput struct {
	Topic string `json:"topic" jsonschema:"the topic to get news about (e.g. technology, finance, sports)"`
}

type GetNewsOutput struct {
	Topic     string   `json:"topic"`
	Headlines []string `json:"headlines"`
}

func getNews(ctx context.Context, req *mcp.CallToolRequest, input GetNewsInput) (*mcp.CallToolResult, GetNewsOutput, error) {
	// Simulated news data
	return nil, GetNewsOutput{
		Topic: input.Topic,
		Headlines: []string{
			fmt.Sprintf("Breaking: Major developments in %s sector", input.Topic),
			fmt.Sprintf("Analysis: What's next for %s industry", input.Topic),
			fmt.Sprintf("Expert opinion: %s trends to watch", input.Topic),
		},
	}, nil
}

type WeatherResponse struct {
	City        string  `json:"city"`
	Temperature float64 `json:"temperature"`
	Conditions  string  `json:"conditions"`
	Unit        string  `json:"unit"`
}

type RestaurantRequest struct {
	City string `json:"city"`
	Food string `json:"food"`
}

type RestaurantResponse struct {
	City        string   `json:"city"`
	Food        string   `json:"food"`
	Restaurants []string `json:"restaurants"`
	Note        string   `json:"note"`
}

// Resource handler for MCP resources
func resourceHandler(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := req.Params.URI

	// Find the matching resource
	for _, r := range resources {
		if r.URI == uri {
			data, _ := json.Marshal(r)
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      uri,
						MIMEType: "application/json",
						Text:     string(data),
					},
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("resource not found: %s", uri)
}

// getFacilitatorURL returns the facilitator URL from environment or default
func getFacilitatorURL() string {
	if url := os.Getenv("FACILITATOR_URL"); url != "" {
		return url
	}
	return "http://localhost:8003/v2/x402"
}

// x402 middleware instance configured for Base Sepolia with real facilitator
// Price: 0.01 USDC per tool call
var x402Middleware = x402mcp.NewMiddleware(
	serverBaseURL, // Server URL
	"0x8D170Db9aB247E7013d024566093E13dc7b0f181", // Payee address
	"eip155:84532", // Base Sepolia testnet
	"0x036CbD53842c5426634e7929541eC2318f3dCF7e", // USDC on Base Sepolia
	getFacilitatorURL(),                          // Facilitator URL
)

func init() {
	// Configure tool pricing - 0.01 USDC = 10000 (6 decimals)
	x402Middleware.SetToolPrice("get_weather", "10000")     // 0.01 USDC
	x402Middleware.SetToolPrice("get_stock_quote", "10000") // 0.01 USDC
	x402Middleware.SetToolPrice("get_news", "10000")        // 0.01 USDC
}

func createMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "x402-discovery",
		Version: "0.1.0",
	}, nil)

	// Register MCP resources (discovered via listResources)
	for _, r := range resources {
		server.AddResource(&mcp.Resource{
			URI:         r.URI,
			Name:        r.Name,
			Description: r.Description,
			MIMEType:    "application/json",
		}, resourceHandler)
	}

	// Register tools with x402 payment middleware
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_weather",
		Description: "Get current weather for a location",
	}, x402mcp.WrapToolHandler(x402Middleware, "get_weather", getWeather))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_stock_quote",
		Description: "Get the current stock price for a ticker symbol",
	}, x402mcp.WrapToolHandler(x402Middleware, "get_stock_quote", getStockQuote))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_news",
		Description: "Get latest news headlines for a topic",
	}, x402mcp.WrapToolHandler(x402Middleware, "get_news", getNews))

	return server
}

func main() {
	r := gin.Default()

	// Debug: log payment headers for protected endpoints (toy repo)
	r.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/weather") || strings.HasPrefix(c.Request.URL.Path, "/restaurants") {
			log.Printf("Headers: %+v", c.Request.Header)
			// log.Printf("x402 request headers (method=%s path=%s): %+v", c.Request.Method, c.Request.URL.Path, c.Request.Header)
		}
		c.Next()
	})

	// Protect HTTP endpoints with x402 payment middleware
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
		fmt.Printf("❌ Failed to create bazaar extension: %v\n", err)
		os.Exit(1)
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
			Resource:           fmt.Sprintf("%s/weather", serverBaseURL),
			Description:        "Get synthetic weather data for a city",
			MimeType:           "application/json",
			UnpaidResponseBody: unpaidJSON("Payment required to access /weather"),
			Extensions: map[string]interface{}{
				types.BAZAAR: discoveryExtension,
			},
		},
		// "POST /restaurants": {
		// 	Accepts: []x402http.PaymentOption{
		// 		{
		// 			Scheme: "exact",
		// 			PayTo:  "0x8D170Db9aB247E7013d024566093E13dc7b0f181",
		// 			Price: map[string]interface{}{
		// 				"amount": "10000",
		// 				"asset":  "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
		// 				"extra": map[string]interface{}{
		// 					"name":    "USDC",
		// 					"version": "2",
		// 				},
		// 			},
		// 			Network:           x402sdk.Network("eip155:84532"),
		// 			MaxTimeoutSeconds: 300,
		// 		},
		// 	},
		// 	Resource:           fmt.Sprintf("%s/restaurants", serverBaseURL),
		// 	Description:        "Get synthetic restaurant suggestions by city and food",
		// 	MimeType:           "application/json",
		// 	UnpaidResponseBody: unpaidJSON("Payment required to access /restaurants"),
		// 	Extensions: map[string]interface{}{
		// 		types.BAZAAR: discoveryExtension,
		// 	},
		// },
	}

	facilitator := x402http.NewHTTPFacilitatorClient(&x402http.FacilitatorConfig{
		URL: getFacilitatorURL(),
	})

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

	// REST API endpoint
	// GET /discovery/resources - Returns list of available resources
	r.GET("/discovery/resources", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"resources": resources,
		})
	})

	// GET /discovery/x402 - Returns x402 entries for available HTTP endpoints
	r.GET("/discovery/x402", func(c *gin.Context) {
		lastUpdated := time.Now().UTC().Format(time.RFC3339Nano)
		entries := []X402EndpointEntry{
			{
				Accepts: []X402AcceptRequirement{
					{
						Asset:       "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
						Description: "Get synthetic weather data for a city",
						Extra: map[string]string{
							"name":    "USDC",
							"version": "2",
						},
						MaxAmountRequired: "10000",
						MaxTimeoutSeconds: 300,
						MimeType:          "application/json",
						Network:           "base-sepolia",
						OutputSchema: X402OutputSchema{
							Input: X402InputSchema{
								Method: "GET",
								QueryParams: map[string]string{
									"city": "string",
								},
								Type: "http",
							},
						},
						PayTo:    "0x8D170Db9aB247E7013d024566093E13dc7b0f181",
						Resource: fmt.Sprintf("%s/weather", serverBaseURL),
						Scheme:   "exact",
					},
				},
				LastUpdated: lastUpdated,
				Resource:    fmt.Sprintf("%s/weather", serverBaseURL),
				Type:        "http",
				X402Version: 1,
			},
		}

		c.JSON(http.StatusOK, gin.H{
			"entries": entries,
		})
	})

	// GET /weather?city=CityName - Returns synthetic weather data
	r.GET("/weather", func(c *gin.Context) {
		city := c.Query("city")
		if city == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "city query param is required",
			})
			return
		}

		c.JSON(http.StatusOK, WeatherResponse{
			City:        city,
			Temperature: 71.2,
			Conditions:  "Partly cloudy",
			Unit:        "fahrenheit",
		})
	})

	// POST /restaurants - Body: { "city": "...", "food": "..." }
	// Returns synthetic restaurant suggestions
	// r.POST("/restaurants", func(c *gin.Context) {
	// 	var req RestaurantRequest
	// 	if err := c.ShouldBindJSON(&req); err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{
	// 			"error": "invalid JSON body",
	// 		})
	// 		return
	// 	}
	// 	if req.City == "" || req.Food == "" {
	// 		c.JSON(http.StatusBadRequest, gin.H{
	// 			"error": "city and food are required",
	// 		})
	// 		return
	// 	}
	//
	// 	c.JSON(http.StatusOK, RestaurantResponse{
	// 		City: req.City,
	// 		Food: req.Food,
	// 		Restaurants: []string{
	// 			fmt.Sprintf("%s %s House", req.City, req.Food),
	// 			fmt.Sprintf("%s %s Bistro", req.City, req.Food),
	// 			fmt.Sprintf("%s %s Kitchen", req.City, req.Food),
	// 		},
	// 		Note: "Synthetic recommendations for demo purposes",
	// 	})
	// })

	// MCP SSE endpoint
	// GET/POST /mcp - MCP server using SSE transport
	mcpHandler := mcp.NewSSEHandler(func(req *http.Request) *mcp.Server {
		return createMCPServer()
	}, nil)

	r.Any("/mcp", gin.WrapH(mcpHandler))
	r.Any("/mcp/*path", gin.WrapH(http.StripPrefix("/mcp", mcpHandler)))

	r.Run(":8080")
}
