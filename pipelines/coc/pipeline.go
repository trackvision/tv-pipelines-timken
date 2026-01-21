package coc

import (
	"context"
	"fmt"

	"timken-etl/pipelines"
	"timken-etl/tasks"
	"timken-etl/types"

	"github.com/fieldryand/goflow/v2"
	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

func init() {
	pipelines.RegisterDescriptor(pipelines.Descriptor{
		Name:        "coc",
		Description: "Certificate of Conformance pipeline",
		Flags:       []string{"--sscc"},
	})
}

// State keys for COC pipeline data
const (
	KeySSCC           = "sscc"
	KeyCOCConfig      = "coc_config"
	KeyCOCData        = "coc_data"
	KeyPDFData        = "pdf_data"
	KeyPreparedData   = "prepared_data"
	KeyCertResult     = "cert_result"
	KeyUploadResult   = "upload_result"
	KeyPipelineResult = "pipeline_result"

	// VisualizationSSCC is a sentinel value used when creating pipelines for DAG visualization
	VisualizationSSCC = "__VISUALIZATION__"
)

// Pipeline implements the COC certificate pipeline
type Pipeline struct {
	state  *pipelines.State
	config *Config
}

// New creates a new COC pipeline instance
func New(state *pipelines.State, sscc string) (*Pipeline, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading COC config: %w", err)
	}

	state.Set(KeySSCC, sscc)
	state.Set(KeyCOCConfig, cfg)

	return &Pipeline{
		state:  state,
		config: cfg,
	}, nil
}

// Name returns the pipeline identifier
func (p *Pipeline) Name() string {
	return "coc"
}

// Description returns a human-readable description
func (p *Pipeline) Description() string {
	return "Certificate of Conformance pipeline"
}

// ValidateConfig validates that all required configuration is present
func (p *Pipeline) ValidateConfig() error {
	if err := p.config.Validate(); err != nil {
		return err
	}
	if p.state.GetString(KeySSCC) == "" {
		return fmt.Errorf("SSCC is required")
	}
	return nil
}

// Job returns a goflow job factory function
func (p *Pipeline) Job() func() *goflow.Job {
	return func() *goflow.Job {
		j := &goflow.Job{
			Name:     "coc-pipeline",
			Schedule: "@manual",
			Active:   true,
		}

		// Task 1: Fetch COC data
		j.Add(&goflow.Task{
			Name:       "fetch_coc_data",
			Operator:   &FetchCOCDataOp{pipeline: p},
			Retries:    2,
			RetryDelay: goflow.ConstantDelay{Period: 5},
		})

		// Task 2: Generate PDF (parallel with Task 1)
		j.Add(&goflow.Task{
			Name:       "generate_pdf",
			Operator:   &GeneratePDFOp{pipeline: p},
			Retries:    2,
			RetryDelay: goflow.ConstantDelay{Period: 5},
		})

		// Task 3: Prepare record (depends on 1 & 2)
		j.Add(&goflow.Task{
			Name:     "prepare_record",
			Operator: &PrepareRecordOp{pipeline: p},
		})

		// Task 4: Create certification (depends on 3)
		j.Add(&goflow.Task{
			Name:       "create_certification",
			Operator:   &CreateCertificationOp{pipeline: p},
			Retries:    2,
			RetryDelay: goflow.ConstantDelay{Period: 5},
		})

		// Task 5: Upload PDF (depends on 4)
		j.Add(&goflow.Task{
			Name:       "upload_pdf",
			Operator:   &UploadPDFOp{pipeline: p},
			Retries:    2,
			RetryDelay: goflow.ConstantDelay{Period: 5},
		})

		// Task 6: Send email (depends on 5)
		j.Add(&goflow.Task{
			Name:       "send_email",
			Operator:   &SendEmailOp{pipeline: p},
			Retries:    2,
			RetryDelay: goflow.ConstantDelay{Period: 5},
		})

		setupDAGEdges(j)
		return j
	}
}

// VisualizationJob returns a goflow job for UI visualization only (not for execution)
func (p *Pipeline) VisualizationJob() func() *goflow.Job {
	return func() *goflow.Job {
		j := &goflow.Job{
			Name:   "coc-pipeline",
			Active: false, // Visualization only
		}

		// Add tasks with no-op operators (just for DAG display)
		j.Add(&goflow.Task{Name: "fetch_coc_data", Operator: &noopOp{}})
		j.Add(&goflow.Task{Name: "generate_pdf", Operator: &noopOp{}})
		j.Add(&goflow.Task{Name: "prepare_record", Operator: &noopOp{}})
		j.Add(&goflow.Task{Name: "create_certification", Operator: &noopOp{}})
		j.Add(&goflow.Task{Name: "upload_pdf", Operator: &noopOp{}})
		j.Add(&goflow.Task{Name: "send_email", Operator: &noopOp{}})

		setupDAGEdges(j)
		return j
	}
}

