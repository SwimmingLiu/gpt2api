package accountpool

import (
	"context"
	"testing"
)

type fakeStore struct {
	route   *ModelRoute
	err     error
	pools   map[uint64]*Pool
	members map[uint64][]*Member
}

func (f *fakeStore) GetRouteByModelID(_ context.Context, _ uint64) (*ModelRoute, error) {
	return f.route, f.err
}

func (f *fakeStore) ListPools(_ context.Context) ([]*Pool, error)                  { return nil, nil }
func (f *fakeStore) GetPoolByID(_ context.Context, id uint64) (*Pool, error)       { return f.pools[id], nil }
func (f *fakeStore) CreatePool(_ context.Context, _ *Pool) error                   { return nil }
func (f *fakeStore) UpdatePool(_ context.Context, _ *Pool) error                   { return nil }
func (f *fakeStore) SoftDeletePool(_ context.Context, _ uint64) error              { return nil }
func (f *fakeStore) ListMembers(_ context.Context, poolID uint64) ([]*Member, error) {
	return f.members[poolID], nil
}
func (f *fakeStore) UpsertMember(_ context.Context, _ *Member) error               { return nil }
func (f *fakeStore) DeleteMember(_ context.Context, _, _ uint64) error             { return nil }
func (f *fakeStore) ListRoutes(_ context.Context) ([]*ModelRoute, error)           { return nil, nil }
func (f *fakeStore) UpsertRoute(_ context.Context, _ *ModelRoute) error            { return nil }
func (f *fakeStore) DeleteRoute(_ context.Context, _ uint64) error                 { return nil }

func TestValidatePoolCode(t *testing.T) {
	if err := validatePoolCode("image-main"); err != nil {
		t.Fatalf("expected valid code, got %v", err)
	}
	if err := validatePoolCode("bad code"); err == nil {
		t.Fatal("expected invalid pool code")
	}
}

func TestResolveModelRouteFallsBackToGlobalWhenUnset(t *testing.T) {
	svc := NewService(&fakeStore{})

	got, err := svc.ResolveModelRoute(context.Background(), 42)
	if err != nil {
		t.Fatalf("ResolveModelRoute returned error: %v", err)
	}
	if got.PoolID != 0 || got.FallbackPoolID != 0 || !got.LegacyGlobal {
		t.Fatalf("unexpected route: %+v", got)
	}
}

func TestResolveDispatchRouteUsesModelRoute(t *testing.T) {
	svc := NewService(&fakeStore{
		route: &ModelRoute{
			ModelID:        42,
			PoolID:         7,
			FallbackPoolID: 8,
			Enabled:        true,
		},
	})

	got, err := svc.ResolveDispatchRoute(context.Background(), 42, 1, 2)
	if err != nil {
		t.Fatalf("ResolveDispatchRoute returned error: %v", err)
	}
	if got.PrimaryPoolID != 7 || got.FallbackPoolID != 8 || !got.AllowFallback || got.Source != "model_route" {
		t.Fatalf("unexpected dispatch route: %+v", got)
	}
}

func TestResolveDispatchRouteFallsBackToDefaultPool(t *testing.T) {
	svc := NewService(&fakeStore{})

	got, err := svc.ResolveDispatchRoute(context.Background(), 42, 3, 4)
	if err != nil {
		t.Fatalf("ResolveDispatchRoute returned error: %v", err)
	}
	if got.PrimaryPoolID != 3 || got.FallbackPoolID != 4 || !got.AllowFallback || got.Source != "default_pool" {
		t.Fatalf("unexpected dispatch route: %+v", got)
	}
}

func TestResolvePoolRejectsDisabledPool(t *testing.T) {
	svc := NewService(&fakeStore{
		pools: map[uint64]*Pool{
			9: {ID: 9, Enabled: false},
		},
	})

	_, err := svc.ResolvePool(context.Background(), 9)
	if err == nil {
		t.Fatal("expected ResolvePool to reject disabled pool")
	}
}
