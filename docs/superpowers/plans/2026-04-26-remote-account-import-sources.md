# Remote Account Import Sources Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add remote sub2api and CPA source management plus remote browse/import flows that feed the existing unified account import core.

**Architecture:** Introduce a new resource-style backend module for persisted remote source configs, separate from `settings` and from local parsing under `internal/account/importsource`. Keep imports synchronous: browse remote accounts/files, select rows, then convert them to `importcore.ImportCandidate` and return the existing `ImportSummary` response shape. Extend the existing Accounts import dialog with remote source selection and source-specific browsing instead of adding a new admin page first.

**Tech Stack:** Go, Gin, sqlx, MySQL, Vue 3, TypeScript, Element Plus

---

## File Structure Map

### New backend files

- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/model.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/dao.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/service.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/http_client.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/handler.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/service_test.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/handler_test.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/sql/migrations/20260426000001_account_import_sources.sql`

### Modified backend files

- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/server/router.go`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/cmd/server/main.go`

### New frontend files

- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/api/account-import-sources.ts`

### Modified frontend files

- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/types.ts`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/CPAImportPane.vue`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/Sub2APIImportPane.vue`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/AccountImportDialog.vue`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/views/admin/Accounts.vue`

## Milestones

### Task 1: Backend remote source resource and remote browse/import

**Files:**
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/model.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/dao.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/service.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/http_client.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/handler.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/service_test.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/handler_test.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/sql/migrations/20260426000001_account_import_sources.sql`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/server/router.go`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/cmd/server/main.go`

- [ ] Write failing service and handler tests for create/list masking, sub2api remote list parsing, CPA remote list parsing, and import option fallback.
- [ ] Run the focused accountsource tests and verify they fail for missing package/symbols.
- [ ] Implement the migration and accountsource DAO/model layer.
- [ ] Implement service logic, encrypted secret handling, remote HTTP calls, and synchronous import delegation to `importcore`.
- [ ] Implement handler endpoints and wire them into router/main.
- [ ] Run the focused accountsource tests until they pass.

### Task 2: Frontend remote source API and import dialog integration

**Files:**
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/api/account-import-sources.ts`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/types.ts`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/CPAImportPane.vue`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/Sub2APIImportPane.vue`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/AccountImportDialog.vue`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/views/admin/Accounts.vue`

- [ ] Add TS API types for source CRUD, sub2api accounts/groups, CPA files, and remote import submission.
- [ ] Update the CPA and sub2api panes to support local-vs-remote mode, source selection, remote browsing, and row selection without breaking existing local import behavior.
- [ ] Update `AccountImportDialog.vue` and `Accounts.vue` so remote imports submit through the new backend endpoint while still reusing advanced options and the existing summary modal.
- [ ] Build the web app and fix any type or template regressions.

### Task 3: Verification

**Files:**
- Modify only as needed based on fixes from verification.

- [ ] Run `rtk go vet ./...`.
- [ ] Run `rtk go test ./...`.
- [ ] Run `rtk sh -lc 'cd /Users/swimmingliu/data/github-proj/gpt2api/web && npm run build'`.
- [ ] Do local runtime verification against the existing local server instead of Docker where possible:
  - health check on `http://127.0.0.1:18080/healthz`
  - if an authenticated browser session is available, verify the remote import UI path in-browser
  - otherwise record the exact auth blocker instead of claiming success
