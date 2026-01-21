package coc

import (
	"fmt"
	"os"
)

// Config holds COC-specific configuration
type Config struct {
	// TimkenCOCAPIURL is the URL of the Timken COC API
	TimkenCOCAPIURL string

	// COCViewerBaseURL is the base URL for the COC viewer (PDF generation)
	COCViewerBaseURL string

	// COCPDFFolderID is the Directus folder ID for storing PDFs
	COCPDFFolderID string

	// COCFromEmail is the sender email address for notifications
	COCFromEmail string
}

// LoadConfig loads COC-specific config from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		TimkenCOCAPIURL:  os.Getenv("TIMKEN_COC_API_URL"),
		COCViewerBaseURL: os.Getenv("COC_VIEWER_BASE_URL"),
		COCPDFFolderID:   os.Getenv("COC_PDF_FOLDER_ID"),
		COCFromEmail:     os.Getenv("COC_FROM_EMAIL"),
	}

	return cfg, nil
}

// Validate checks that all required COC configuration is present
func (c *Config) Validate() error {
	if c.TimkenCOCAPIURL == "" {
		return fmt.Errorf("TIMKEN_COC_API_URL is required")
	}
	if c.COCViewerBaseURL == "" {
		return fmt.Errorf("COC_VIEWER_BASE_URL is required")
	}
	if c.COCPDFFolderID == "" {
		return fmt.Errorf("COC_PDF_FOLDER_ID is required")
	}
	return nil
}
