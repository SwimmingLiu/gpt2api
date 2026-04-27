package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/432539/gpt2api/internal/account"
	"github.com/432539/gpt2api/internal/accountpool"
	"github.com/432539/gpt2api/internal/config"
	"github.com/432539/gpt2api/internal/proxy"
)

type fakePoolResolver struct {
	pools map[uint64]accountpool.ResolvedPool
}

func (f *fakePoolResolver) ResolvePool(_ context.Context, poolID uint64) (accountpool.ResolvedPool, error) {
	pool, ok := f.pools[poolID]
	if !ok {
		return accountpool.ResolvedPool{}, accountpool.ErrNotFound
	}
	return pool, nil
}

type fakeAccountDAO struct {
	accounts map[uint64]*account.Account
}

func (f *fakeAccountDAO) GetByID(_ context.Context, id uint64) (*account.Account, error) {
	acc, ok := f.accounts[id]
	if !ok {
		return nil, account.ErrNotFound
	}
	return acc, nil
}

func (f *fakeAccountDAO) EnsureDeviceID(_ context.Context, id uint64, deviceID string) (string, error) {
	f.accounts[id].OAIDeviceID = deviceID
	return deviceID, nil
}

func (f *fakeAccountDAO) EnsureSessionID(_ context.Context, id uint64, sessionID string) (string, error) {
	f.accounts[id].OAISessionID = sessionID
	return sessionID, nil
}

func (f *fakeAccountDAO) MarkUsed(_ context.Context, _ uint64, _ time.Time) error { return nil }

func (f *fakeAccountDAO) SetStatus(_ context.Context, id uint64, status string, _ *time.Time) error {
	f.accounts[id].Status = status
	return nil
}

type fakeAccountRuntime struct {
	dao *fakeAccountDAO
}

func (f *fakeAccountRuntime) DAO() accountSchedulerDAO { return f.dao }

func (f *fakeAccountRuntime) DecryptAuthToken(_ *account.Account) (string, error) {
	return "auth-token", nil
}

func (f *fakeAccountRuntime) GetBinding(_ context.Context, _ uint64) (*account.Binding, error) {
	return nil, nil
}

type fakeProxyRuntime struct{}

func (f *fakeProxyRuntime) Get(_ context.Context, _ uint64) (*proxy.Proxy, error) { return nil, nil }
func (f *fakeProxyRuntime) BuildURL(_ *proxy.Proxy) (string, error)               { return "", nil }

type fakeLock struct {
	failKeys map[string]bool
}

func (f *fakeLock) Acquire(_ context.Context, key, _ string, _ time.Duration) error {
	if f.failKeys[key] {
		return errors.New("lock not acquired")
	}
	return nil
}

func (f *fakeLock) Release(_ context.Context, _, _ string) error { return nil }

func TestDispatchUsesPrimaryPool(t *testing.T) {
	sched := newTestScheduler(map[uint64]*account.Account{
		1: newHealthyAccount(1),
	}, map[uint64]accountpool.ResolvedPool{
		10: {
			Pool:    &accountpool.Pool{ID: 10, Enabled: true},
			Members: []*accountpool.Member{{PoolID: 10, AccountID: 1, Enabled: true, Priority: 10, Weight: 100, MaxParallel: 1}},
		},
	})

	lease, err := sched.Dispatch(context.Background(), accountpool.DispatchRoute{PrimaryPoolID: 10})
	if err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}
	if lease.Account.ID != 1 {
		t.Fatalf("expected account 1, got %d", lease.Account.ID)
	}
}

func TestDispatchSkipsDisabledMember(t *testing.T) {
	sched := newTestScheduler(map[uint64]*account.Account{
		1: newHealthyAccount(1),
		2: newHealthyAccount(2),
	}, map[uint64]accountpool.ResolvedPool{
		10: {
			Pool: &accountpool.Pool{ID: 10, Enabled: true},
			Members: []*accountpool.Member{
				{PoolID: 10, AccountID: 1, Enabled: false, Priority: 10, Weight: 100, MaxParallel: 1},
				{PoolID: 10, AccountID: 2, Enabled: true, Priority: 20, Weight: 100, MaxParallel: 1},
			},
		},
	})

	lease, err := sched.Dispatch(context.Background(), accountpool.DispatchRoute{PrimaryPoolID: 10})
	if err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}
	if lease.Account.ID != 2 {
		t.Fatalf("expected account 2, got %d", lease.Account.ID)
	}
}

func TestDispatchFallsBackWhenPrimaryPoolHasNoAvailableAccount(t *testing.T) {
	primary := newHealthyAccount(1)
	primary.Status = account.StatusDead
	sched := newTestScheduler(map[uint64]*account.Account{
		1: primary,
		2: newHealthyAccount(2),
	}, map[uint64]accountpool.ResolvedPool{
		10: {
			Pool:    &accountpool.Pool{ID: 10, Enabled: true},
			Members: []*accountpool.Member{{PoolID: 10, AccountID: 1, Enabled: true, Priority: 10, Weight: 100, MaxParallel: 1}},
		},
		20: {
			Pool:    &accountpool.Pool{ID: 20, Enabled: true},
			Members: []*accountpool.Member{{PoolID: 20, AccountID: 2, Enabled: true, Priority: 10, Weight: 100, MaxParallel: 1}},
		},
	})

	lease, err := sched.Dispatch(context.Background(), accountpool.DispatchRoute{
		PrimaryPoolID:  10,
		FallbackPoolID: 20,
		AllowFallback:  true,
	})
	if err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}
	if lease.Account.ID != 2 {
		t.Fatalf("expected fallback account 2, got %d", lease.Account.ID)
	}
}

func TestDispatchFailsWhenNoPoolMemberAvailable(t *testing.T) {
	expired := newHealthyAccount(1)
	expired.TokenExpiresAt = sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true}
	sched := newTestScheduler(map[uint64]*account.Account{
		1: expired,
	}, map[uint64]accountpool.ResolvedPool{
		10: {
			Pool:    &accountpool.Pool{ID: 10, Enabled: true},
			Members: []*accountpool.Member{{PoolID: 10, AccountID: 1, Enabled: true, Priority: 10, Weight: 100, MaxParallel: 1}},
		},
	})

	_, err := sched.Dispatch(context.Background(), accountpool.DispatchRoute{PrimaryPoolID: 10})
	if !errors.Is(err, ErrNoAvailable) {
		t.Fatalf("expected ErrNoAvailable, got %v", err)
	}
}

func newTestScheduler(accounts map[uint64]*account.Account, pools map[uint64]accountpool.ResolvedPool) *Scheduler {
	s := New(
		&fakeAccountRuntime{dao: &fakeAccountDAO{accounts: accounts}},
		&fakeProxyRuntime{},
		&fakePoolResolver{pools: pools},
		&fakeLock{failKeys: map[string]bool{}},
		config.SchedulerConfig{
			MinIntervalSec:  0,
			DailyUsageRatio: 1,
			LockTTLSec:      60,
		},
	)
	s.SetRuntime(RuntimeParams{
		QueueWaitSec: func() int { return 0 },
	})
	return s
}

func newHealthyAccount(id uint64) *account.Account {
	return &account.Account{
		ID:           id,
		AuthTokenEnc: "enc",
		Status:       account.StatusHealthy,
		PlanType:     "plus",
		OAIDeviceID:  "did",
		OAISessionID: "sid",
	}
}
