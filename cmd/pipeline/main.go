package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/trackvision/tv-pipelines-template/configs"
	"github.com/trackvision/tv-pipelines-template/pipelines"
	_ "github.com/trackvision/tv-pipelines-template/pipelines/template" // Register template pipeline
	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/pipelines", pipelinesHandler)
	http.HandleFunc("/run/template", runTemplateHandler)

	logger.Info("Starting pipeline service", zap.String("port", port))
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func pipelinesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pipelines": pipelines.List(),
	})
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
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	_ = state // suppress unused warning for template

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runResponse{
		Success:  true,
		Pipeline: "template",
		ID:       req.ID,
	})
}

func respondError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(runResponse{
		Success: false,
		Error:   msg,
	})
}
