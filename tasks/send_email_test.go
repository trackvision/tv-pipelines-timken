package tasks

import (
	"strings"
	"testing"
	"github.com/trackvision/tv-pipelines-template/types"
)

func TestSendEmail_EmailDisabled(t *testing.T) {
	cfg := SMTPConfig{
		Host:     "smtp.test.com",
		Port:     "587",
		User:     "user",
		Password: "pass",
		From:     "from@test.com",
	}

	data := &types.UploadResult{
		CertificationResult: types.CertificationResult{
			PreparedData: types.PreparedData{
				SendEmail:      false, // Email disabled
				EmailAddresses: []string{"test@example.com"},
				SSCC:           "100538930005550017",
			},
			CertificationID: "cert-123",
		},
		FileID: "file-456",
	}

	result, err := SendEmail(cfg, data)
	if err != nil {
		t.Fatalf("SendEmail failed: %v", err)
	}

	if result.EmailSent {
		t.Error("expected EmailSent to be false when disabled")
	}
	if result.EmailSkipped != "send_coc_emails not set to 1" {
		t.Errorf("expected skip reason, got: %s", result.EmailSkipped)
	}
}

func TestSendEmail_NoRecipients(t *testing.T) {
	cfg := SMTPConfig{
		Host:     "smtp.test.com",
		Port:     "587",
		User:     "user",
		Password: "pass",
		From:     "from@test.com",
	}

	data := &types.UploadResult{
		CertificationResult: types.CertificationResult{
			PreparedData: types.PreparedData{
				SendEmail:      true,
				EmailAddresses: []string{}, // No recipients
				SSCC:           "100538930005550017",
			},
			CertificationID: "cert-123",
		},
		FileID: "file-456",
	}

	_, err := SendEmail(cfg, data)
	if err == nil {
		t.Error("expected error for no recipients")
	}
	if !strings.Contains(err.Error(), "no email recipients") {
		t.Errorf("expected no recipients error, got: %v", err)
	}
}

func TestSendEmail_InvalidEmails(t *testing.T) {
	cfg := SMTPConfig{
		Host:     "smtp.test.com",
		Port:     "587",
		User:     "user",
		Password: "pass",
		From:     "from@test.com",
	}

	data := &types.UploadResult{
		CertificationResult: types.CertificationResult{
			PreparedData: types.PreparedData{
				SendEmail:      true,
				EmailAddresses: []string{"invalid-email", "also invalid"}, // All invalid
				SSCC:           "100538930005550017",
			},
			CertificationID: "cert-123",
		},
		FileID: "file-456",
	}

	_, err := SendEmail(cfg, data)
	if err == nil {
		t.Error("expected error for no valid emails")
	}
	if !strings.Contains(err.Error(), "no valid email") {
		t.Errorf("expected no valid email error, got: %v", err)
	}
}

func TestEmailRegex(t *testing.T) {
	validEmails := []string{
		"test@example.com",
		"user.name@domain.org",
		"user+tag@sub.domain.co.uk",
		"name123@test.io",
	}

	invalidEmails := []string{
		"invalid",
		"@nodomain.com",
		"noat.com",
		"spaces in@email.com",
		"",
	}

	for _, email := range validEmails {
		if !emailRegex.MatchString(email) {
			t.Errorf("expected %q to be valid", email)
		}
	}

	for _, email := range invalidEmails {
		if emailRegex.MatchString(email) {
			t.Errorf("expected %q to be invalid", email)
		}
	}
}

func TestSendEmail_PassesDataThrough(t *testing.T) {
	cfg := SMTPConfig{
		Host:     "smtp.test.com",
		Port:     "587",
		User:     "user",
		Password: "pass",
		From:     "from@test.com",
	}

	data := &types.UploadResult{
		CertificationResult: types.CertificationResult{
			PreparedData: types.PreparedData{
				SendEmail:      false, // Disabled so we don't actually try to send
				EmailAddresses: []string{"test@example.com"},
				SSCC:           "100538930005550017",
				PDFFilename:    "test.pdf",
			},
			CertificationID: "cert-123",
		},
		FileID: "file-456",
	}

	result, err := SendEmail(cfg, data)
	if err != nil {
		t.Fatalf("SendEmail failed: %v", err)
	}

	// Verify data is passed through
	if result.SSCC != "100538930005550017" {
		t.Errorf("expected SSCC to be passed through")
	}
	if result.CertificationID != "cert-123" {
		t.Errorf("expected CertificationID to be passed through")
	}
	if result.FileID != "file-456" {
		t.Errorf("expected FileID to be passed through")
	}
}
