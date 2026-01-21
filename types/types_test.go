package types

import (
	"encoding/json"
	"testing"
)

func TestCOCItem_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"sscc": "123456789",
		"serial": "SN001",
		"product_id": "PROD-001",
		"send_coc_emails": 1,
		"ship_to_notification_emails": ["ship@example.com"],
		"sold_to_notification_emails": ["sold@example.com"]
	}`

	var item COCItem
	err := json.Unmarshal([]byte(jsonData), &item)
	if err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if item.SSCC != "123456789" {
		t.Errorf("SSCC = %q, want %q", item.SSCC, "123456789")
	}
	if item.SendCOCEmails != 1 {
		t.Errorf("SendCOCEmails = %d, want 1", item.SendCOCEmails)
	}
	if len(item.ShipToNotificationEmails) != 1 {
		t.Errorf("ShipToNotificationEmails count = %d, want 1", len(item.ShipToNotificationEmails))
	}
}

func TestCertificationRecord_JSONMarshal(t *testing.T) {
	record := CertificationRecord{
		CertificationType:           "Conformance",
		CertificationIdentification: "DOC-001",
		SSCC:                        "123456789",
		CoveredProducts:             []CoveredProduct{{ProductID: "PROD-001"}},
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	// Verify JSON structure
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if result["certification_type"] != "Conformance" {
		t.Errorf("certification_type = %v, want Conformance", result["certification_type"])
	}
}

func TestPipelineResponse_JSONMarshal(t *testing.T) {
	response := PipelineResponse{
		Success:         true,
		CertificationID: "cert-123",
		FileID:          "file-456",
		EmailSent:       true,
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if result["success"] != true {
		t.Errorf("success = %v, want true", result["success"])
	}
	if result["certification_id"] != "cert-123" {
		t.Errorf("certification_id = %v, want cert-123", result["certification_id"])
	}
}

func TestPipelineResponse_OmitEmpty(t *testing.T) {
	response := PipelineResponse{
		Success: false,
		Error:   "something went wrong",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// certification_id and file_id should be omitted when empty
	if _, exists := result["certification_id"]; exists {
		t.Error("certification_id should be omitted when empty")
	}
}
