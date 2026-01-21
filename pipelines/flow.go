package pipelines

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Flow orchestrates task execution with dependency management
type Flow struct {
	name  string
	tasks map[string]*task
	order []string
}

type task struct {
	name    string
	fn      func() error
	deps    []string
	done    bool
	running bool
	err     error
}

// NewFlow creates a new pipeline flow
func NewFlow(name string) *Flow {
	return &Flow{
		name:  name,
		tasks: make(map[string]*task),
	}
}

// AddTask adds a task to the flow with optional dependencies
func (f *Flow) AddTask(name string, fn func() error, deps ...string) {
	f.tasks[name] = &task{
		name: name,
		fn:   fn,
		deps: deps,
	}
	f.order = append(f.order, name)
}

// Run executes all tasks in dependency order
func (f *Flow) Run(ctx context.Context) error {
	logger := zap.L().With(zap.String("pipeline", f.name))
	startTime := time.Now()

	// Log pipeline start with task list
	taskNames := make([]string, 0, len(f.order))
	for _, name := range f.order {
		taskNames = append(taskNames, name)
	}
	logger.Info("pipeline started",
		zap.Int("task_count", len(f.tasks)),
		zap.Strings("tasks", taskNames))

	completedCount := 0

	for {
		// Find tasks that can run (all deps satisfied, not done, not running)
		ready := f.findReadyTasks()
		if len(ready) == 0 {
			// Check if we're done or stuck
			if f.allDone() {
				elapsed := time.Since(startTime)
				logger.Info("pipeline completed",
					zap.Duration("duration", elapsed),
					zap.Int("tasks_completed", completedCount))
				return nil
			}
			// Check for errors
			for _, t := range f.tasks {
				if t.err != nil {
					return t.err
				}
			}
			return fmt.Errorf("pipeline %s: deadlock detected", f.name)
		}

		// Run ready tasks in parallel
		var wg sync.WaitGroup
		errChan := make(chan error, len(ready))

		for _, t := range ready {
			t.running = true
			wg.Add(1)
			go func(t *task) {
				defer wg.Done()
				taskStart := time.Now()
				logger.Info("step started", zap.String("step", t.name))

				if err := t.fn(); err != nil {
					t.err = fmt.Errorf("%s: %w", t.name, err)
					errChan <- t.err
					logger.Error("step failed",
						zap.String("step", t.name),
						zap.Error(err),
						zap.Duration("duration", time.Since(taskStart)))
				} else {
					t.done = true
					completedCount++
					logger.Info("step completed",
						zap.String("step", t.name),
						zap.Duration("duration", time.Since(taskStart)))
				}
				t.running = false
			}(t)
		}

		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			if err != nil {
				elapsed := time.Since(startTime)
				logger.Error("pipeline failed",
					zap.Duration("duration", elapsed),
					zap.Int("tasks_completed", completedCount),
					zap.Error(err))
				return err
			}
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}

func (f *Flow) findReadyTasks() []*task {
	var ready []*task
	for _, t := range f.tasks {
		if t.done || t.running || t.err != nil {
			continue
		}
		// Check if all deps are done
		allDepsDone := true
		for _, dep := range t.deps {
			if dt, ok := f.tasks[dep]; !ok || !dt.done {
				allDepsDone = false
				break
			}
		}
		if allDepsDone {
			ready = append(ready, t)
		}
	}
	return ready
}

func (f *Flow) allDone() bool {
	for _, t := range f.tasks {
		if !t.done {
			return false
		}
	}
	return true
}
