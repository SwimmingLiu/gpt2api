package accountsource

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeHandlerService struct {
	listFn   func(ctx *gin.Context) ([]*SourceView, error)
	createFn func(ctx *gin.Context, in CreateInput) (*SourceView, error)
	importFn func(ctx *gin.Context, sourceID uint64, in ImportSelectedInput) (*ImportSummary, error)
}

func (f *fakeHandlerService) List(ctx *gin.Context) ([]*SourceView, error) {
	if f.listFn == nil {
		return nil, nil
	}
	return f.listFn(ctx)
}

func (f *fakeHandlerService) Create(ctx *gin.Context, in CreateInput) (*SourceView, error) {
	if f.createFn == nil {
		return nil, nil
	}
	return f.createFn(ctx, in)
}

func (f *fakeHandlerService) Get(_ *gin.Context, _ uint64) (*SourceView, error) { return nil, nil }
func (f *fakeHandlerService) Update(_ *gin.Context, _ uint64, _ UpdateInput) (*SourceView, error) {
	return nil, nil
}
func (f *fakeHandlerService) Delete(_ *gin.Context, _ uint64) error { return nil }
func (f *fakeHandlerService) ListSub2APIGroups(_ *gin.Context, _ uint64) ([]*Sub2APIGroup, error) {
	return nil, nil
}
func (f *fakeHandlerService) ListSub2APIAccounts(_ *gin.Context, _ uint64) ([]*Sub2APIAccount, error) {
	return nil, nil
}
func (f *fakeHandlerService) ListCPAFiles(_ *gin.Context, _ uint64) ([]*CPAFile, error) {
	return nil, nil
}
func (f *fakeHandlerService) ImportSelected(ctx *gin.Context, sourceID uint64, in ImportSelectedInput) (*ImportSummary, error) {
	if f.importFn == nil {
		return nil, nil
	}
	return f.importFn(ctx, sourceID, in)
}

func TestCreateAndListEndpointsKeepMaskedSecretFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)

	item := &SourceView{
		ID:             9,
		SourceType:     "sub2api",
		Name:           "demo",
		BaseURL:        "https://example.com",
		Enabled:        true,
		AuthMode:       "password",
		Email:          "owner@example.com",
		GroupID:        "grp-1",
		DefaultProxyID: 5,
		TargetPoolID:   6,
		HasAPIKey:      true,
		HasPassword:    true,
		HasSecretKey:   true,
	}
	svc := &fakeHandlerService{
		createFn: func(_ *gin.Context, in CreateInput) (*SourceView, error) {
			if in.Password != "secret-pass" || in.SecretKey != "secret-bearer" || in.APIKey != "api-key-1" {
				t.Fatalf("expected plaintext secret input to reach service, got %+v", in)
			}
			return item, nil
		},
		listFn: func(_ *gin.Context) ([]*SourceView, error) {
			return []*SourceView{item}, nil
		},
	}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/sources", h.Create)
	r.GET("/sources", h.List)

	req := httptest.NewRequest(http.MethodPost, "/sources", strings.NewReader(`{
		"source_type":"sub2api",
		"name":"demo",
		"base_url":"https://example.com",
		"auth_mode":"password",
		"email":"owner@example.com",
		"password":"secret-pass",
		"secret_key":"secret-bearer",
		"api_key":"api-key-1"
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"has_password":true`) || !strings.Contains(body, `"has_secret_key":true`) || !strings.Contains(body, `"has_api_key":true`) {
		t.Fatalf("expected masked secret flags in create response, got %s", body)
	}
	if strings.Contains(body, "secret-pass") || strings.Contains(body, "secret-bearer") || strings.Contains(body, "api-key-1") {
		t.Fatalf("expected create response to hide plaintext secrets, got %s", body)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/sources", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listW.Code, listW.Body.String())
	}
	if !strings.Contains(listW.Body.String(), `"total":1`) {
		t.Fatalf("expected list total in response, got %s", listW.Body.String())
	}
}

func TestImportEndpointBindsSourceIDAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var gotID uint64
	var gotReq ImportSelectedInput
	svc := &fakeHandlerService{
		importFn: func(_ *gin.Context, sourceID uint64, in ImportSelectedInput) (*ImportSummary, error) {
			gotID = sourceID
			gotReq = in
			return &ImportSummary{
				Total:   1,
				Created: 1,
				Results: []ImportSummaryResultRow{{
					Index:      1,
					Email:      "user@example.com",
					Status:     "created",
					ID:         88,
					SourceType: "sub2api_remote",
					SourceRef:  "remote:acc-1",
				}},
			}, nil
		},
	}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/sources/:id/import", h.ImportSelected)

	req := httptest.NewRequest(http.MethodPost, "/sources/42/import", strings.NewReader(`{
		"account_ids":["acc-1"],
		"update_existing":false,
		"default_proxy_id":7,
		"target_pool_id":8,
		"resolve_identity":false,
		"kick_refresh":false,
		"kick_quota_probe":false
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if gotID != 42 {
		t.Fatalf("expected route source id to bind, got %d", gotID)
	}
	if len(gotReq.AccountIDs) != 1 || gotReq.AccountIDs[0] != "acc-1" {
		t.Fatalf("unexpected import request binding: %+v", gotReq)
	}
	if gotReq.UpdateExisting == nil || *gotReq.UpdateExisting {
		t.Fatalf("expected update_existing=false binding, got %+v", gotReq)
	}
	if !strings.Contains(w.Body.String(), `"created":1`) || !strings.Contains(w.Body.String(), `"source_ref":"remote:acc-1"`) {
		t.Fatalf("unexpected import response body: %s", w.Body.String())
	}
}
