# Unified Account Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unify manual account creation, access-token text import, CPA file import, and local sub2api JSON import behind one backend import core that handles identity resolution, expired-token filtering, email-based upsert, default proxy binding, pool assignment, and refresh/quota post-processing.

**Architecture:** Add two leaf packages under `internal/account`: `importcore` for lifecycle policy, identity resolution, persistence orchestration, and post-processing; `importsource` for source-specific parsing into a shared `ImportCandidate` shape. Keep `internal/account` as the wiring layer that adapts existing DAO/service/refresher/prober/account-pool dependencies into narrow interfaces, and preserve all current admin routes while making them call the unified core. On the frontend, replace the monolithic import section in `Accounts.vue` with a shared import dialog and source panes that all submit the same advanced options.

**Tech Stack:** Go, Gin, sqlx, MySQL, Redis, Vue 3, TypeScript, Element Plus

---

## File Structure Map

### New backend files

- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/types.go`
  - Shared `ImportCandidate`, `ImportOptions`, `ImportResult`, credential lifecycle enums.
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/identity.go`
  - JWT claim parsing, lifecycle classification, optional remote identity resolution contract.
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/persist.go`
  - Email-based create/update logic and non-empty field merge rules.
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/postprocess.go`
  - Default proxy binding, pool membership upsert, refresh/probe kicks.
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/service.go`
  - End-to-end import orchestration.
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/service_test.go`
  - Core unit tests for expired-token filtering, upsert rules, and post-processing.
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/access_token.go`
  - Parse multiline access-token text / TXT content.
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/cpa.go`
  - Parse CPA JSON files.
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/sub2api_json.go`
  - Parse local sub2api export JSON.
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/manual.go`
  - Convert manual form input into a candidate without importing `account` and causing package cycles.
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/access_token_test.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/cpa_test.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/sub2api_json_test.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore_adapter.go`
  - `account` package adapters that satisfy `importcore` interfaces without creating an import cycle.
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/handler_import_test.go`
  - Handler compatibility tests for `/api/admin/accounts`, `/import`, and `/import-tokens`.

### Modified backend files

- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/handler.go`
  - Replace inline import logic with adapter + importcore orchestration.
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importer.go`
  - Downgrade to compatibility shims or remove business logic now covered by `importsource`.
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importer_tokens.go`
  - Keep RT/ST exchange helpers, add candidate builders, stop performing direct batch upsert.
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/cmd/server/main.go`
  - Instantiate import core and inject adapters/refresher/prober/pool service into handlers.

### New frontend files

- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/types.ts`
  - Shared pane props, advanced options, result row types.
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/AccountImportDialog.vue`
  - Main import dialog shell with tab switching and result view.
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/AccessTokenImportPane.vue`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/CPAImportPane.vue`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/Sub2APIImportPane.vue`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/ManualAccountPane.vue`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/ImportAdvancedOptions.vue`

### Modified frontend files

- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/api/accounts.ts`
  - Extend payload types with `target_pool_id`, `kick_refresh`, `kick_quota_probe`, `resolve_identity`.
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/views/admin/Accounts.vue`
  - Replace inline import UI with the shared dialog and keep list/refresh/probe behavior intact.

---

### Task 1: Build import-core types and credential lifecycle policy

**Files:**
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/types.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/identity.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/service_test.go`

- [ ] **Step 1: Write failing tests for lifecycle classification and expired AT-only filtering**

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `rtk go test ./internal/account/importcore -run 'TestClassifyCredentialStateSkipsExpiredATOnly|TestClassifyCredentialStateKeepsExpiredRefreshableCandidate' -v`

Expected: FAIL with errors such as `cannot find package`, `undefined: ImportCandidate`, or `undefined: ClassifyCredentialState`.

- [ ] **Step 3: Add import types and lifecycle helpers**

