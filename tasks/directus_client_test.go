package tasks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDirectusClient_PostItem_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/items/certification" {
			t.Errorf("expected /items/certification, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Bearer auth header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}

		// Read and verify body
		body, _ := io.ReadAll(r.Body)
		var item map[string]any
		if err := json.Unmarshal(body, &item); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if item["name"] != "test item" {
			t.Errorf("expected name 'test item', got %v", item["name"])
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":   "created-id-123",
				"name": "test item",
			},
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDirectusClient(server.URL, "test-api-key")
	ctx := context.Background()

	result, err := client.PostItem(ctx, "certification", map[string]any{"name": "test item"})
	if err != nil {
		t.Fatalf("PostItem failed: %v", err)
	}

	if result["id"] != "created-id-123" {
		t.Errorf("expected id 'created-id-123', got %v", result["id"])
	}
}

func TestDirectusClient_PostItem_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"errors": [{"message": "validation failed"}]}`)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDirectusClient(server.URL, "test-api-key")
	ctx := context.Background()

	_, err := client.PostItem(ctx, "certification", map[string]any{"name": ""})
	if err == nil {
		t.Error("expected error for bad request, got nil")
	}
}

func TestDirectusClient_PatchItem_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/items/certification/item-123" {
			t.Errorf("expected /items/certification/item-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": "item-123", "updated": true},
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDirectusClient(server.URL, "test-api-key")
	ctx := context.Background()

	err := client.PatchItem(ctx, "certification", "item-123", map[string]any{"status": "active"})
	if err != nil {
		t.Fatalf("PatchItem failed: %v", err)
	}
}

func TestDirectusClient_UploadFile_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/files" {
			t.Errorf("expected /files, got %s", r.URL.Path)
		}

		// Verify multipart form
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			t.Fatalf("failed to parse multipart: %v", err)
		}

		if r.FormValue("folder") != "folder-123" {
			t.Errorf("expected folder 'folder-123', got %s", r.FormValue("folder"))
		}
		if r.FormValue("title") != "Test PDF" {
			t.Errorf("expected title 'Test PDF', got %s", r.FormValue("title"))
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("failed to get file: %v", err)
		}
		defer func() {
			if cerr := file.Close(); cerr != nil {
				t.Errorf("failed to close file: %v", cerr)
			}
		}()

		if header.Filename != "test.pdf" {
			t.Errorf("expected filename 'test.pdf', got %s", header.Filename)
		}

		content, _ := io.ReadAll(file)
		if string(content) != "fake pdf content" {
			t.Errorf("file content mismatch")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": "file-id-456"},
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDirectusClient(server.URL, "test-api-key")
	ctx := context.Background()

	result, err := client.UploadFile(ctx, UploadFileParams{
		Filename:    "test.pdf",
		Content:     []byte("fake pdf content"),
		FolderID:    "folder-123",
		Title:       "Test PDF",
		ContentType: "application/pdf",
	})
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}

	if result.ID != "file-id-456" {
		t.Errorf("expected file ID 'file-id-456', got %s", result.ID)
	}
}

func TestDirectusClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wait for context cancellation
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewDirectusClient(server.URL, "test-api-key")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.PostItem(ctx, "test", map[string]any{})
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
