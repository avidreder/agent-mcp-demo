package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/andrewreder/agent-poc/go-api/x402"
)

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

// x402 middleware instance (configured with synthetic payment details)
// TODO: Load these from environment variables or config
var x402Middleware = x402.NewMiddleware(
	"http://localhost:8080",                               // Server URL
	"0x1234567890abcdef1234567890abcdef12345678",          // TODO: Real payTo address
	"eip155:84532",                                         // Base Sepolia testnet
	"0x036CbD53842c5426634e7929541eC2318f3dCF7e",          // TODO: Real USDC contract address
)

func init() {
	// Configure tool pricing
	// TODO: Load pricing from config or database
	x402Middleware.SetToolPrice("get_weather", "1000")      // 0.001 USDC (6 decimals)
	x402Middleware.SetToolPrice("get_stock_quote", "5000")  // 0.005 USDC
	x402Middleware.SetToolPrice("get_news", "2000")         // 0.002 USDC
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
	}, x402.WrapToolHandler(x402Middleware, "get_weather", getWeather))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_stock_quote",
		Description: "Get the current stock price for a ticker symbol",
	}, x402.WrapToolHandler(x402Middleware, "get_stock_quote", getStockQuote))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_news",
		Description: "Get latest news headlines for a topic",
	}, x402.WrapToolHandler(x402Middleware, "get_news", getNews))

	return server
}

func main() {
	r := gin.Default()

	// REST API endpoint
	// GET /discovery/resources - Returns list of available resources
	r.GET("/discovery/resources", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"resources": resources,
		})
	})

	// MCP SSE endpoint
	// GET/POST /mcp - MCP server using SSE transport
	mcpHandler := mcp.NewSSEHandler(func(req *http.Request) *mcp.Server {
		return createMCPServer()
	}, nil)

	r.Any("/mcp", gin.WrapH(mcpHandler))
	r.Any("/mcp/*path", gin.WrapH(http.StripPrefix("/mcp", mcpHandler)))

	r.Run(":8080")
}
