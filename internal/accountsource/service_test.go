package accountsource

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/432539/gpt2api/internal/account/importcore"
	"github.com/432539/gpt2api/pkg/crypto"
)

type memoryStore struct {
	nextID uint64
	items  map[uint64]*Source
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		nextID: 1,
		items:  map[uint64]*Source{},
	}
}

func (m *memoryStore) ListSources(_ context.Context) ([]*Source, error) {
	out := make([]*Source, 0, len(m.items))
	for id := uint64(1); id < m.nextID; id++ {
		if item, ok := m.items[id]; ok && item.DeletedAt == nil {
			out = append(out, cloneSource(item))
		}
	}
	return out, nil
}

func (m *memoryStore) GetSourceByID(_ context.Context, id uint64) (*Source, error) {
	item, ok := m.items[id]
	if !ok || item.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return cloneSource(item), nil
}

func (m *memoryStore) CreateSource(_ context.Context, src *Source) error {
	now := time.Now().UTC()
	src.ID = m.nextID
	src.CreatedAt = now
	src.UpdatedAt = now
	m.nextID++
	m.items[src.ID] = cloneSource(src)
	return nil
}

func (m *memoryStore) UpdateSource(_ context.Context, src *Source) error {
	if _, ok := m.items[src.ID]; !ok {
		return ErrNotFound
	}
	current := cloneSource(src)
	current.UpdatedAt = time.Now().UTC()
	m.items[src.ID] = current
	return nil
}

func (m *memoryStore) SoftDeleteSource(_ context.Context, id uint64) error {
	item, ok := m.items[id]
	if !ok || item.DeletedAt != nil {
		return ErrNotFound
	}
	now := time.Now().UTC()
	item.DeletedAt = &now
	item.Enabled = false
	m.items[id] = cloneSource(item)
	return nil
}

func cloneSource(src *Source) *Source {
	if src == nil {
		return nil
	}
	out := *src
	if src.DeletedAt != nil {
		ts := *src.DeletedAt
		out.DeletedAt = &ts
	}
	return &out
}

type fakeImporter struct {
	result        *importcore.ImportResult
	err           error
	gotCandidates []importcore.ImportCandidate
	gotOptions    importcore.ImportOptions
}

