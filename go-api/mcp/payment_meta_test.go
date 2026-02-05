package mcp

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestInjectPaymentSignature(t *testing.T) {
	t.Parallel()

	params, err := injectPaymentSignature(nil, map[string]any{
		"x402Version": 2,
		"resource": map[string]any{
			"url": "mcp://tool/financial_analysis",
		},
		"accepted": map[string]any{
			"scheme":  "exact",
			"network": "eip155:84532",
		},
		"payload": map[string]any{
			"signature": "0xdeadbeef",
		},
	})
	if err != nil {
		t.Fatalf("injectPaymentSignature error: %v", err)
	}

	headers, ok := params["headers"].(map[string]any)
	if !ok {
		t.Fatalf("expected headers object, got %T", params["headers"])
	}
	rawHeader, ok := headers["PAYMENT-SIGNATURE"].(string)
	if !ok || rawHeader == "" {
		t.Fatalf("expected PAYMENT-SIGNATURE to be set")
	}
	decoded, err := base64.StdEncoding.DecodeString(rawHeader)
	if err != nil {
		t.Fatalf("decode PAYMENT-SIGNATURE header: %v", err)
	}
	var headerPayload map[string]any
	if err := json.Unmarshal(decoded, &headerPayload); err != nil {
		t.Fatalf("unmarshal X-PAYMENT payload: %v", err)
	}
	if headerPayload["x402Version"] != float64(2) {
		t.Fatalf("expected x402Version to be 2, got %v", headerPayload["x402Version"])
	}
	resource, ok := headerPayload["resource"].(map[string]any)
	if !ok || resource["url"] != "mcp://tool/financial_analysis" {
		t.Fatalf("expected resource url to be set")
	}
	accepted, ok := headerPayload["accepted"].(map[string]any)
	if !ok || accepted["scheme"] != "exact" || accepted["network"] != "eip155:84532" {
		t.Fatalf("expected accepted scheme/network to be set")
	}
	payload, ok := headerPayload["payload"].(map[string]any)
	if !ok || payload["signature"] != "0xdeadbeef" {
		t.Fatalf("expected payload signature to be set")
	}
}

func TestInjectPaymentSignatureV1UsesXPayment(t *testing.T) {
	t.Parallel()

	params, err := injectPaymentSignature(nil, map[string]any{
		"x402Version": 1,
		"scheme":      "exact",
		"network":     "base-sepolia",
		"payload": map[string]any{
			"signature": "0xdeadbeef",
		},
	})
	if err != nil {
		t.Fatalf("injectPaymentSignature error: %v", err)
	}

	headers, ok := params["headers"].(map[string]any)
	if !ok {
		t.Fatalf("expected headers object, got %T", params["headers"])
	}
	rawHeader, ok := headers["X-PAYMENT"].(string)
	if !ok || rawHeader == "" {
		t.Fatalf("expected X-PAYMENT to be set")
	}
	decoded, err := base64.StdEncoding.DecodeString(rawHeader)
	if err != nil {
		t.Fatalf("decode X-PAYMENT header: %v", err)
	}
	var headerPayload map[string]any
	if err := json.Unmarshal(decoded, &headerPayload); err != nil {
		t.Fatalf("unmarshal X-PAYMENT payload: %v", err)
	}
	if headerPayload["x402Version"] != float64(1) {
		t.Fatalf("expected x402Version to be 1, got %v", headerPayload["x402Version"])
	}
	if headerPayload["scheme"] != "exact" || headerPayload["network"] != "base-sepolia" {
		t.Fatalf("expected scheme/network to be set")
	}
	payload, ok := headerPayload["payload"].(map[string]any)
	if !ok || payload["signature"] != "0xdeadbeef" {
		t.Fatalf("expected payload signature to be set")
	}
}

func TestHTTPResponseToMCPResultAddsPaymentMeta(t *testing.T) {
	t.Parallel()

	paymentResponse := map[string]any{
		"success": true,
		"network": "eip155:84532",
	}
	payload, err := json.Marshal(paymentResponse)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Payment-Response": []string{base64.StdEncoding.EncodeToString(payload)},
		},
		Body: io.NopCloser(strings.NewReader(`{"ok":true}`)),
	}

	result, err := httpResponseToMCPResult(resp)
	if err != nil {
		t.Fatalf("httpResponseToMCPResult error: %v", err)
	}
	if result.Meta == nil {
		t.Fatalf("expected meta to be set")
	}
	if _, ok := result.Meta["x402/payment-response"]; !ok {
		t.Fatalf("expected x402/payment-response in meta")
	}
}

