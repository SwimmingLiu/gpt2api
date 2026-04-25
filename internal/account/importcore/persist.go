package importcore

import (
	"context"
	"strings"
	"time"
)

type AccountRecord struct {
	ID               uint64
	Email            string
	AuthToken        string
	RefreshToken     string
	SessionToken     string
	ClientID         string
	ChatGPTAccountID string
	AccountType      string
	PlanType         string
	TokenExpiresAt   *time.Time
	OAISessionID     string
	OAIDeviceID      string
	Cookies          string
	Notes            string
	Status           string
}

type Store interface {
	FindByEmail(ctx context.Context, email string) (*AccountRecord, error)
	Create(ctx context.Context, candidate ImportCandidate) (uint64, error)
	Update(ctx context.Context, id uint64, candidate ImportCandidate, existing *AccountRecord) error
	BindDefaultProxy(ctx context.Context, accountID, proxyID uint64) error
}

type persistDecision struct {
	accountID uint64
	status    string
	reason    string
	email     string
}

func normalizeCandidates(candidates []ImportCandidate) []ImportCandidate {
	out := make([]ImportCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		item := candidate
		item.Email = strings.TrimSpace(strings.ToLower(item.Email))
		item.SourceType = strings.TrimSpace(item.SourceType)
		item.SourceRef = strings.TrimSpace(item.SourceRef)
		out = append(out, item)
	}
	return out
}

func deduplicateByEmail(candidates []ImportCandidate) []ImportCandidate {
	seen := make(map[string]int, len(candidates))
	out := make([]ImportCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		key := candidate.Email
		if key == "" {
			out = append(out, candidate)
			continue
		}
		if idx, ok := seen[key]; ok {
			out[idx] = candidate
			continue
		}
		seen[key] = len(out)
		out = append(out, candidate)
	}
	return out
}

func (s *Service) persistOne(ctx context.Context, candidate ImportCandidate, opt ImportOptions) ImportLineResult {
	line := ImportLineResult{
		Email:  candidate.Email,
		Source: candidate.SourceRef,
	}

	if candidate.Email == "" {
		line.Status = "failed"
		line.Reason = "email_required"
		return line
	}

	state := ClassifyCredentialState(candidate, s.now(), s.refreshAheadSec())
	if opt.SkipExpiredATOnly && state.SkipImport {
		line.Status = "skipped"
		line.Reason = "expired_access_token_only"
		return line
	}
	if state.Warning != "" {
		line.Warning = state.Warning
	}
	if s.store == nil {
		line.Status = "failed"
		line.Reason = errStoreRequired.Error()
		return line
	}

	existing, err := s.store.FindByEmail(ctx, candidate.Email)
	if err != nil {
		line.Status = "failed"
		line.Reason = err.Error()
		return line
	}

	decision := persistDecision{email: candidate.Email}
	switch {
	case existing == nil:
		accountID, err := s.store.Create(ctx, candidate)
		if err != nil {
			line.Status = "failed"
			line.Reason = err.Error()
			return line
		}
		decision.accountID = accountID
		decision.status = "created"
	case opt.UpdateExisting:
		if err := s.store.Update(ctx, existing.ID, candidate, existing); err != nil {
			line.Status = "failed"
			line.Reason = err.Error()
			return line
		}
		decision.accountID = existing.ID
		decision.status = "updated"
	default:
		decision.accountID = existing.ID
		decision.status = "skipped"
		decision.reason = "account_exists"
	}

	if decision.accountID != 0 && decision.status == "created" && opt.DefaultProxyID != 0 {
		if err := s.store.BindDefaultProxy(ctx, decision.accountID, opt.DefaultProxyID); err != nil {
			line.ID = decision.accountID
			line.Status = "failed"
			line.Reason = "default_proxy_bind_failed: " + err.Error()
			return line
		}
	}

	line.ID = decision.accountID
	line.Status = decision.status
	line.Reason = decision.reason
	return line
}
