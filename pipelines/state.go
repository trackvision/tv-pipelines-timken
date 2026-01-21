package pipelines

import (
	"sync"

	"timken-etl/configs"
	"timken-etl/tasks"
)

// State holds shared state between pipeline tasks
// Common fields are defined here, pipeline-specific data goes in the Data map
// State is safe for concurrent access via Get/Set methods
type State struct {
	// Config holds common environment configuration
	Config *configs.Env

	// DirectusClient is the shared Directus API client
	DirectusClient *tasks.DirectusClient

	// mu protects Data from concurrent access
	mu sync.RWMutex

	// Data holds pipeline-specific state that can be set by individual pipelines
	// Use type assertions to retrieve typed values
	Data map[string]interface{}
}

// NewState creates a new pipeline state with initialized maps
func NewState(cfg *configs.Env) *State {
	return &State{
		Config:         cfg,
		DirectusClient: tasks.NewDirectusClient(cfg.CMSBaseURL, cfg.DirectusCMSAPIKey),
		Data:           make(map[string]interface{}),
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
