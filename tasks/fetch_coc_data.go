package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
	"timken-etl/types"

	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

// Shared HTTP client for reuse across calls
var httpClient = &http.Client{Timeout: 30 * time.Second}

// FetchCOCData fetches COC data from the Timken COC API
func FetchCOCData(ctx context.Context, apiURL, apiKey, sscc string) (*types.COCData, error) {
	logger.Info("Fetching COC data", zap.String("sscc", sscc))

	if sscc == "" {
		return nil, fmt.Errorf("missing required 'sscc' parameter")
	}

	requestURL := fmt.Sprintf("%s?sscc=%s", apiURL, url.QueryEscape(sscc))

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var items []types.COCItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("empty response from API for SSCC: %s", sscc)
	}

	logger.Info("Fetched COC data", zap.Int("items", len(items)))

	return &types.COCData{Items: items, SSCC: sscc}, nil
}
