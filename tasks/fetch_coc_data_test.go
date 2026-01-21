package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"tv-pipelines-timken/configs"
	"tv-pipelines-timken/types"
)

func init() {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func TestFetchCOCData_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sscc := r.URL.Query().Get("sscc")
		if sscc != "test-sscc-123" {
			t.Errorf("sscc query param = %q, want %q", sscc, "test-sscc-123")
		}

		// API returns array directly, not {"data": [...]}
		response := []types.COCItem{
			{
				SSCC:            "test-sscc-123",
				Serial:          "SN001",
				ProductID:       "PROD-001",
				COCDocumentID:   "DOC-001",
				COCDocumentDate: "2024-01-15",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &configs.Config{
		COCDataAPIURL: server.URL,
	}

	data, err := FetchCOCData(context.Background(), cfg, "test-sscc-123")
	if err != nil {
		t.Fatalf("FetchCOCData() error = %v", err)
	}

	if len(data.Items) != 1 {
		t.Errorf("Items count = %d, want 1", len(data.Items))
	}
	if data.Items[0].SSCC != "test-sscc-123" {
		t.Errorf("SSCC = %q, want %q", data.Items[0].SSCC, "test-sscc-123")
	}
}

func TestFetchCOCData_NoRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API returns empty array
		json.NewEncoder(w).Encode([]types.COCItem{})
	}))
	defer server.Close()

	cfg := &configs.Config{
		COCDataAPIURL: server.URL,
	}

	_, err := FetchCOCData(context.Background(), cfg, "nonexistent-sscc")
	if err == nil {
		t.Error("FetchCOCData() expected error for empty response")
	}
}

func TestFetchCOCData_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	cfg := &configs.Config{
		COCDataAPIURL: server.URL,
	}

	_, err := FetchCOCData(context.Background(), cfg, "test-sscc")
	if err == nil {
		t.Error("FetchCOCData() expected error for 500 response")
	}
}

func TestFetchCOCData_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	cfg := &configs.Config{
		COCDataAPIURL: server.URL,
	}

	_, err := FetchCOCData(context.Background(), cfg, "test-sscc")
	if err == nil {
		t.Error("FetchCOCData() expected error for invalid JSON")
	}
}
