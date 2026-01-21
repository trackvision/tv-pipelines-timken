package template

import (
	"fmt"
	"time"

	"github.com/trackvision/tv-pipelines-template/pipelines"

	"github.com/fieldryand/goflow/v2"
	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

func init() {
	pipelines.RegisterDescriptor(pipelines.Descriptor{
		Name:        "template",
		Description: "Template pipeline - copy and customize for your use case",
		Flags:       []string{"--id"},
	})
}

// State keys for pipeline data
const (
	KeyID             = "id"
	KeyConfig         = "template_config"
	KeyFetchedData    = "fetched_data"
	KeyProcessedData  = "processed_data"
	KeyPipelineResult = "pipeline_result"

	// VisualizationID is a sentinel value used when creating pipelines for DAG visualization
	VisualizationID = "__VISUALIZATION__"
)

// Pipeline implements the template pipeline
type Pipeline struct {
	state  *pipelines.State
	config *Config
}

// New creates a new pipeline instance
func New(state *pipelines.State, id string) (*Pipeline, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	state.Set(KeyID, id)
	state.Set(KeyConfig, cfg)

	return &Pipeline{
		state:  state,
		config: cfg,
	}, nil
}

// Name returns the pipeline identifier
func (p *Pipeline) Name() string {
	return "template"
}

// Description returns a human-readable description
func (p *Pipeline) Description() string {
	return "Template pipeline - copy and customize for your use case"
}

// ValidateConfig validates that all required configuration is present
func (p *Pipeline) ValidateConfig() error {
	if err := p.config.Validate(); err != nil {
		return err
	}
	if p.state.GetString(KeyID) == "" {
		return fmt.Errorf("ID is required")
	}
	return nil
}

// Job returns a goflow job factory function
func (p *Pipeline) Job() func() *goflow.Job {
	return func() *goflow.Job {
		j := &goflow.Job{
			Name:     "template-pipeline",
			Schedule: "@manual",
			Active:   true,
		}

		// Task 1: Fetch data
		j.Add(&goflow.Task{
			Name:       "fetch_data",
			Operator:   &FetchDataOp{pipeline: p},
			Retries:    2,
			RetryDelay: goflow.ConstantDelay{Period: 5},
		})

		// Task 2: Process data (depends on Task 1)
		j.Add(&goflow.Task{
			Name:       "process_data",
			Operator:   &ProcessDataOp{pipeline: p},
			Retries:    2,
			RetryDelay: goflow.ConstantDelay{Period: 5},
		})

		// Task 3: Save results (depends on Task 2)
		j.Add(&goflow.Task{
			Name:       "save_results",
			Operator:   &SaveResultsOp{pipeline: p},
			Retries:    2,
			RetryDelay: goflow.ConstantDelay{Period: 5},
		})

		setupDAGEdges(j)
		return j
	}
}

// VisualizationJob returns a goflow job for UI visualization only (not for execution)
func (p *Pipeline) VisualizationJob() func() *goflow.Job {
	return func() *goflow.Job {
		j := &goflow.Job{
			Name:   "template-pipeline",
			Active: false, // Visualization only
		}

		// Add tasks with no-op operators (just for DAG display)
		j.Add(&goflow.Task{Name: "fetch_data", Operator: &noopOp{}})
		j.Add(&goflow.Task{Name: "process_data", Operator: &noopOp{}})
		j.Add(&goflow.Task{Name: "save_results", Operator: &noopOp{}})

		setupDAGEdges(j)
		return j
	}
}

// setupDAGEdges defines the task dependencies for the pipeline
func setupDAGEdges(j *goflow.Job) {
	// Retrieve tasks with validation
	fetchData := j.Task("fetch_data")
	processData := j.Task("process_data")
	saveResults := j.Task("save_results")

	// Validate all tasks exist to catch typos early
	for name, task := range map[string]*goflow.Task{
		"fetch_data":   fetchData,
		"process_data": processData,
		"save_results": saveResults,
	} {
		if task == nil {
			panic(fmt.Sprintf("task %q not found in job - check task name spelling", name))
		}
	}

	j.SetDownstream(fetchData, processData)
	j.SetDownstream(processData, saveResults)
}

// noopOp is a no-operation operator for visualization
type noopOp struct{}

func (o *noopOp) Run() (any, error) { return nil, nil }

// taskConfig defines retry configuration for a task
type taskConfig struct {
	name       string
	op         goflow.Operator
	retries    int
	retryDelay time.Duration
}

// RunOnce executes the pipeline synchronously with retry logic matching Job() config
func (p *Pipeline) RunOnce() error {
	id := p.state.GetString(KeyID)
	logger.Info("Running template pipeline", zap.String("id", id))

	// Task configuration matches Job() definition
	tasks := []taskConfig{
		{"fetch_data", &FetchDataOp{pipeline: p}, 2, 5 * time.Second},
		{"process_data", &ProcessDataOp{pipeline: p}, 2, 5 * time.Second},
		{"save_results", &SaveResultsOp{pipeline: p}, 2, 5 * time.Second},
	}

	for _, t := range tasks {
		if err := p.runTaskWithRetry(t); err != nil {
			return err
		}
	}

	logger.Info("Pipeline complete", zap.String("id", id))
	return nil
}

// runTaskWithRetry executes a task with retry logic
func (p *Pipeline) runTaskWithRetry(t taskConfig) error {
	var lastErr error

	for attempt := 0; attempt <= t.retries; attempt++ {
		// Check for cancellation before each attempt
		if err := p.state.Ctx.Err(); err != nil {
			return fmt.Errorf("task %s cancelled: %w", t.name, err)
		}

		if attempt > 0 {
			logger.Info("Retrying task",
				zap.String("task", t.name),
				zap.Int("attempt", attempt+1),
				zap.Int("max_attempts", t.retries+1),
			)
			time.Sleep(t.retryDelay)
		}

		if _, err := t.op.Run(); err != nil {
			lastErr = err
			logger.Warn("Task failed",
				zap.String("task", t.name),
				zap.Int("attempt", attempt+1),
				zap.Error(err),
			)
			continue
		}

		return nil // Success
	}

	return fmt.Errorf("task %s failed after %d attempts: %w", t.name, t.retries+1, lastErr)
}

// --- Custom Operators ---

// FetchDataOp fetches data from an external source
type FetchDataOp struct {
	pipeline *Pipeline
}

func (o *FetchDataOp) Run() (interface{}, error) {
	// Check for cancellation before starting
	if err := o.pipeline.state.Ctx.Err(); err != nil {
		return nil, fmt.Errorf("fetch_data cancelled: %w", err)
	}

	id := o.pipeline.state.GetString(KeyID)
	logger.Info("Task: fetch_data", zap.String("id", id))

	// TODO: Implement your data fetching logic here
	// Example:
	// data, err := tasks.FetchData(o.pipeline.state.Ctx, o.pipeline.config.ExampleAPIURL, id)
	// if err != nil {
	//     return nil, fmt.Errorf("fetch_data failed: %w", err)
	// }
	// o.pipeline.state.Set(KeyFetchedData, data)

	logger.Info("Task: fetch_data complete")
	return nil, nil
}

// ProcessDataOp processes the fetched data
type ProcessDataOp struct {
	pipeline *Pipeline
}

func (o *ProcessDataOp) Run() (interface{}, error) {
	// Check for cancellation before starting
	if err := o.pipeline.state.Ctx.Err(); err != nil {
		return nil, fmt.Errorf("process_data cancelled: %w", err)
	}

	id := o.pipeline.state.GetString(KeyID)
	logger.Info("Task: process_data", zap.String("id", id))

	// TODO: Implement your data processing logic here
	// Example:
	// fetchedData, err := getStateValue[types.FetchedData](o.pipeline, KeyFetchedData)
	// if err != nil {
	//     return nil, fmt.Errorf("process_data: %w", err)
	// }
	// processed := transform(fetchedData)
	// o.pipeline.state.Set(KeyProcessedData, processed)

	logger.Info("Task: process_data complete")
	return nil, nil
}

// SaveResultsOp saves the processed results
type SaveResultsOp struct {
	pipeline *Pipeline
}

func (o *SaveResultsOp) Run() (interface{}, error) {
	// Check for cancellation before starting
	if err := o.pipeline.state.Ctx.Err(); err != nil {
		return nil, fmt.Errorf("save_results cancelled: %w", err)
	}

	id := o.pipeline.state.GetString(KeyID)
	logger.Info("Task: save_results", zap.String("id", id))

	// TODO: Implement your save logic here
	// Example:
	// processedData, err := getStateValue[types.ProcessedData](o.pipeline, KeyProcessedData)
	// if err != nil {
	//     return nil, fmt.Errorf("save_results: %w", err)
	// }
	// result, err := tasks.SaveToDirectus(o.pipeline.state.Ctx, o.pipeline.state.DirectusClient, processedData)
	// if err != nil {
	//     return nil, fmt.Errorf("save_results failed: %w", err)
	// }
	// o.pipeline.state.Set(KeyPipelineResult, result)

	logger.Info("Task: save_results complete")
	return nil, nil
}
