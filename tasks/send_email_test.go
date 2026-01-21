package tasks

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"tv-pipelines-timken/configs"
	"tv-pipelines-timken/types"
)

func init() {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func TestCollectEmailAddresses(t *testing.T) {
	tests := []struct {
		name   string
		shipTo []string
		soldTo []string
		want   int
	}{
		{
			name:   "both arrays with values",
			shipTo: []string{"ship1@example.com", "ship2@example.com"},
			soldTo: []string{"sold1@example.com"},
			want:   3,
		},
		{
			name:   "deduplicate",
			shipTo: []string{"same@example.com"},
			soldTo: []string{"same@example.com"},
			want:   1,
		},
		{
			name:   "empty arrays",
			shipTo: []string{},
			soldTo: []string{},
			want:   0,
		},
		{
			name:   "whitespace trimmed",
			shipTo: []string{"  email@example.com  "},
			soldTo: []string{},
			want:   1,
		},
		{
			name:   "empty strings filtered",
			shipTo: []string{"", "  ", "valid@example.com"},
			soldTo: []string{""},
			want:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectEmailAddresses(tt.shipTo, tt.soldTo)
			if len(got) != tt.want {
				t.Errorf("collectEmailAddresses() = %d addresses, want %d", len(got), tt.want)
			}
		})
	}
}

func TestSendEmail_NotEnabled(t *testing.T) {
	cocData := &types.COCData{
		Items: []types.COCItem{
			{
				SendCOCEmails: 0, // Not enabled
			},
		},
	}

	cfg := &configs.Config{}

	sent, err := SendEmail(context.Background(), cfg, cocData, []byte("pdf"), "test.pdf")
	if err != nil {
		t.Fatalf("SendEmail() error = %v", err)
	}
	if sent {
		t.Error("SendEmail() = true, want false when not enabled")
	}
}

func TestSendEmail_NoRecipients(t *testing.T) {
	cocData := &types.COCData{
		Items: []types.COCItem{
			{
				SendCOCEmails:            1, // Enabled
				ShipToNotificationEmails: []string{},
				SoldToNotificationEmails: []string{},
			},
		},
	}

	cfg := &configs.Config{}

	_, err := SendEmail(context.Background(), cfg, cocData, []byte("pdf"), "test.pdf")
	if err == nil {
		t.Error("SendEmail() expected error when no recipients")
	}
}

func TestSendEmail_InvalidEmail(t *testing.T) {
	cocData := &types.COCData{
		Items: []types.COCItem{
			{
				SendCOCEmails:            1,
				ShipToNotificationEmails: []string{"not-an-email"},
			},
		},
	}

	cfg := &configs.Config{}

	_, err := SendEmail(context.Background(), cfg, cocData, []byte("pdf"), "test.pdf")
	if err == nil {
		t.Error("SendEmail() expected error for invalid email address")
	}
}

func TestSendEmail_NilData(t *testing.T) {
	cfg := &configs.Config{}

	_, err := SendEmail(context.Background(), cfg, nil, []byte("pdf"), "test.pdf")
	if err == nil {
		t.Error("SendEmail() expected error for nil COC data")
	}
}
