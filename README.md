# Pipeline Service Template

Go pipeline service template for Cloud Run. Uses Goflow for DAG-based pipeline execution with HTTP API.

## Features

- HTTP API for pipeline execution
- Goflow DAG orchestration with retries
- Directus CMS integration (REST API client)
- TiDB/MySQL database access via sqlx
- PDF generation via headless Chrome
- Email sending via SMTP
- Structured JSON logging

## Quick Start

```bash
# Clone and rename
git clone https://github.com/trackvision/tv-pipelines-template my-pipeline-service
cd my-pipeline-service

# Update module name in go.mod
# Update import paths throughout codebase

# Install dependencies
make deps

# Run locally
export CMS_BASE_URL="https://your-directus.com"
export DIRECTUS_CMS_API_KEY="your-api-key"
make run

# Test endpoint
curl -X POST http://localhost:8080/run/template -d '{"id":"test-123"}'
```

## Project Structure

```
tv-pipelines-template/
├── cmd/pipeline/main.go      # HTTP API server
├── configs/env.go            # Shared configuration
├── pipelines/
│   ├── registry.go           # Pipeline interface and registry
│   ├── state.go              # Shared state (config, DB, Directus)
│   └── template/             # Example pipeline - copy for new pipelines
│       ├── config.go         # Pipeline-specific config
│       └── pipeline.go       # Pipeline with Goflow operators
├── tasks/                    # Reusable task implementations
│   ├── directus_client.go    # Directus REST API (POST, PATCH, upload)
│   ├── fetch_directus_data.go
│   ├── fetch_tidb_data.go    # TiDB queries with sqlx
│   ├── send_email.go
│   └── generate_pdf.go
├── types/types.go            # Shared type definitions
├── Dockerfile
├── Makefile
└── .github/workflows/ci.yml
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/pipelines` | List available pipelines |
| POST | `/run/<pipeline>` | Execute pipeline with JSON body |

### Example Request

```bash
curl -X POST http://localhost:8080/run/template \
  -H "Content-Type: application/json" \
  -d '{"id": "test-123"}'
```

### Example Response

```json
{
  "success": true,
  "pipeline": "template",
  "id": "test-123"
}
```

## Creating a New Pipeline

1. **Copy the template**
   ```bash
   cp -r pipelines/template pipelines/mypipeline
   ```

2. **Update `pipelines/mypipeline/config.go`**
   - Change package name to `mypipeline`
   - Define pipeline-specific environment variables
   - Implement `Validate()` for required config

3. **Update `pipelines/mypipeline/pipeline.go`**
   - Change package name to `mypipeline`
   - Update `init()` with pipeline name, description, flags
   - Define state key constants
   - Implement task operators (`FetchDataOp`, `ProcessDataOp`, etc.)
   - Configure DAG edges in `setupDAGEdges()`

4. **Register the pipeline in `cmd/pipeline/main.go`**
   ```go
   import _ "github.com/trackvision/tv-pipelines-template/pipelines/mypipeline"
   
   http.HandleFunc("/run/mypipeline", runMyPipelineHandler)
   ```

5. **Add any shared types to `types/types.go`**

## Task Examples

### Directus API

```go
client := state.DirectusClient

// Create item
result, err := client.PostItem(ctx, "products", map[string]any{
    "name": "Widget",
    "status": "published",
})

// Update item
err := client.PatchItem(ctx, "products", "item-uuid", map[string]any{
    "status": "archived",
})

// Upload file
result, err := client.UploadFile(ctx, tasks.UploadFileParams{
    Filename: "report.pdf",
    Content:  pdfBytes,
    FolderID: "folder-uuid",
})
```

### TiDB/MySQL

```go
// Single record
var product types.Product
err := db.GetContext(ctx, &product, 
    "SELECT * FROM product WHERE gtin = ?", gtin)

// Multiple records with IN clause
query, args, err := sqlx.In(
    "SELECT * FROM product WHERE gtin IN (?)", gtins)
query = db.Rebind(query)
err = db.SelectContext(ctx, &products, query, args...)
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | 8080 | HTTP server port |
| `CMS_BASE_URL` | Yes | - | Directus CMS base URL |
| `DIRECTUS_CMS_API_KEY` | Yes | - | Directus API key |
| `DATABASE_HOST` | No | - | TiDB/MySQL host |
| `DATABASE_PORT` | No | 4000 | Database port |
| `DATABASE_NAME` | No | - | Database name |
| `DATABASE_USER` | No | - | Database user |
| `DATABASE_SSL` | No | true | Enable SSL |
| `EMAIL_SMTP_HOST` | No | smtp.resend.com | SMTP host |
| `EMAIL_SMTP_PORT` | No | 587 | SMTP port |
| `EMAIL_SMTP_USER` | No | resend | SMTP user |
| `EMAIL_SMTP_PASSWORD` | No | - | SMTP password |

## Development

```bash
# Build
make build

# Run HTTP server
make run

# List pipelines
make list

# Run tests
make test

# Format code
make fmt

# Lint
make lint
```

## CI/CD

GitHub Actions workflow (`.github/workflows/ci.yml`):

- **Test job**: Runs on all pushes and PRs
- **Build job**: Builds and pushes Docker image on main/master

### Required Secrets

| Secret | Description |
|--------|-------------|
| `GH_PAT` | GitHub PAT for private Go modules (`github.com/trackvision/*`) |
| `GITHUB_TOKEN` | Auto-provided for GHCR push |

### Docker Image

Images are pushed to `ghcr.io/trackvision/tv-pipelines-template` with tags:
- `latest` - latest main/master build
- `<sha>` - commit SHA for specific versions

## Deployment

Deploy as a Cloud Run Service:

```bash
gcloud run deploy my-pipeline-service \
  --image ghcr.io/trackvision/tv-pipelines-template:latest \
  --region us-central1 \
  --timeout=3600 \
  --memory=512Mi \
  --set-env-vars="CMS_BASE_URL=https://your-directus.com" \
  --set-secrets="DIRECTUS_CMS_API_KEY=directus-key:latest"
```

## Dependencies

Key dependencies from `go.mod`:

- `github.com/fieldryand/goflow/v2` - DAG workflow engine
- `github.com/jmoiron/sqlx` - SQL extensions for Go
- `github.com/chromedp/chromedp` - Headless Chrome for PDF generation
- `github.com/trackvision/tv-shared-go/*` - Shared TrackVision libraries

## License

Internal TrackVision template - not for public distribution.
