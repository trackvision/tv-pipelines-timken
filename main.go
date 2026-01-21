package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"

	"tv-pipelines-timken/configs"
	"tv-pipelines-timken/pipelines/coc"
	"tv-pipelines-timken/tasks"
	"tv-pipelines-timken/types"
)

// PipelineFunc is the standard signature for all pipelines
type PipelineFunc func(ctx context.Context, cms *tasks.DirectusClient, cfg *configs.Config, sscc string) (*types.PipelineResult, error)

// Pipeline registry - simple map
var pipelines = map[string]PipelineFunc{
	"coc": coc.Run,
}

func main() {
	// Load configuration
	cfg, err := configs.Load()
	if err != nil {
		logger.Fatal("failed to load configuration", zap.Error(err))
	}

	// Create Directus client
	cms := tasks.NewDirectusClient(cfg)

	// Create HTTP server
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	mux.HandleFunc("/run/coc", handlePipeline("coc", cms, cfg))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting server", zap.String("port", cfg.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}
	logger.Info("server stopped")
}

func handlePipeline(name string, cms *tasks.DirectusClient, cfg *configs.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req types.PipelineRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.SSCC == "" {
			writeError(w, http.StatusBadRequest, "sscc is required")
			return
		}

		pipeline, ok := pipelines[name]
		if !ok {
			writeError(w, http.StatusInternalServerError, "pipeline not found")
			return
		}

		logger.Info("pipeline started", zap.String("pipeline", name), zap.String("sscc", req.SSCC))

		result, err := pipeline(r.Context(), cms, cfg, req.SSCC)
		if err != nil {
			logger.Error("pipeline failed", zap.String("pipeline", name), zap.Error(err))
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("pipeline complete", zap.String("pipeline", name), zap.Bool("success", result.Success))

		w.Header().Set("Content-Type", "application/json")
		if !result.Success {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(types.PipelineResponse{
			Success:         result.Success,
			CertificationID: result.CertificationID,
			FileID:          result.FileID,
			EmailSent:       result.EmailSent,
			Error:           result.Error,
		})
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(types.PipelineResponse{
		Success: false,
		Error:   message,
	})
}
