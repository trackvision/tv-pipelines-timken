package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"timken-etl/types"
)

func TestFetchCOCData_Success(t *testing.T) {
	expectedItems := []types.COCItem{
		{
			SSCC:            "100538930005550017",
			Serial:          "SN0001",
			ProductID:       "PROD001",
			COCDocumentID:   "DOC123",
			COCDocumentDate: "2025-10-16",
		},
		{
			SSCC:   "100538930005550017",
			Serial: "SN0002",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Verify SSCC parameter (URL encoded)
		sscc := r.URL.Query().Get("sscc")
		if sscc != "100538930005550017" {
			t.Errorf("expected sscc '100538930005550017', got %s", sscc)
		}

		// Verify auth header
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Bearer auth header")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedItems)
	}))
	defer server.Close()

	ctx := context.Background()
	result, err := FetchCOCData(ctx, server.URL, "test-api-key", "100538930005550017")
	if err != nil {
		t.Fatalf("FetchCOCData failed: %v", err)
	}

	if result.SSCC != "100538930005550017" {
		t.Errorf("expected SSCC '100538930005550017', got %s", result.SSCC)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].Serial != "SN0001" {
		t.Errorf("expected first serial 'SN0001', got %s", result.Items[0].Serial)
	}
}

func TestFetchCOCData_EmptySSCC(t *testing.T) {
	ctx := context.Background()
	_, err := FetchCOCData(ctx, "http://example.com", "key", "")
	if err == nil {
		t.Error("expected error for empty SSCC")
	}
	if !strings.Contains(err.Error(), "missing required 'sscc' parameter") {
		t.Errorf("expected missing sscc error, got: %v", err)
	}
}

func TestFetchCOCData_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]types.COCItem{})
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := FetchCOCData(ctx, server.URL, "key", "100538930005550017")
	if err == nil {
		t.Error("expected error for empty response")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("expected empty response error, got: %v", err)
	}
}

func TestFetchCOCData_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := FetchCOCData(ctx, server.URL, "key", "100538930005550017")
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status 500 in error, got: %v", err)
	}
}

func TestFetchCOCData_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := FetchCOCData(ctx, server.URL, "key", "100538930005550017")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFetchCOCData_URLEncoding(t *testing.T) {
	var receivedSSCC string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSSCC = r.URL.Query().Get("sscc")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]types.COCItem{{SSCC: receivedSSCC, Serial: "SN001"}})
	}))
	defer server.Close()

	ctx := context.Background()
	// Test with special characters that need encoding
	sscc := "test+value&special=chars"
	_, err := FetchCOCData(ctx, server.URL, "key", sscc)
	if err != nil {
		t.Fatalf("FetchCOCData failed: %v", err)
	}

	// The server should receive the decoded value
	if receivedSSCC != sscc {
		t.Errorf("URL encoding issue: expected %q, got %q", sscc, receivedSSCC)
	}
}

func TestFetchCOCData_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := FetchCOCData(ctx, server.URL, "key", "100538930005550017")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
