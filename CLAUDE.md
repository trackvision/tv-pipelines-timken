# Pipeline Service Template - Claude Code Instructions

## IMPORTANT: Code Quality Checks

**Before any commit or push, always run:**
```bash
make check   # Runs go vet, golangci-lint, and tests
```

**First time setup (after cloning):**
```bash
make setup-hooks   # Installs pre-push hook that enforces checks
```

If the user asks to commit or push changes:
1. First run `make check` to verify all checks pass
2. If checks fail, fix the issues before committing
3. Remind the user to run `make setup-hooks` if they haven't already

The pre-push hook automatically blocks pushes if `go vet`, `golangci-lint`, or tests fail.

## Project Overview

Go pipeline service template for Cloud Run. HTTP API accepts POST requests with pipeline parameters, executes using Goflow for DAG orchestration, and returns JSON results.

## Architecture

```
cmd/pipeline/main.go     - HTTP API server (health, list, run endpoints)
configs/env.go           - Shared environment configuration
pipelines/
  registry.go            - Pipeline interface and thread-safe registry
  state.go               - Shared state between tasks (config, DB, Directus client)
  template/              - Example pipeline to copy for new pipelines
    config.go            - Pipeline-specific configuration
    pipeline.go          - Pipeline implementation with Goflow operators
tasks/                   - Reusable task implementations
  directus_client.go     - Directus REST API client (POST, PATCH, upload)
  fetch_directus_data.go - Fetch data from Directus API
  fetch_tidb_data.go     - Fetch data from TiDB using sqlx
  send_email.go          - SMTP email sending
  generate_pdf.go        - PDF generation via headless Chrome
types/types.go           - Shared type definitions
```

## Key Patterns

### Pipeline Interface

All pipelines implement `pipelines.Pipeline`:

```go
type Pipeline interface {
    Name() string
    Description() string
    ValidateConfig() error
    Job() func() *goflow.Job
    RunOnce() error
}
```

### Adding a New Pipeline

1. Copy `pipelines/template/` to `pipelines/<name>/`

2. Update `config.go` with pipeline-specific env vars:
```go
package mypipeline

type Config struct {
    APIEndpoint string
}

func LoadConfig() (*Config, error) {
    return &Config{
        APIEndpoint: os.Getenv("MYPIPELINE_API_ENDPOINT"),
    }, nil
}

func (c *Config) Validate() error {
    if c.APIEndpoint == "" {
        return fmt.Errorf("MYPIPELINE_API_ENDPOINT is required")
    }
    return nil
}
```

3. Update `pipeline.go`:
   - Change package name
   - Update `init()` to register descriptor with correct name/description/flags
   - Define state keys as constants
   - Implement operators for each task
   - Wire up DAG edges in `setupDAGEdges()`

4. Add HTTP handler in `cmd/pipeline/main.go`:
```go
import _ "github.com/trackvision/tv-pipelines-template/pipelines/mypipeline"

http.HandleFunc("/run/mypipeline", runMyPipelineHandler)
```

### Pipeline State

Thread-safe state sharing between tasks:

```go
// Define constants for state keys
const (
    KeyFetchedData   = "fetched_data"
    KeyProcessedData = "processed_data"
)

// Set data (thread-safe)
state.Set(KeyFetchedData, data)

// Get data with type assertion
data := state.Get(KeyFetchedData).(*MyType)

// Context for cancellation is available via state.Ctx
if err := state.Ctx.Err(); err != nil {
    return nil, fmt.Errorf("cancelled: %w", err)
}
```

### Goflow Operators

Each task is a struct implementing `goflow.Operator`:

```go
type FetchDataOp struct {
    pipeline *Pipeline
}

func (o *FetchDataOp) Run() (interface{}, error) {
    // Access config via o.pipeline.config
    // Access shared state via o.pipeline.state
    // Access DB via o.pipeline.state.DB
    // Access Directus via o.pipeline.state.DirectusClient
    return nil, nil
}
```

### Database Access (TiDB/MySQL)

Use `sqlx` with parameterized queries:

```go
// Single record
var product types.Product
err := db.GetContext(ctx, &product, "SELECT * FROM product WHERE gtin = ?", gtin)

// Multiple records with IN clause
query, args, err := sqlx.In("SELECT * FROM product WHERE gtin IN (?)", gtins)
query = db.Rebind(query)
err = db.SelectContext(ctx, &products, query, args...)
```

### Directus API

Use the shared client from state:

```go
client := state.DirectusClient

// Create item
result, err := client.PostItem(ctx, "collection_name", item)

// Update item
err := client.PatchItem(ctx, "collection_name", "item-id", updates)

// Upload file
result, err := client.UploadFile(ctx, tasks.UploadFileParams{
    Filename: "file.pdf",
    Content:  pdfBytes,
    FolderID: "folder-uuid",
})
```

## Getting Started

```bash
# Install dependencies
make deps

# Install pre-push git hooks (runs go vet, golangci-lint, tests before push)
make setup-hooks

# Install golangci-lint if not already installed
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Development Workflow

```bash
# Run all checks (vet, lint, test) before committing
make check

