package tasks

import (
	"context"
	"fmt"
	"github.com/trackvision/tv-pipelines-template/types"

	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

// UploadPDF uploads the PDF to Directus and updates the certification record
func UploadPDF(ctx context.Context, client *DirectusClient, folderID string, data *types.CertificationResult) (*types.UploadResult, error) {
	logger.Info("Uploading PDF",
		zap.String("sscc", data.SSCC),
		zap.String("filename", data.PDFFilename),
		zap.Int("bytes", len(data.PDFBytes)),
	)

	// Upload file to Directus
	fileResult, err := client.UploadFile(ctx, UploadFileParams{
		Filename:    data.PDFFilename,
		Content:     data.PDFBytes,
		FolderID:    folderID,
		Title:       fmt.Sprintf("Certificate of Conformance - %s", data.SSCC),
		ContentType: "application/pdf",
	})
	if err != nil {
		return nil, fmt.Errorf("uploading PDF: %w", err)
	}

	fileID := fileResult.ID
	logger.Info("PDF uploaded", zap.String("fileID", fileID))

	// Update certification with attachment
	err = client.PatchItem(ctx, "certification", data.CertificationID, map[string]any{
		"primary_attachment": fileID,
	})
	if err != nil {
		return nil, fmt.Errorf("updating certification with attachment: %w", err)
	}

	logger.Info("Certification updated with attachment")

	return &types.UploadResult{
		CertificationResult: *data,
		FileID:              fileID,
	}, nil
}
