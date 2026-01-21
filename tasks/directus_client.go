package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"tv-pipelines-timken/configs"
	"tv-pipelines-timken/types"
)

// DirectusClient handles communication with the Directus API
type DirectusClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewDirectusClient creates a new Directus API client
func NewDirectusClient(cfg *configs.Config) *DirectusClient {
	return &DirectusClient{
		baseURL: cfg.CMSBaseURL,
		apiKey:  cfg.DirectusAPIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PostItem creates a new item in a collection. Returns the item ID.
func (c *DirectusClient) PostItem(ctx context.Context, collection string, item interface{}) (string, error) {
	url := fmt.Sprintf("%s/items/%s", c.baseURL, collection)

	body, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("marshal item: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("post item: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("directus returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result types.DirectusResponse[struct {
		ID string `json:"id"`
	}]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.Data.ID, nil
}

// PatchItem updates an existing item in a collection
func (c *DirectusClient) PatchItem(ctx context.Context, collection, id string, updates map[string]interface{}) error {
	url := fmt.Sprintf("%s/items/%s/%s", c.baseURL, collection, id)

	body, err := json.Marshal(updates)
	if err != nil {
		return fmt.Errorf("marshal updates: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("patch item: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("directus returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// UploadFileParams holds parameters for file upload
type UploadFileParams struct {
	Filename string
	Content  []byte
	FolderID string
}

// UploadFile uploads a file to Directus. Returns the file ID.
func (c *DirectusClient) UploadFile(ctx context.Context, params UploadFileParams) (string, error) {
	url := fmt.Sprintf("%s/files", c.baseURL)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if params.FolderID != "" {
		if err := writer.WriteField("folder", params.FolderID); err != nil {
			return "", fmt.Errorf("write folder field: %w", err)
		}
	}

	part, err := writer.CreateFormFile("file", params.Filename)
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(params.Content); err != nil {
		return "", fmt.Errorf("write file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(req)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("directus returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result types.DirectusResponse[types.DirectusFileResponse]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.Data.ID, nil
}

func (c *DirectusClient) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
}
