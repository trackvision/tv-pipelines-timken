package coc

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"tv-pipelines-timken/configs"
	"tv-pipelines-timken/pipelines"
	"tv-pipelines-timken/tasks"
	"tv-pipelines-timken/types"
)

// Run executes the COC pipeline
func Run(ctx context.Context, cms *tasks.DirectusClient, cfg *configs.Config, sscc string) (*types.PipelineResult, error) {
	logger := zap.L().With(zap.String("sscc", sscc))
	logger.Info("coc pipeline started")

	// Shared state via closures
	var (
		pdfData         []byte
		pdfFilename     string
		cocData         *types.COCData
		certRecord      *types.CertificationRecord
		certificationID string
		fileID          string
		emailSent       bool
	)

	flow := pipelines.NewFlow("coc")

	// Task: generate_pdf (no deps)
	flow.AddTask("generate_pdf", func() error {
		data, filename, err := tasks.GeneratePDF(ctx, cfg, sscc)
		if err != nil {
			return fmt.Errorf("generate PDF: %w", err)
		}
		pdfData = data
		pdfFilename = filename
		return nil
	})

	// Task: fetch_coc_data (no deps, runs in parallel with generate_pdf)
	flow.AddTask("fetch_coc_data", func() error {
		data, err := tasks.FetchCOCData(ctx, cfg, sscc)
		if err != nil {
			return fmt.Errorf("fetch COC data: %w", err)
		}
		cocData = data
		return nil
	})

	// Task: prepare_record (depends on fetch_coc_data)
	flow.AddTask("prepare_record", func() error {
		record, err := prepareRecord(cocData)
		if err != nil {
			return fmt.Errorf("prepare record: %w", err)
		}
		certRecord = record
		return nil
	}, "fetch_coc_data")

	// Task: create_certification (depends on prepare_record)
	flow.AddTask("create_certification", func() error {
		id, err := cms.PostItem(ctx, "certification", certRecord)
		if err != nil {
			return fmt.Errorf("create certification: %w", err)
		}
		certificationID = id
		return nil
	}, "prepare_record")

	// Task: upload_pdf (depends on create_certification and generate_pdf)
	flow.AddTask("upload_pdf", func() error {
		fid, err := cms.UploadFile(ctx, tasks.UploadFileParams{
			Filename: pdfFilename,
			Content:  pdfData,
			FolderID: cfg.COCFolderID,
		})
		if err != nil {
			return fmt.Errorf("upload PDF: %w", err)
		}
		fileID = fid

		// Attach to certification
		err = cms.PatchItem(ctx, "certification", certificationID, map[string]interface{}{
			"primary_attachment": fileID,
		})
		if err != nil {
			return fmt.Errorf("attach PDF to certification: %w", err)
		}
		return nil
	}, "create_certification", "generate_pdf")

	// Task: send_email (depends on upload_pdf)
	flow.AddTask("send_email", func() error {
		sent, err := tasks.SendEmail(ctx, cfg, cocData, pdfData, pdfFilename)
		if err != nil {
			return fmt.Errorf("send email: %w", err)
		}
		emailSent = sent
		return nil
	}, "upload_pdf")

	// Run the flow
	if err := flow.Run(ctx); err != nil {
		return &types.PipelineResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	logger.Info("coc pipeline complete",
		zap.String("certification_id", certificationID),
		zap.String("file_id", fileID),
		zap.Bool("email_sent", emailSent))

	return &types.PipelineResult{
		Success:         true,
		CertificationID: certificationID,
		FileID:          fileID,
		EmailSent:       emailSent,
	}, nil
}

// prepareRecord transforms COC data into a certification record
func prepareRecord(cocData *types.COCData) (*types.CertificationRecord, error) {
	if cocData == nil || len(cocData.Items) == 0 {
		return nil, fmt.Errorf("no COC data available")
	}

	first := cocData.Items[0]

	// Collect all serial numbers
	var serials []string
	for _, item := range cocData.Items {
		if item.Serial != "" {
			serials = append(serials, item.Serial)
		}
	}

	return &types.CertificationRecord{
		CertificationType:           "Conformance",
		CertificationIdentification: first.COCDocumentID,
		SSCC:                        first.SSCC,
		DeliveryNote:                extractLastPathSegment(first.DeliveryNoteURI),
		CustomerPO:                  extractLastPathSegment(first.PurchaseOrderURI),
		InitialCertificationDate:    first.COCDocumentDate,
		CoveredSerials:              strings.Join(serials, "\n"),
		CoveredProducts:             []types.CoveredProduct{{ProductID: first.ProductID}},
		EventID:                     first.ShippingEventID,
	}, nil
}

// extractLastPathSegment extracts the last segment from a URI path
func extractLastPathSegment(uri string) string {
	if uri == "" {
		return ""
	}
	// Handle relative paths without scheme
	if !strings.Contains(uri, "://") {
		uri = strings.TrimSuffix(uri, "/")
		if idx := strings.LastIndex(uri, "/"); idx != -1 {
			return uri[idx+1:]
		}
		return ""
	}
	// For absolute URLs, find path after scheme://host
	schemeEnd := strings.Index(uri, "://")
	if schemeEnd == -1 {
		return ""
	}
	rest := uri[schemeEnd+3:] // Skip "://"
	slashIdx := strings.Index(rest, "/")
	if slashIdx == -1 {
		return "" // No path, just domain
	}
	path := rest[slashIdx:]
	path = strings.TrimSuffix(path, "/")
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		return path[idx+1:]
	}
	return strings.TrimPrefix(path, "/")
}
