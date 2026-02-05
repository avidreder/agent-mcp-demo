package x402

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	cdpjwt "github.com/coinbase/cdp-sdk/go/auth"
	x402http "github.com/coinbase/x402/go/http"
)

const (
	CoinbaseFacilitatorBaseURL = "https://api.cdp.coinbase.com"
	CoinbaseFacilitatorV2Route = "/platform/v2/x402"

	X402SDKVersion = "0.7.3"
	CDPSDKVersion  = "1.29.0"
)

// CoinbaseAuthProvider generates auth headers for Coinbase facilitator requests.
type CoinbaseAuthProvider struct {
	apiKeyID     string
	apiKeySecret string
	requestHost  string
}

// NewCoinbaseAuthProvider builds a provider for Coinbase facilitator auth.
func NewCoinbaseAuthProvider(apiKeyID, apiKeySecret string) *CoinbaseAuthProvider {
	return &CoinbaseAuthProvider{
		apiKeyID:     apiKeyID,
		apiKeySecret: apiKeySecret,
		requestHost:  coinbaseRequestHost(),
	}
}

// GetAuthHeaders implements the x402 HTTP AuthProvider interface.
func (p *CoinbaseAuthProvider) GetAuthHeaders(ctx context.Context) (x402http.AuthHeaders, error) {
	headers := x402http.AuthHeaders{
		Verify: map[string]string{
			"Correlation-Context": createCorrelationHeader(),
		},
		Settle: map[string]string{
			"Correlation-Context": createCorrelationHeader(),
		},
		Supported: map[string]string{
			"Correlation-Context": createCorrelationHeader(),
		},
	}

	if p.apiKeyID == "" || p.apiKeySecret == "" {
		return headers, nil
	}

	verify, err := createAuthHeader(p.apiKeyID, p.apiKeySecret, "POST", p.requestHost, CoinbaseFacilitatorV2Route+"/verify")
	if err != nil {
		return x402http.AuthHeaders{}, err
	}
	settle, err := createAuthHeader(p.apiKeyID, p.apiKeySecret, "POST", p.requestHost, CoinbaseFacilitatorV2Route+"/settle")
	if err != nil {
		return x402http.AuthHeaders{}, err
	}
	supported, err := createAuthHeader(p.apiKeyID, p.apiKeySecret, "GET", p.requestHost, CoinbaseFacilitatorV2Route+"/supported")
	if err != nil {
		return x402http.AuthHeaders{}, err
	}

	headers.Verify["Authorization"] = verify
	headers.Settle["Authorization"] = settle
	headers.Supported["Authorization"] = supported

	return headers, nil
}

// FacilitatorConfigFromEnv builds a facilitator config using env vars when present.
func FacilitatorConfigFromEnv(defaultURL string) *x402http.FacilitatorConfig {
	apiKeyID := strings.TrimSpace(os.Getenv("CDP_API_KEY"))
	apiKeySecret := strings.TrimSpace(os.Getenv("CDP_API_KEY_SECRET"))
	facilitatorURL := strings.TrimSpace(os.Getenv("FACILITATOR_URL"))

	if facilitatorURL == "" {
		if apiKeyID != "" || apiKeySecret != "" {
			facilitatorURL = CoinbaseFacilitatorBaseURL + CoinbaseFacilitatorV2Route
		} else {
			facilitatorURL = defaultURL
		}
	}

	config := &x402http.FacilitatorConfig{
		URL: facilitatorURL,
	}

	if apiKeyID != "" && apiKeySecret != "" {
		config.AuthProvider = NewCoinbaseAuthProvider(apiKeyID, apiKeySecret)
	}

	return config
}

func createAuthHeader(apiKeyID, apiKeySecret, requestMethod, requestHost, requestPath string) (string, error) {
	jwt, err := cdpjwt.GenerateJWT(cdpjwt.JwtOptions{
		KeyID:         apiKeyID,
		KeySecret:     apiKeySecret,
		RequestMethod: requestMethod,
		RequestHost:   requestHost,
		RequestPath:   requestPath,
	})
	if err != nil {
		return "", fmt.Errorf("generate JWT: %w", err)
	}
	return "Bearer " + jwt, nil
}

func createCorrelationHeader() string {
	data := map[string]string{
		"sdk_version":    CDPSDKVersion,
		"sdk_language":   "go",
		"source":         "x402",
		"source_version": X402SDKVersion,
	}

	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, url.QueryEscape(data[key])))
	}
	return strings.Join(parts, ",")
}

func coinbaseRequestHost() string {
	parsed, err := url.Parse(CoinbaseFacilitatorBaseURL)
	if err != nil || parsed.Host == "" {
		return strings.TrimPrefix(CoinbaseFacilitatorBaseURL, "https://")
	}
	return parsed.Host
}

func GetFacilitatorClient() *x402http.HTTPFacilitatorClient {
	facilitatorURL := os.Getenv("FACILITATOR_URL")
	if facilitatorURL == "" {
		facilitatorURL = CoinbaseFacilitatorBaseURL + CoinbaseFacilitatorV2Route
	}
	if strings.Contains(facilitatorURL, "coinbase") {
		return x402http.NewHTTPFacilitatorClient(&x402http.FacilitatorConfig{
			URL:          facilitatorURL,
			AuthProvider: NewCoinbaseAuthProvider(os.Getenv("CDP_API_KEY"), os.Getenv("CDP_API_KEY_SECRET")),
		})
	}
	return x402http.NewHTTPFacilitatorClient(&x402http.FacilitatorConfig{
		URL: facilitatorURL,
	})
}
