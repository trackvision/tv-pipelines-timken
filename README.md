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

## Project Structure

```
tv-pipelines-template/
+-- cmd/
|   +-- pipeline/
|       +-- main.go              # HTTP API server
+-- pipelines/
|   +-- registry.go              # Pipeline interface + registry
|   +-- state.go                 # Generic pipeline state
|   +-- <name>/
|       +-- pipeline.go          # Pipeline implementation
|       +-- config.go            # Pipeline-specific config
+-- configs/
|   +-- env.go                   # Common config (Directus, SMTP)
+-- tasks/                       # Reusable task implementations
+-- types/                       # Shared types
+-- Dockerfile
+-- Makefile
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
```

## Deployment

Deploy as a Cloud Run Service:

```bash
gcloud run deploy <service-name> \
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