// setupDAGEdges defines the task dependencies for the COC pipeline
func setupDAGEdges(j *goflow.Job) {
	j.SetDownstream(j.Task("fetch_coc_data"), j.Task("prepare_record"))
	j.SetDownstream(j.Task("generate_pdf"), j.Task("prepare_record"))
	j.SetDownstream(j.Task("prepare_record"), j.Task("create_certification"))
	j.SetDownstream(j.Task("create_certification"), j.Task("upload_pdf"))
	j.SetDownstream(j.Task("upload_pdf"), j.Task("send_email"))
}

// noopOp is a no-operation operator for visualization
type noopOp struct{}

func (o *noopOp) Run() (any, error) { return nil, nil }

// getStateValue safely retrieves a typed value from pipeline state
func getStateValue[T any](p *Pipeline, key string) (*T, error) {
	val := p.state.Get(key)
	if val == nil {
		return nil, fmt.Errorf("state key %q not set", key)
	}
	typed, ok := val.(*T)
	if !ok {
		return nil, fmt.Errorf("state key %q has unexpected type %T", key, val)
	}
	return typed, nil
}

// RunOnce executes the pipeline synchronously
func (p *Pipeline) RunOnce() error {
	sscc := p.state.GetString(KeySSCC)
	logger.Info("Running COC pipeline in once mode", zap.String("sscc", sscc))

	// Execute parallel tasks first (fetch_coc_data and generate_pdf)
	type taskResult struct {
		name string
		err  error
	}
	results := make(chan taskResult, 2)

	go func() {
		_, err := (&FetchCOCDataOp{pipeline: p}).Run()
		results <- taskResult{"fetch_coc_data", err}
	}()
	go func() {
		_, err := (&GeneratePDFOp{pipeline: p}).Run()
		results <- taskResult{"generate_pdf", err}
	}()

	// Wait for parallel tasks
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err != nil {
			return fmt.Errorf("task %s failed: %w", r.name, r.err)
		}
	}

	// Execute sequential tasks
	sequentialOps := []struct {
		name string
		op   goflow.Operator
	}{
		{"prepare_record", &PrepareRecordOp{pipeline: p}},
		{"create_certification", &CreateCertificationOp{pipeline: p}},
		{"upload_pdf", &UploadPDFOp{pipeline: p}},
		{"send_email", &SendEmailOp{pipeline: p}},
	}

	for _, t := range sequentialOps {
		if _, err := t.op.Run(); err != nil {
			return fmt.Errorf("task %s failed: %w", t.name, err)
		}
	}

	result, err := getStateValue[types.PipelineResult](p, KeyPipelineResult)
	if err != nil {
		return fmt.Errorf("getting pipeline result: %w", err)
	}
	logger.Info("Pipeline complete",
		zap.String("sscc", result.SSCC),
		zap.String("certificationID", result.CertificationID),
		zap.String("fileID", result.FileID),
		zap.Bool("emailSent", result.EmailSent),
	)

	return nil
}

// --- Custom Operators ---

// FetchCOCDataOp fetches COC data from the API
type FetchCOCDataOp struct {
	pipeline *Pipeline
}

func (o *FetchCOCDataOp) Run() (interface{}, error) {
	sscc := o.pipeline.state.GetString(KeySSCC)
	logger.Info("Task: fetch_coc_data", zap.String("sscc", sscc))

	ctx := context.Background()
	data, err := tasks.FetchCOCData(
		ctx,
		o.pipeline.config.TimkenCOCAPIURL,
		o.pipeline.state.Config.DirectusCMSAPIKey,
		sscc,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch_coc_data failed: %w", err)
	}

	o.pipeline.state.Set(KeyCOCData, data)
	logger.Info("Task: fetch_coc_data complete", zap.Int("items", len(data.Items)))
	return len(data.Items), nil
}

// GeneratePDFOp generates PDF from the COC viewer
type GeneratePDFOp struct {
	pipeline *Pipeline
}

