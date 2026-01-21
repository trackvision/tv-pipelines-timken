package tasks

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"

	"tv-pipelines-timken/configs"
)

// silentLogger suppresses chromedp's internal error logs (e.g., unmarshal warnings)
type silentLogger struct{}

func (s silentLogger) Printf(format string, args ...interface{}) {
	// Suppress known harmless warnings about newer Chrome protocol features
	msg := fmt.Sprintf(format, args...)
	if strings.Contains(msg, "could not unmarshal event") {
		return
	}
	// Log other errors normally
	log.Printf(format, args...)
}

// GeneratePDF generates a PDF from the COC viewer webpage using chromedp
func GeneratePDF(ctx context.Context, cfg *configs.Config, sscc string) ([]byte, string, error) {
	viewerURL, err := url.Parse(cfg.COCViewerBaseURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid COC viewer URL: %w", err)
	}

	q := viewerURL.Query()
	q.Set("sscc", sscc)
	viewerURL.RawQuery = q.Encode()

	// Configure Chrome options for Cloud Run (headless-shell)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", "new"),
		chromedp.Flag("disable-gpu", true),
		chromedp.NoSandbox,
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	// Use silent logger to suppress unmarshal warnings
	chromeCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithErrorf(silentLogger{}.Printf))
	defer cancel()

	var pdfData []byte

	logger.Info("navigating to COC viewer",
		zap.String("sscc", sscc),
		zap.String("url", viewerURL.String()))

	err = chromedp.Run(chromeCtx,
		chromedp.Navigate(viewerURL.String()),
		// Wait for the certificate content to render
		chromedp.WaitVisible(`#certificate`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfData, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).   // A4 width in inches
				WithPaperHeight(11.69). // A4 height in inches
				WithMarginTop(0.39).    // ~10mm in inches
				WithMarginBottom(0.39).
				WithMarginLeft(0.39).
				WithMarginRight(0.39).
				Do(ctx)
			return err
		}),
	)
	if err != nil {
		return nil, "", fmt.Errorf("generate PDF: %w", err)
	}

	if len(pdfData) == 0 {
		return nil, "", fmt.Errorf("generated PDF is empty")
	}

	filename := fmt.Sprintf("COC-%s.pdf", sscc)
	logger.Info("PDF generated",
		zap.String("sscc", sscc),
		zap.Int("size_bytes", len(pdfData)),
		zap.String("filename", filename))

	return pdfData, filename, nil
}
