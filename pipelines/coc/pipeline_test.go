package coc

import (
	"testing"

	"tv-pipelines-timken/types"
)

func TestExtractLastPathSegment(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string
	}{
		{
			name: "normal URI",
			uri:  "https://desadv.sap.timken.com/bt/ASN123",
			want: "ASN123",
		},
		{
			name: "trailing slash",
			uri:  "https://example.com/path/segment/",
			want: "segment",
		},
		{
			name: "empty string",
			uri:  "",
			want: "",
		},
		{
			name: "no path",
			uri:  "https://example.com",
			want: "",
		},
		{
			name: "single segment",
			uri:  "/segment",
			want: "segment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLastPathSegment(tt.uri)
			if got != tt.want {
				t.Errorf("extractLastPathSegment(%q) = %q, want %q", tt.uri, got, tt.want)
			}
		})
	}
}

func TestPrepareRecord(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cocData := &types.COCData{
			Items: []types.COCItem{
				{
					SSCC:             "123456789",
					Serial:           "SN001",
					ProductID:        "PROD-001",
					COCDocumentID:    "DOC-001",
					COCDocumentDate:  "2024-01-15",
					DeliveryNoteURI:  "https://desadv.sap.timken.com/bt/ASN123",
					PurchaseOrderURI: "https://po.sap.timken.com/bt/PO456",
					ShippingEventID:  "EVENT-001",
				},
				{
					SSCC:   "123456789",
					Serial: "SN002",
				},
			},
		}

		record, err := prepareRecord(cocData)
		if err != nil {
			t.Fatalf("prepareRecord() error = %v", err)
		}

		if record.CertificationType != "Conformance" {
			t.Errorf("CertificationType = %q, want %q", record.CertificationType, "Conformance")
		}
		if record.CertificationIdentification != "DOC-001" {
			t.Errorf("CertificationIdentification = %q, want %q", record.CertificationIdentification, "DOC-001")
		}
		if record.DeliveryNote != "ASN123" {
			t.Errorf("DeliveryNote = %q, want %q", record.DeliveryNote, "ASN123")
		}
		if record.CustomerPO != "PO456" {
			t.Errorf("CustomerPO = %q, want %q", record.CustomerPO, "PO456")
		}
		if record.CoveredSerials != "SN001\nSN002" {
			t.Errorf("CoveredSerials = %q, want %q", record.CoveredSerials, "SN001\nSN002")
		}
		if len(record.CoveredProducts) != 1 || record.CoveredProducts[0].ProductID != "PROD-001" {
			t.Errorf("CoveredProducts = %v, want [{ProductID: PROD-001}]", record.CoveredProducts)
		}
	})

	t.Run("nil data", func(t *testing.T) {
		_, err := prepareRecord(nil)
		if err == nil {
			t.Error("prepareRecord(nil) expected error")
		}
	})

	t.Run("empty items", func(t *testing.T) {
		cocData := &types.COCData{Items: []types.COCItem{}}
		_, err := prepareRecord(cocData)
		if err == nil {
			t.Error("prepareRecord with empty items expected error")
		}
	})
}
