package importcore

import (
	"context"
	"testing"
	"time"
)

type fakeStore struct {
	findByEmail map[string]*AccountRecord
	lastCreated ImportCandidate
	lastUpdated struct {
		id        uint64
		candidate ImportCandidate
		existing  *AccountRecord
	}
	createCalls int
	updateCalls int
	bindCalls   []struct {
		accountID uint64
		proxyID   uint64
	}
	nextID uint64
}

func (s *fakeStore) FindByEmail(_ context.Context, email string) (*AccountRecord, error) {
	if s.findByEmail == nil {
		return nil, nil
	}
	return s.findByEmail[email], nil
}

func (s *fakeStore) Create(_ context.Context, candidate ImportCandidate) (uint64, error) {
	s.createCalls++
	s.lastCreated = candidate
	if s.nextID == 0 {
		s.nextID = 1
	}
	return s.nextID, nil
}

func (s *fakeStore) Update(_ context.Context, id uint64, candidate ImportCandidate, existing *AccountRecord) error {
	s.updateCalls++
	s.lastUpdated.id = id
	s.lastUpdated.candidate = candidate
	s.lastUpdated.existing = existing
	return nil
}

func (s *fakeStore) BindDefaultProxy(_ context.Context, accountID, proxyID uint64) error {
	s.bindCalls = append(s.bindCalls, struct {
		accountID uint64
		proxyID   uint64
	}{accountID: accountID, proxyID: proxyID})
	return nil
}

type fakePoolMembership struct {
	calls []struct {
		poolID    uint64
		accountID uint64
	}
}

func (p *fakePoolMembership) AddDefaultMember(_ context.Context, poolID, accountID uint64) error {
	p.calls = append(p.calls, struct {
		poolID    uint64
		accountID uint64
	}{poolID: poolID, accountID: accountID})
	return nil
}

type fakeHooks struct {
	refreshCalls    int
	quotaProbeCalls int
}

func (h *fakeHooks) KickRefresh() {
	h.refreshCalls++
}

func (h *fakeHooks) KickQuotaProbe() {
	h.quotaProbeCalls++
}

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

func TestImportSkipsExpiredATOnlyCandidate(t *testing.T) {
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	exp := now.Add(-time.Minute)

	core := NewService(ServiceDeps{Now: func() time.Time { return now }, RefreshAheadSec: func() int { return 900 }})
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
	pools := &fakePoolMembership{}
	hooks := &fakeHooks{}
	core := NewService(ServiceDeps{
		Store:           store,
		PoolMembership:  pools,
		Hooks:           hooks,
		Now:             time.Now,
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
	if len(pools.calls) != 1 || pools.calls[0].poolID != 7 || pools.calls[0].accountID != 1 {
		t.Fatalf("unexpected pool membership calls: %+v", pools.calls)
	}
	if hooks.refreshCalls != 1 || hooks.quotaProbeCalls != 1 {
		t.Fatalf("unexpected hooks state: %+v", hooks)
	}
}

func TestImportUpdatesExistingAccount(t *testing.T) {
	store := &fakeStore{
		findByEmail: map[string]*AccountRecord{
			"user@example.com": {ID: 9, Email: "user@example.com", AuthToken: "old"},
		},
	}
	core := NewService(ServiceDeps{
		Store:           store,
		Now:             time.Now,
		RefreshAheadSec: func() int { return 900 },
	})

	result, err := core.Import(context.Background(), []ImportCandidate{{
		SourceType:  "manual",
		SourceRef:   "admin_form",
		AccessToken: "new-at",
		Email:       "user@example.com",
	}}, ImportOptions{UpdateExisting: true})
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if result.Updated != 1 || store.updateCalls != 1 || store.lastUpdated.id != 9 {
		t.Fatalf("expected update path, got result=%+v store=%+v", result, store)
	}
}

func TestImportSkipsExistingAccountWhenUpdatesDisabled(t *testing.T) {
	store := &fakeStore{
		findByEmail: map[string]*AccountRecord{
			"user@example.com": {ID: 12, Email: "user@example.com", AuthToken: "old"},
		},
	}
	core := NewService(ServiceDeps{
		Store:           store,
		Now:             time.Now,
		RefreshAheadSec: func() int { return 900 },
	})

	result, err := core.Import(context.Background(), []ImportCandidate{{
		SourceType:  "manual",
		SourceRef:   "admin_form",
		AccessToken: "new-at",
		Email:       "user@example.com",
	}}, ImportOptions{UpdateExisting: false})
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if result.Skipped != 1 || store.createCalls != 0 || store.updateCalls != 0 {
		t.Fatalf("expected skip path, got result=%+v store=%+v", result, store)
	}
	if result.Results[0].Reason == "" {
		t.Fatalf("expected skip reason, got %+v", result.Results[0])
	}
}
