package tasks

import (
	"context"
	"fmt"
	"net/url"
	"time"
	"github.com/trackvision/tv-pipelines-template/types"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

// GeneratePDF generates a PDF from the COC viewer URL using headless Chrome
func GeneratePDF(ctx context.Context, cocViewerBaseURL, sscc string) (*types.PDFData, error) {
	logger.Info("Generating PDF", zap.String("sscc", sscc))

	targetURL := fmt.Sprintf("%s?sscc=%s", cocViewerBaseURL, url.QueryEscape(sscc))

	// Create chromedp context with headless options
	allocCtx, cancel := chromedp.NewExecAllocator(ctx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-setuid-sandbox", true),
		)...,
	)
	defer cancel()

	chromeCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set overall timeout
	chromeCtx, cancel = context.WithTimeout(chromeCtx, 90*time.Second)
	defer cancel()

	var pdfBytes []byte

	// CSS to fix print pagination issues
	printCSS := `
		var style = document.createElement('style');
		style.textContent = '@media print { ' +
			'table { page-break-inside: avoid !important; } ' +
			'tr { page-break-inside: avoid !important; page-break-after: auto !important; } ' +
			'thead { display: table-header-group !important; } ' +
			'.table-title { page-break-after: avoid !important; } ' +
			'.tagline { page-break-inside: avoid !important; margin-top: 20px !important; } ' +
			'#inspection-reports-container { page-break-before: always !important; } ' +
			'* { orphans: 3 !important; widows: 3 !important; } ' +
		'}';
		document.head.appendChild(style);

		// Keep Product Specifications table with its title
		var tableTitles = document.querySelectorAll('.table-title');
		tableTitles.forEach(function(el) {
			if (el.textContent.includes('Product Specification')) {
				// Wrap title and following table in a container to keep together
				var nextTable = el.nextElementSibling;
				if (nextTable && nextTable.tagName === 'TABLE') {
					var wrapper = document.createElement('div');
					wrapper.style.pageBreakInside = 'avoid';
					el.parentNode.insertBefore(wrapper, el);
					wrapper.appendChild(el);
					wrapper.appendChild(nextTable);
				}
			}
		});
	`

	err := chromedp.Run(chromeCtx,
		chromedp.Navigate(targetURL),
		chromedp.WaitVisible("#certificate", chromedp.ByID),
		chromedp.Sleep(2*time.Second), // Wait for dynamic content
		chromedp.Evaluate(printCSS, nil),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBytes, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				WithPaperWidth(8.27).   // A4 width in inches
				WithPaperHeight(11.69). // A4 height in inches
				WithMarginTop(0.4).
				WithMarginBottom(0.8). // Larger bottom margin to avoid footer overlap
				WithMarginLeft(0.4).
				WithMarginRight(0.4).
				Do(ctx)
			return err
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("PDF generation failed: %w", err)
	}

	logger.Info("PDF generated", zap.Int("bytes", len(pdfBytes)))

	return &types.PDFData{
		PDFBytes:    pdfBytes,
		PDFFilename: fmt.Sprintf("coc_%s.pdf", sscc),
		SSCC:        sscc,
	}, nil
}
