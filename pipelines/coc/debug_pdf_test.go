//go:build integration

package coc

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"

	"tv-pipelines-timken/configs"
	"tv-pipelines-timken/tasks"
)

// TestDebug_GeneratePDF generates a PDF and saves it locally for inspection.
// Run with: go test -v -tags=integration -run TestDebug_GeneratePDF ./pipelines/coc/
func TestDebug_GeneratePDF(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	cfg, err := configs.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	sscc := "100538930005550017"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Logf("Generating PDF for SSCC: %s", sscc)

	pdfData, filename, err := tasks.GeneratePDF(ctx, cfg, sscc)
	if err != nil {
		t.Fatalf("GeneratePDF failed: %v", err)
	}

	// Save locally for inspection
	localPath := "/tmp/" + filename
	if err := os.WriteFile(localPath, pdfData, 0644); err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}

	t.Logf("PDF saved to: %s (%d bytes)", localPath, len(pdfData))
}
