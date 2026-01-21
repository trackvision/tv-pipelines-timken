package main

import (
	"context"
	"fmt"
	"os"
	"time"
	"timken-etl/tasks"

	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

func main() {
	logger.Trace()

	sscc := "100538930005550017"
	cocViewerBaseURL := "https://timken-coc-viewer.netlify.app/html/sscc-coc/"

	logger.Info("Testing PDF generation",
		zap.String("sscc", sscc),
		zap.String("url", cocViewerBaseURL),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pdfData, err := tasks.GeneratePDF(ctx, cocViewerBaseURL, sscc)
	if err != nil {
		logger.Fatal("PDF generation failed", zap.Error(err))
	}

	outputPath := fmt.Sprintf("/tmp/coc_%s.pdf", sscc)
	err = os.WriteFile(outputPath, pdfData.PDFBytes, 0644)
	if err != nil {
		logger.Fatal("Failed to write PDF", zap.Error(err))
	}

	logger.Info("PDF saved successfully",
		zap.String("path", outputPath),
		zap.Int("bytes", len(pdfData.PDFBytes)),
	)

	fmt.Printf("\nPDF saved to: %s\n", outputPath)
}
