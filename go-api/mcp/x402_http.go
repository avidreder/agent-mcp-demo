package mcp

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	x402types "github.com/coinbase/x402/go/types"
)

type paymentHeader struct {
	Name    string
	Value   string
	Version int
}

func encodePaymentHeader(payment any) (*paymentHeader, error) {
	payloadBytes, err := json.Marshal(payment)
	if err != nil {
		return nil, err
	}

	version, err := x402types.DetectVersion(payloadBytes)
	if err != nil {
		if version, ok := normalizeX402Version(extractX402Version(payment)); ok {
			return buildPaymentHeader(version, payloadBytes), nil
		}
		return nil, err
	}

	return buildPaymentHeader(version, payloadBytes), nil
}

func buildPaymentHeader(version int, payloadBytes []byte) *paymentHeader {
	headerName := "X-PAYMENT"
	if version >= 2 {
		headerName = "PAYMENT-SIGNATURE"
	}
	return &paymentHeader{
		Name:    headerName,
		Value:   base64.StdEncoding.EncodeToString(payloadBytes),
		Version: version,
	}
}

func decodePaymentRequired(resp *http.Response, body []byte) map[string]any {
	if resp == nil {
		return nil
	}
	if paymentRequired := decodePaymentHeader(resp.Header.Get("PAYMENT-REQUIRED")); paymentRequired != nil {
		return paymentRequired
	}
	if resp.StatusCode != http.StatusPaymentRequired || len(body) == 0 {
		return nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil
	}
	version, err := x402types.DetectVersion(body)
	if err != nil {
		version, ok := normalizeX402Version(decoded["x402Version"])
		if !ok || version != 1 {
			return nil
		}
	} else if version != 1 {
		return nil
	}
	if _, ok := decoded["accepts"]; !ok {
		return nil
	}
	return decoded
}

func decodePaymentResponse(resp *http.Response) map[string]any {
	if resp == nil {
		return nil
	}
	if paymentResponse := decodePaymentHeader(resp.Header.Get("PAYMENT-RESPONSE")); paymentResponse != nil {
		return paymentResponse
	}
	return decodePaymentHeader(resp.Header.Get("X-PAYMENT-RESPONSE"))
}

func extractX402Version(payment any) any {
	paymentMap, ok := payment.(map[string]any)
	if !ok {
		return nil
	}
	return paymentMap["x402Version"]
}
