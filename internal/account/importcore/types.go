package importcore

import "time"

type ImportCandidate struct {
	SourceType       string
	SourceRef        string
	AccessToken      string
	RefreshToken     string
	SessionToken     string
	Email            string
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

type ImportOptions struct {
	UpdateExisting    bool
	DefaultProxyID    uint64
	TargetPoolID      uint64
	SkipExpiredATOnly bool
	ResolveIdentity   bool
	KickRefresh       bool
	KickQuotaProbe    bool
}

type CredentialState struct {
	Capability string
	Lifecycle  string
	SkipImport bool
	Warning    string
}
