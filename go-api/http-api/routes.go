package httpapi

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	mcpserver "github.com/andrewreder/agent-poc/go-api/mcp"
	"github.com/gin-gonic/gin"
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

// NewRouter builds the Gin router with all HTTP routes registered.
func NewRouter() (*gin.Engine, error) {
	r := gin.Default()

	attachDebugLogging(r)
	if err := ConfigurePayments(r, serverBaseURL); err != nil {
		return nil, err
	}
	registerDiscoveryRoutes(r, serverBaseURL)
	registerWeatherRoutes(r)
	if err := registerMCPRoute(r); err != nil {
		return nil, err
	}

	return r, nil
}

func attachDebugLogging(r *gin.Engine) {
	// Debug: log payment headers for protected endpoints (toy repo)
	r.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/weather") || strings.HasPrefix(c.Request.URL.Path, "/restaurants") {
			log.Printf("Headers: %+v", c.Request.Header)
		}
		c.Next()
	})
}

func registerDiscoveryRoutes(r *gin.Engine, baseURL string) {
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
						Resource: fmt.Sprintf("%s/weather", baseURL),
						Scheme:   "exact",
					},
				},
				LastUpdated: lastUpdated,
				Resource:    fmt.Sprintf("%s/weather", baseURL),
				Type:        "http",
				X402Version: 1,
			},
		}

		c.JSON(http.StatusOK, gin.H{
			"entries": entries,
		})
	})
}

func registerWeatherRoutes(r *gin.Engine) {
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
}

func registerMCPRoute(r *gin.Engine) error {
	// MCP streamable HTTP endpoint
	discoveryServer, err := mcpserver.NewServer()
	if err != nil {
		return fmt.Errorf("failed to initialize MCP discovery server: %w", err)
	}
	r.Any("/discovery/mcp", gin.WrapH(discoveryServer.Handler()))
	return nil
}
