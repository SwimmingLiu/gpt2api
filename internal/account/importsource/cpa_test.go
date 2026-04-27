package importsource

import "testing"

func TestParseCPAFileExtractsAccessToken(t *testing.T) {
	candidates, err := ParseCPAFile("sample.json", []byte(`{"access_token":"tok-a","email":"a@example.com"}`))
	if err != nil {
		t.Fatalf("ParseCPAFile returned error: %v", err)
	}
	if len(candidates) != 1 || candidates[0].AccessToken != "tok-a" {
		t.Fatalf("unexpected CPA candidates: %+v", candidates)
	}
}
