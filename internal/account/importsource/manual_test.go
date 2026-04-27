package importsource

import (
	"testing"
	"time"
)

func TestParseManualMapsFields(t *testing.T) {
	exp := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	candidates := ParseManual(ManualInput{
		Email:            " user@example.com ",
		AccessToken:      " tok-a ",
		RefreshToken:     " rt ",
		SessionToken:     " st ",
		ClientID:         " app_x ",
		ChatGPTAccountID: " acc-1 ",
		AccountType:      " chatgpt ",
		PlanType:         " plus ",
		TokenExpiresAt:   &exp,
		OAISessionID:     " sid ",
		OAIDeviceID:      " did ",
		Cookies:          " cookie=1 ",
		Notes:            " note ",
	})
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	got := candidates[0]
	if got.SourceType != "manual" || got.SourceRef != "admin_form" {
		t.Fatalf("unexpected source metadata: %+v", got)
	}
	if got.Email != "user@example.com" || got.AccessToken != "tok-a" || got.RefreshToken != "rt" {
		t.Fatalf("unexpected normalized fields: %+v", got)
	}
	if got.TokenExpiresAt == nil || !got.TokenExpiresAt.Equal(exp) {
		t.Fatalf("unexpected token expiry: %+v", got.TokenExpiresAt)
	}
}
