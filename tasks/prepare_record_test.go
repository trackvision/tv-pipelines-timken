package tasks

import (
	"testing"
	"timken-etl/types"
)

func TestPrepareRecord_Success(t *testing.T) {
	cocData := &types.COCData{
		SSCC: "100538930005550017",
		Items: []types.COCItem{
			{
				SSCC:                     "100538930005550017",
				Serial:                   "SN0001",
				ProductID:                "PROD001",
				COCDocumentID:            "DOC123",
				COCDocumentDate:          "2025-10-16",
				DeliveryNoteURI:          "https://example.com/delivery/ASN123",
				PurchaseOrderURI:         "https://example.com/po/PO123",
				ShippingEventID:          "EVENT001",
				SendCOCEmails:            1,
				ShipToNotificationEmails: []string{"shipto@example.com"},
				SoldToNotificationEmails: []string{"soldto@example.com"},
			},
			{
				SSCC:   "100538930005550017",
				Serial: "SN0002",
			},
		},
	}

	pdfData := &types.PDFData{
		PDFBytes:    []byte("fake pdf content"),
		PDFFilename: "coc_100538930005550017.pdf",
		SSCC:        "100538930005550017",
	}

	result, err := PrepareRecord(cocData, pdfData)
	if err != nil {
		t.Fatalf("PrepareRecord failed: %v", err)
	}

	// Verify certification record
	if result.Certification.CertificationType != "Conformance" {
		t.Errorf("expected CertificationType 'Conformance', got %q", result.Certification.CertificationType)
	}
	if result.Certification.CertificationIdentification != "DOC123" {
		t.Errorf("expected CertificationIdentification 'DOC123', got %q", result.Certification.CertificationIdentification)
	}
	if result.Certification.SSCC != "100538930005550017" {
		t.Errorf("expected SSCC '100538930005550017', got %q", result.Certification.SSCC)
	}
	if result.Certification.DeliveryNote != "ASN123" {
		t.Errorf("expected DeliveryNote 'ASN123', got %q", result.Certification.DeliveryNote)
	}
	if result.Certification.CustomerPO != "PO123" {
		t.Errorf("expected CustomerPO 'PO123', got %q", result.Certification.CustomerPO)
	}

	// Verify serials are joined
	expectedSerials := "SN0001\nSN0002"
	if result.Certification.CoveredSerials != expectedSerials {
		t.Errorf("expected CoveredSerials %q, got %q", expectedSerials, result.Certification.CoveredSerials)
	}

	// Verify email settings
	if !result.SendEmail {
		t.Error("expected SendEmail to be true")
	}
	if len(result.EmailAddresses) != 2 {
		t.Errorf("expected 2 email addresses, got %d", len(result.EmailAddresses))
	}

	// Verify PDF data passed through
	if string(result.PDFBytes) != "fake pdf content" {
		t.Error("PDF bytes not passed through correctly")
	}
}

func TestPrepareRecord_EmptyItems(t *testing.T) {
	cocData := &types.COCData{
		SSCC:  "100538930005550017",
		Items: []types.COCItem{},
	}

	pdfData := &types.PDFData{
		PDFBytes:    []byte("fake pdf"),
		PDFFilename: "test.pdf",
		SSCC:        "100538930005550017",
	}

	_, err := PrepareRecord(cocData, pdfData)
	if err == nil {
		t.Error("expected error for empty items, got nil")
	}
}

func TestPrepareRecord_EmailDisabled(t *testing.T) {
	cocData := &types.COCData{
		SSCC: "100538930005550017",
		Items: []types.COCItem{
			{
				SSCC:          "100538930005550017",
				Serial:        "SN0001",
				SendCOCEmails: 0, // Disabled
			},
		},
	}

	pdfData := &types.PDFData{
		PDFBytes:    []byte("fake pdf"),
		PDFFilename: "test.pdf",
		SSCC:        "100538930005550017",
	}

	result, err := PrepareRecord(cocData, pdfData)
	if err != nil {
		t.Fatalf("PrepareRecord failed: %v", err)
	}

	if result.SendEmail {
		t.Error("expected SendEmail to be false when SendCOCEmails is 0")
	}
}

func TestExtractLastSegment(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/delivery/ASN123", "ASN123"},
		{"https://example.com/po/PO123/", "PO123"},
		{"/simple/path", "path"},
		{"", ""},
		{"nopath", "nopath"},
	}

	for _, tc := range tests {
		result := extractLastSegment(tc.input)
		if result != tc.expected {
			t.Errorf("extractLastSegment(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}