func (o *GeneratePDFOp) Run() (interface{}, error) {
	sscc := o.pipeline.state.GetString(KeySSCC)
	logger.Info("Task: generate_pdf", zap.String("sscc", sscc))

	ctx := context.Background()
	data, err := tasks.GeneratePDF(ctx, o.pipeline.config.COCViewerBaseURL, sscc)
	if err != nil {
		return nil, fmt.Errorf("generate_pdf failed: %w", err)
	}

	o.pipeline.state.Set(KeyPDFData, data)
	logger.Info("Task: generate_pdf complete", zap.Int("bytes", len(data.PDFBytes)))
	return len(data.PDFBytes), nil
}

// PrepareRecordOp prepares the certification record
type PrepareRecordOp struct {
	pipeline *Pipeline
}

func (o *PrepareRecordOp) Run() (interface{}, error) {
	sscc := o.pipeline.state.GetString(KeySSCC)
	logger.Info("Task: prepare_record", zap.String("sscc", sscc))

	cocData, err := getStateValue[types.COCData](o.pipeline, KeyCOCData)
	if err != nil {
		return nil, fmt.Errorf("prepare_record: %w", err)
	}
	pdfData, err := getStateValue[types.PDFData](o.pipeline, KeyPDFData)
	if err != nil {
		return nil, fmt.Errorf("prepare_record: %w", err)
	}

	data, err := tasks.PrepareRecord(cocData, pdfData)
	if err != nil {
		return nil, fmt.Errorf("prepare_record failed: %w", err)
	}

	o.pipeline.state.Set(KeyPreparedData, data)
	logger.Info("Task: prepare_record complete", zap.Int("serials", len(cocData.Items)))
	return data.Certification.CertificationIdentification, nil
}

// CreateCertificationOp creates the certification in Directus
type CreateCertificationOp struct {
	pipeline *Pipeline
}

func (o *CreateCertificationOp) Run() (interface{}, error) {
	sscc := o.pipeline.state.GetString(KeySSCC)
	logger.Info("Task: create_certification", zap.String("sscc", sscc))

	ctx := context.Background()
	preparedData, err := getStateValue[types.PreparedData](o.pipeline, KeyPreparedData)
	if err != nil {
		return nil, fmt.Errorf("create_certification: %w", err)
	}

	result, err := tasks.CreateCertification(ctx, o.pipeline.state.DirectusClient, preparedData)
	if err != nil {
		return nil, fmt.Errorf("create_certification failed: %w", err)
	}

	o.pipeline.state.Set(KeyCertResult, result)
	logger.Info("Task: create_certification complete", zap.String("certificationID", result.CertificationID))
	return result.CertificationID, nil
}

// UploadPDFOp uploads the PDF to Directus
type UploadPDFOp struct {
	pipeline *Pipeline
}

func (o *UploadPDFOp) Run() (interface{}, error) {
	sscc := o.pipeline.state.GetString(KeySSCC)
	logger.Info("Task: upload_pdf", zap.String("sscc", sscc))

	ctx := context.Background()
	certResult, err := getStateValue[types.CertificationResult](o.pipeline, KeyCertResult)
	if err != nil {
		return nil, fmt.Errorf("upload_pdf: %w", err)
	}

	result, err := tasks.UploadPDF(
		ctx,
		o.pipeline.state.DirectusClient,
		o.pipeline.config.COCPDFFolderID,
		certResult,
	)
	if err != nil {
		return nil, fmt.Errorf("upload_pdf failed: %w", err)
	}

	o.pipeline.state.Set(KeyUploadResult, result)
	logger.Info("Task: upload_pdf complete", zap.String("fileID", result.FileID))
	return result.FileID, nil
}

// SendEmailOp sends the notification email
type SendEmailOp struct {
	pipeline *Pipeline
}

func (o *SendEmailOp) Run() (interface{}, error) {
	sscc := o.pipeline.state.GetString(KeySSCC)
	logger.Info("Task: send_email", zap.String("sscc", sscc))

	uploadResult, err := getStateValue[types.UploadResult](o.pipeline, KeyUploadResult)
	if err != nil {
		return nil, fmt.Errorf("send_email: %w", err)
	}
	cfg := o.pipeline.state.Config

	smtpCfg := tasks.SMTPConfig{
		Host:     cfg.EmailSMTPHost,
		Port:     cfg.EmailSMTPPort,
		User:     cfg.EmailSMTPUser,
		Password: cfg.EmailSMTPPassword,
		From:     o.pipeline.config.COCFromEmail,
	}

	result, err := tasks.SendEmail(smtpCfg, uploadResult)
	if err != nil {
		return nil, fmt.Errorf("send_email failed: %w", err)
	}

	o.pipeline.state.Set(KeyPipelineResult, result)
	logger.Info("Task: send_email complete", zap.Bool("emailSent", result.EmailSent))
	return result.EmailSent, nil
}
