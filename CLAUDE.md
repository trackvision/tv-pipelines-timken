# Timken Pipeline Service - Claude Code Instructions

## IMPORTANT: Code Quality Checks

**Before any commit or push, always run:**
```bash
make check   # Runs go vet, golangci-lint, and tests
```

**First time setup (after cloning):**
```bash
make setup-hooks   # Installs pre-push hook that enforces checks
```

## Project Overview

Go pipeline service for Cloud Run that generates Certificate of Conformance (COC) documents for Timken shipments. HTTP API triggers pipelines via `POST /run/coc`.

## Architecture

```
main.go                  - HTTP server + pipeline registry + UI
pipelines/
  flow.go                - Fluent AddTask API with goflow (retries, skip steps)
  coc/pipeline.go        - COC certificate generation pipeline
tasks/                   - Reusable task implementations
  directus.go            - Directus CMS client
  pdf.go                 - PDF generation with chromedp
  email.go               - Email sending
  coc_data.go            - COC data fetching
  gcp_logging.go         - GCP Cloud Logging integration
configs/                 - Environment configuration
types/                   - Shared type definitions
templates/               - HTML templates for web UI
```

## COC Pipeline

The COC pipeline generates Certificate of Conformance documents:

1. **generate_pdf** - Render COC viewer page to PDF using chromedp
2. **fetch_coc_data** - Fetch shipment data from COC API (runs in parallel with generate_pdf)
3. **prepare_record** - Transform COC data into certification record
4. **create_certification** - Create certification record in Directus CMS
5. **upload_pdf** - Upload PDF to Directus and attach to certification
6. **send_email** - Email PDF to notification recipients

## Flow API

```go
flow := pipelines.NewFlow("name")

flow.AddTask("fetch", fetchFunc)                           // No dependencies
flow.AddTask("process", processFunc, "fetch")              // Depends on fetch
flow.AddTask("combine", combineFunc, "fetch1", "fetch2")   // Multiple deps

return flow.Run(ctx)
```

Features:
- Automatic retries (2 retries with 5s delay)
- Skip steps via context (for dry-run mode)
- Comprehensive logging per step

## HTTP API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/jobs` | GET | List all pipelines |
| `/jobs/{name}` | GET | Get pipeline details (steps, schedule) |
| `/run/coc` | POST | Run COC pipeline with `{"sscc": "...", "skip_steps": [...]}` |
| `/logs` | GET | Query GCP Cloud Logging |
| `/ui/` | GET | Web UI - pipeline list |
| `/ui/jobs/{name}` | GET | Web UI - pipeline details |
| `/ui/logs` | GET | Web UI - logs viewer |

## Directus API

```go
cms.PostItem(ctx, "collection", item)
cms.PatchItem(ctx, "collection", "id", updates)
cms.UploadFile(ctx, tasks.UploadFileParams{Filename: "f.pdf", Content: bytes})
```

## Development

```bash
make check    # Run vet, lint, tests
make run      # Start server locally
curl -X POST http://localhost:8080/run/coc -d '{"sscc":"123456789012345678"}'
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | No | HTTP port (default: 8080) |
| `CMS_API_KEY` | No | API key for request authentication |
| `CMS_BASE_URL` | Yes | Directus CMS base URL |
| `DIRECTUS_CMS_API_KEY` | Yes | Directus API key |
| `COC_VIEWER_BASE_URL` | Yes | COC viewer URL for PDF generation |
| `COC_DATA_API_URL` | Yes | COC data API endpoint |
| `COC_FOLDER_ID` | No | Directus folder ID for PDF storage |
| `EMAIL_FROM_ADDRESS` | Yes | Sender email address |
| `EMAIL_SMTP_HOST` | No | SMTP host (default: smtp.resend.com) |
| `EMAIL_SMTP_PORT` | No | SMTP port (default: 587) |
| `EMAIL_SMTP_USER` | No | SMTP user (default: resend) |
| `EMAIL_SMTP_PASSWORD` | No | SMTP password |
| `GCP_PROJECT_ID` | No | GCP project for logs viewer |
| `CLOUD_RUN_SERVICE` | No | Cloud Run service name for logs |

## Cloud Run

- **Stateless**: No persistent state between requests
- **Timeout**: Up to 60 minutes per request (PDF generation can be slow)
- **Concurrency**: State is per-request via closures
- **chromedp**: Uses headless Chrome for PDF generation (via chromedp/headless-shell base image)
