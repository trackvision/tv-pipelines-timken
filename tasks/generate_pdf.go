package tasks

import (
	"context"
	"fmt"
	"net/url"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"go.uber.org/zap"

	"tv-pipelines-timken/configs"
)

// GeneratePDF generates a PDF from the COC viewer webpage using chromedp
func GeneratePDF(ctx context.Context, cfg *configs.Config, sscc string) ([]byte, string, error) {
	logger := zap.L().With(zap.String("task", "generate_pdf"), zap.String("sscc", sscc))
	logger.Info("generate_pdf started")

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

	chromeCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var pdfData []byte

	logger.Info("navigating to viewer", zap.String("url", viewerURL.String()))

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
	logger.Info("generate_pdf complete", zap.Int("size_bytes", len(pdfData)))

	return pdfData, filename, nil
}
