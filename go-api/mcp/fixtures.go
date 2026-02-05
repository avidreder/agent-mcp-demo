package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type fixtureResponse struct {
	Items []X402DiscoveryResource `json:"items"`
}

var (
	fixtureOnce      sync.Once
	fixtureResources []X402DiscoveryResource
	fixtureErr       error
)

func loadDiscoveryResources() ([]X402DiscoveryResource, error) {
	fixtureOnce.Do(func() {
		path, err := fixturePath()
		if err != nil {
			fixtureErr = err
			return
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			fixtureErr = fmt.Errorf("read fixtures: %w", err)
			return
		}
		var decoded fixtureResponse
		if err := json.Unmarshal(payload, &decoded); err != nil {
			fixtureErr = fmt.Errorf("parse fixtures: %w", err)
			return
		}
		fixtureResources = decoded.Items
	})
	return fixtureResources, fixtureErr
}

func fixturePath() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to locate fixtures directory")
	}
	baseDir := filepath.Dir(currentFile)
	return filepath.Clean(filepath.Join(baseDir, "..", "fixtures", "x402-endpoints.json")), nil
}

func filterWeatherResources(items []X402DiscoveryResource) []X402DiscoveryResource {
	filtered := make([]X402DiscoveryResource, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Resource), "/weather") {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func paginateResources(
	items []X402DiscoveryResource,
	limit *int,
	offset *int,
) ([]X402DiscoveryResource, SearchResourcesPagination) {
	total := len(items)
	start := 0
	if offset != nil && *offset > 0 {
		start = *offset
		if start > total {
			start = total
		}
	}
	end := total
	if limit != nil && *limit >= 0 {
		end = start + *limit
		if end > total {
			end = total
		}
	}
	paged := items[start:end]

	var limitPtr *int
	if limit != nil {
		value := *limit
		limitPtr = &value
	}
	var offsetPtr *int
	if offset != nil {
		value := *offset
		offsetPtr = &value
	}
	totalPtr := total

	return paged, SearchResourcesPagination{
		Limit:  limitPtr,
		Offset: offsetPtr,
		Total:  &totalPtr,
	}
}