```go
package importcore

import "time"

type ImportCandidate struct {
	SourceType       string
	SourceRef        string
	AccessToken      string
	RefreshToken     string
	SessionToken     string
	Email            string
	ClientID         string
	ChatGPTAccountID string
	AccountType      string
	PlanType         string
	TokenExpiresAt   *time.Time
	OAISessionID     string
	OAIDeviceID      string
	Cookies          string
	Notes            string
}

type ImportOptions struct {
	UpdateExisting    bool
	DefaultProxyID    uint64
	TargetPoolID      uint64
	SkipExpiredATOnly bool
	ResolveIdentity   bool
	KickRefresh       bool
	KickQuotaProbe    bool
}

type CredentialState struct {
	Capability string
	Lifecycle  string
	SkipImport bool
	Warning    string
}

func ClassifyCredentialState(candidate ImportCandidate, now time.Time, refreshAheadSec int) CredentialState {
	hasRT := candidate.RefreshToken != ""
	hasST := candidate.SessionToken != ""
	refreshable := hasRT || hasST

	state := CredentialState{Capability: "at_only", Lifecycle: "unknown"}
	switch {
	case hasRT && hasST:
		state.Capability = "refreshable_full"
	case hasRT:
		state.Capability = "refreshable_rt"
	case hasST:
		state.Capability = "refreshable_st"
	}

	if candidate.TokenExpiresAt == nil {
		if refreshable {
			state.Warning = "token_expiry_unknown"
		}
		return state
	}

	exp := candidate.TokenExpiresAt.UTC()
	if !exp.After(now.UTC()) {
		state.Lifecycle = "expired"
		if !refreshable {
			state.SkipImport = true
			return state
		}
		state.Warning = "access_token_expired_but_refreshable"
		return state
	}

	if exp.Before(now.UTC().Add(time.Duration(refreshAheadSec) * time.Second)) {
		state.Lifecycle = "expiring_soon"
		if refreshable {
			state.Warning = "access_token_expiring_soon"
		} else {
			state.Warning = "access_token_expiring_soon_unrefreshable"
		}
		return state
	}

	state.Lifecycle = "active"
	return state
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `rtk go test ./internal/account/importcore -run 'TestClassifyCredentialStateSkipsExpiredATOnly|TestClassifyCredentialStateKeepsExpiredRefreshableCandidate' -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/account/importcore/types.go internal/account/importcore/identity.go internal/account/importcore/service_test.go
git commit -m "feat: add import core lifecycle policy"
```

### Task 2: Implement source adapters for access-token text, CPA JSON, sub2api JSON, and manual input

**Files:**
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/access_token.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/cpa.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/sub2api_json.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/manual.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/access_token_test.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/cpa_test.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/sub2api_json_test.go`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importer.go`

- [ ] **Step 1: Write failing parser tests for CRLF token text, CPA token extraction, and sub2api local JSON**

```go
package importsource

import "testing"

func TestParseAccessTokenTextHandlesCRLF(t *testing.T) {
	candidates := ParseAccessTokenText("tok-a\r\ntok-b\r\n\r\n")
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].SourceRef != "line:1" || candidates[1].SourceRef != "line:2" {
		t.Fatalf("unexpected source refs: %+v", candidates)
	}
}

func TestParseCPAFileExtractsAccessToken(t *testing.T) {
	candidates, err := ParseCPAFile("sample.json", []byte(`{"access_token":"tok-a","email":"a@example.com"}`))
	if err != nil {
		t.Fatalf("ParseCPAFile returned error: %v", err)
	}
	if len(candidates) != 1 || candidates[0].AccessToken != "tok-a" {
		t.Fatalf("unexpected CPA candidates: %+v", candidates)
	}
}

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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `rtk go test ./internal/account/importsource -run 'TestParseAccessTokenTextHandlesCRLF|TestParseCPAFileExtractsAccessToken|TestParseSub2APIJSONExtractsCredentials' -v`

Expected: FAIL with `undefined: ParseAccessTokenText`, `undefined: ParseCPAFile`, or missing package errors.

- [ ] **Step 3: Implement the adapters and downgrade `internal/account/importer.go` to compatibility wrappers**

```go
package importsource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/432539/gpt2api/internal/account/importcore"
)

func ParseAccessTokenText(raw string) []importcore.ImportCandidate {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	out := make([]importcore.ImportCandidate, 0, len(lines))
	for i, line := range lines {
		token := strings.TrimSpace(line)
		if token == "" {
			continue
		}
		out = append(out, importcore.ImportCandidate{
			SourceType:  "access_token_text",
			SourceRef:   fmt.Sprintf("line:%d", i+1),
			AccessToken: token,
		})
	}
	return out
}

func ParseCPAFile(name string, raw []byte) ([]importcore.ImportCandidate, error) {
	var obj map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(raw), &obj); err != nil {
		return nil, err
	}
	token, _ := obj["access_token"].(string)
	if token == "" {
		token, _ = obj["accessToken"].(string)
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("%s missing access token", name)
	}
	return []importcore.ImportCandidate{{
		SourceType:  "cpa_file",
		SourceRef:   "file:" + name,
		AccessToken: strings.TrimSpace(token),
	}}, nil
}
```

