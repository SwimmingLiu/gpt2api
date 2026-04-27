# Phase 1 Account Pool Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add first-class account pools, model-to-pool routing, and pool-aware dispatch while preserving the current global account scheduling behavior when no route is configured.

**Architecture:** Introduce a new `internal/accountpool` module for pools, pool memberships, and model routes. Keep `internal/account` as the source of account truth, extend scheduler and gateway to accept an optional pool constraint, and add admin UI/API for pool management and model route management.

**Tech Stack:** Go, Gin, sqlx, Goose, MySQL, Redis, Vue 3, TypeScript, Element Plus

---

### Task 1: Add schema and core account-pool domain types

**Files:**
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/sql/migrations/20260425000001_account_pools.sql`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountpool/model.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountpool/service.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountpool/service_test.go`

- [ ] **Step 1: Write the failing tests for pool code validation and legacy fallback**

```go
package accountpool

import (
	"context"
	"testing"
)

type fakeStore struct {
	route *ModelRoute
}

func (f *fakeStore) GetRouteByModelID(_ context.Context, modelID uint64) (*ModelRoute, error) {
	return f.route, nil
}

func TestValidatePoolCode(t *testing.T) {
	if err := validatePoolCode("image-main"); err != nil {
		t.Fatalf("expected valid code, got %v", err)
	}
	if err := validatePoolCode("bad code"); err == nil {
		t.Fatal("expected invalid code error")
	}
}

func TestResolveModelRouteFallsBackToGlobalWhenUnset(t *testing.T) {
	svc := NewService(&fakeStore{})
	got, err := svc.ResolveModelRoute(context.Background(), 42)
	if err != nil {
		t.Fatalf("ResolveModelRoute returned error: %v", err)
	}
	if got.PoolID != 0 || got.FallbackPoolID != 0 || !got.LegacyGlobal {
		t.Fatalf("unexpected fallback route: %+v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `rtk go test ./internal/accountpool -run 'TestValidatePoolCode|TestResolveModelRouteFallsBackToGlobalWhenUnset' -v`

Expected: FAIL with errors such as `undefined: ModelRoute`, `undefined: NewService`, or package missing.

- [ ] **Step 3: Add the migration and domain models**

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE `account_pools` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(64) NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `pool_type` VARCHAR(32) NOT NULL DEFAULT 'mixed',
  `description` VARCHAR(255) NOT NULL DEFAULT '',
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `dispatch_strategy` VARCHAR(32) NOT NULL DEFAULT 'least_recently_used',
  `sticky_ttl_sec` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_account_pools_code` (`code`)
);

CREATE TABLE `account_pool_members` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `pool_id` BIGINT UNSIGNED NOT NULL,
  `account_id` BIGINT UNSIGNED NOT NULL,
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `weight` INT NOT NULL DEFAULT 100,
  `priority` INT NOT NULL DEFAULT 100,
  `max_parallel` INT NOT NULL DEFAULT 1,
  `note` VARCHAR(255) NOT NULL DEFAULT '',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_account_pool_member` (`pool_id`, `account_id`),
  KEY `idx_account_pool_members_account_id` (`account_id`)
);

CREATE TABLE `model_pool_routes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `model_id` BIGINT UNSIGNED NOT NULL,
  `pool_id` BIGINT UNSIGNED NOT NULL,
  `fallback_pool_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_model_pool_routes_model_id` (`model_id`)
);
-- +goose StatementEnd
```

```go
package accountpool

type Pool struct {
	ID               uint64 `db:"id" json:"id"`
	Code             string `db:"code" json:"code"`
	Name             string `db:"name" json:"name"`
	PoolType         string `db:"pool_type" json:"pool_type"`
	Description      string `db:"description" json:"description"`
	Enabled          bool   `db:"enabled" json:"enabled"`
	DispatchStrategy string `db:"dispatch_strategy" json:"dispatch_strategy"`
	StickyTTLSec     int    `db:"sticky_ttl_sec" json:"sticky_ttl_sec"`
}

type Member struct {
	ID          uint64 `db:"id" json:"id"`
	PoolID      uint64 `db:"pool_id" json:"pool_id"`
	AccountID   uint64 `db:"account_id" json:"account_id"`
	Enabled     bool   `db:"enabled" json:"enabled"`
	Weight      int    `db:"weight" json:"weight"`
	Priority    int    `db:"priority" json:"priority"`
	MaxParallel int    `db:"max_parallel" json:"max_parallel"`
	Note        string `db:"note" json:"note"`
}

