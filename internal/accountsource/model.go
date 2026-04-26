package accountsource

import (
	"context"
	"errors"
	"time"

	"github.com/432539/gpt2api/internal/account/importcore"
)

const (
	SourceTypeSub2API = "sub2api"
	SourceTypeCPA     = "cpa"

	AuthModePassword = "password"
	AuthModeAPIKey   = "api_key"
	AuthModeBearer   = "bearer"
)

var ErrNotFound = errors.New("accountsource: not found")
var ErrBadRequest = errors.New("accountsource: bad request")

type Source struct {
	ID             uint64     `db:"id" json:"id"`
	SourceType     string     `db:"source_type" json:"source_type"`
	Name           string     `db:"name" json:"name"`
	BaseURL        string     `db:"base_url" json:"base_url"`
	Enabled        bool       `db:"enabled" json:"enabled"`
	AuthMode       string     `db:"auth_mode" json:"auth_mode"`
	Email          string     `db:"email" json:"email"`
	GroupID        string     `db:"group_id" json:"group_id"`
	APIKeyEnc      string     `db:"api_key_enc" json:"-"`
	PasswordEnc    string     `db:"password_enc" json:"-"`
	SecretKeyEnc   string     `db:"secret_key_enc" json:"-"`
	DefaultProxyID uint64     `db:"default_proxy_id" json:"default_proxy_id"`
	TargetPoolID   uint64     `db:"target_pool_id" json:"target_pool_id"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt      *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

type SourceView struct {
	ID             uint64     `json:"id"`
	SourceType     string     `json:"source_type"`
	Name           string     `json:"name"`
	BaseURL        string     `json:"base_url"`
	Enabled        bool       `json:"enabled"`
	AuthMode       string     `json:"auth_mode"`
	Email          string     `json:"email"`
	GroupID        string     `json:"group_id"`
	DefaultProxyID uint64     `json:"default_proxy_id"`
	TargetPoolID   uint64     `json:"target_pool_id"`
	HasAPIKey      bool       `json:"has_api_key"`
	HasPassword    bool       `json:"has_password"`
	HasSecretKey   bool       `json:"has_secret_key"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

type CreateInput struct {
	SourceType     string `json:"source_type"`
	Name           string `json:"name"`
	BaseURL        string `json:"base_url"`
	Enabled        *bool  `json:"enabled"`
	AuthMode       string `json:"auth_mode"`
	Email          string `json:"email"`
	GroupID        string `json:"group_id"`
	APIKey         string `json:"api_key"`
	Password       string `json:"password"`
	SecretKey      string `json:"secret_key"`
	DefaultProxyID uint64 `json:"default_proxy_id"`
	TargetPoolID   uint64 `json:"target_pool_id"`
}

type UpdateInput struct {
	Name           *string `json:"name"`
	BaseURL        *string `json:"base_url"`
	Enabled        *bool   `json:"enabled"`
	AuthMode       *string `json:"auth_mode"`
	Email          *string `json:"email"`
	GroupID        *string `json:"group_id"`
	APIKey         *string `json:"api_key"`
	Password       *string `json:"password"`
	SecretKey      *string `json:"secret_key"`
	DefaultProxyID *uint64 `json:"default_proxy_id"`
	TargetPoolID   *uint64 `json:"target_pool_id"`
}

type Sub2APIGroup struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description,omitempty"`
	Platform           string `json:"platform,omitempty"`
	Status             string `json:"status,omitempty"`
	AccountCount       int    `json:"account_count,omitempty"`
	ActiveAccountCount int    `json:"active_account_count,omitempty"`
}

type Sub2APIAccount struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	PlanType        string `json:"plan_type,omitempty"`
	Status          string `json:"status,omitempty"`
	ExpiresAt       string `json:"expires_at,omitempty"`
	HasRefreshToken bool   `json:"has_refresh_token,omitempty"`
}

type CPAFile struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type ImportSelectedInput struct {
	AccountIDs      []string `json:"account_ids"`
	FileNames       []string `json:"file_names"`
	UpdateExisting  *bool    `json:"update_existing"`
	DefaultProxyID  *uint64  `json:"default_proxy_id"`
	TargetPoolID    *uint64  `json:"target_pool_id"`
	ResolveIdentity *bool    `json:"resolve_identity"`
	KickRefresh     *bool    `json:"kick_refresh"`
	KickQuotaProbe  *bool    `json:"kick_quota_probe"`
}

type ImportSummaryResultRow struct {
	Index      int      `json:"index"`
	Email      string   `json:"email"`
	Status     string   `json:"status"`
	Reason     string   `json:"reason,omitempty"`
	ID         uint64   `json:"id,omitempty"`
	SourceType string   `json:"source_type,omitempty"`
	SourceRef  string   `json:"source_ref,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

type ImportSummary struct {
	Total   int                      `json:"total"`
	Created int                      `json:"created"`
	Updated int                      `json:"updated"`
	Skipped int                      `json:"skipped"`
	Failed  int                      `json:"failed"`
	Results []ImportSummaryResultRow `json:"results"`
}

type Store interface {
	ListSources(ctx context.Context) ([]*Source, error)
	GetSourceByID(ctx context.Context, id uint64) (*Source, error)
	CreateSource(ctx context.Context, src *Source) error
	UpdateSource(ctx context.Context, src *Source) error
	SoftDeleteSource(ctx context.Context, id uint64) error
}

type Importer interface {
	Import(ctx context.Context, candidates []importcore.ImportCandidate, opt importcore.ImportOptions) (*importcore.ImportResult, error)
}