```go
package account

import (
	"github.com/432539/gpt2api/internal/account/importcore"
	"github.com/432539/gpt2api/internal/account/importsource"
)

func parseCandidatesFromJSON(raw []byte) ([]importcore.ImportCandidate, error) {
	return importsource.ParseAutoJSON(raw)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `rtk go test ./internal/account/importsource -run 'TestParseAccessTokenTextHandlesCRLF|TestParseCPAFileExtractsAccessToken|TestParseSub2APIJSONExtractsCredentials' -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/account/importsource/access_token.go internal/account/importsource/cpa.go internal/account/importsource/sub2api_json.go internal/account/importsource/manual.go internal/account/importsource/access_token_test.go internal/account/importsource/cpa_test.go internal/account/importsource/sub2api_json_test.go internal/account/importer.go
git commit -m "feat: add unified account import source adapters"
```

### Task 3: Implement the import-core persistence and post-processing pipeline

**Files:**
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/persist.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/postprocess.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/service.go`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/service_test.go`

- [ ] **Step 1: Write failing orchestration tests for create/update/skip, expired-token handling, and pool membership post-processing**

```go
func TestImportSkipsExpiredATOnlyCandidate(t *testing.T) {
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	exp := now.Add(-time.Minute)

	core := NewService(fakeDeps{Now: func() time.Time { return now }, RefreshAheadSec: func() int { return 900 }})
	result, err := core.Import(context.Background(), []ImportCandidate{{
		SourceType:     "access_token_text",
		SourceRef:      "line:1",
		AccessToken:    "expired-at",
		Email:          "user@example.com",
		TokenExpiresAt: &exp,
	}}, DefaultOptions())
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if result.Skipped != 1 || result.Results[0].Reason == "" {
		t.Fatalf("expected skipped expired candidate, got %+v", result)
	}
}

