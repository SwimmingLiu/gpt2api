package account

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

	h := NewHandler(nil)
	h.importCore = &fakeImportCore{
		result: &importcore.ImportResult{
			Total:   1,
			Created: 1,
			Results: []importcore.ImportLineResult{{Email: "user@example.com", Status: "created", ID: 9}},
		},
	}
	r := gin.New()
	r.POST("/api/admin/accounts/import-tokens", h.ImportTokens)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/accounts/import-tokens",
		strings.NewReader(`{"mode":"at","tokens":"tok-a","target_pool_id":7}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"created":1`) || !strings.Contains(w.Body.String(), `"total":1`) {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
}

func TestImportEndpointUsesUnifiedImportCore(t *testing.T) {
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

	r := gin.New()
	r.POST("/api/admin/accounts/import", h.Import)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/accounts/import",
		strings.NewReader(`{"text":"{\"access_token\":\"tok-a\",\"email\":\"user@example.com\"}","update_existing":false,"default_proxy_id":5,"target_pool_id":7}`))
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
	if !strings.Contains(w.Body.String(), `"created":1`) || !strings.Contains(w.Body.String(), `"total":1`) {
		t.Fatalf("unexpected response body: %s", w.Body.String())
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
			ID:          id,
			Email:       "user@example.com",
			ClientID:    "app_manual",
			AccountType: "codex",
			PlanType:    "plus",
			Status:      StatusHealthy,
		}, nil
	}

	r := gin.New()
	r.POST("/api/admin/accounts", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/accounts",
		strings.NewReader(`{"email":"user@example.com","auth_token":"tok-a","client_id":"app_manual","proxy_id":3,"target_pool_id":7}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if len(fakeCore.gotCandidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(fakeCore.gotCandidates))
	}
	if fakeCore.gotOptions.DefaultProxyID != 3 || fakeCore.gotOptions.TargetPoolID != 7 || !fakeCore.gotOptions.UpdateExisting {
		t.Fatalf("unexpected import options: %+v", fakeCore.gotOptions)
	}
	if !strings.Contains(w.Body.String(), `"id":9`) || !strings.Contains(w.Body.String(), `"email":"user@example.com"`) {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
}
