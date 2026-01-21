# Timken ETL - Claude Code Instructions

## Project Overview

Multi-pipeline ETL service deployed as a Cloud Run Service. HTTP API accepts POST requests with pipeline name and parameters, executes the pipeline synchronously, and returns results as JSON.

## Architecture

- **HTTP API Server** (`cmd/pipeline/main.go`): Handles requests, routes to pipelines
- **Pipeline Registry** (`pipelines/registry.go`): Pipeline interface and discovery
- **Pipeline State** (`pipelines/state.go`): Shared state between pipeline tasks
- **Individual Pipelines** (`pipelines/<name>/`): Each pipeline has its own config and implementation
- **Tasks** (`tasks/`): Reusable task implementations (Directus, email, PDF generation)

## Key Patterns

### Adding a New Pipeline

1. Create `pipelines/<name>/config.go`:
   ```go
   package name

   type Config struct {
       SomeField string
   }

   func LoadConfig() (*Config, error) {
       return &Config{
           SomeField: os.Getenv("NAME_SOME_FIELD"),
       }, nil
   }
   ```

2. Create `pipelines/<name>/pipeline.go`:
   ```go
   package name

   func init() {
       pipelines.RegisterDescriptor(pipelines.Descriptor{
           Name:        "name",
           Description: "Description here",
           Flags:       []string{"--param"},
       })
   }

   type Pipeline struct {
       state  *pipelines.State
       config *Config
   }

   func New(state *pipelines.State, param string) (*Pipeline, error) { ... }
   func (p *Pipeline) Name() string { return "name" }
   func (p *Pipeline) Description() string { return "..." }
   func (p *Pipeline) ValidateConfig() error { ... }
   func (p *Pipeline) Job() func() *goflow.Job { ... }
   func (p *Pipeline) RunOnce() error { ... }
   ```

3. Add HTTP handler in `cmd/pipeline/main.go`

### Pipeline State Keys

Use constants for state keys to avoid typos:
```go
const (
    KeyMyData = "my_data"
)
state.Set(KeyMyData, data)
data := state.Get(KeyMyData).(*MyType)
```

## Testing

```bash
# Run all tests
go test ./...

# Test specific SSCC
make run-once SSCC=100538930005550017

# Test via HTTP API locally
make run
curl -X POST http://localhost:8080/run/coc -d '{"sscc":"100538930005550017"}'
```

## Environment

- **Runtime**: Cloud Run Service (max 60 min timeout)
- **Config**: Environment variables loaded via `github.com/trackvision/tv-shared-go/env`
- **Logging**: Structured JSON via `github.com/trackvision/tv-shared-go/logger`

## COC Pipeline Flow

1. `fetch_coc_data` - Fetch data from Timken COC API (parallel)
2. `generate_pdf` - Generate PDF from COC viewer (parallel)
3. `prepare_record` - Prepare certification record
4. `create_certification` - Create in Directus
5. `upload_pdf` - Upload PDF to Directus
6. `send_email` - Send notification emails
