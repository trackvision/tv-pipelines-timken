package configs

import (
	"fmt"
	"os"
)

// Config holds all environment configuration
type Config struct {
	Port              string
	CMSBaseURL        string
	DirectusAPIKey    string
	COCViewerBaseURL  string
	COCDataAPIURL     string
	COCFolderID       string
	EmailFromAddress  string
	EmailSMTPHost     string
	EmailSMTPPort     string
	EmailSMTPUser     string
	EmailSMTPPassword string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		CMSBaseURL:        os.Getenv("CMS_BASE_URL"),
		DirectusAPIKey:    os.Getenv("DIRECTUS_CMS_API_KEY"),
		COCViewerBaseURL:  os.Getenv("COC_VIEWER_BASE_URL"),
		COCDataAPIURL:     os.Getenv("COC_DATA_API_URL"),
		COCFolderID:       os.Getenv("COC_FOLDER_ID"),
		EmailFromAddress:  os.Getenv("EMAIL_FROM_ADDRESS"),
		EmailSMTPHost:     getEnv("EMAIL_SMTP_HOST", "smtp.gmail.com"),
		EmailSMTPPort:     getEnv("EMAIL_SMTP_PORT", "587"),
		EmailSMTPUser:     os.Getenv("EMAIL_SMTP_USER"),
		EmailSMTPPassword: os.Getenv("EMAIL_SMTP_PASSWORD"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"CMS_BASE_URL":         c.CMSBaseURL,
		"DIRECTUS_CMS_API_KEY": c.DirectusAPIKey,
		"COC_VIEWER_BASE_URL":  c.COCViewerBaseURL,
		"COC_DATA_API_URL":     c.COCDataAPIURL,
		"EMAIL_FROM_ADDRESS":   c.EmailFromAddress,
	}

	for name, value := range required {
		if value == "" {
			return fmt.Errorf("required environment variable %s is not set", name)
		}
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
