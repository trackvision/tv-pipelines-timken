package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchDirectusData_Success(t *testing.T) {
	expectedItems := []DirectusItem{
		{
			ID:          "item-001",
			Status:      "published",
			DateCreated: "2025-01-01T00:00:00Z",
		},
		{
			ID:     "item-002",
			Status: "draft",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		q := r.URL.Query().Get("q")
		if q != "test-query" {
			t.Errorf("expected query 'test-query', got %s", q)
		}

		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Bearer auth header")
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(expectedItems); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	result, err := FetchDirectusData(ctx, nil, server.URL, "test-api-key", "test-query")
	if err != nil {
		t.Fatalf("FetchDirectusData failed: %v", err)
	}

	if result.Query != "test-query" {
		t.Errorf("expected Query 'test-query', got %s", result.Query)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].ID != "item-001" {
		t.Errorf("expected first ID 'item-001', got %s", result.Items[0].ID)
	}
}

func TestFetchDirectusData_EmptyQuery(t *testing.T) {
	ctx := context.Background()
	_, err := FetchDirectusData(ctx, nil, "http://example.com", "key", "")
	if err == nil {
		t.Error("expected error for empty query")
	}
	if !strings.Contains(err.Error(), "missing required query parameter") {
		t.Errorf("expected missing query error, got: %v", err)
	}
}

func TestFetchDirectusData_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]DirectusItem{}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := FetchDirectusData(ctx, nil, server.URL, "key", "test-query")
	if err == nil {
		t.Error("expected error for empty response")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("expected empty response error, got: %v", err)
	}
}

func TestFetchDirectusData_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("internal server error")); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := FetchDirectusData(ctx, nil, server.URL, "key", "test-query")
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status 500 in error, got: %v", err)
	}
}

func TestFetchDirectusData_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("not valid json")); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := FetchDirectusData(ctx, nil, server.URL, "key", "test-query")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFetchDirectusData_URLEncoding(t *testing.T) {
	var receivedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]DirectusItem{{ID: "item-1", Status: "published"}}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	query := "test+value&special=chars"
	_, err := FetchDirectusData(ctx, nil, server.URL, "key", query)
	if err != nil {
		t.Fatalf("FetchDirectusData failed: %v", err)
	}

	if receivedQuery != query {
		t.Errorf("URL encoding issue: expected %q, got %q", query, receivedQuery)
	}
}

func TestFetchDirectusData_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wait for context cancellation
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := FetchDirectusData(ctx, nil, server.URL, "key", "test-query")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
