package accountpool

import (
	"context"
	"testing"
)

type fakeStore struct {
	route *ModelRoute
	err   error
}

func (f *fakeStore) GetRouteByModelID(_ context.Context, _ uint64) (*ModelRoute, error) {
	return f.route, f.err
}

func (f *fakeStore) ListPools(_ context.Context) ([]*Pool, error)                  { return nil, nil }
func (f *fakeStore) GetPoolByID(_ context.Context, _ uint64) (*Pool, error)        { return nil, nil }
func (f *fakeStore) CreatePool(_ context.Context, _ *Pool) error                   { return nil }
func (f *fakeStore) UpdatePool(_ context.Context, _ *Pool) error                   { return nil }
func (f *fakeStore) SoftDeletePool(_ context.Context, _ uint64) error              { return nil }
func (f *fakeStore) ListMembers(_ context.Context, _ uint64) ([]*Member, error)    { return nil, nil }
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
