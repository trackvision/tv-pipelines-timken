package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tv-pipelines-timken/configs"
	"tv-pipelines-timken/types"
)

func TestNewDirectusClient(t *testing.T) {
	cfg := &configs.Config{
		CMSBaseURL:     "https://cms.example.com",
		DirectusAPIKey: "test-key",
	}

	client := NewDirectusClient(cfg)

	if client.baseURL != cfg.CMSBaseURL {
		t.Errorf("baseURL = %q, want %q", client.baseURL, cfg.CMSBaseURL)
	}
	if client.apiKey != cfg.DirectusAPIKey {
		t.Errorf("apiKey = %q, want %q", client.apiKey, cfg.DirectusAPIKey)
	}
}

func TestDirectusClient_PostItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/items/test-collection" {
			t.Errorf("Path = %q, want /items/test-collection", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization header missing or incorrect")
		}

		response := types.DirectusResponse[struct {
			ID string `json:"id"`
		}]{
			Data: struct {
				ID string `json:"id"`
			}{ID: "created-id-123"},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &DirectusClient{
		baseURL:    server.URL,
		apiKey:     "test-key",
		httpClient: http.DefaultClient,
	}

	item := map[string]string{"name": "test"}
	id, err := client.PostItem(context.Background(), "test-collection", item)
	if err != nil {
		t.Fatalf("PostItem() error = %v", err)
	}
	if id != "created-id-123" {
		t.Errorf("PostItem() = %q, want %q", id, "created-id-123")
	}
}

func TestDirectusClient_PatchItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Method = %q, want PATCH", r.Method)
		}
		if r.URL.Path != "/items/test-collection/item-123" {
			t.Errorf("Path = %q, want /items/test-collection/item-123", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &DirectusClient{
		baseURL:    server.URL,
		apiKey:     "test-key",
		httpClient: http.DefaultClient,
	}

	err := client.PatchItem(context.Background(), "test-collection", "item-123", map[string]interface{}{"field": "value"})
	if err != nil {
		t.Fatalf("PatchItem() error = %v", err)
	}
}

func TestDirectusClient_UploadFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/files" {
			t.Errorf("Path = %q, want /files", r.URL.Path)
		}

		response := types.DirectusResponse[types.DirectusFileResponse]{
			Data: types.DirectusFileResponse{ID: "file-id-456"},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &DirectusClient{
		baseURL:    server.URL,
		apiKey:     "test-key",
		httpClient: http.DefaultClient,
	}

	params := UploadFileParams{
		Filename: "test.pdf",
		Content:  []byte("test content"),
		FolderID: "folder-123",
	}

	id, err := client.UploadFile(context.Background(), params)
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if id != "file-id-456" {
		t.Errorf("UploadFile() = %q, want %q", id, "file-id-456")
	}
}

func TestDirectusClient_PostItem_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := &DirectusClient{
		baseURL:    server.URL,
		apiKey:     "test-key",
		httpClient: http.DefaultClient,
	}

	_, err := client.PostItem(context.Background(), "test-collection", map[string]string{})
	if err == nil {
		t.Error("PostItem() expected error for 500 response")
	}
}
