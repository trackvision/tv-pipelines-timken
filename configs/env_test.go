package configs

import (
	"os"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	// Set required environment variables
	os.Setenv("CMS_BASE_URL", "https://cms.example.com")
	os.Setenv("DIRECTUS_CMS_API_KEY", "test-api-key")
	os.Setenv("COC_VIEWER_BASE_URL", "https://viewer.example.com")
	os.Setenv("COC_DATA_API_URL", "https://api.example.com/coc")
	os.Setenv("EMAIL_FROM_ADDRESS", "test@example.com")
	defer func() {
		os.Unsetenv("CMS_BASE_URL")
		os.Unsetenv("DIRECTUS_CMS_API_KEY")
		os.Unsetenv("COC_VIEWER_BASE_URL")
		os.Unsetenv("COC_DATA_API_URL")
		os.Unsetenv("EMAIL_FROM_ADDRESS")
	}()

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
	// Clear all required env vars
	os.Unsetenv("CMS_BASE_URL")
	os.Unsetenv("DIRECTUS_CMS_API_KEY")
	os.Unsetenv("COC_VIEWER_BASE_URL")
	os.Unsetenv("COC_DATA_API_URL")
	os.Unsetenv("EMAIL_FROM_ADDRESS")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing required vars")
	}
}

func TestGetEnv_Default(t *testing.T) {
	os.Unsetenv("TEST_VAR_NOT_SET")

	got := getEnv("TEST_VAR_NOT_SET", "default-value")
	if got != "default-value" {
		t.Errorf("getEnv() = %q, want %q", got, "default-value")
	}
}

func TestGetEnv_FromEnv(t *testing.T) {
	os.Setenv("TEST_VAR_SET", "env-value")
	defer os.Unsetenv("TEST_VAR_SET")

	got := getEnv("TEST_VAR_SET", "default-value")
	if got != "env-value" {
		t.Errorf("getEnv() = %q, want %q", got, "env-value")
	}
}
