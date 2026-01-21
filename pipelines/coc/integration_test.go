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

func init() {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

// TestIntegration_COCPipeline runs the full COC pipeline end-to-end.
// Run with: go test -v -tags=integration -run TestIntegration_COCPipeline ./pipelines/coc/
func TestIntegration_COCPipeline(t *testing.T) {
	// Load config from environment
	cfg, err := configs.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Create Directus client
	cms := tasks.NewDirectusClient(cfg)

	// Test SSCC
	sscc := os.Getenv("TEST_SSCC")
	if sscc == "" {
		sscc = "100538930005550017"
	}

	t.Logf("Running COC pipeline for SSCC: %s", sscc)

	// Run with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := Run(ctx, cms, cfg, sscc)
	if err != nil {
		t.Fatalf("pipeline returned error: %v", err)
	}

	if !result.Success {
		t.Fatalf("pipeline failed: %s", result.Error)
	}

	t.Logf("Pipeline completed successfully!")
	t.Logf("  Certification ID: %s", result.CertificationID)
	t.Logf("  File ID: %s", result.FileID)
	t.Logf("  Email Sent: %v", result.EmailSent)
}
