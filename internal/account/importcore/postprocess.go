package importcore

import "context"

type PoolMembership interface {
	AddDefaultMember(ctx context.Context, poolID, accountID uint64) error
}

type Hooks interface {
	KickRefresh()
	KickQuotaProbe()
}

func (s *Service) runPostprocess(ctx context.Context, result *ImportResult, opt ImportOptions) {
	createdOrUpdated := false
	for i := range result.Results {
		line := &result.Results[i]
		if line.ID == 0 {
			continue
		}
		if opt.TargetPoolID != 0 && s.pools != nil && (line.Status == "created" || line.Status == "updated") {
			if err := s.pools.AddDefaultMember(ctx, opt.TargetPoolID, line.ID); err != nil {
				switch line.Status {
				case "created":
					result.Created--
				case "updated":
					result.Updated--
				}
				result.Failed++
				line.Status = "failed"
				line.Reason = "pool_membership_failed: " + err.Error()
				continue
			}
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
