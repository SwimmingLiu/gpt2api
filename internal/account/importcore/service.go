package importcore

import (
	"context"
	"errors"
	"time"
)

type ImportLineResult struct {
	Email   string
	Source  string
	Status  string
	Reason  string
	Warning string
	ID      uint64
}

type ImportResult struct {
	Total   int
	Created int
	Updated int
	Skipped int
	Failed  int
	Results []ImportLineResult
}

type ServiceDeps struct {
	Store            Store
	PoolMembership   PoolMembership
	Hooks            Hooks
	IdentityResolver IdentityResolver
	Now              func() time.Time
	RefreshAheadSec  func() int
}

type Service struct {
	store            Store
	pools            PoolMembership
	hooks            Hooks
	identityResolver IdentityResolver
	now              func() time.Time
	refreshAheadSec  func() int
}

var errStoreRequired = errors.New("importcore: store is required")

func NewService(deps ServiceDeps) *Service {
	nowFn := deps.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	refreshAheadFn := deps.RefreshAheadSec
	if refreshAheadFn == nil {
		refreshAheadFn = func() int { return 0 }
	}
	return &Service{
		store:            deps.Store,
		pools:            deps.PoolMembership,
		hooks:            deps.Hooks,
		identityResolver: deps.IdentityResolver,
		now:              nowFn,
		refreshAheadSec:  refreshAheadFn,
	}
}

func DefaultOptions() ImportOptions {
	return ImportOptions{
		SkipExpiredATOnly: true,
		ResolveIdentity:   true,
	}
}

func (s *Service) Import(ctx context.Context, candidates []ImportCandidate, opt ImportOptions) (*ImportResult, error) {
	result := &ImportResult{Results: make([]ImportLineResult, 0, len(candidates))}
	normalized := normalizeCandidates(candidates)
	resolved := s.resolveCandidates(ctx, normalized, opt)
	deduped := deduplicatePreparedByEmail(resolved)
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
	s.runPostprocess(ctx, result, opt)
	return result, nil
}
