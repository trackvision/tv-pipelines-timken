package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"

	"tv-pipelines-timken/configs"
	"tv-pipelines-timken/pipelines"
	"tv-pipelines-timken/pipelines/coc"
	"tv-pipelines-timken/tasks"
	"tv-pipelines-timken/types"
)

//go:embed templates/*.html
var templatesFS embed.FS

// PipelineFunc is the standard signature for all pipelines
type PipelineFunc func(ctx context.Context, cms *tasks.DirectusClient, cfg *configs.Config, sscc string) (*types.PipelineResult, error)

// Pipeline registry - simple map
var pipelineRegistry = map[string]PipelineFunc{
	"coc": coc.Run,
}

// pipelineSteps maps pipeline names to their step names (for API discovery)
var pipelineSteps = map[string][]string{
	"coc": coc.Steps,
}

// API response types
type jobListResponse struct {
	Jobs []string `json:"jobs"`
}

type jobInfoResponse struct {
	Name     string   `json:"name"`
	Tasks    []string `json:"tasks"`
	Schedule string   `json:"schedule"`
}

// authMiddleware checks for valid API key in Authorization header or X-API-Key header
func authMiddleware(apiKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If no API key configured, skip auth
		if apiKey == "" {
			next(w, r)
			return
		}

		// Check Authorization: Bearer <key>
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == apiKey {
				next(w, r)
				return
			}
		}

		// Check X-API-Key header
		if r.Header.Get("X-API-Key") == apiKey {
			next(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	}
}

func main() {
	// Load configuration
	cfg, err := configs.Load()
	if err != nil {
		logger.Fatal("failed to load configuration", zap.Error(err))
	}

	// Create Directus client
	cms := tasks.NewDirectusClient(cfg)

	// Parse templates
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		logger.Fatal("failed to parse templates", zap.Error(err))
	}

	// Create HTTP server
	mux := http.NewServeMux()

	// Health check (no auth required)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	// API endpoints (auth required)
	mux.HandleFunc("/jobs", authMiddleware(cfg.APIKey, jobsHandler))
	mux.HandleFunc("/jobs/", authMiddleware(cfg.APIKey, jobInfoHandler))
	mux.HandleFunc("/run/coc", authMiddleware(cfg.APIKey, handlePipeline("coc", cms, cfg)))

	// Logs endpoint (auth required)
	mux.HandleFunc("/logs", authMiddleware(cfg.APIKey, makeLogsHandler(cfg)))

	// UI endpoints (no auth - for browser access)
	mux.HandleFunc("/", redirectToUI)
	mux.HandleFunc("/ui/", makeUIIndexHandler(tmpl))
	mux.HandleFunc("/ui/jobs/", makeUIJobHandler(tmpl))
	mux.HandleFunc("/ui/logs", makeUILogsHandler(tmpl, cfg))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting server",
			zap.String("port", cfg.Port),
			zap.Strings("pipelines", getPipelineNames()),
			zap.Bool("auth_enabled", cfg.APIKey != ""))
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

// jobsHandler returns list of all pipeline names (GET /jobs)
func jobsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jobListResponse{Jobs: getPipelineNames()})
}

// jobInfoHandler returns pipeline details (GET /jobs/{name})
func jobInfoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/jobs/")
	if name == "" {
		http.Error(w, "pipeline name required", http.StatusBadRequest)
		return
	}

	steps, ok := pipelineSteps[name]
	if !ok {
		http.Error(w, "unknown pipeline: "+name, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jobInfoResponse{
		Name:     name,
		Tasks:    steps,
		Schedule: "@manual",
	})
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

		pipeline, ok := pipelineRegistry[name]
		if !ok {
			writeError(w, http.StatusInternalServerError, "pipeline not found")
			return
		}

		// Build context with skip steps if provided
		ctx := r.Context()
		if len(req.SkipSteps) > 0 {
			ctx = context.WithValue(ctx, pipelines.SkipStepsKey, req.SkipSteps)
		}

		logger.Info("pipeline started",
			zap.String("pipeline", name),
			zap.String("sscc", req.SSCC),
			zap.Strings("skip_steps", req.SkipSteps))

		result, err := pipeline(ctx, cms, cfg, req.SSCC)
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
		_ = json.NewEncoder(w).Encode(types.PipelineResponse{
			Success:         result.Success,
			CertificationID: result.CertificationID,
			FileID:          result.FileID,
			EmailSent:       result.EmailSent,
			Error:           result.Error,
		})
	}
}

