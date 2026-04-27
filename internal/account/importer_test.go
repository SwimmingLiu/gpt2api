package account

import (
	"strings"
	"testing"
)

func TestParseJSONBlobSupportsJSONArray(t *testing.T) {
	raw := `[{"access_token":"tok-a","email":"a@example.com"},{"accessToken":"tok-b","email":"b@example.com"}]`
	items, err := ParseJSONBlob(raw)
	if err != nil {
		t.Fatalf("ParseJSONBlob returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Email != "a@example.com" || items[1].Email != "b@example.com" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestParseJSONBlobSupportsJSONL(t *testing.T) {
	raw := "{\"access_token\":\"tok-a\",\"email\":\"a@example.com\"}\n{\"accessToken\":\"tok-b\",\"email\":\"b@example.com\"}\n"
	items, err := ParseJSONBlob(raw)
	if err != nil {
		t.Fatalf("ParseJSONBlob returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestParseJSONBlobSupportsCamelAndSnakeCaseTokenJSON(t *testing.T) {
	snake, err := ParseJSONBlob(`{"access_token":"tok-a","refresh_token":"rt-a","email":"a@example.com","client_id":"app_a"}`)
	if err != nil {
		t.Fatalf("ParseJSONBlob snake_case error: %v", err)
	}
	camel, err := ParseJSONBlob(`{"accessToken":"tok-b","refreshToken":"rt-b","email":"b@example.com","clientId":"app_b"}`)
	if err != nil {
		t.Fatalf("ParseJSONBlob camelCase error: %v", err)
	}
	if len(snake) != 1 || len(camel) != 1 {
		t.Fatalf("unexpected lengths: snake=%d camel=%d", len(snake), len(camel))
	}
	if snake[0].RefreshToken != "rt-a" || camel[0].RefreshToken != "rt-b" || camel[0].ClientID != "app_b" {
		t.Fatalf("unexpected parsed items: snake=%+v camel=%+v", snake, camel)
	}
}

func TestParseJSONBlobReturnsErrorWhenTokenJSONMissingEmail(t *testing.T) {
	_, err := ParseJSONBlob(`{"access_token":"tok-a"}`)
	if err == nil {
		t.Fatal("expected error for token json without email")
	}
	if !strings.Contains(err.Error(), "email") {
		t.Fatalf("expected email-related error, got %v", err)
	}
}
