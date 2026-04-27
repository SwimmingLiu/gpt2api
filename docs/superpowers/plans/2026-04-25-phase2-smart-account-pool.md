# Phase 2 Smart Account Pool Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add pool-aware weighted scheduling, per-member concurrency limits, sticky-session routing, and minimal pool health/operations APIs on top of the Phase 1 account-pool foundation.

**Architecture:** Keep the existing account-pool tables and routing path, but make `account_pool_members` participate in runtime selection. Scheduler will load eligible member rows, score them by priority and recent usage, enforce `max_parallel` with Redis keys, and optionally reuse the same account through a sticky key stored in Redis. Admin APIs will expose minimal pool health summaries and batch membership operations.

**Tech Stack:** Go, Gin, sqlx, Redis, Vue 3, TypeScript, Element Plus

---

### Task 1: Extend account-pool DAO for smart scheduling inputs

**Files:**
- Modify: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/codex-account-pool-phase1/internal/accountpool/model.go`
- Modify: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/codex-account-pool-phase1/internal/accountpool/dao.go`
- Create: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/codex-account-pool-phase1/internal/accountpool/dao_scheduling_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestNormalizeMemberDefaults(t *testing.T) {
	m := &Member{Weight: 0, Priority: 0, MaxParallel: 0}
	normalizeMemberDefaults(m)
	if m.Weight != 100 || m.Priority != 100 || m.MaxParallel != 1 {
		t.Fatalf("unexpected defaults: %+v", m)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `rtk go test ./internal/accountpool -run 'TestNormalizeMemberDefaults' -v`

Expected: FAIL with `undefined: normalizeMemberDefaults`

- [ ] **Step 3: Add minimal implementation**

```go
type PoolMemberCandidate struct {
	Member
	AccountEmail  string `db:"account_email"`
	AccountStatus string `db:"account_status"`
}

func normalizeMemberDefaults(m *Member) {
	if m == nil {
		return
	}
	if m.Weight <= 0 {
		m.Weight = 100
	}
	if m.Priority <= 0 {
		m.Priority = 100
	}
	if m.MaxParallel <= 0 {
		m.MaxParallel = 1
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `rtk go test ./internal/accountpool -run 'TestNormalizeMemberDefaults' -v`

Expected: PASS

### Task 2: Implement scheduler smart selection and sticky routing

**Files:**
- Modify: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/codex-account-pool-phase1/internal/scheduler/scheduler.go`
- Modify: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/codex-account-pool-phase1/internal/scheduler/scheduler_pool_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestPickCandidatePrefersLowerPriorityValue(t *testing.T) {
	candidates := []*memberRuntimeCandidate{
		{AccountID: 1, Priority: 200, Weight: 100},
		{AccountID: 2, Priority: 100, Weight: 100},
	}
	got := sortCandidates(candidates)
	if got[0].AccountID != 2 {
		t.Fatalf("expected account 2 first, got %+v", got)
	}
}

func TestStickyKeyBuildsStableRedisKey(t *testing.T) {
	key := buildStickyRedisKey(3, "user:42:model:gpt-5")
	if key != "acctpool:sticky:3:user:42:model:gpt-5" {
		t.Fatalf("unexpected sticky key: %s", key)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `rtk go test ./internal/scheduler -run 'TestPickCandidatePrefersLowerPriorityValue|TestStickyKeyBuildsStableRedisKey' -v`

Expected: FAIL with undefined helpers

- [ ] **Step 3: Add minimal implementation**

```go
type DispatchOptions struct {
	PoolID       uint64
	FallbackPoolID uint64
	StickyKey    string
	StickyTTL    time.Duration
}

func buildStickyRedisKey(poolID uint64, stickyKey string) string {
	return fmt.Sprintf("acctpool:sticky:%d:%s", poolID, stickyKey)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `rtk go test ./internal/scheduler -run 'TestPickCandidatePrefersLowerPriorityValue|TestStickyKeyBuildsStableRedisKey' -v`

Expected: PASS

### Task 3: Add minimal pool-health and batch member APIs

**Files:**
- Modify: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/codex-account-pool-phase1/internal/accountpool/service.go`
- Modify: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/codex-account-pool-phase1/internal/accountpool/handler.go`
- Modify: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/codex-account-pool-phase1/internal/server/router.go`

- [ ] **Step 1: Write the failing test**

```go
func TestHealthSummaryShape(t *testing.T) {
	s := PoolHealthSummary{PoolID: 1, TotalMembers: 3, EnabledMembers: 2}
	if s.PoolID != 1 || s.TotalMembers != 3 || s.EnabledMembers != 2 {
		t.Fatalf("unexpected summary: %+v", s)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `rtk go test ./internal/accountpool -run 'TestHealthSummaryShape' -v`

Expected: FAIL with undefined type

- [ ] **Step 3: Add implementation**

```go
type PoolHealthSummary struct {
	PoolID          uint64 `json:"pool_id"`
	TotalMembers    int    `json:"total_members"`
	EnabledMembers  int    `json:"enabled_members"`
	HealthyAccounts int    `json:"healthy_accounts"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `rtk go test ./internal/accountpool -run 'TestHealthSummaryShape' -v`

Expected: PASS

### Task 4: Wire sticky key into gateway and image runner

**Files:**
- Modify: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/codex-account-pool-phase1/internal/gateway/chat.go`
- Modify: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/codex-account-pool-phase1/internal/gateway/images.go`
- Modify: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/codex-account-pool-phase1/internal/image/runner.go`

- [ ] **Step 1: Add sticky-key builder helper and use request IDs / task IDs**

```go
func buildGatewayStickyKey(userID uint64, modelSlug, requestID string) string {
	return fmt.Sprintf("user:%d:model:%s:req:%s", userID, modelSlug, requestID)
}
```

- [ ] **Step 2: Verify by running focused package tests**

Run: `rtk go test ./internal/gateway ./internal/image ./internal/scheduler ./internal/accountpool`

Expected: PASS

### Task 5: Final verification for Phase 2

**Files:**
- Modify if needed after verification

- [ ] **Step 1: Run backend verification**

Run: `rtk go vet ./...`

Expected: exit 0

Run: `rtk go test ./...`

Expected: all tests PASS

- [ ] **Step 2: Run frontend build**

Run: `cd /Users/swimmingliu/.config/superpowers/worktrees/gpt2api/codex-account-pool-phase1/web && rtk npm run build`

Expected: build succeeds

- [ ] **Step 3: Summarize remaining environment blockers**

Capture whether Docker smoke and UI screenshots are executable in the current environment. If not, report exact blockers rather than claiming completion.
