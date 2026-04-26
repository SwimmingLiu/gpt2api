package account

import (
	"encoding/json"
	"testing"
)

func TestCreateInputJSONAllowsEmptyTokenExpiresAt(t *testing.T) {
	var in CreateInput
	err := json.Unmarshal([]byte(`{
		"email":"test@example.com",
		"auth_token":"dummy-token",
		"token_expires_at":""
	}`), &in)
	if err != nil {
		t.Fatalf("expected empty token_expires_at to decode successfully, got error: %v", err)
	}
}

func TestUpdateInputJSONAllowsEmptyTokenExpiresAt(t *testing.T) {
	var in UpdateInput
	err := json.Unmarshal([]byte(`{
		"email":"test@example.com",
		"token_expires_at":""
	}`), &in)
	if err != nil {
		t.Fatalf("expected empty token_expires_at to decode successfully, got error: %v", err)
	}
}