# Individual commands
make vet      # Run go vet
make lint     # Run golangci-lint
make test     # Run tests
make fmt      # Format code
```

## Testing

```bash
# Run all tests
go test ./...

# Run specific test
go test -v ./tasks/... -run TestFetchProduct

# Run HTTP server locally
make run

# Test endpoint
curl -X POST http://localhost:8080/run/template -d '{"id":"test-123"}'
```

## CI/CD

- GitHub Actions workflow in `.github/workflows/ci.yml`
- Uses `GH_PAT` secret for private Go modules (`github.com/trackvision/*`)
- Docker images pushed to `ghcr.io/trackvision/tv-pipelines-template`
- Builds on push to main/master branches

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | No | HTTP port (default: 8080) |
| `CMS_BASE_URL` | Yes | Directus CMS base URL |
| `DIRECTUS_CMS_API_KEY` | Yes | Directus API key |
| `DATABASE_HOST` | No | TiDB/MySQL host |
| `DATABASE_PORT` | No | Database port (default: 4000) |
| `DATABASE_NAME` | No | Database name |
| `DATABASE_USER` | No | Database user |
| `DATABASE_SSL` | No | Enable SSL (default: true) |
| `EMAIL_SMTP_HOST` | No | SMTP host (default: smtp.resend.com) |
| `EMAIL_SMTP_PORT` | No | SMTP port (default: 587) |
| `EMAIL_SMTP_USER` | No | SMTP user |
| `EMAIL_SMTP_PASSWORD` | No | SMTP password |

## Cloud Run Considerations

- **Stateless**: No persistent state between requests; use DB or external storage
- **Timeout**: Up to 60 minutes per request for long-running pipelines (configurable via `gcloud run services update --timeout=3600`)
- **Cold starts**: First request may be slower; keep binary small, minimize init work
- **Concurrency**: Multiple requests can hit same instance; state is per-request via handler

## Processing Massive Record Sets

When processing large datasets (e.g., 200K records), pipelines may be interrupted mid-execution. Use checkpointing to enable resumption.

### Checkpointing Strategies

| Scenario | Strategy | Description |
|----------|----------|-------------|
| Sequential tasks | Task-level | Track which tasks (fetch, process, save) completed |
| Batch processing | Cursor-based | Track last processed record ID |
| Idempotent ops | Mark-as-processed | Flag records when done |
| Unordered data | Batch numbers | Process in fixed-size chunks |

### Schema for Resumable Pipelines

```sql
-- Pipeline run tracking
CREATE TABLE pipeline_run (
    id VARCHAR(36) PRIMARY KEY,
    pipeline_name VARCHAR(100) NOT NULL,
    input_id VARCHAR(255) NOT NULL,
    status ENUM('running', 'completed', 'failed') NOT NULL DEFAULT 'running',
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    error_message TEXT,
    INDEX idx_resumable (pipeline_name, input_id, status)
);

-- Cursor for batch processing
CREATE TABLE pipeline_cursor (
    run_id VARCHAR(36) PRIMARY KEY,
    last_processed_id VARCHAR(255),
    processed_count INT DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### Cursor-Based Processing Pattern

```go
func (o *ProcessRecordsOp) Run() (interface{}, error) {
    ctx := o.pipeline.state.Ctx
    db := o.pipeline.state.DB
    runID := o.pipeline.state.GetString(KeyRunID)

    // Load cursor (resume point)
    var cursor string
    db.GetContext(ctx, &cursor,
        "SELECT last_processed_id FROM pipeline_cursor WHERE run_id = ?", runID)

    for {
        // Fetch batch after cursor
        var records []Record
        err := db.SelectContext(ctx, &records, `
            SELECT * FROM source_table
            WHERE id > ?
            ORDER BY id
            LIMIT 1000`, cursor)
        if err != nil || len(records) == 0 {
            break
        }

        for _, record := range records {
            if err := ctx.Err(); err != nil {
                return nil, fmt.Errorf("cancelled: %w", err)
            }

            process(record)
            cursor = record.ID

            // Checkpoint every 100 records
            if processedCount % 100 == 0 {
                db.ExecContext(ctx, `
                    INSERT INTO pipeline_cursor (run_id, last_processed_id, processed_count)
                    VALUES (?, ?, ?)
                    ON DUPLICATE KEY UPDATE
                        last_processed_id = VALUES(last_processed_id),
                        processed_count = VALUES(processed_count)`,
                    runID, cursor, processedCount)
            }
        }
    }
    return nil, nil
}
```

### Alternative: Mark-as-Processed

If operations are idempotent, mark records as processed:

```go
// Query only unprocessed records
records, _ := db.SelectContext(ctx, &records, `
    SELECT * FROM product
    WHERE pipeline_processed_at IS NULL
    LIMIT 1000`)

// After processing each record
db.ExecContext(ctx,
    "UPDATE product SET pipeline_processed_at = NOW() WHERE id = ?",
    record.ID)
```

This approach naturally resumes from where it left off without explicit cursor tracking.
