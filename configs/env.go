package configs

import "github.com/trackvision/tv-shared-go/database"

// Env holds common configuration shared across all pipelines
type Env struct {
	Port string `env:"PORT" envDefault:"8080"`

	// Directus Configuration (required for all pipelines)
	CMSBaseURL        string `env:"CMS_BASE_URL,required"`
	DirectusCMSAPIKey string `env:"DIRECTUS_CMS_API_KEY,required"`

	// Email Configuration (common SMTP settings)
	EmailSMTPHost     string `env:"EMAIL_SMTP_HOST" envDefault:"smtp.resend.com"`
	EmailSMTPPort     string `env:"EMAIL_SMTP_PORT" envDefault:"587"`
	EmailSMTPUser     string `env:"EMAIL_SMTP_USER" envDefault:"resend"`
	EmailSMTPPassword string `env:"EMAIL_SMTP_PASSWORD"`

	// Database Configuration (TiDB)
	Database *database.DB
}

// NewDatabaseConfig creates a database configuration from environment variables
// Set DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD, DB_SSL environment variables
func NewDatabaseConfig(host, port, dbName, user string, ssl bool) *database.DB {
	return &database.DB{
		Host:     host,
		Port:     port,
		Database: dbName,
		Username: user,
		SSL:      ssl,
	}
}
