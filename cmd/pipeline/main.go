package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/trackvision/tv-pipelines-template/configs"
	"github.com/trackvision/tv-pipelines-template/pipelines"
	_ "github.com/trackvision/tv-pipelines-template/pipelines/template" // Register template pipeline
	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

// maxRequestBodySize limits request body to prevent memory exhaustion
const maxRequestBodySize = 1 << 20 // 1 MB

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/pipelines", pipelinesHandler)
	mux.HandleFunc("/run/template", runTemplateHandler)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Minute, // Long timeout for pipeline execution
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	done := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		logger.Info("Shutting down server...")

		// Cloud Run gives 10 seconds for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("Server shutdown error", zap.Error(err))
		}
		close(done)
	}()

	logger.Info("Starting pipeline service", zap.String("port", port))
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Fatal("Server failed", zap.Error(err))
	}

	<-done
	logger.Info("Server stopped")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "healthy"}); err != nil {
		logger.Error("Failed to encode health response", zap.Error(err))
	}
}

func pipelinesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"pipelines": pipelines.List(),
	}); err != nil {
		logger.Error("Failed to encode pipelines response", zap.Error(err))
	}
}

type runRequest struct {
	ID string `json:"id"`
}

type runResponse struct {
	Success  bool   `json:"success"`
	Pipeline string `json:"pipeline"`
	ID       string `json:"id"`
	Error    string `json:"error,omitempty"`
}

func runTemplateHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context() // Use request context for cancellation propagation

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size to prevent memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		respondError(w, "id is required", http.StatusBadRequest)
		return
	}

	// Load config - in a real app, parse from environment
	cfg := &configs.Env{
		CMSBaseURL:        os.Getenv("CMS_BASE_URL"),
		DirectusCMSAPIKey: os.Getenv("DIRECTUS_CMS_API_KEY"),
	}

	// Create pipeline state
	state := pipelines.NewState(cfg)
	defer state.Close() // Clean up resources when done

	// Create and run pipeline
	// TODO: Import and use your pipeline package
	// pipeline, err := template.New(state, req.ID)
	// if err != nil {
	//     respondError(w, err.Error(), http.StatusInternalServerError)
	//     return
	// }
	// if err := pipeline.RunOnce(); err != nil {
	//     respondError(w, err.Error(), http.StatusInternalServerError)
	//     return
	// }

	_ = ctx   // Pass ctx to pipeline operations for cancellation
	_ = state // suppress unused warning for template

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(runResponse{
		Success:  true,
		Pipeline: "template",
		ID:       req.ID,
	}); err != nil {
		logger.Error("Failed to encode run response", zap.Error(err))
	}
}

func respondError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(runResponse{
		Success: false,
		Error:   msg,
	}); err != nil {
		logger.Error("Failed to encode error response", zap.Error(err))
	}
}
