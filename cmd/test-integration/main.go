package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"
	"timken-etl/tasks"
	"timken-etl/types"

	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

const (
	defaultSSCC           = "100538930005550017"
	defaultCOCViewerURL   = "https://timken-coc-viewer.netlify.app/html/sscc-coc/"
	defaultTimkenCOCAPI   = "https://timkendev.trackvision.ai/flows/trigger/705d83de-7f24-4c84-be1c-39ce49cf1677"
)

func main() {
	logger.Trace()

	sscc := flag.String("sscc", defaultSSCC, "SSCC to process")
	sendEmail := flag.Bool("send-email", false, "Actually send the email (requires SMTP config)")
	skipPDF := flag.Bool("skip-pdf", false, "Skip PDF generation")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	apiKey := os.Getenv("DIRECTUS_CMS_API_KEY")
	if apiKey == "" {
		fmt.Println("ERROR: DIRECTUS_CMS_API_KEY environment variable is required")
		fmt.Println("")
		fmt.Println("Set it with:")
		fmt.Println("  export DIRECTUS_CMS_API_KEY=your-api-key")
		fmt.Println("")
		fmt.Println("Or run with:")
		fmt.Println("  DIRECTUS_CMS_API_KEY=your-key go run cmd/test-integration/main.go")
		os.Exit(1)
	}

	logger.Info("Starting integration test",
		zap.String("sscc", *sscc),
		zap.Bool("sendEmail", *sendEmail),
	)

	// Step 1: Fetch COC data
	logger.Info("Step 1: Fetching COC data...")
	cocData, err := tasks.FetchCOCData(ctx, defaultTimkenCOCAPI, apiKey, *sscc)
	if err != nil {
		logger.Fatal("Failed to fetch COC data", zap.Error(err))
	}
	logger.Info("COC data fetched",
		zap.Int("items", len(cocData.Items)),
		zap.String("firstSerial", cocData.Items[0].Serial),
	)

	// Step 2: Generate PDF (optional)
	var pdfData *types.PDFData
	if !*skipPDF {
		logger.Info("Step 2: Generating PDF...")
		pdfData, err = tasks.GeneratePDF(ctx, defaultCOCViewerURL, *sscc)
		if err != nil {
			logger.Fatal("Failed to generate PDF", zap.Error(err))
		}
		logger.Info("PDF generated",
			zap.String("filename", pdfData.PDFFilename),
			zap.Int("bytes", len(pdfData.PDFBytes)),
		)

		// Save PDF for inspection
		pdfPath := fmt.Sprintf("/tmp/%s", pdfData.PDFFilename)
		if err := os.WriteFile(pdfPath, pdfData.PDFBytes, 0644); err != nil {
			logger.Error("Failed to save PDF", zap.Error(err))
		} else {
			logger.Info("PDF saved", zap.String("path", pdfPath))
		}
	} else {
		logger.Info("Step 2: Skipping PDF generation")
		pdfData = &types.PDFData{
			PDFBytes:    []byte("dummy pdf for testing"),
			PDFFilename: fmt.Sprintf("coc_%s.pdf", *sscc),
			SSCC:        *sscc,
		}
	}

	// Step 3: Prepare record
	logger.Info("Step 3: Preparing record...")
	preparedData, err := tasks.PrepareRecord(cocData, pdfData)
	if err != nil {
		logger.Fatal("Failed to prepare record", zap.Error(err))
	}

	// Display prepared data
	fmt.Println("\n========== PREPARED RECORD ==========")
	fmt.Printf("SSCC: %s\n", preparedData.SSCC)
	fmt.Printf("Document ID: %s\n", preparedData.Certification.CertificationIdentification)
	fmt.Printf("Document Date: %s\n", preparedData.Certification.InitialCertificationDate)
	fmt.Printf("Delivery Note: %s\n", preparedData.Certification.DeliveryNote)
	fmt.Printf("Customer PO: %s\n", preparedData.Certification.CustomerPO)
	fmt.Printf("Covered Serials: %s\n", preparedData.Certification.CoveredSerials)
	fmt.Printf("PDF Filename: %s\n", preparedData.PDFFilename)
	fmt.Println("======================================")

	// Step 4: Email details
	fmt.Println("========== EMAIL CONFIGURATION ==========")
	fmt.Printf("Send Email Enabled: %v\n", preparedData.SendEmail)
	fmt.Printf("Email Recipients: %v\n", preparedData.EmailAddresses)
	fmt.Println("==========================================")

	if !preparedData.SendEmail {
		logger.Info("Email sending is disabled in COC data (send_coc_emails != 1)")
		fmt.Println("To test email sending, the COC data must have send_coc_emails = 1")
		return
	}

	if len(preparedData.EmailAddresses) == 0 {
		logger.Info("No email recipients configured")
		return
	}

	// Step 5: Send email (if requested)
	if *sendEmail {
		smtpHost := os.Getenv("EMAIL_SMTP_HOST")
		smtpPort := os.Getenv("EMAIL_SMTP_PORT")
		smtpUser := os.Getenv("EMAIL_SMTP_USER")
		smtpPass := os.Getenv("EMAIL_SMTP_PASSWORD")
		smtpFrom := os.Getenv("COC_FROM_EMAIL")

		if smtpHost == "" {
			logger.Fatal("EMAIL_SMTP_HOST not set. Required env vars: EMAIL_SMTP_HOST, EMAIL_SMTP_PORT, EMAIL_SMTP_USER, EMAIL_SMTP_PASSWORD, COC_FROM_EMAIL")
		}

		cfg := tasks.SMTPConfig{
			Host:     smtpHost,
			Port:     smtpPort,
			User:     smtpUser,
			Password: smtpPass,
			From:     smtpFrom,
		}

		// Create upload result to pass to SendEmail
		uploadResult := &types.UploadResult{
			CertificationResult: types.CertificationResult{
				PreparedData:    *preparedData,
				CertificationID: "test-cert-id",
			},
			FileID: "test-file-id",
		}

		logger.Info("Step 5: Sending email...",
			zap.Strings("to", preparedData.EmailAddresses),
			zap.String("from", smtpFrom),
		)

		result, err := tasks.SendEmail(cfg, uploadResult)
		if err != nil {
			logger.Fatal("Failed to send email", zap.Error(err))
		}

		if result.EmailSent {
			logger.Info("Email sent successfully!",
				zap.Strings("recipients", preparedData.EmailAddresses),
			)
		} else {
			logger.Info("Email was skipped", zap.String("reason", result.EmailSkipped))
		}
	} else {
		fmt.Println("To actually send the email, run with -send-email flag.")
		fmt.Println("SMTP vars are already in .env (EMAIL_SMTP_HOST, EMAIL_SMTP_PORT, etc.)")
		fmt.Println("")
		fmt.Println("NOTE: You need a valid Resend API key in EMAIL_SMTP_PASSWORD")
		fmt.Println("")
		fmt.Println("Run with:")
		fmt.Println("  source .env && go run cmd/test-integration/main.go -send-email")
	}

	logger.Info("Integration test complete")
}
