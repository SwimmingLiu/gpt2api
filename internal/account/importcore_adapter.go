package account

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/432539/gpt2api/internal/account/importcore"
	"github.com/432539/gpt2api/internal/accountpool"
)

type importCoreAdapter struct {
	svc *Service
}

func (a *importCoreAdapter) FindByEmail(ctx context.Context, email string) (*importcore.AccountRecord, error) {
	if a == nil || a.svc == nil || a.svc.dao == nil {
		return nil, errors.New("account service unavailable")
	}
	existing, err := a.svc.dao.GetByEmail(ctx, strings.TrimSpace(strings.ToLower(email)))
	if err != nil || existing == nil {
		return nil, err
	}
	record := &importcore.AccountRecord{
		ID:               existing.ID,
		Email:            existing.Email,
		ClientID:         existing.ClientID,
		ChatGPTAccountID: existing.ChatGPTAccountID,
		AccountType:      existing.AccountType,
		PlanType:         existing.PlanType,
		OAISessionID:     existing.OAISessionID,
		OAIDeviceID:      existing.OAIDeviceID,
		Notes:            existing.Notes,
		Status:           existing.Status,
	}
	if existing.TokenExpiresAt.Valid {
		ts := existing.TokenExpiresAt.Time.UTC()
		record.TokenExpiresAt = &ts
	}
	return record, nil
}

