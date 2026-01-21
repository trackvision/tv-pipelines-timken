package tasks

import (
	"context"
	"fmt"
	"net/url"

	"github.com/playwright-community/playwright-go"
	"go.uber.org/zap"

	"tv-pipelines-timken/configs"
)

// GeneratePDF generates a PDF from the COC viewer webpage using Playwright
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

	pw, err := playwright.Run()
	if err != nil {
		return nil, "", fmt.Errorf("start playwright: %w", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args:     []string{"--no-sandbox", "--disable-setuid-sandbox"},
	})
	if err != nil {
		return nil, "", fmt.Errorf("launch browser: %w", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		return nil, "", fmt.Errorf("create page: %w", err)
	}

	logger.Info("navigating to viewer", zap.String("url", viewerURL.String()))

	resp, err := page.Goto(viewerURL.String(), playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(60000),
	})
	if err != nil {
		return nil, "", fmt.Errorf("navigate to page: %w", err)
	}
	if resp.Status() >= 400 {
		return nil, "", fmt.Errorf("page returned status %d", resp.Status())
	}

	// Wait for certificate content to load
	if _, err := page.WaitForSelector("#certificate", playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(30000),
	}); err != nil {
		return nil, "", fmt.Errorf("wait for certificate: %w", err)
	}

	// Generate PDF with A4 format - same as Python version
	pdfData, err := page.PDF(playwright.PagePdfOptions{
		Format:          playwright.String("A4"),
		PrintBackground: playwright.Bool(true),
		Margin: &playwright.Margin{
			Top:    playwright.String("10mm"),
			Bottom: playwright.String("10mm"),
			Left:   playwright.String("10mm"),
			Right:  playwright.String("10mm"),
		},
	})
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
