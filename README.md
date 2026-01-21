# Pipeline Service Template

Multi-pipeline ETL service template. Runs as a Cloud Run Service with HTTP API endpoints.

## Architecture

```
+-------------------------------------------------------------+
|                    Cloud Run Service                        |
+-------------------------------------------------------------+
|  HTTP API (:8080)                                           |
|  +-- /health, /pipelines      -> Service info               |
|  +-- /run/<pipeline>          -> Execute pipeline           |
|  +-- /api/*, /stream          -> Goflow DAG API             |
+-------------------------------------------------------------+
|  Goflow Engine (:8181 internal)                             |
|  +-- Pipeline visualization & execution tracking            |
+-------------------------------------------------------------+
```

## Getting Started

1. Clone this template
2. Update `go.mod` module name
3. Create your pipeline in `pipelines/<name>/`
4. Add HTTP handler in `cmd/pipeline/main.go`
5. Configure environment variables

## Project Structure

```
tv-pipelines-template/
+-- cmd/
|   +-- pipeline/
|       +-- main.go              # HTTP API server
+-- pipelines/
|   +-- registry.go              # Pipeline interface + registry
|   +-- state.go                 # Shared pipeline state (config, DB, Directus client)
|   +-- template/                # Example pipeline - copy for new pipelines
|       +-- pipeline.go
|       +-- config.go
+-- configs/
|   +-- env.go                   # Common config (Directus, SMTP, Database)
+-- tasks/                       # Reusable task implementations
|   +-- directus_client.go       # Directus REST API client (POST, PATCH, upload)
|   +-- fetch_directus_data.go   # Example: fetch data from Directus API
|   +-- fetch_tidb_data.go       # Example: fetch data from TiDB using sqlx
|   +-- send_email.go            # Send emails via SMTP
|   +-- generate_pdf.go          # Generate PDFs via headless Chrome
|   +-- ...
+-- types/                       # Shared types
+-- Dockerfile
+-- Makefile
```

## Tasks

### Directus Client (`directus_client.go`)

Reusable client for Directus REST API operations:

```go
client := tasks.NewDirectusClient(baseURL, apiKey)

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

### TiDB/MySQL Access (`fetch_tidb_data.go`)

For database access, use `sqlx` directly. The example shows common patterns:

```go
// Single record by field
product, err := tasks.FetchProductByGTIN(ctx, db, "01234567890123")

// Multiple records with IN clause
products, err := tasks.FetchProductsByGTINs(ctx, db, []string{"gtin1", "gtin2"})
```

Connection is managed via `tv-shared-go/database` in `pipelines/state.go`.

## API Endpoints

### Pipeline Execution

```
GET  /health          # Health check
GET  /pipelines       # List available pipelines
POST /run/<pipeline>  # Run pipeline with JSON body
```

### Goflow DAG API

```
GET  /api/jobs                    # List registered jobs
GET  /api/jobs/{name}             # Job details + DAG structure
GET  /api/executions              # List job executions
POST /api/jobs/{name}/submit      # Submit job for execution
GET  /stream                      # SSE real-time execution updates
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | No | HTTP server port (default: 8080) |
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

## Development

```bash
# Build
make build

# Run HTTP server locally
make run

# List available pipelines
make list

# Run tests
make test

# Run integration tests
go run cmd/test-integration/main.go --help
```

## Deployment

Docker image is automatically built and pushed to `ghcr.io/trackvision/tv-pipelines-template` on push to master.

Deploy as a Cloud Run Service:

```bash
gcloud run deploy <service-name> \
  --image ghcr.io/trackvision/tv-pipelines-template:latest \
  --timeout=3600 \
  --set-env-vars="CMS_BASE_URL=...,DIRECTUS_CMS_API_KEY=..."
```

## Adding New Pipelines

1. Copy `pipelines/template/` to `pipelines/<name>/`
2. Update package name and pipeline descriptor
3. Implement your pipeline logic using tasks from `tasks/`
4. Add HTTP handler in `cmd/pipeline/main.go`
5. Register any new environment variables in `configs/env.go`