// redirectToUI redirects root to UI
func redirectToUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.Redirect(w, r, "/ui/", http.StatusFound)
		return
	}
	http.NotFound(w, r)
}

// makeUIIndexHandler returns UI index page showing all pipelines
func makeUIIndexHandler(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ui/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.ExecuteTemplate(w, "index.html", map[string]any{
			"Jobs": getPipelineNames(),
		})
	}
}

// makeUIJobHandler returns UI page for a specific pipeline
func makeUIJobHandler(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/ui/jobs/")
		if name == "" {
			http.Error(w, "pipeline name required", http.StatusBadRequest)
			return
		}

		steps, ok := pipelineSteps[name]
		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.ExecuteTemplate(w, "job.html", map[string]any{
			"Name":  name,
			"Tasks": steps,
		})
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(types.PipelineResponse{
		Success: false,
		Error:   message,
	})
}

// logsResponse is the response format for the /logs API
type logsResponse struct {
	Runs  []tasks.PipelineRun `json:"runs"`
	Count int                 `json:"count"`
	Query map[string]any      `json:"query"`
}

// makeLogsHandler returns logs from GCP Cloud Logging
func makeLogsHandler(cfg *configs.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check if logging is configured
		if cfg.GCPProjectID == "" || cfg.CloudRunService == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "logs not configured: set GCP_PROJECT_ID and CLOUD_RUN_SERVICE",
			})
			return
		}

		// Parse query parameters
		query := r.URL.Query()
		pipeline := query.Get("pipeline")
		severity := query.Get("severity")
		sinceStr := query.Get("since")
		limitStr := query.Get("limit")

		// Parse since duration
		since := time.Hour
		if sinceStr != "" {
			if d, err := time.ParseDuration(sinceStr); err == nil {
				since = d
			}
		}

		// Parse limit
		limit := 100
		if limitStr != "" {
			if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil && limit > 0 && limit <= 500 {
				// valid
			} else {
				limit = 100
			}
		}

		// Create log client
		ctx := r.Context()
		logClient, err := tasks.NewLogClient(ctx, cfg.GCPProjectID, cfg.CloudRunService)
		if err != nil {
			logger.Error("failed to create log client", zap.Error(err))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		defer func() { _ = logClient.Close() }()

		// Query logs
		logs, err := logClient.QueryLogs(ctx, tasks.LogQuery{
			ProjectID:   cfg.GCPProjectID,
			ServiceName: cfg.CloudRunService,
			Pipeline:    pipeline,
			Severity:    severity,
			Since:       since,
			Limit:       limit,
		})
		if err != nil {
			logger.Error("failed to query logs", zap.Error(err))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		// Group logs by pipeline run
		runs := tasks.GroupByRun(logs, cfg.GCPProjectID, cfg.CloudRunService)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(logsResponse{
			Runs:  runs,
			Count: len(runs),
			Query: map[string]any{
				"pipeline": pipeline,
				"severity": severity,
				"since":    sinceStr,
				"limit":    limit,
			},
		})
	}
}

// makeUILogsHandler returns the logs viewer UI page
func makeUILogsHandler(tmpl *template.Template, cfg *configs.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.ExecuteTemplate(w, "logs.html", map[string]any{
			"Configured":  cfg.GCPProjectID != "" && cfg.CloudRunService != "",
			"ProjectID":   cfg.GCPProjectID,
			"ServiceName": cfg.CloudRunService,
			"Pipelines":   getPipelineNames(),
		})
	}
}

func getPipelineNames() []string {
	names := make([]string, 0, len(pipelineRegistry))
	for name := range pipelineRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
