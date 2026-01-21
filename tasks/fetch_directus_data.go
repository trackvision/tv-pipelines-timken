package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

// DirectusItem represents a generic item from a Directus API response
// Customize this struct for your specific use case
type DirectusItem struct {
	ID          string `json:"id"`
	Status      string `json:"status,omitempty"`
	DateCreated string `json:"date_created,omitempty"`
	DateUpdated string `json:"date_updated,omitempty"`
	// Add your custom fields here
}

// DirectusData contains the collection of items from Directus API
type DirectusData struct {
	Items []DirectusItem `json:"items"`
	Query string         `json:"query"`
}

// FetchDirectusData fetches data from a Directus Flow or API endpoint
// This is a template - customize the URL pattern and response handling for your use case
// Pass nil for client to use a default client (respects context timeout)
func FetchDirectusData(ctx context.Context, client *http.Client, apiURL, apiKey, queryParam string) (*DirectusData, error) {
	logger.Info("Fetching Directus data", zap.String("query", queryParam))

	if queryParam == "" {
		return nil, fmt.Errorf("missing required query parameter")
	}

	if client == nil {
		client = http.DefaultClient
	}

	requestURL := fmt.Sprintf("%s?q=%s", apiURL, url.QueryEscape(queryParam))

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			logger.Warn("Failed to close response body", zap.Error(cerr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var items []DirectusItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("empty response from API for query: %s", queryParam)
	}

	logger.Info("Fetched Directus data", zap.Int("items", len(items)))

	return &DirectusData{Items: items, Query: queryParam}, nil
}