func TestImportCreatesAccountAndAddsPoolMember(t *testing.T) {
	store := &fakeStore{}
	core := NewService(fakeDeps{
		Store:          store,
		PoolMembership: &fakePoolMembership{},
		Now:            time.Now,
		RefreshAheadSec: func() int { return 900 },
	})
	result, err := core.Import(context.Background(), []ImportCandidate{{
		SourceType:  "manual",
		SourceRef:   "admin_form",
		AccessToken: "at",
		Email:       "user@example.com",
	}}, ImportOptions{UpdateExisting: true, TargetPoolID: 7, KickRefresh: true, KickQuotaProbe: true, SkipExpiredATOnly: true})
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if result.Created != 1 {
		t.Fatalf("expected one created account, got %+v", result)
	}
	if store.lastCreated.Email != "user@example.com" {
		t.Fatalf("unexpected created record: %+v", store.lastCreated)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `rtk go test ./internal/account/importcore -run 'TestImportSkipsExpiredATOnlyCandidate|TestImportCreatesAccountAndAddsPoolMember' -v`

Expected: FAIL with `undefined: NewService`, missing fake interfaces, or missing persistence hooks.

- [ ] **Step 3: Implement the import service with narrow interfaces to avoid package cycles**

```go
package importcore

import "context"

type AccountRecord struct {
	ID               uint64
	Email            string
	AuthToken        string
	RefreshToken     string
	SessionToken     string
	ClientID         string
	ChatGPTAccountID string
	AccountType      string
	PlanType         string
	TokenExpiresAt   *time.Time
	OAISessionID     string
	OAIDeviceID      string
	Cookies          string
	Notes            string
	Status           string
}

type Store interface {
	FindByEmail(ctx context.Context, email string) (*AccountRecord, error)
	Create(ctx context.Context, candidate ImportCandidate) (uint64, error)
	Update(ctx context.Context, id uint64, candidate ImportCandidate, existing *AccountRecord) error
	BindDefaultProxy(ctx context.Context, accountID, proxyID uint64) error
}

type PoolMembership interface {
	AddDefaultMember(ctx context.Context, poolID, accountID uint64) error
}

type Hooks interface {
	KickRefresh()
	KickQuotaProbe()
}

type Service struct {
	store          Store
	pools          PoolMembership
	hooks          Hooks
	now            func() time.Time
	refreshAheadSec func() int
}

func (s *Service) Import(ctx context.Context, candidates []ImportCandidate, opt ImportOptions) (*ImportResult, error) {
	result := &ImportResult{Results: make([]ImportLineResult, 0, len(candidates))}
	normalized := NormalizeCandidates(candidates)
	resolved := s.resolveIdentityAndClassify(ctx, normalized, opt)
	deduped := DeduplicateByEmail(resolved)
	for _, item := range deduped {
		line := s.persistOne(ctx, item, opt)
		result.Results = append(result.Results, line)
		switch line.Status {
		case "created":
			result.Created++
		case "updated":
			result.Updated++
		case "skipped":
			result.Skipped++
		case "failed":
			result.Failed++
		}
	}
	result.Total = len(result.Results)
	s.runPostprocess(ctx, result.Results, opt)
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `rtk go test ./internal/account/importcore -run 'TestClassifyCredentialState.*|TestImportSkipsExpiredATOnlyCandidate|TestImportCreatesAccountAndAddsPoolMember' -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/account/importcore/persist.go internal/account/importcore/postprocess.go internal/account/importcore/service.go internal/account/importcore/service_test.go
git commit -m "feat: add unified account import core service"
```

### Task 4: Wire the existing admin handlers to the unified import core while preserving route compatibility

**Files:**
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore_adapter.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/handler_import_test.go`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/handler.go`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importer_tokens.go`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/cmd/server/main.go`

- [ ] **Step 1: Write failing handler compatibility tests for `/api/admin/accounts`, `/import`, and `/import-tokens`**

```go
package account

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestImportTokensEndpointPreservesSummaryShape(t *testing.T) {
	h := NewHandler(nil)
	h.importCore = &fakeImportCore{
		result: &ImportResult{
			Total: 1, Created: 1,
			Results: []ImportLineResult{{Index: 0, Email: "user@example.com", Status: "created", ID: 9}},
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `rtk go test ./internal/account -run 'TestImportTokensEndpointPreservesSummaryShape' -v`

Expected: FAIL with missing fake import core wiring or response mismatch.

- [ ] **Step 3: Implement adapters and route wiring**

```go
package account

import (
	"context"

	"github.com/432539/gpt2api/internal/account/importcore"
)

type importCoreAdapter struct {
	svc *Service
}

func (a *importCoreAdapter) FindByEmail(ctx context.Context, email string) (*importcore.AccountRecord, error) {
	existing, err := a.svc.dao.GetByEmail(ctx, email)
	if err != nil || existing == nil {
		return nil, err
	}
	return &importcore.AccountRecord{
		ID:               existing.ID,
		Email:            existing.Email,
		ClientID:         existing.ClientID,
		ChatGPTAccountID: existing.ChatGPTAccountID,
		AccountType:      existing.AccountType,
		PlanType:         existing.PlanType,
		Status:           existing.Status,
	}, nil
}
```

```go
func (h *Handler) ImportTokens(c *gin.Context) {
	// decode request -> build candidates using importsource / importer_tokens helper -> call importcore -> resp.OK(summary)
}

func (h *Handler) Create(c *gin.Context) {
	// map manual form to importsource.ManualInput -> importcore.Import -> return created/updated account
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `rtk go test ./internal/account -run 'TestImportTokensEndpointPreservesSummaryShape|TestCreateEndpointUsesUnifiedImportCore' -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/account/importcore_adapter.go internal/account/handler_import_test.go internal/account/handler.go internal/account/importer_tokens.go cmd/server/main.go
git commit -m "feat: route account imports through unified core"
```

### Task 5: Add shared frontend import types, API payloads, and dialog panes

**Files:**
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/types.ts`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/AccountImportDialog.vue`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/AccessTokenImportPane.vue`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/CPAImportPane.vue`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/Sub2APIImportPane.vue`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/ManualAccountPane.vue`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/ImportAdvancedOptions.vue`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/api/accounts.ts`

- [ ] **Step 1: Introduce a compile-failing frontend regression that references the shared import dialog**

```vue
<!-- /Users/swimmingliu/data/github-proj/gpt2api/web/src/views/admin/Accounts.vue -->
<script setup lang="ts">
import AccountImportDialog from '@/components/admin/account-import/AccountImportDialog.vue'
</script>

<template>
  <AccountImportDialog />
</template>
```

- [ ] **Step 2: Run the frontend build to verify it fails**

Run: `rtk sh -lc 'cd /Users/swimmingliu/data/github-proj/gpt2api/web && npm run build'`

Expected: FAIL with errors such as `Cannot find module '@/components/admin/account-import/AccountImportDialog.vue'`.

- [ ] **Step 3: Implement shared types, advanced options, and pane components**

```ts
// /Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/types.ts
export interface ImportAdvancedOptions {
  update_existing: boolean
  default_proxy_id?: number
  target_pool_id?: number
  resolve_identity: boolean
  kick_refresh: boolean
  kick_quota_probe: boolean
}

export interface ImportDialogResultRow {
  index: number
  source_type?: string
  source_ref?: string
  email: string
  status: 'created' | 'updated' | 'skipped' | 'failed'
  reason?: string
  warnings?: string[]
  id?: number
}
```

```ts
// /Users/swimmingliu/data/github-proj/gpt2api/web/src/api/accounts.ts
export interface ImportTokensBody {
  mode: 'at' | 'rt' | 'st'
  tokens: string | string[]
  client_id?: string
  update_existing?: boolean
  default_proxy_id?: number
  target_pool_id?: number
  resolve_identity?: boolean
  kick_refresh?: boolean
  kick_quota_probe?: boolean
}
```

```vue
<!-- /Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/ImportAdvancedOptions.vue -->
<template>
  <el-form label-width="120px">
    <el-form-item label="更新已有邮箱">
      <el-switch v-model="model.update_existing" />
    </el-form-item>
    <el-form-item label="默认代理">
      <el-select v-model="model.default_proxy_id" clearable />
    </el-form-item>
    <el-form-item label="导入到账号池">
      <el-select v-model="model.target_pool_id" clearable />
    </el-form-item>
  </el-form>
</template>
```

- [ ] **Step 4: Run the frontend build to verify it passes**

Run: `rtk sh -lc 'cd /Users/swimmingliu/data/github-proj/gpt2api/web && npm run build'`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/admin/account-import/types.ts web/src/components/admin/account-import/AccountImportDialog.vue web/src/components/admin/account-import/AccessTokenImportPane.vue web/src/components/admin/account-import/CPAImportPane.vue web/src/components/admin/account-import/Sub2APIImportPane.vue web/src/components/admin/account-import/ManualAccountPane.vue web/src/components/admin/account-import/ImportAdvancedOptions.vue web/src/api/accounts.ts
git commit -m "feat: add shared account import dialog components"
```

### Task 6: Integrate the unified import dialog into the accounts page and run repository verification

**Files:**
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/views/admin/Accounts.vue`

- [ ] **Step 1: Replace the inline import UI in `Accounts.vue` with the shared dialog and keep existing refresh/probe/list behavior**

```vue
<script setup lang="ts">
import AccountImportDialog from '@/components/admin/account-import/AccountImportDialog.vue'

function handleImportDone() {
  importDlg.value = false
  fetchList()
}
</script>

<template>
  <div class="actions">
    <AccountImportDialog
      :proxies="proxies"
      :on-done="handleImportDone"
    />
    <el-button type="primary" @click="openCreate">新建账号</el-button>
  </div>
</template>
```

- [ ] **Step 2: Run the focused account tests**

Run: `rtk go test ./internal/account ./internal/account/importcore ./internal/account/importsource -v`

Expected: PASS

- [ ] **Step 3: Run repository-wide backend verification**

Run: `rtk go vet ./...`

Expected: PASS

Run: `rtk go test ./...`

Expected: PASS

- [ ] **Step 4: Run frontend verification**

Run: `rtk sh -lc 'cd /Users/swimmingliu/data/github-proj/gpt2api/web && npm run build'`

Expected: PASS

- [ ] **Step 5: Run docker smoke verification**

Run: `rtk sh -lc 'cd /Users/swimmingliu/data/github-proj/gpt2api && docker compose -f deploy/docker-compose.yml up -d --wait'`

Expected: containers become healthy.

Run: `rtk sh -lc 'cd /Users/swimmingliu/data/github-proj/gpt2api/scripts && npm run smoke:docker'`

Expected: PASS

- [ ] **Step 6: Capture UI evidence for the new import dialog**

Artifact: save an after screenshot to `/Users/swimmingliu/data/github-proj/gpt2api/docs/screenshots/account-import-unified.png` showing the accounts page with the unified import dialog open and advanced options visible.

- [ ] **Step 7: Commit**

```bash
git add web/src/views/admin/Accounts.vue docs/screenshots/account-import-unified.png
git commit -m "feat: unify account import workflow in admin console"
```
