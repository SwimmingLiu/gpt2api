package account

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/432539/gpt2api/internal/account/importcore"
)

type fakeImportCore struct {
	result        *importcore.ImportResult
	err           error
	gotCandidates []importcore.ImportCandidate
	gotOptions    importcore.ImportOptions
}

func (f *fakeImportCore) Import(_ context.Context, candidates []importcore.ImportCandidate, opt importcore.ImportOptions) (*importcore.ImportResult, error) {
	f.gotCandidates = append([]importcore.ImportCandidate(nil), candidates...)
	f.gotOptions = opt
	if f.result == nil {
		f.result = &importcore.ImportResult{}
	}
	return f.result, f.err
}

func TestImportTokensEndpointPreservesSummaryShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	token := testJWT(t, map[string]any{
		"email":              "user@example.com",
		"chatgpt_account_id": "acct-9",
		"exp":                float64(time.Now().Add(time.Hour).Unix()),
	})

	h := NewHandler(nil)
	h.importCore = &fakeImportCore{
		result: &importcore.ImportResult{
			Total:   1,
			Created: 1,
			Results: []importcore.ImportLineResult{{
				Email:   "user@example.com",
				Source:  "line:1",
				Status:  "created",
				Warning: "token_expiry_unknown",
				ID:      9,
			}},
		},
	}
	r := gin.New()
	r.POST("/api/admin/accounts/import-tokens", h.ImportTokens)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/accounts/import-tokens",
		strings.NewReader(`{"mode":"at","tokens":"`+token+`","target_pool_id":7,"resolve_identity":false,"kick_refresh":false,"kick_quota_probe":false}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"created":1`) || !strings.Contains(w.Body.String(), `"total":1`) {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
	if h.importCore.(*fakeImportCore).gotOptions.TargetPoolID != 7 {
		t.Fatalf("expected token import target_pool_id to forward, got %+v", h.importCore.(*fakeImportCore).gotOptions)
	}
	if h.importCore.(*fakeImportCore).gotOptions.ResolveIdentity || h.importCore.(*fakeImportCore).gotOptions.KickRefresh || h.importCore.(*fakeImportCore).gotOptions.KickQuotaProbe {
		t.Fatalf("expected token import advanced options to honor request, got %+v", h.importCore.(*fakeImportCore).gotOptions)
	}
	body := decodeImportSummaryBody(t, w.Body.Bytes())
	if len(body.Data.Results) != 1 {
		t.Fatalf("expected 1 result row, got %+v", body.Data.Results)
	}
	row := body.Data.Results[0]
	if row.SourceType != "access_token_text" || row.SourceRef != "line:1" {
		t.Fatalf("expected metadata fields preserved, got %+v", row)
	}
	if len(row.Warnings) != 1 || row.Warnings[0] != "token_expiry_unknown" {
		t.Fatalf("expected warnings preserved, got %+v", row.Warnings)
	}
}

func TestImportTokensEndpointATKeepsChatGPTAccountID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token := testJWT(t, map[string]any{
		"email":              "user@example.com",
		"chatgpt_account_id": "acct-123",
		"exp":                float64(time.Now().Add(time.Hour).Unix()),
	})
	fakeCore := &fakeImportCore{
		result: &importcore.ImportResult{
			Total:   1,
			Created: 1,
			Results: []importcore.ImportLineResult{{Email: "user@example.com", Status: "created", ID: 9}},
		},
	}
	h := NewHandler(nil)
	h.importCore = fakeCore

	r := gin.New()
	r.POST("/api/admin/accounts/import-tokens", h.ImportTokens)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/accounts/import-tokens",
		strings.NewReader(`{"mode":"at","tokens":"`+token+`"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if len(fakeCore.gotCandidates) != 1 || fakeCore.gotCandidates[0].ChatGPTAccountID != "acct-123" {
		t.Fatalf("expected AT candidate to keep chatgpt_account_id, got %+v", fakeCore.gotCandidates)
	}
}

func TestImportEndpointUsesUnifiedImportCore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fakeCore := &fakeImportCore{
		result: &importcore.ImportResult{
			Total:   1,
			Created: 1,
			Results: []importcore.ImportLineResult{{
				Email:   "user@example.com",
				Source:  "inline",
				Status:  "created",
				Warning: "access_token_expiring_soon",
				ID:      10,
			}},
		},
	}
	h := NewHandler(nil)
	h.importCore = fakeCore

	r := gin.New()
	r.POST("/api/admin/accounts/import", h.Import)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/accounts/import",
		strings.NewReader(`{"text":"{\"access_token\":\"tok-a\",\"email\":\"user@example.com\"}","source_kind":"auto","update_existing":false,"default_proxy_id":5,"target_pool_id":7,"resolve_identity":false,"kick_refresh":false,"kick_quota_probe":false}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if len(fakeCore.gotCandidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(fakeCore.gotCandidates))
	}
	if fakeCore.gotOptions.DefaultProxyID != 5 || fakeCore.gotOptions.TargetPoolID != 7 {
		t.Fatalf("unexpected import options: %+v", fakeCore.gotOptions)
	}
	if fakeCore.gotOptions.ResolveIdentity || fakeCore.gotOptions.KickRefresh || fakeCore.gotOptions.KickQuotaProbe {
		t.Fatalf("expected advanced options to honor request, got %+v", fakeCore.gotOptions)
	}
	if !strings.Contains(w.Body.String(), `"created":1`) || !strings.Contains(w.Body.String(), `"total":1`) {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
	body := decodeImportSummaryBody(t, w.Body.Bytes())
	if len(body.Data.Results) != 1 {
		t.Fatalf("expected 1 result row, got %+v", body.Data.Results)
	}
	row := body.Data.Results[0]
	if row.SourceType != "cpa_file" || row.SourceRef != "inline" {
		t.Fatalf("expected import metadata fields preserved, got %+v", row)
	}
	if len(row.Warnings) != 1 || row.Warnings[0] != "access_token_expiring_soon" {
		t.Fatalf("expected warnings preserved, got %+v", row.Warnings)
	}
}