func TestHTTPResponseToMCPResultPaymentRequired(t *testing.T) {
	t.Parallel()

	paymentRequired := map[string]any{
		"x402Version": 2,
		"error":       "Payment required to access this resource",
		"resource": map[string]any{
			"url":         "mcp://tool/financial_analysis",
			"description": "Advanced financial analysis tool",
			"mimeType":    "application/json",
		},
		"accepts": []any{
			map[string]any{
				"scheme":            "exact",
				"network":           "eip155:84532",
				"amount":            "10000",
				"asset":             "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
				"payTo":             "0x209693Bc6afc0C5328bA36FaF03C514EF312287C",
				"maxTimeoutSeconds": 60,
				"extra": map[string]any{
					"name":    "USDC",
					"version": "2",
				},
			},
		},
	}
	payload, err := json.Marshal(paymentRequired)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	resp := &http.Response{
		StatusCode: http.StatusPaymentRequired,
		Header: http.Header{
			"Payment-Required": []string{base64.StdEncoding.EncodeToString(payload)},
		},
		Body: io.NopCloser(strings.NewReader(`{"error":"payment required"}`)),
	}

	result, err := httpResponseToMCPResult(resp)
	if err != nil {
		t.Fatalf("httpResponseToMCPResult error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected payment required to return IsError=true")
	}
	if result.StructuredContent == nil {
		t.Fatalf("expected structuredContent to be set")
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structuredContent to be map, got %T", result.StructuredContent)
	}
	if structured["error"] != "Payment required to access this resource" {
		t.Fatalf("expected structuredContent error to be set")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected content to be present")
	}
	var textPayload map[string]any
	textContent, ok := result.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if err := json.Unmarshal([]byte(textContent.Text), &textPayload); err != nil {
		t.Fatalf("expected content text to be JSON: %v", err)
	}
	if textPayload["x402Version"] != float64(2) {
		t.Fatalf("expected content x402Version to be 2, got %v", textPayload["x402Version"])
	}
}

func TestHTTPResponseToMCPResultPaymentRequiredV1Body(t *testing.T) {
	t.Parallel()

	paymentRequired := map[string]any{
		"x402Version": 1,
		"error":       "Payment required to access this resource",
		"accepts": []any{
			map[string]any{
				"scheme":            "exact",
				"network":           "base-sepolia",
				"maxAmountRequired": "10000",
				"asset":             "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
				"payTo":             "0x209693Bc6afc0C5328bA36FaF03C514EF312287C",
				"resource":          "https://api.example.com/premium-data",
				"description":       "Access to premium market data",
				"mimeType":          "application/json",
				"maxTimeoutSeconds": 60,
				"extra": map[string]any{
					"name":    "USDC",
					"version": "2",
				},
			},
		},
	}
	payload, err := json.Marshal(paymentRequired)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	resp := &http.Response{
		StatusCode: http.StatusPaymentRequired,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(string(payload))),
	}

	result, err := httpResponseToMCPResult(resp)
	if err != nil {
		t.Fatalf("httpResponseToMCPResult error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected payment required to return IsError=true")
	}
	if result.StructuredContent == nil {
		t.Fatalf("expected structuredContent to be set")
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structuredContent to be map, got %T", result.StructuredContent)
	}
	if structured["x402Version"] != float64(1) {
		t.Fatalf("expected structuredContent x402Version to be 1")
	}
}

func TestHTTPResponseToMCPResultAddsPaymentMetaV1Header(t *testing.T) {
	t.Parallel()

	paymentResponse := map[string]any{
		"success": true,
		"network": "base-sepolia",
	}
	payload, err := json.Marshal(paymentResponse)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"X-Payment-Response": []string{base64.StdEncoding.EncodeToString(payload)},
		},
		Body: io.NopCloser(strings.NewReader(`{"ok":true}`)),
	}

	result, err := httpResponseToMCPResult(resp)
	if err != nil {
		t.Fatalf("httpResponseToMCPResult error: %v", err)
	}
	if result.Meta == nil {
		t.Fatalf("expected meta to be set")
	}
	if _, ok := result.Meta["x402/payment-response"]; !ok {
		t.Fatalf("expected x402/payment-response in meta")
	}
}
