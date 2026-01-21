package pipelines

import (
	"context"
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/trackvision/tv-pipelines-template/configs"
	"github.com/trackvision/tv-pipelines-template/tasks"
	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

// State holds shared state between pipeline tasks
// Common fields are defined here, pipeline-specific data goes in the Data map
// State is safe for concurrent access via Get/Set methods
type State struct {
	// Ctx is the context for cancellation and timeout propagation
	Ctx context.Context

	// Config holds common environment configuration
	Config *configs.Env

	// DirectusClient is the shared Directus API client
	DirectusClient *tasks.DirectusClient

	// DB is the shared database connection (TiDB)
	DB *sqlx.DB

	// mu protects Data from concurrent access
	mu sync.RWMutex

	// Data holds pipeline-specific state that can be set by individual pipelines
	// Use type assertions to retrieve typed values
	Data map[string]interface{}
}

// NewState creates a new pipeline state with initialized maps
func NewState(ctx context.Context, cfg *configs.Env) *State {
	state := &State{
		Ctx:            ctx,
		Config:         cfg,
		DirectusClient: tasks.NewDirectusClient(cfg.CMSBaseURL, cfg.DirectusCMSAPIKey),
		Data:           make(map[string]interface{}),
	}

	return state
}

// InitDB initializes the database connection. Returns error if connection fails.
// Call this separately from NewState to allow pipelines that don't need DB to skip it.
func (s *State) InitDB() error {
	if s.Config.Database == nil {
		return fmt.Errorf("database configuration not set")
	}
	db := s.Config.Database.Open()
	if db == nil {
		return fmt.Errorf("database.Open() returned nil")
	}
	// Verify connection is working
	if err := db.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	s.DB = db
	return nil
}

// Close releases resources held by the State.
// Should be called when the pipeline is done.
func (s *State) Close() {
	if s.DB != nil {
		if err := s.DB.Close(); err != nil {
			logger.Error("Failed to close database connection", zap.Error(err))
		}
	}
}

// Set stores a value in the pipeline state (thread-safe)
func (s *State) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[key] = value
}

// Get retrieves a value from the pipeline state (thread-safe)
func (s *State) Get(key string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Data[key]
}

// GetString retrieves a string value from the pipeline state (thread-safe)
func (s *State) GetString(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.Data[key].(string); ok {
		return v
	}
	return ""
}
