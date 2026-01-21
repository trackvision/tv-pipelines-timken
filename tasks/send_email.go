package tasks

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"regexp"
	"strings"
	"github.com/trackvision/tv-pipelines-template/types"

	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

// SendEmail sends the COC notification email with PDF attachment
func SendEmail(cfg SMTPConfig, data *types.UploadResult) (*types.PipelineResult, error) {
	result := &types.PipelineResult{
		UploadResult: *data,
		EmailSent:    false,
	}

	if !data.SendEmail {
		logger.Info("Email sending disabled", zap.String("sscc", data.SSCC))
		result.EmailSkipped = "send_coc_emails not set to 1"
		return result, nil
	}

	if len(data.EmailAddresses) == 0 {
		return nil, fmt.Errorf("no email recipients configured for SSCC: %s", data.SSCC)
	}

	// Validate emails
	var validEmails []string
	for _, email := range data.EmailAddresses {
		email = strings.TrimSpace(email)
		if emailRegex.MatchString(email) {
			validEmails = append(validEmails, email)
		} else {
			logger.Warn("Skipping invalid email", zap.String("email", email))
		}
	}

	if len(validEmails) == 0 {
		return nil, fmt.Errorf("no valid email addresses for SSCC: %s", data.SSCC)
	}

	logger.Info("Sending COC email",
		zap.String("sscc", data.SSCC),
		zap.Strings("recipients", validEmails),
	)

	err := sendEmailWithAttachment(cfg, validEmails, data.PDFBytes, data.PDFFilename, data.SSCC)
	if err != nil {
		return nil, fmt.Errorf("sending email: %w", err)
	}

	logger.Info("Email sent successfully")
	result.EmailSent = true

	return result, nil
}

func sendEmailWithAttachment(cfg SMTPConfig, recipients []string, pdfBytes []byte, filename, sscc string) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Email headers
	headers := make(textproto.MIMEHeader)
	headers.Set("From", cfg.From)
	headers.Set("To", strings.Join(recipients, ", "))
	headers.Set("Subject", fmt.Sprintf("Certificate of Conformance - SSCC %s", sscc))
	headers.Set("MIME-Version", "1.0")
	headers.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", writer.Boundary()))

	// Write headers
	var headerBuf bytes.Buffer
	for key, values := range headers {
		for _, value := range values {
			headerBuf.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
		}
	}
	headerBuf.WriteString("\r\n")

	// Text body part
	textHeaders := make(textproto.MIMEHeader)
	textHeaders.Set("Content-Type", "text/plain; charset=utf-8")
	textPart, err := writer.CreatePart(textHeaders)
	if err != nil {
		return fmt.Errorf("creating text part: %w", err)
	}
	textBody := fmt.Sprintf("Please find attached the Certificate of Conformance for SSCC: %s\n\nThis is an automated message.", sscc)
	textPart.Write([]byte(textBody))

	// PDF attachment part
	attachHeaders := make(textproto.MIMEHeader)
	attachHeaders.Set("Content-Type", "application/pdf")
	attachHeaders.Set("Content-Transfer-Encoding", "base64")
	attachHeaders.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	attachPart, err := writer.CreatePart(attachHeaders)
	if err != nil {
		return fmt.Errorf("creating attachment part: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(pdfBytes)
	attachPart.Write([]byte(encoded))

	writer.Close()

	// Combine headers and body
	var message bytes.Buffer
	message.Write(headerBuf.Bytes())
	message.Write(buf.Bytes())

	// Send email
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)

	err = smtp.SendMail(addr, auth, cfg.From, recipients, message.Bytes())
	if err != nil {
		return fmt.Errorf("SMTP send failed: %w", err)
	}

	return nil
}
