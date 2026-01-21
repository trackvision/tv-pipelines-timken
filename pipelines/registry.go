package pipelines

import (
	"fmt"
	"sort"
	"sync"

	"github.com/fieldryand/goflow/v2"
)

// Pipeline defines the interface that all pipelines must implement
type Pipeline interface {
	// Name returns the unique identifier for this pipeline
	Name() string

	// Description returns a human-readable description of the pipeline
	Description() string

	// ValidateConfig validates that all required configuration is present
	ValidateConfig() error

	// Job returns a goflow job factory function
	Job() func() *goflow.Job

	// RunOnce executes the pipeline synchronously and returns any error
	RunOnce() error
}

// Descriptor provides metadata about a pipeline for listing/discovery
type Descriptor struct {
	Name        string
	Description string
	Flags       []string // Required flags for this pipeline
}

var (
	registry    = make(map[string]Pipeline)
	descriptors = make(map[string]Descriptor)
	mu          sync.RWMutex
)

// RegisterDescriptor registers a pipeline descriptor for discovery
func RegisterDescriptor(d Descriptor) {
	mu.Lock()
	defer mu.Unlock()
	descriptors[d.Name] = d
}

// Register adds a pipeline instance to the registry
func Register(p Pipeline) {
	mu.Lock()
	defer mu.Unlock()
	registry[p.Name()] = p
}

// Get returns a pipeline by name, or nil if not found
func Get(name string) Pipeline {
	mu.RLock()
	defer mu.RUnlock()
	return registry[name]
}

// GetDescriptor returns a pipeline descriptor by name
func GetDescriptor(name string) (Descriptor, bool) {
	mu.RLock()
	defer mu.RUnlock()
	d, ok := descriptors[name]
	return d, ok
}

// listNamesLocked returns sorted descriptor names. Caller must hold mu.
func listNamesLocked() []string {
	names := make([]string, 0, len(descriptors))
	for name := range descriptors {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// List returns a sorted list of all registered pipeline descriptor names
func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	return listNamesLocked()
}

// All returns all registered pipelines
func All() []Pipeline {
	mu.RLock()
	defer mu.RUnlock()
	pipelines := make([]Pipeline, 0, len(registry))
	for _, p := range registry {
		pipelines = append(pipelines, p)
	}
	return pipelines
}

// ListWithDescriptions returns a formatted string of all pipelines with descriptions
func ListWithDescriptions() string {
	mu.RLock()
	defer mu.RUnlock()

	if len(descriptors) == 0 {
		return "No pipelines registered"
	}

	names := listNamesLocked()
	result := "Available pipelines:\n"
	for _, name := range names {
		d := descriptors[name]
		flagInfo := ""
		if len(d.Flags) > 0 {
			flagInfo = fmt.Sprintf(" (requires: %v)", d.Flags)
		}
		result += fmt.Sprintf("  %s - %s%s\n", name, d.Description, flagInfo)
	}
	return result
}
