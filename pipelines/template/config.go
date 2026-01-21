package template

import (
	"fmt"
	"os"
)

// Config holds pipeline-specific configuration
type Config struct {
	// ExampleAPIURL is a placeholder for your API endpoint
	ExampleAPIURL string

	// ExampleFolderID is a placeholder for a folder/resource ID
	ExampleFolderID string
}

// LoadConfig loads pipeline-specific config from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		ExampleAPIURL:   os.Getenv("EXAMPLE_API_URL"),
		ExampleFolderID: os.Getenv("EXAMPLE_FOLDER_ID"),
	}

	return cfg, nil
}

// Validate checks that all required configuration is present
func (c *Config) Validate() error {
	if c.ExampleAPIURL == "" {
		return fmt.Errorf("EXAMPLE_API_URL is required")
	}
	return nil
}
