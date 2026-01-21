package configs

import "github.com/trackvision/tv-shared-go/database"

// Env holds common configuration shared across all pipelines
type Env struct {
	Port string

	// Directus Configuration (required for all pipelines)
	CMSBaseURL        string
	DirectusCMSAPIKey string

	// Email Configuration (common SMTP settings)
	EmailSMTPHost     string
	EmailSMTPPort     string
	EmailSMTPUser     string
	EmailSMTPPassword string

	// Database Configuration (TiDB)
	Database *database.DB
}

// NewDatabaseConfig creates a database configuration
func NewDatabaseConfig(host, port, dbName, user string, ssl bool) *database.DB {
	return &database.DB{
		Host:     host,
		Port:     port,
		Database: dbName,
		Username: user,
		SSL:      ssl,
	}
}