type ModelRoute struct {
	ID             uint64 `db:"id" json:"id"`
	ModelID        uint64 `db:"model_id" json:"model_id"`
	PoolID         uint64 `db:"pool_id" json:"pool_id"`
	FallbackPoolID uint64 `db:"fallback_pool_id" json:"fallback_pool_id"`
	Enabled        bool   `db:"enabled" json:"enabled"`
}

type ResolvedRoute struct {
	PoolID         uint64
	FallbackPoolID uint64
	LegacyGlobal   bool
}
```

- [ ] **Step 4: Implement the minimal service to pass the tests**

```go
package accountpool

import (
	"context"
	"errors"
	"regexp"
)

var poolCodeRe = regexp.MustCompile(`^[a-z][a-z0-9\\-]{1,63}$`)

type routeReader interface {
	GetRouteByModelID(ctx context.Context, modelID uint64) (*ModelRoute, error)
}

type Service struct {
	store routeReader
}

func NewService(store routeReader) *Service { return &Service{store: store} }

func validatePoolCode(code string) error {
	if !poolCodeRe.MatchString(code) {
		return errors.New("invalid pool code")
	}
	return nil
}

func (s *Service) ResolveModelRoute(ctx context.Context, modelID uint64) (ResolvedRoute, error) {
	route, err := s.store.GetRouteByModelID(ctx, modelID)
	if err != nil || route == nil || !route.Enabled {
		return ResolvedRoute{LegacyGlobal: true}, err
	}
	return ResolvedRoute{
		PoolID:         route.PoolID,
		FallbackPoolID: route.FallbackPoolID,
		LegacyGlobal:   false,
	}, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `rtk go test ./internal/accountpool -run 'TestValidatePoolCode|TestResolveModelRouteFallsBackToGlobalWhenUnset' -v`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add sql/migrations/20260425000001_account_pools.sql internal/accountpool/model.go internal/accountpool/service.go internal/accountpool/service_test.go
git commit -m "feat: add account pool schema and core route service"
```

### Task 2: Implement account-pool DAO, service, handlers, and server wiring

**Files:**
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountpool/dao.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountpool/handler.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountpool/handler_test.go`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/server/router.go`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/cmd/server/main.go`

- [ ] **Step 1: Write failing handler tests for pool CRUD and model route upsert**

```go
func TestPutRouteCreatesOrUpdatesModelRoute(t *testing.T) {
	svc := &fakeService{
		putRouteFn: func(_ context.Context, modelID uint64, in PutRouteInput) (*ModelRoute, error) {
			return &ModelRoute{ModelID: modelID, PoolID: in.PoolID, FallbackPoolID: in.FallbackPoolID, Enabled: true}, nil
		},
	}
	h := NewHandler(svc)
	r := gin.New()
	r.PUT("/routes/:modelId", h.PutRoute)

	req := httptest.NewRequest(http.MethodPut, "/routes/9", strings.NewReader(`{"pool_id":1,"fallback_pool_id":2}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `rtk go test ./internal/accountpool -run 'TestPutRouteCreatesOrUpdatesModelRoute' -v`

Expected: FAIL with `undefined: NewHandler` or fake service method mismatch.

- [ ] **Step 3: Implement DAO methods and service inputs**

```go
type DAO struct{ db *sqlx.DB }

func (d *DAO) ListPools(ctx context.Context) ([]*Pool, error) { /* SELECT * FROM account_pools */ }
func (d *DAO) CreatePool(ctx context.Context, p *Pool) error { /* INSERT */ }
func (d *DAO) UpdatePool(ctx context.Context, p *Pool) error { /* UPDATE */ }
func (d *DAO) SoftDeletePool(ctx context.Context, id uint64) error { /* soft delete */ }
func (d *DAO) ListMembers(ctx context.Context, poolID uint64) ([]*Member, error) { /* SELECT */ }
func (d *DAO) UpsertMember(ctx context.Context, in *Member) error { /* INSERT ... ON DUPLICATE KEY UPDATE */ }
func (d *DAO) DeleteMember(ctx context.Context, poolID, memberID uint64) error { /* DELETE */ }
func (d *DAO) GetRouteByModelID(ctx context.Context, modelID uint64) (*ModelRoute, error) { /* SELECT */ }
func (d *DAO) UpsertRoute(ctx context.Context, in *ModelRoute) error { /* INSERT ... ON DUPLICATE KEY UPDATE */ }
func (d *DAO) DeleteRoute(ctx context.Context, modelID uint64) error { /* DELETE */ }
```

- [ ] **Step 4: Implement handlers and route registration**

```go
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) PutRoute(c *gin.Context) {
	modelID, _ := strconv.ParseUint(c.Param("modelId"), 10, 64)
	var req PutRouteInput
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error())
		return
	}
	route, err := h.svc.PutRoute(c.Request.Context(), modelID, req)
	if err != nil {
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, route)
}
```

```go
// cmd/server/main.go
poolDAO := accountpool.NewDAO(sqldb)
poolSvc := accountpool.NewService(poolDAO)
poolH := accountpool.NewHandler(poolSvc)

// internal/server/router.go
pg := admin.Group("/account-pools", middleware.RequirePerm(rbac.PermAccountRead, rbac.PermAccountWrite))
{
	pg.GET("", d.AccountPoolH.ListPools)
	pg.POST("", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountPoolH.CreatePool)
	pg.GET("/:id", d.AccountPoolH.GetPool)
	pg.PATCH("/:id", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountPoolH.UpdatePool)
	pg.DELETE("/:id", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountPoolH.DeletePool)
	pg.GET("/:id/members", d.AccountPoolH.ListMembers)
	pg.POST("/:id/members", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountPoolH.UpsertMember)
	pg.PATCH("/:id/members/:memberId", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountPoolH.UpdateMember)
	pg.DELETE("/:id/members/:memberId", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountPoolH.DeleteMember)
}
admin.GET("/account-pool-routes", middleware.RequirePerm(rbac.PermModelRead), d.AccountPoolH.ListRoutes)
admin.PUT("/account-pool-routes/:modelId", middleware.RequirePerm(rbac.PermModelWrite), d.AccountPoolH.PutRoute)
admin.DELETE("/account-pool-routes/:modelId", middleware.RequirePerm(rbac.PermModelWrite), d.AccountPoolH.DeleteRoute)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `rtk go test ./internal/accountpool -run 'TestPutRouteCreatesOrUpdatesModelRoute' -v`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/accountpool/dao.go internal/accountpool/handler.go internal/accountpool/handler_test.go internal/server/router.go cmd/server/main.go
git commit -m "feat: add account pool admin api and server wiring"
```

### Task 3: Make scheduler and gateway pool-aware

**Files:**
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/scheduler/scheduler.go`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/internal/scheduler/scheduler_pool_test.go`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/gateway/chat.go`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/gateway/images.go`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/dao.go`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/service.go`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/handler.go`

- [ ] **Step 1: Write failing scheduler test for pool-constrained dispatch**

```go
func TestFilterCandidatesByPoolOnlyKeepsPoolMembers(t *testing.T) {
	candidates := []*account.Account{{ID: 1}, {ID: 2}, {ID: 3}}
	allowed := map[uint64]struct{}{2: {}, 3: {}}

	got := filterByAllowedAccounts(candidates, allowed)

	if len(got) != 2 || got[0].ID != 2 || got[1].ID != 3 {
		t.Fatalf("unexpected filtered result: %+v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `rtk go test ./internal/scheduler -run 'TestFilterCandidatesByPoolOnlyKeepsPoolMembers' -v`

Expected: FAIL with `undefined: filterByAllowedAccounts`.

- [ ] **Step 3: Implement pool-aware account lookup and scheduler dispatch options**

```go
// internal/account/dao.go
func (d *DAO) ListDispatchableByPool(ctx context.Context, poolID uint64, limit int) ([]*Account, error) {
	rows := make([]*Account, 0, limit)
	err := d.db.SelectContext(ctx, &rows, `
SELECT a.*
  FROM oai_accounts a
  JOIN account_pool_members m ON m.account_id = a.id
  JOIN account_pools p ON p.id = m.pool_id
 WHERE a.deleted_at IS NULL
   AND a.status = 'healthy'
   AND (a.cooldown_until IS NULL OR a.cooldown_until <= NOW())
   AND (a.token_expires_at IS NULL OR a.token_expires_at > NOW())
   AND p.deleted_at IS NULL
   AND p.enabled = 1
   AND m.enabled = 1
   AND m.pool_id = ?
 ORDER BY CASE WHEN a.last_used_at IS NULL THEN 0 ELSE 1 END, a.last_used_at ASC
 LIMIT ?`, poolID, limit)
	fillAll(rows)
	return rows, err
}
```

```go
// internal/scheduler/scheduler.go
type DispatchOptions struct {
	PoolID uint64
}

func (s *Scheduler) Dispatch(ctx context.Context, modelType string, opt DispatchOptions) (*Lease, error) {
	// old logic preserved; when opt.PoolID > 0 use ListDispatchableByPool
}
```

- [ ] **Step 4: Resolve model route inside gateway and fall back cleanly**

```go
route, err := h.PoolSvc.ResolveModelRoute(c.Request.Context(), m.ID)
if err != nil {
	fail("pool_route_error")
	openAIError(c, http.StatusInternalServerError, "internal_error", "账号池路由解析失败:"+err.Error())
	return
}

lease, err := h.Scheduler.Dispatch(c.Request.Context(), modelpkg.TypeImage, scheduler.DispatchOptions{
	PoolID: route.PoolID,
})
if errors.Is(err, scheduler.ErrNoAvailable) && route.FallbackPoolID > 0 {
	lease, err = h.Scheduler.Dispatch(c.Request.Context(), modelpkg.TypeImage, scheduler.DispatchOptions{
		PoolID: route.FallbackPoolID,
	})
}
```

- [ ] **Step 5: Extend account listing/import to understand pools**

```go
// internal/account/service.go
type ImportOptions struct {
	UpdateExisting  bool
	DefaultClientID string
	DefaultProxyID  uint64
	DefaultPoolID   uint64
	BatchSize       int
}
```

```go
// internal/account/handler.go
var req struct {
	Text            string `json:"text"`
	UpdateExisting  *bool  `json:"update_existing"`
	DefaultClientID string `json:"default_client_id"`
	DefaultProxyID  uint64 `json:"default_proxy_id"`
	DefaultPoolID   uint64 `json:"default_pool_id"`
}
```

- [ ] **Step 6: Run focused tests to verify pool-aware behavior**

Run: `rtk go test ./internal/accountpool ./internal/scheduler ./internal/account -run 'Test.*Pool|Test.*Route' -v`

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/scheduler/scheduler.go internal/scheduler/scheduler_pool_test.go internal/gateway/chat.go internal/gateway/images.go internal/account/dao.go internal/account/service.go internal/account/handler.go
git commit -m "feat: route models through account pools during dispatch"
```

### Task 4: Add admin UI for pool management and model-pool routing

**Files:**
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/api/accountPools.ts`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/views/admin/AccountPools.vue`
- Create: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/views/admin/ModelPoolRoutes.vue`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/views/admin/Accounts.vue`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/api/accounts.ts`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/web/src/router/index.ts`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/internal/rbac/menu.go`

- [ ] **Step 1: Add the API client types first**

```ts
export interface AccountPool {
  id: number
  code: string
  name: string
  pool_type: string
  description: string
  enabled: boolean
  dispatch_strategy: string
  sticky_ttl_sec: number
}

export interface ModelPoolRoute {
  model_id: number
  pool_id: number
  fallback_pool_id: number
  enabled: boolean
}

export function listAccountPools() {
  return http.get<any, { items: AccountPool[]; total: number }>('/api/admin/account-pools')
}

export function putModelPoolRoute(modelId: number, body: { pool_id: number; fallback_pool_id?: number; enabled?: boolean }) {
  return http.put<any, ModelPoolRoute>(`/api/admin/account-pool-routes/${modelId}`, body)
}
```

- [ ] **Step 2: Run the frontend build to verify imports are currently missing**

Run: `cd /Users/swimmingliu/data/github-proj/gpt2api/web && rtk npm run build`

Expected: FAIL after the next route/component imports are added but before files exist, with messages such as `Failed to resolve import`.

- [ ] **Step 3: Implement the admin pages and navigation**

```ts
// web/src/router/index.ts
{ path: 'account-pools', component: () => import('@/views/admin/AccountPools.vue'),
  meta: { title: '账号池管理', perm: 'account:read' } },
{ path: 'model-pool-routes', component: () => import('@/views/admin/ModelPoolRoutes.vue'),
  meta: { title: '模型池路由', perm: ['model:read', 'model:write'] } },
```

```go
// internal/rbac/menu.go
{Key: "admin.account-pools", Title: "账号池管理", Icon: "CollectionTag", Path: "/admin/account-pools",
	Perms: []Permission{PermAccountRead}},
{Key: "admin.model-pool-routes", Title: "模型池路由", Icon: "Share", Path: "/admin/model-pool-routes",
	Perms: []Permission{PermModelRead, PermModelWrite}},
```

```vue
<!-- AccountPools.vue -->
<el-table :data="rows">
  <el-table-column prop="code" label="池编码" min-width="180" />
  <el-table-column prop="name" label="名称" min-width="180" />
  <el-table-column prop="pool_type" label="类型" width="120" />
  <el-table-column prop="enabled" label="启用" width="100" />
</el-table>
```

- [ ] **Step 4: Extend the existing Accounts page with pool filtering and default-pool import**

```ts
// web/src/api/accounts.ts
export function listAccounts(params: {
  page?: number
  page_size?: number
  status?: string
  keyword?: string
  pool_id?: number
} = {}) {
  return http.get<any, Page<Account>>('/api/admin/accounts', { params })
}
```

```ts
// Accounts.vue
const filter = reactive<{ status?: string; keyword?: string; pool_id?: number }>({
  status: '',
  keyword: '',
  pool_id: undefined,
})
```

- [ ] **Step 5: Run the frontend build and capture UI evidence**

Run: `cd /Users/swimmingliu/data/github-proj/gpt2api/web && rtk npm run build`

Expected: PASS

Then start the app and capture screenshots for:

- `/admin/account-pools`
- `/admin/model-pool-routes`
- `/admin/accounts` with pool filter visible

Save screenshots under:

- `/Users/swimmingliu/data/github-proj/gpt2api/docs/screenshots/account-pools-phase1.png`
- `/Users/swimmingliu/data/github-proj/gpt2api/docs/screenshots/model-pool-routes-phase1.png`
- `/Users/swimmingliu/data/github-proj/gpt2api/docs/screenshots/accounts-pool-filter-phase1.png`

- [ ] **Step 6: Commit**

```bash
git add web/src/api/accountPools.ts web/src/views/admin/AccountPools.vue web/src/views/admin/ModelPoolRoutes.vue web/src/views/admin/Accounts.vue web/src/api/accounts.ts web/src/router/index.ts internal/rbac/menu.go docs/screenshots/account-pools-phase1.png docs/screenshots/model-pool-routes-phase1.png docs/screenshots/accounts-pool-filter-phase1.png
git commit -m "feat: add admin ui for account pools and model routes"
```

### Task 5: Full verification, packaging, and delivery evidence

**Files:**
- Modify if needed after fixes from verification

- [ ] **Step 1: Run static checks and unit tests**

Run: `cd /Users/swimmingliu/data/github-proj/gpt2api && rtk go vet ./...`

Expected: exit 0

Run: `cd /Users/swimmingliu/data/github-proj/gpt2api && rtk go test ./...`

Expected: all tests PASS

- [ ] **Step 2: Run the frontend build**

Run: `cd /Users/swimmingliu/data/github-proj/gpt2api/web && rtk npm run build`

Expected: build succeeds and outputs `web/dist/`

- [ ] **Step 3: Run docker smoke tests**

Run: `cd /Users/swimmingliu/data/github-proj/gpt2api && rtk docker compose -f deploy/docker-compose.yml up -d --wait`

Expected: mysql, redis, server all healthy

Run: `cd /Users/swimmingliu/data/github-proj/gpt2api/scripts && rtk npm run smoke:docker`

Expected: smoke checks PASS

- [ ] **Step 4: Build deliverables**

Run: `cd /Users/swimmingliu/data/github-proj/gpt2api && rtk make build`

Expected: `/Users/swimmingliu/data/github-proj/gpt2api/bin/gpt2api`

Run: `cd /Users/swimmingliu/data/github-proj/gpt2api && rtk bash deploy/build-local.sh`

Expected:

- `/Users/swimmingliu/data/github-proj/gpt2api/deploy/bin/gpt2api`
- `/Users/swimmingliu/data/github-proj/gpt2api/deploy/bin/goose`
- `/Users/swimmingliu/data/github-proj/gpt2api/web/dist/`

- [ ] **Step 5: Publish branch/PR and request review**

Run:

```bash
git checkout -b codex/account-pool-phase1
git add .
git commit -m "feat: add phase 1 account pool foundation"
git push -u origin codex/account-pool-phase1
```

Create a PR that includes:

- Motivation: move from global account list to governable account pools
- Key implementation points: pool schema, admin APIs, model routing, pool-aware scheduler, admin UI
- Test results: `go vet`, `go test`, `web build`, docker smoke, build artifacts
- UI screenshots: the three screenshots from Task 4

Then request `@Copolot` review. If review causes file changes, restart verification from Step 1.
