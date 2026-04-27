package importsource

import (
	"strings"
	"time"

	"github.com/432539/gpt2api/internal/account/importcore"
)

type ManualInput struct {
	Email            string
	AccessToken      string
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
}

func ParseManual(input ManualInput) []importcore.ImportCandidate {
	candidate := importcore.ImportCandidate{
		SourceType:       "manual",
		SourceRef:        "admin_form",
		AccessToken:      strings.TrimSpace(input.AccessToken),
		RefreshToken:     strings.TrimSpace(input.RefreshToken),
		SessionToken:     strings.TrimSpace(input.SessionToken),
		Email:            strings.TrimSpace(input.Email),
		ClientID:         strings.TrimSpace(input.ClientID),
		ChatGPTAccountID: strings.TrimSpace(input.ChatGPTAccountID),
		AccountType:      strings.TrimSpace(input.AccountType),
		PlanType:         strings.TrimSpace(input.PlanType),
		TokenExpiresAt:   input.TokenExpiresAt,
		OAISessionID:     strings.TrimSpace(input.OAISessionID),
		OAIDeviceID:      strings.TrimSpace(input.OAIDeviceID),
		Cookies:          strings.TrimSpace(input.Cookies),
		Notes:            strings.TrimSpace(input.Notes),
	}
	return []importcore.ImportCandidate{candidate}
}
