package tasks

import (
	"context"
	"fmt"
	"github.com/trackvision/tv-pipelines-template/types"

	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

// CreateCertification creates a certification record in Directus
func CreateCertification(ctx context.Context, client *DirectusClient, data *types.PreparedData) (*types.CertificationResult, error) {
	logger.Info("Creating certification record",
		zap.String("sscc", data.SSCC),
		zap.String("certification_id", data.Certification.CertificationIdentification),
	)

	result, err := client.PostItem(ctx, "certification", data.Certification)
	if err != nil {
		return nil, fmt.Errorf("creating certification: %w", err)
	}

	certID, ok := result["id"].(string)
	if !ok {
		return nil, fmt.Errorf("failed to get certification ID from response")
	}

	logger.Info("Certification created", zap.String("id", certID))

	return &types.CertificationResult{
		PreparedData:    *data,
		CertificationID: certID,
	}, nil
}