func (a *importCoreAdapter) Create(ctx context.Context, candidate importcore.ImportCandidate) (uint64, error) {
	if a == nil || a.svc == nil || a.svc.dao == nil || a.svc.cipher == nil {
		return 0, errors.New("account service unavailable")
	}
	atEnc, err := a.svc.cipher.EncryptString(candidate.AccessToken)
	if err != nil {
		return 0, err
	}
	account := &Account{
		Email:            candidate.Email,
		AuthTokenEnc:     atEnc,
		ClientID:         defaultClientID(candidate.ClientID),
		ChatGPTAccountID: candidate.ChatGPTAccountID,
		AccountType:      defaultAccountType(candidate.AccountType),
		PlanType:         defaultPlanType(candidate.PlanType),
		DailyImageQuota:  100,
		Status:           StatusHealthy,
		OAISessionID:     candidate.OAISessionID,
		OAIDeviceID:      candidate.OAIDeviceID,
		Notes:            candidate.Notes,
	}
	if candidate.RefreshToken != "" {
		rtEnc, err := a.svc.cipher.EncryptString(candidate.RefreshToken)
		if err != nil {
			return 0, err
		}
		account.RefreshTokenEnc = sql.NullString{String: rtEnc, Valid: true}
	}
	if candidate.SessionToken != "" {
		stEnc, err := a.svc.cipher.EncryptString(candidate.SessionToken)
		if err != nil {
			return 0, err
		}
		account.SessionTokenEnc = sql.NullString{String: stEnc, Valid: true}
	}
	if candidate.TokenExpiresAt != nil {
		account.TokenExpiresAt = sql.NullTime{Time: candidate.TokenExpiresAt.UTC(), Valid: true}
	} else if expAt := parseJWTExp(candidate.AccessToken); !expAt.IsZero() {
		account.TokenExpiresAt = sql.NullTime{Time: expAt.UTC(), Valid: true}
	}
	id, err := a.svc.dao.Create(ctx, account)
	if err != nil {
		return 0, err
	}
	if candidate.Cookies != "" {
		cookieEnc, err := a.svc.cipher.EncryptString(candidate.Cookies)
		if err != nil {
			return 0, err
		}
		if err := a.svc.dao.UpsertCookies(ctx, id, cookieEnc); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func (a *importCoreAdapter) Update(ctx context.Context, id uint64, candidate importcore.ImportCandidate, _ *importcore.AccountRecord) error {
	if a == nil || a.svc == nil || a.svc.dao == nil || a.svc.cipher == nil {
		return errors.New("account service unavailable")
	}
	current, err := a.svc.dao.GetByID(ctx, id)
	if err != nil {
		return err
	}
	atEnc, err := a.svc.cipher.EncryptString(candidate.AccessToken)
	if err != nil {
		return err
	}
	current.AuthTokenEnc = atEnc
	if candidate.RefreshToken != "" {
		rtEnc, err := a.svc.cipher.EncryptString(candidate.RefreshToken)
		if err != nil {
			return err
		}
		current.RefreshTokenEnc = sql.NullString{String: rtEnc, Valid: true}
	}
	if candidate.SessionToken != "" {
		stEnc, err := a.svc.cipher.EncryptString(candidate.SessionToken)
		if err != nil {
			return err
		}
		current.SessionTokenEnc = sql.NullString{String: stEnc, Valid: true}
	}
	if candidate.TokenExpiresAt != nil {
		current.TokenExpiresAt = sql.NullTime{Time: candidate.TokenExpiresAt.UTC(), Valid: true}
	} else if expAt := parseJWTExp(candidate.AccessToken); !expAt.IsZero() {
		current.TokenExpiresAt = sql.NullTime{Time: expAt.UTC(), Valid: true}
	}
	if candidate.ClientID != "" {
		current.ClientID = candidate.ClientID
	}
	if candidate.ChatGPTAccountID != "" {
		current.ChatGPTAccountID = candidate.ChatGPTAccountID
	}
	if candidate.AccountType != "" {
		current.AccountType = candidate.AccountType
	}
	if candidate.PlanType != "" {
		current.PlanType = candidate.PlanType
	}
	if candidate.OAISessionID != "" {
		current.OAISessionID = candidate.OAISessionID
	}
	if candidate.OAIDeviceID != "" {
		current.OAIDeviceID = candidate.OAIDeviceID
	}
	if candidate.Notes != "" {
		current.Notes = candidate.Notes
	}
	if current.Status == StatusDead || current.Status == StatusSuspicious {
		current.Status = StatusHealthy
	}
	if err := a.svc.dao.Update(ctx, current); err != nil {
		return err
	}
	if candidate.Cookies != "" {
		cookieEnc, err := a.svc.cipher.EncryptString(candidate.Cookies)
		if err != nil {
			return err
		}
		if err := a.svc.dao.UpsertCookies(ctx, id, cookieEnc); err != nil {
			return err
		}
	}
	return nil
}

func (a *importCoreAdapter) BindDefaultProxy(ctx context.Context, accountID, proxyID uint64) error {
	if a == nil || a.svc == nil || a.svc.dao == nil {
		return errors.New("account service unavailable")
	}
	return a.svc.dao.SetBinding(ctx, accountID, proxyID)
}

type importCorePoolAdapter struct {
	svc *accountpool.Service
}

func (a *importCorePoolAdapter) AddDefaultMember(ctx context.Context, poolID, accountID uint64) error {
	if a == nil || a.svc == nil {
		return nil
	}
	_, err := a.svc.UpsertMember(ctx, poolID, 0, accountpool.UpsertMemberInput{AccountID: accountID})
	return err
}

type importCoreHooks struct {
	refresher *Refresher
	prober    *QuotaProber
}

func (h *importCoreHooks) KickRefresh() {
	if h != nil && h.refresher != nil {
		h.refresher.Kick()
	}
}

func (h *importCoreHooks) KickQuotaProbe() {
	if h != nil && h.prober != nil {
		h.prober.Kick()
	}
}

type importCoreIdentityResolver struct{}

func (r *importCoreIdentityResolver) ResolveEmail(_ context.Context, candidate importcore.ImportCandidate) (string, error) {
	if strings.TrimSpace(candidate.AccessToken) == "" {
		return "", errors.New("email_required_identity_unresolved")
	}
	email, _, _, err := decodeATClaims(candidate.AccessToken)
	if err != nil {
		return "", err
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return "", errors.New("email_required_identity_unresolved")
	}
	return email, nil
}

func NewUnifiedImportCore(
	svc *Service,
	poolSvc *accountpool.Service,
	refresher *Refresher,
	prober *QuotaProber,
	refreshAheadSec func() int,
) *importcore.Service {
	return importcore.NewService(importcore.ServiceDeps{
		Store:            &importCoreAdapter{svc: svc},
		PoolMembership:   &importCorePoolAdapter{svc: poolSvc},
		Hooks:            &importCoreHooks{refresher: refresher, prober: prober},
		IdentityResolver: &importCoreIdentityResolver{},
		RefreshAheadSec:  refreshAheadSec,
	})
}

func defaultClientID(clientID string) string {
	if strings.TrimSpace(clientID) == "" {
		return "app_EMoamEEZ73f0CkXaXp7hrann"
	}
	return strings.TrimSpace(clientID)
}

func defaultAccountType(accountType string) string {
	if strings.TrimSpace(accountType) == "" {
		return "codex"
	}
	return strings.TrimSpace(accountType)
}

func defaultPlanType(planType string) string {
	if strings.TrimSpace(planType) == "" {
		return "free"
	}
	return strings.TrimSpace(planType)
}
