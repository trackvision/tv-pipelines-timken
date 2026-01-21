package tasks

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"

	"go.uber.org/zap"

	"tv-pipelines-timken/configs"
	"tv-pipelines-timken/types"
)

const (
	emailSubject = "Timken Certificate of Conformance"
	emailBody    = `Please find the attached certificate of conformance for your Timken products.

Kind regards,
Timken support team.`
)

// SendEmail sends the COC email with the PDF attachment. Returns true if email was sent.
func SendEmail(ctx context.Context, cfg *configs.Config, cocData *types.COCData, pdfData []byte, pdfFilename string) (bool, error) {
	logger := zap.L().With(zap.String("task", "send_email"))
	logger.Info("send_email started")

	if cocData == nil || len(cocData.Items) == 0 {
		return false, fmt.Errorf("no COC data available")
	}

	first := cocData.Items[0]

	// Check if email sending is enabled
	if first.SendCOCEmails != 1 {
		logger.Info("send_email skipped", zap.String("reason", "send_coc_emails not set"))
		return false, nil
	}

	// Collect email addresses
	recipients := collectEmailAddresses(first.ShipToNotificationEmails, first.SoldToNotificationEmails)

	if len(recipients) == 0 {
		return false, fmt.Errorf("send_coc_emails is 1 but no email addresses provided")
	}

	// Validate all email addresses
	for _, email := range recipients {
		if _, err := mail.ParseAddress(email); err != nil {
			return false, fmt.Errorf("invalid email address %q: %w", email, err)
		}
	}

	// Send the email
	if err := sendEmailWithAttachment(cfg, recipients, emailSubject, emailBody, pdfFilename, pdfData); err != nil {
		return false, fmt.Errorf("send email: %w", err)
	}

	logger.Info("send_email complete", zap.Int("recipient_count", len(recipients)))
	return true, nil
}

func collectEmailAddresses(shipTo, soldTo []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, email := range append(shipTo, soldTo...) {
		email = strings.TrimSpace(email)
		if email != "" && !seen[email] {
			seen[email] = true
			result = append(result, email)
		}
	}

	return result
}

func sendEmailWithAttachment(cfg *configs.Config, to []string, subject, body, attachmentName string, attachmentData []byte) error {
	boundary := "----=_Part_0_1234567890"

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", cfg.EmailFromAddress))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
	msg.WriteString("\r\n")

	// Body part
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)
	msg.WriteString("\r\n")

	// Attachment part
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString(fmt.Sprintf("Content-Type: application/pdf; name=\"%s\"\r\n", attachmentName))
	msg.WriteString("Content-Transfer-Encoding: base64\r\n")
	msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", attachmentName))
	msg.WriteString("\r\n")
	msg.WriteString(base64.StdEncoding.EncodeToString(attachmentData))
	msg.WriteString("\r\n")

	// End boundary
	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	auth := smtp.PlainAuth("", cfg.EmailSMTPUser, cfg.EmailSMTPPassword, cfg.EmailSMTPHost)
	addr := fmt.Sprintf("%s:%s", cfg.EmailSMTPHost, cfg.EmailSMTPPort)

	return smtp.SendMail(addr, auth, cfg.EmailFromAddress, to, []byte(msg.String()))
}