func (f *fakeImporter) Import(_ context.Context, candidates []importcore.ImportCandidate, opt importcore.ImportOptions) (*importcore.ImportResult, error) {
	f.gotCandidates = append([]importcore.ImportCandidate(nil), candidates...)
	f.gotOptions = opt
	if f.result == nil {
		f.result = &importcore.ImportResult{}
	}
	return f.result, f.err
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestCreateAndListSourcesHideSecrets(t *testing.T) {
	store := newMemoryStore()
	svc := NewService(store, mustCipher(t), nil)

	created, err := svc.Create(context.Background(), CreateInput{
		SourceType:     "sub2api",
		Name:           "Remote One",
		BaseURL:        " https://example.com/root/ ",
		AuthMode:       "password",
		Email:          "admin@example.com",
		GroupID:        "group-1",
		APIKey:         "api-key-1",
		Password:       "pass-1",
		SecretKey:      "secret-1",
		DefaultProxyID: 11,
		TargetPoolID:   22,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if !created.Enabled {
		t.Fatalf("expected source enabled by default, got %+v", created)
	}
	if created.BaseURL != "https://example.com/root" {
		t.Fatalf("expected normalized base_url, got %q", created.BaseURL)
	}
	if !created.HasAPIKey || !created.HasPassword || !created.HasSecretKey {
		t.Fatalf("expected masked secret flags, got %+v", created)
	}

	stored := store.items[created.ID]
	if stored == nil {
		t.Fatal("expected stored source")
	}
	if stored.APIKeyEnc == "" || stored.APIKeyEnc == "api-key-1" {
		t.Fatalf("expected api key encrypted, got %+v", stored)
	}
	if stored.PasswordEnc == "" || stored.PasswordEnc == "pass-1" {
		t.Fatalf("expected password encrypted, got %+v", stored)
	}
	if stored.SecretKeyEnc == "" || stored.SecretKeyEnc == "secret-1" {
		t.Fatalf("expected secret key encrypted, got %+v", stored)
	}

	got, err := svc.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !got.HasAPIKey || !got.HasPassword || !got.HasSecretKey {
		t.Fatalf("expected Get to keep secret flags only, got %+v", got)
	}

	items, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 source, got %d", len(items))
	}
	if !items[0].HasAPIKey || !items[0].HasPassword || !items[0].HasSecretKey {
		t.Fatalf("expected List to keep secret flags only, got %+v", items[0])
	}
}

func TestListSub2APIGroupsAndAccountsParsesEnvelope(t *testing.T) {
	var loginCalled bool
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
			loginCalled = true
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode login body: %v", err)
			}
			if body["email"] != "owner@example.com" || body["password"] != "top-secret" {
				t.Fatalf("unexpected login body: %+v", body)
			}
			return jsonResponse(t, map[string]any{
				"code":    0,
				"message": "ok",
				"data": map[string]any{
					"access_token": "token-1",
				},
			}), nil
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/groups":
			if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
				t.Fatalf("unexpected groups auth header: %q", got)
			}
			return jsonResponse(t, map[string]any{
				"code": 0,
				"data": map[string]any{
					"items": []map[string]any{
						{"id": "grp-1", "name": "Primary"},
						{"id": 2, "name": "Backup"},
					},
				},
			}), nil
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/accounts":
			if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
				t.Fatalf("unexpected accounts auth header: %q", got)
			}
			if got := r.URL.Query().Get("group"); got != "group-9" {
				t.Fatalf("expected group query to use source default, got %q", got)
			}
			return jsonResponse(t, map[string]any{
				"list": []map[string]any{
					{
						"id":       "acc-1",
						"name":     "codex_user_example.com",
						"platform": "openai",
						"type":     "oauth",
						"extra": map[string]any{
							"email": "user@example.com",
						},
					},
					{
						"id":       "acc-2",
						"name":     "chatgpt_jane_example.com",
						"platform": "openai",
						"type":     "oauth",
						"email":    "jane@example.com",
					},
				},
			}), nil
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("not found"))}, nil
		}
	})}

	store := newMemoryStore()
	svc := NewService(store, mustCipher(t), nil)
	svc.SetHTTPClient(client)

	source, err := svc.Create(context.Background(), CreateInput{
		SourceType: "sub2api",
		Name:       "sub2api-source",
		BaseURL:    "https://sub2api.example.test",
		AuthMode:   "password",
		Email:      "owner@example.com",
		Password:   "top-secret",
		GroupID:    "group-9",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	groups, err := svc.ListSub2APIGroups(context.Background(), source.ID)
	if err != nil {
		t.Fatalf("ListSub2APIGroups returned error: %v", err)
	}
	if !loginCalled {
		t.Fatal("expected login flow for password auth")
	}
	if len(groups) != 2 || groups[0].ID != "grp-1" || groups[1].ID != "2" {
		t.Fatalf("unexpected groups: %+v", groups)
	}

	accounts, err := svc.ListSub2APIAccounts(context.Background(), source.ID)
	if err != nil {
		t.Fatalf("ListSub2APIAccounts returned error: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %+v", accounts)
	}
	if accounts[0].Email != "user@example.com" || accounts[1].Email != "jane@example.com" {
		t.Fatalf("unexpected account emails: %+v", accounts)
	}
}

func TestListCPAFilesParsesEnvelope(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v0/management/auth-files" {
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("not found"))}, nil
		}
		if got := r.Header.Get("Authorization"); got != "Bearer bearer-1" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		return jsonResponse(t, map[string]any{
			"code": 0,
			"data": map[string]any{
				"files": []string{"alpha.json", "beta.json"},
			},
		}), nil
	})}

	store := newMemoryStore()
	svc := NewService(store, mustCipher(t), nil)
	svc.SetHTTPClient(client)

	source, err := svc.Create(context.Background(), CreateInput{
		SourceType: "cpa",
		Name:       "cpa-source",
		BaseURL:    "https://cpa.example.test",
		AuthMode:   "bearer",
		SecretKey:  "bearer-1",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	files, err := svc.ListCPAFiles(context.Background(), source.ID)
	if err != nil {
		t.Fatalf("ListCPAFiles returned error: %v", err)
	}
	if len(files) != 2 || files[0].Name != "alpha.json" || files[1].Name != "beta.json" {
		t.Fatalf("unexpected files: %+v", files)
	}
}

func TestListCPAFilesAcceptsObjectArrayWithoutDuplicateGarbageRows(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v0/management/auth-files" {
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("not found"))}, nil
		}
		return jsonResponse(t, map[string]any{
			"files": []map[string]any{
				{"name": "alpha.json", "email": "alpha@example.com"},
				{"name": "beta.json", "email": "beta@example.com"},
			},
		}), nil
	})}

	store := newMemoryStore()
	svc := NewService(store, mustCipher(t), nil)
	svc.SetHTTPClient(client)

	source, err := svc.Create(context.Background(), CreateInput{
		SourceType: "cpa",
		Name:       "cpa-source",
		BaseURL:    "https://cpa.example.test",
		AuthMode:   "api_key",
		APIKey:     "api-key-1",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	files, err := svc.ListCPAFiles(context.Background(), source.ID)
	if err != nil {
		t.Fatalf("ListCPAFiles returned error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected exactly 2 files, got %+v", files)
	}
	if files[0].Name != "alpha.json" || files[1].Name != "beta.json" {
		t.Fatalf("unexpected files: %+v", files)
	}
}

func TestImportSelectedSub2APIForwardsOptionsAndBuildsCandidates(t *testing.T) {
	importer := &fakeImporter{
		result: &importcore.ImportResult{
			Total:   1,
			Created: 1,
			Results: []importcore.ImportLineResult{{
				Email:  "user@example.com",
				Source: "remote:acc-1",
				Status: "created",
				ID:     99,
			}},
		},
	}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
			return jsonResponse(t, map[string]any{
				"code": 0,
				"data": map[string]any{
					"access_token": "token-2",
				},
			}), nil
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/accounts/acc-1":
			if got := r.Header.Get("Authorization"); got != "Bearer token-2" {
				t.Fatalf("unexpected detail auth header: %q", got)
			}
			return jsonResponse(t, map[string]any{
				"data": map[string]any{
					"id":       "acc-1",
					"name":     "codex_user_example.com",
					"platform": "openai",
					"type":     "oauth",
					"plan":     "plus",
					"extra": map[string]any{
						"email": "user@example.com",
					},
					"credentials": map[string]any{
						"access_token":       "at-1",
						"refresh_token":      "rt-1",
						"session_token":      "st-1",
						"client_id":          "client-1",
						"chatgpt_account_id": "chatgpt-1",
					},
				},
			}), nil
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("not found"))}, nil
		}
	})}

	store := newMemoryStore()
	svc := NewService(store, mustCipher(t), importer)
	svc.SetHTTPClient(client)

	source, err := svc.Create(context.Background(), CreateInput{
		SourceType: "sub2api",
		Name:       "sub2api-import",
		BaseURL:    "https://sub2api.example.test",
		AuthMode:   "password",
		Email:      "owner@example.com",
		Password:   "pw-2",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	summary, err := svc.ImportSelected(context.Background(), source.ID, ImportSelectedInput{
		AccountIDs:      []string{"acc-1"},
		UpdateExisting:  boolPtr(false),
		DefaultProxyID:  uint64Ptr(7),
		TargetPoolID:    uint64Ptr(8),
		ResolveIdentity: boolPtr(false),
		KickRefresh:     boolPtr(false),
		KickQuotaProbe:  boolPtr(false),
	})
	if err != nil {
		t.Fatalf("ImportSelected returned error: %v", err)
	}
	if summary.Created != 1 || summary.Total != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if importer.gotOptions.UpdateExisting || importer.gotOptions.DefaultProxyID != 7 || importer.gotOptions.TargetPoolID != 8 {
		t.Fatalf("unexpected import options: %+v", importer.gotOptions)
	}
	if importer.gotOptions.ResolveIdentity || importer.gotOptions.KickRefresh || importer.gotOptions.KickQuotaProbe {
		t.Fatalf("expected explicit false import flags, got %+v", importer.gotOptions)
	}
	if len(importer.gotCandidates) != 1 {
		t.Fatalf("expected 1 candidate, got %+v", importer.gotCandidates)
	}
	candidate := importer.gotCandidates[0]
	if candidate.AccessToken != "at-1" || candidate.RefreshToken != "rt-1" || candidate.SessionToken != "st-1" {
		t.Fatalf("unexpected candidate tokens: %+v", candidate)
	}
	if candidate.Email != "user@example.com" || candidate.ClientID != "client-1" || candidate.ChatGPTAccountID != "chatgpt-1" {
		t.Fatalf("unexpected candidate identity: %+v", candidate)
	}
	if candidate.PlanType != "plus" {
		t.Fatalf("expected plan type from remote detail, got %+v", candidate)
	}
}

func TestImportSelectedFallsBackToSourceDefaults(t *testing.T) {
	importer := &fakeImporter{
		result: &importcore.ImportResult{
			Total:   1,
			Created: 1,
			Results: []importcore.ImportLineResult{{
				Email:  "cpa@example.com",
				Source: "file:bundle.json",
				Status: "created",
				ID:     100,
			}},
		},
	}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v0/management/auth-files/download":
			if got := r.Header.Get("X-API-Key"); got != "api-key-9" {
				t.Fatalf("unexpected api key header: %q", got)
			}
			if got := r.URL.Query().Get("name"); got != "bundle.json" {
				t.Fatalf("unexpected download query name: %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"at-cpa","email":"cpa@example.com"}`)),
			}, nil
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("not found"))}, nil
		}
	})}

	store := newMemoryStore()
	svc := NewService(store, mustCipher(t), importer)
	svc.SetHTTPClient(client)

	source, err := svc.Create(context.Background(), CreateInput{
		SourceType:     "cpa",
		Name:           "cpa-import",
		BaseURL:        "https://cpa.example.test",
		AuthMode:       "api_key",
		APIKey:         "api-key-9",
		DefaultProxyID: 31,
		TargetPoolID:   41,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	summary, err := svc.ImportSelected(context.Background(), source.ID, ImportSelectedInput{
		FileNames: []string{"bundle.json"},
	})
	if err != nil {
		t.Fatalf("ImportSelected returned error: %v", err)
	}
	if summary.Created != 1 || summary.Total != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if importer.gotOptions.DefaultProxyID != 31 || importer.gotOptions.TargetPoolID != 41 {
		t.Fatalf("expected source defaults to backfill proxy/pool, got %+v", importer.gotOptions)
	}
	if !importer.gotOptions.UpdateExisting || !importer.gotOptions.ResolveIdentity || !importer.gotOptions.KickRefresh || !importer.gotOptions.KickQuotaProbe {
		t.Fatalf("expected default import options, got %+v", importer.gotOptions)
	}
	if len(importer.gotCandidates) != 1 || importer.gotCandidates[0].Email != "cpa@example.com" {
		t.Fatalf("unexpected imported candidates: %+v", importer.gotCandidates)
	}
}

func TestImportSelectedRejectsDisabledSource(t *testing.T) {
	importer := &fakeImporter{}
	store := newMemoryStore()
	svc := NewService(store, mustCipher(t), importer)

	source, err := svc.Create(context.Background(), CreateInput{
		SourceType: "cpa",
		Name:       "disabled-cpa",
		BaseURL:    "https://cpa.example.test",
		AuthMode:   "api_key",
		APIKey:     "api-key-1",
		Enabled:    boolPtr(false),
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	_, err = svc.ImportSelected(context.Background(), source.ID, ImportSelectedInput{
		FileNames: []string{"bundle.json"},
	})
	if err == nil || !errors.Is(err, ErrBadRequest) {
		t.Fatalf("expected disabled source import to fail with bad request, got %v", err)
	}
	if len(importer.gotCandidates) != 0 {
		t.Fatalf("expected importer not to run for disabled source, got %+v", importer.gotCandidates)
	}
}

func mustCipher(t *testing.T) *crypto.AESGCM {
	t.Helper()
	c, err := crypto.NewAESGCM(strings.Repeat("11", 32))
	if err != nil {
		t.Fatalf("NewAESGCM: %v", err)
	}
	return c
}

func jsonResponse(t *testing.T, body any) *http.Response {
	t.Helper()
	buf := &strings.Builder{}
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		t.Fatalf("Encode response: %v", err)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(buf.String())),
	}
}

func boolPtr(v bool) *bool       { return &v }
func uint64Ptr(v uint64) *uint64 { return &v }