func TestImportMultipartEndpointForwardsTargetPoolID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fakeCore := &fakeImportCore{
		result: &importcore.ImportResult{
			Total:   1,
			Created: 1,
			Results: []importcore.ImportLineResult{{Email: "user@example.com", Status: "created", ID: 10}},
		},
	}
	h := NewHandler(nil)
	h.importCore = fakeCore

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("target_pool_id", "7"); err != nil {
		t.Fatalf("WriteField target_pool_id: %v", err)
	}
	part, err := writer.CreateFormFile("files", "accounts.json")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte(`{"access_token":"tok-a","email":"user@example.com"}`)); err != nil {
		t.Fatalf("part.Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	r := gin.New()
	r.POST("/api/admin/accounts/import", h.Import)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/accounts/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if fakeCore.gotOptions.TargetPoolID != 7 {
		t.Fatalf("expected multipart target_pool_id to forward, got %+v", fakeCore.gotOptions)
	}
}

func TestCreateEndpointUsesUnifiedImportCore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fakeCore := &fakeImportCore{
		result: &importcore.ImportResult{
			Total:   1,
			Created: 1,
			Results: []importcore.ImportLineResult{{Email: "user@example.com", Status: "created", ID: 9}},
		},
	}
	h := NewHandler(nil)
	h.importCore = fakeCore
	h.accountLookup = func(_ context.Context, id uint64) (*Account, error) {
		return &Account{
			ID:              id,
			Email:           "user@example.com",
			ClientID:        "app_manual",
			AccountType:     "codex",
			PlanType:        "plus",
			Status:          StatusHealthy,
			DailyImageQuota: 234,
			OAIDeviceID:     "generated-device-id",
		}, nil
	}

	r := gin.New()
	r.POST("/api/admin/accounts", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/accounts",
		strings.NewReader(`{"email":"user@example.com","auth_token":"tok-a","client_id":"app_manual","proxy_id":3,"target_pool_id":7,"daily_image_quota":234,"notes":"operator-note","resolve_identity":false,"kick_refresh":false,"kick_quota_probe":false}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if len(fakeCore.gotCandidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(fakeCore.gotCandidates))
	}
	if fakeCore.gotOptions.DefaultProxyID != 3 || fakeCore.gotOptions.TargetPoolID != 7 || fakeCore.gotOptions.UpdateExisting {
		t.Fatalf("unexpected import options: %+v", fakeCore.gotOptions)
	}
	if fakeCore.gotOptions.ResolveIdentity || fakeCore.gotOptions.KickRefresh || fakeCore.gotOptions.KickQuotaProbe {
		t.Fatalf("expected manual create to forward supported options, got %+v", fakeCore.gotOptions)
	}
	notes, quota, ok := decodeManualCreateCompat(fakeCore.gotCandidates[0])
	if !ok || notes != "operator-note" || quota != 234 {
		t.Fatalf("expected manual create compat metadata, got ok=%v notes=%q quota=%d candidate=%+v", ok, notes, quota, fakeCore.gotCandidates[0])
	}
	if fakeCore.gotCandidates[0].OAIDeviceID == "" {
		t.Fatalf("expected manual create to auto-populate oai_device_id, got %+v", fakeCore.gotCandidates[0])
	}
	if !strings.Contains(w.Body.String(), `"id":9`) || !strings.Contains(w.Body.String(), `"email":"user@example.com"`) {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
}

func TestCreateEndpointDuplicateEmailDoesNotSilentlyUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fakeCore := &fakeImportCore{
		result: &importcore.ImportResult{
			Total:   1,
			Skipped: 1,
			Results: []importcore.ImportLineResult{{Email: "user@example.com", Status: "skipped", Reason: "account_exists", ID: 9}},
		},
	}
	h := NewHandler(nil)
	h.importCore = fakeCore

	r := gin.New()
	r.POST("/api/admin/accounts", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/accounts",
		strings.NewReader(`{"email":"user@example.com","auth_token":"tok-a"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected duplicate create to fail, got %d body=%s", w.Code, w.Body.String())
	}
	if fakeCore.gotOptions.UpdateExisting {
		t.Fatalf("expected create route to keep create-only semantics, got %+v", fakeCore.gotOptions)
	}
}

func TestMergeImportedNotesKeepsExistingOperatorNotesForSub2APIUpdate(t *testing.T) {
	candidate := importcore.ImportCandidate{
		SourceType: "sub2api",
		Notes:      "chatgpt-user_example.com",
	}
	got := mergeImportedNotes(candidate, "operator-note")
	if got != "operator-note" {
		t.Fatalf("expected existing notes to win for sub2api update, got %q", got)
	}
}

func testJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "none", "typ": "JWT"})
	if err != nil {
		t.Fatalf("json.Marshal header: %v", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("json.Marshal claims: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

type importSummaryBody struct {
	Data struct {
		Results []struct {
			SourceType string   `json:"source_type"`
			SourceRef  string   `json:"source_ref"`
			Warnings   []string `json:"warnings"`
		} `json:"results"`
	} `json:"data"`
}

func decodeImportSummaryBody(t *testing.T, raw []byte) importSummaryBody {
	t.Helper()
	var body importSummaryBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("json.Unmarshal response: %v body=%s", err, string(raw))
	}
	return body
}
