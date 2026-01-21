package tasks

import (
	"fmt"
	"strings"
	"github.com/trackvision/tv-pipelines-template/types"

	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

// PrepareRecord combines COC data and PDF data into a prepared certification record
func PrepareRecord(cocData *types.COCData, pdfData *types.PDFData) (*types.PreparedData, error) {
	logger.Info("Preparing certification record", zap.String("sscc", cocData.SSCC))

	if len(cocData.Items) == 0 {
		return nil, fmt.Errorf("empty items list from API")
	}

	first := cocData.Items[0]

	// Collect all serial numbers
	var serials []string
	for _, item := range cocData.Items {
		if item.Serial != "" {
			serials = append(serials, item.Serial)
		}
	}

	// Build covered products from first item only
	var coveredProducts []types.CoveredProduct
	if first.ProductID != "" {
		coveredProducts = append(coveredProducts, types.CoveredProduct{ProductID: first.ProductID})
	}

	cert := types.CertificationRecord{
		CertificationType:           "Conformance",
		CertificationIdentification: first.COCDocumentID,
		SSCC:                        first.SSCC,
		DeliveryNote:                extractLastSegment(first.DeliveryNoteURI),
		CustomerPO:                  extractLastSegment(first.PurchaseOrderURI),
		InitialCertificationDate:    first.COCDocumentDate,
		CoveredSerials:              strings.Join(serials, "\n"),
		CoveredProducts:             coveredProducts,
		EventID:                     first.ShippingEventID,
	}

	// Collect email addresses
	var emailAddresses []string
	emailAddresses = append(emailAddresses, first.ShipToNotificationEmails...)
	emailAddresses = append(emailAddresses, first.SoldToNotificationEmails...)

	sendEmail := first.SendCOCEmails == 1

	logger.Info("Record prepared",
		zap.Int("serials", len(serials)),
		zap.Bool("sendEmail", sendEmail),
		zap.Int("emailAddresses", len(emailAddresses)),
	)

	return &types.PreparedData{
		Certification:  cert,
		PDFBytes:       pdfData.PDFBytes,
		PDFFilename:    pdfData.PDFFilename,
		SendEmail:      sendEmail,
		EmailAddresses: emailAddresses,
		SSCC:           cocData.SSCC,
	}, nil
}

// extractLastSegment extracts the last segment from a URI path
func extractLastSegment(uri string) string {
	if uri == "" {
		return ""
	}
	uri = strings.TrimRight(uri, "/")
	parts := strings.Split(uri, "/")
	return parts[len(parts)-1]
}
