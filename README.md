# Timken ETL

Multi-pipeline ETL service for Timken data processing. Runs as a Cloud Run Service with HTTP API endpoints.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Cloud Run Service                        │
├─────────────────────────────────────────────────────────────┤
│  HTTP API (:8080)                                           │
│  ├── /health, /pipelines      → Service info               │
│  ├── /run/coc                 → Execute pipeline           │
│  └── /api/*, /stream          → Goflow DAG API             │
├─────────────────────────────────────────────────────────────┤
│  Goflow Engine (:8181 internal)                             │
│  └── Pipeline visualization & execution tracking            │
└─────────────────────────────────────────────────────────────┘
```

### COC Pipeline Flow

```
┌──────────────┐     ┌──────────────┐
│ fetch_coc_   │     │ generate_    │
│    data      │     │    pdf       │
└──────┬───────┘     └──────┬───────┘
       │                    │
       └────────┬───────────┘
                ▼
       ┌────────────────┐
       │ prepare_record │
       └───────┬────────┘
               ▼
       ┌────────────────────┐
       │ create_certification│
       └───────┬────────────┘
               ▼
       ┌────────────────┐
       │   upload_pdf   │
       └───────┬────────┘
               ▼
       ┌────────────────┐
       │   send_email   │
       └────────────────┘
```

## Pipelines

| Pipeline | Description | Required Params |
|----------|-------------|-----------------|
| `coc` | Certificate of Conformance | `sscc` |

## API Endpoints

### Pipeline Execution

```
GET  /health          # Health check
GET  /pipelines       # List available pipelines
POST /run/coc         # Run COC pipeline
```

### Goflow DAG API

```
GET  /api/jobs                    # List registered jobs
GET  /api/jobs/{name}             # Job details + DAG structure
GET  /api/executions              # List job executions
POST /api/jobs/{name}/submit      # Submit job for execution
POST /api/jobs/{name}/toggle      # Toggle job schedule
GET  /stream                      # SSE real-time execution updates
```

### Run COC Pipeline

```bash
curl -X POST https://your-service/run/coc \
  -H "Content-Type: application/json" \
  -d '{"sscc": "100538930005550017"}'
```

**Response:**
```json
{
  "success": true,
  "pipeline": "coc",
  "sscc": "100538930005550017",
  "certification_id": "uuid",
  "file_id": "uuid",
  "email_sent": true
}
```

### Get Pipeline DAG

```bash
curl https://your-service/api/jobs/coc-pipeline
```

**Response:**
```json
{
  "job": "coc-pipeline",
  "tasks": ["fetch_coc_data", "generate_pdf", "prepare_record", "create_certification", "upload_pdf", "send_email"],
  "dag": {
    "fetch_coc_data": ["prepare_record"],
    "generate_pdf": ["prepare_record"],
    "prepare_record": ["create_certification"],
    "create_certification": ["upload_pdf"],
    "upload_pdf": ["send_email"],
    "send_email": []
  }
}
```

## Project Structure

```
timken-etl/
├── cmd/
│   └── pipeline/
│       └── main.go              # HTTP API server
├── pipelines/
│   ├── registry.go              # Pipeline interface + registry
│   ├── state.go                 # Generic pipeline state
│   └── coc/
│       ├── pipeline.go          # COC job definition + operators
│       └── config.go            # COC-specific config
├── configs/
│   └── env.go                   # Common config (Directus, SMTP)
├── tasks/                       # Reusable task implementations
├── types/                       # Shared types
└── Makefile
```

## Environment Variables

### Common (all pipelines)

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | No | HTTP server port (default: 8080) |
| `CMS_BASE_URL` | Yes | Directus CMS base URL |
| `DIRECTUS_CMS_API_KEY` | Yes | Directus API key |
| `EMAIL_SMTP_HOST` | No | SMTP host (default: smtp.resend.com) |
| `EMAIL_SMTP_PORT` | No | SMTP port (default: 587) |
| `EMAIL_SMTP_USER` | No | SMTP user (default: resend) |
| `EMAIL_SMTP_PASSWORD` | No | SMTP password |

### COC Pipeline

| Variable | Required | Description |
|----------|----------|-------------|
| `TIMKEN_COC_API_URL` | Yes | Timken COC API URL |
| `COC_VIEWER_BASE_URL` | Yes | COC viewer URL for PDF generation |
| `COC_PDF_FOLDER_ID` | Yes | Directus folder ID for PDFs |
| `COC_FROM_EMAIL` | No | Sender email address |

## Development

```bash
# Build
make build

# Run HTTP server locally
make run

# Run pipeline once via CLI
make run-once SSCC=100538930005550017

# List available pipelines
make list

# Run tests
make test
```

## Deployment

Deploy as a Cloud Run Service:

```bash
gcloud run deploy timken-etl \
  --source . \
  --timeout=3600 \
  --set-env-vars="CMS_BASE_URL=...,DIRECTUS_CMS_API_KEY=..."
```

## Adding New Pipelines

1. Create `pipelines/<name>/config.go` with pipeline-specific config
2. Create `pipelines/<name>/pipeline.go` implementing the `Pipeline` interface
3. Register descriptor in `init()`:
   ```go
   func init() {
       pipelines.RegisterDescriptor(pipelines.Descriptor{
           Name:        "name",
           Description: "Description",
           Flags:       []string{"--param"},
       })
   }
   ```
4. Add handler in `cmd/pipeline/main.go`
