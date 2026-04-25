package importcore

import (
	"testing"
	"time"
)

func TestClassifyCredentialStateSkipsExpiredATOnly(t *testing.T) {
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	exp := now.Add(-1 * time.Minute)

	candidate := ImportCandidate{
		SourceType:     "access_token_text",
		SourceRef:      "line:1",
		AccessToken:    "expired-at",
		TokenExpiresAt: &exp,
	}

	state := ClassifyCredentialState(candidate, now, 900)
	if !state.SkipImport {
		t.Fatalf("expected expired AT-only candidate to be skipped, got %+v", state)
	}
	if state.Warning != "" {
		t.Fatalf("expected no warning for skipped candidate, got %q", state.Warning)
	}
}

func TestClassifyCredentialStateKeepsExpiredRefreshableCandidate(t *testing.T) {
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	exp := now.Add(-1 * time.Minute)

	candidate := ImportCandidate{
		SourceType:     "sub2api_json",
		SourceRef:      "account:0",
		AccessToken:    "expired-at",
		RefreshToken:   "rt-token",
		TokenExpiresAt: &exp,
	}

	state := ClassifyCredentialState(candidate, now, 900)
	if state.SkipImport {
		t.Fatalf("expected refreshable candidate to be kept, got %+v", state)
	}
	if state.Warning != "access_token_expired_but_refreshable" {
		t.Fatalf("unexpected warning: %q", state.Warning)
	}
}
