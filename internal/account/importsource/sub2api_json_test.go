package importsource

import "testing"

func TestParseSub2APIJSONExtractsCredentials(t *testing.T) {
	raw := `{"accounts":[{"name":"chatgpt-user_example.com","platform":"chatgpt","credentials":{"access_token":"tok-a","refresh_token":"rt","session_token":"st","client_id":"app_x","chatgpt_account_id":"acc-1"},"extra":{"email":"user@example.com"}}]}`
	candidates, err := ParseSub2APIJSON([]byte(raw))
	if err != nil {
		t.Fatalf("ParseSub2APIJSON returned error: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Email != "user@example.com" || candidates[0].RefreshToken != "rt" {
		t.Fatalf("unexpected sub2api candidates: %+v", candidates)
	}
}
