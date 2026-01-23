package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"tv-pipelines-timken/configs"
	"tv-pipelines-timken/types"
)

// FetchCOCData fetches COC data from the Timken API
func FetchCOCData(ctx context.Context, cfg *configs.Config, sscc string) (*types.COCData, error) {
	logger := zap.L().With(zap.String("task", "fetch_coc_data"), zap.String("sscc", sscc))
	logger.Info("fetch_coc_data started")

	apiURL, err := url.Parse(cfg.COCDataAPIURL)
	if err != nil {
		return nil, fmt.Errorf("invalid COC API URL: %w", err)
	}

	q := apiURL.Query()
	q.Set("sscc", sscc)
	apiURL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add authorization header for Directus flow trigger
	if cfg.DirectusAPIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.DirectusAPIKey))
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch COC data: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("COC API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// API returns array directly, not {"data": [...]}
	var items []types.COCItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("parse COC data: %w", err)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no rows returned from COC API for SSCC %s", sscc)
	}

	logger.Info("fetch_coc_data complete", zap.Int("item_count", len(items)))

	return &types.COCData{Items: items}, nil
}
