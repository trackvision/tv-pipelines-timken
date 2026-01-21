package pipelines

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func init() {
	// Initialize a no-op logger for tests
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func TestFlow_SimpleTask(t *testing.T) {
	executed := false

	flow := NewFlow("test")
	flow.AddTask("task1", func() error {
		executed = true
		return nil
	})

	err := flow.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !executed {
		t.Error("task1 was not executed")
	}
}

func TestFlow_DependencyOrder(t *testing.T) {
	var order []string

	flow := NewFlow("test")
	flow.AddTask("first", func() error {
		order = append(order, "first")
		return nil
	})
	flow.AddTask("second", func() error {
		order = append(order, "second")
		return nil
	}, "first")

	err := flow.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Errorf("execution order = %v, want [first, second]", order)
	}
}

func TestFlow_ParallelTasks(t *testing.T) {
	started := make(chan string, 2)

	flow := NewFlow("test")
	flow.AddTask("a", func() error {
		started <- "a"
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	flow.AddTask("b", func() error {
		started <- "b"
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	err := flow.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Both tasks should have started (order doesn't matter for parallel)
	close(started)
	count := 0
	for range started {
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 tasks to start, got %d", count)
	}
}

func TestFlow_TaskError(t *testing.T) {
	expectedErr := errors.New("task failed")

	flow := NewFlow("test")
	flow.AddTask("failing", func() error {
		return expectedErr
	})

	err := flow.Run(context.Background())
	if err == nil {
		t.Fatal("Run() expected error")
	}
}

func TestFlow_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	flow := NewFlow("test")
	flow.AddTask("blocking", func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	// Cancel immediately
	cancel()

	err := flow.Run(ctx)
	if err == nil {
		t.Fatal("Run() expected context cancellation error")
	}
}
