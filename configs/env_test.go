package configs

import (
	"testing"
)

func TestLoad_Success(t *testing.T) {
	// Set required environment variables
	t.Setenv("CMS_BASE_URL", "https://cms.example.com")
	t.Setenv("DIRECTUS_CMS_API_KEY", "test-api-key")
	t.Setenv("COC_VIEWER_BASE_URL", "https://viewer.example.com")
	t.Setenv("COC_DATA_API_URL", "https://api.example.com/coc")
	t.Setenv("EMAIL_FROM_ADDRESS", "test@example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.CMSBaseURL != "https://cms.example.com" {
		t.Errorf("CMSBaseURL = %q, want %q", cfg.CMSBaseURL, "https://cms.example.com")
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want default %q", cfg.Port, "8080")
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	// Clear all required env vars by setting them to empty
	t.Setenv("CMS_BASE_URL", "")
	t.Setenv("DIRECTUS_CMS_API_KEY", "test-key") // Required secret, set to valid
	t.Setenv("COC_VIEWER_BASE_URL", "")
	t.Setenv("COC_DATA_API_URL", "")
	t.Setenv("EMAIL_FROM_ADDRESS", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing required vars")
	}
}

func TestGetEnv_Default(t *testing.T) {
	// Use a unique var name that won't be set
	got := getEnv("TEST_VAR_DEFINITELY_NOT_SET_12345", "default-value")
	if got != "default-value" {
		t.Errorf("getEnv() = %q, want %q", got, "default-value")
	}
}

func TestGetEnv_FromEnv(t *testing.T) {
	t.Setenv("TEST_VAR_SET", "env-value")

	got := getEnv("TEST_VAR_SET", "default-value")
	if got != "env-value" {
		t.Errorf("getEnv() = %q, want %q", got, "env-value")
	}
}
