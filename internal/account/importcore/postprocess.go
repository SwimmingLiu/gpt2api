package importcore

import "context"

type PoolMembership interface {
	AddDefaultMember(ctx context.Context, poolID, accountID uint64) error
}

type Hooks interface {
	KickRefresh()
	KickQuotaProbe()
}

func (s *Service) runPostprocess(ctx context.Context, results []ImportLineResult, opt ImportOptions) {
	createdOrUpdated := false
	for _, line := range results {
		if line.ID == 0 {
			continue
		}
		if opt.TargetPoolID != 0 && s.pools != nil && (line.Status == "created" || line.Status == "updated") {
			_ = s.pools.AddDefaultMember(ctx, opt.TargetPoolID, line.ID)
		}
		if line.Status == "created" || line.Status == "updated" {
			createdOrUpdated = true
		}
	}
	if !createdOrUpdated || s.hooks == nil {
		return
	}
	if opt.KickRefresh {
		s.hooks.KickRefresh()
	}
	if opt.KickQuotaProbe {
		s.hooks.KickQuotaProbe()
	}
}
