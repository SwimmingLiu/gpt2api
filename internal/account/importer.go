package account

import (
	"context"
	"strings"
	"time"

	"github.com/432539/gpt2api/internal/account/importcore"
	"github.com/432539/gpt2api/internal/account/importsource"
)

// ImportSource 代表一条待导入记录,来自任意一种 JSON 格式。
type ImportSource struct {
	// 必填
	AccessToken string
	Email       string

	// 可选
	RefreshToken     string
	SessionToken     string // 从 cookie 里提取的 __Secure-next-auth.session-token
	ClientID         string
	ChatGPTAccountID string
	AccountType      string // codex / chatgpt
	ExpiredAt        time.Time
	Name             string // sub2api 里的 name 字段(当 email 缺失时退化为 email)
}

// ImportLineResult 返回给前端,每条记录处理结果。
type ImportLineResult struct {
	Index  int    `json:"index"`
	Email  string `json:"email"`
	Status string `json:"status"` // created / updated / skipped / failed
	Reason string `json:"reason,omitempty"`
	ID     uint64 `json:"id,omitempty"`
}

// ImportSummary 整体统计。
type ImportSummary struct {
	Total   int                `json:"total"`
	Created int                `json:"created"`
	Updated int                `json:"updated"`
	Skipped int                `json:"skipped"`
	Failed  int                `json:"failed"`
	Results []ImportLineResult `json:"results"`
}

// ImportOptions 批量导入选项。
type ImportOptions struct {
	// UpdateExisting 为 true 时 email 已存在则更新 token;false 则 skipped。
	UpdateExisting bool
	// DefaultClientID 当记录里没有 client_id 时填充的值。
	DefaultClientID string
	// DefaultProxyID 新建账号时默认绑定的代理 id(0 = 不绑)。
	DefaultProxyID uint64
	// BatchSize 分批 commit 的大小(仅用于让出 CPU,每批做一次 context check)。默认 200。
	BatchSize int
}

// ParseJSONBlob 尝试把用户上传的文本解析成 ImportSource 列表。
// 同时兼容以下输入:
//  1. 顶层是对象且含 `accounts` 数组 → sub2api 多账号导出
//  2. 顶层是对象且含 `access_token` / `accessToken` → 单账号 token_xxx.json
//  3. 顶层是数组,每个元素同 (1)/(2) 的单个对象
//  4. 多个 JSON 文本用换行/空行分隔(JSONL)
func ParseJSONBlob(raw string) ([]ImportSource, error) {
	candidates, err := parseCandidatesFromJSON([]byte(raw))
	if err != nil {
		return nil, err
	}
	out := make([]ImportSource, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, importSourceFromCandidate(candidate))
	}
	return out, nil
}

func parseCandidatesFromJSON(raw []byte) ([]importcore.ImportCandidate, error) {
	return importsource.ParseAutoJSON(raw)
}

func importSourceFromCandidate(candidate importcore.ImportCandidate) ImportSource {
	var expiredAt time.Time
	if candidate.TokenExpiresAt != nil {
		expiredAt = candidate.TokenExpiresAt.UTC()
	}
	return ImportSource{
		AccessToken:      candidate.AccessToken,
		Email:            candidate.Email,
		RefreshToken:     candidate.RefreshToken,
		SessionToken:     candidate.SessionToken,
		ClientID:         candidate.ClientID,
		ChatGPTAccountID: candidate.ChatGPTAccountID,
		AccountType:      candidate.AccountType,
		ExpiredAt:        expiredAt,
		Name:             candidate.Notes,
	}
}

// ImportBatch 执行批量导入。
// 处理策略:
//   - 同一批内 email 去重(后者覆盖前者)
//   - email 已存在则按 UpdateExisting 决定更新或 skip
//   - 每 BatchSize 条让出一次 CPU,并检查 ctx.Done(),便于大批量
//   - 不做整体事务(失败项不影响成功项);单条失败只影响该条
func (s *Service) ImportBatch(ctx context.Context, items []ImportSource, opt ImportOptions) *ImportSummary {
	if opt.BatchSize <= 0 {
		opt.BatchSize = 200
	}
	if opt.DefaultClientID == "" {
		opt.DefaultClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
	}

	// email 去重(后者覆盖)
	seen := make(map[string]int, len(items))
	dedup := make([]ImportSource, 0, len(items))
	for _, it := range items {
		if it.Email == "" {
			continue
		}
		key := strings.ToLower(it.Email)
		if idx, ok := seen[key]; ok {
			dedup[idx] = it // 覆盖
		} else {
			seen[key] = len(dedup)
			dedup = append(dedup, it)
		}
	}

	sum := &ImportSummary{
		Total:   len(dedup),
		Results: make([]ImportLineResult, 0, len(dedup)),
	}

	for i, it := range dedup {
		if i > 0 && i%opt.BatchSize == 0 {
			// 让出一次 CPU;大批量下防止长时间独占
			select {
			case <-ctx.Done():
				sum.Failed += len(dedup) - i
				sum.Results = append(sum.Results, ImportLineResult{
					Index: i, Email: "", Status: "failed", Reason: "导入被取消",
				})
				return sum
			default:
			}
		}
		res := s.importOne(ctx, i, it, opt)
		switch res.Status {
		case "created":
			sum.Created++
		case "updated":
			sum.Updated++
		case "skipped":
			sum.Skipped++
		case "failed":
			sum.Failed++
		}
		sum.Results = append(sum.Results, res)
	}
	return sum
}

func (s *Service) importOne(ctx context.Context, idx int, it ImportSource, opt ImportOptions) ImportLineResult {
	out := ImportLineResult{Index: idx, Email: it.Email}

	if it.AccessToken == "" {
		out.Status = "failed"
		out.Reason = "缺少 access_token"
		return out
	}

	// 计算过期时间:优先用 JSON 的 expired,其次解析 JWT
	expAt := it.ExpiredAt
	if expAt.IsZero() {
		expAt = parseJWTExp(it.AccessToken)
	}

	clientID := it.ClientID
	if clientID == "" {
		clientID = opt.DefaultClientID
	}
	accountType := it.AccountType
	if accountType == "" {
		accountType = "codex"
	}

	// 查是否已存在
	existing, err := s.dao.GetByEmail(ctx, it.Email)
	if err != nil {
		out.Status = "failed"
		out.Reason = "查询失败:" + err.Error()
		return out
	}

	atEnc, err := s.cipher.EncryptString(it.AccessToken)
	if err != nil {
		out.Status = "failed"
		out.Reason = "AT 加密失败:" + err.Error()
		return out
	}
	var rtEnc, stEnc string
	if it.RefreshToken != "" {
		if v, err := s.cipher.EncryptString(it.RefreshToken); err == nil {
			rtEnc = v
		}
	}
	if it.SessionToken != "" {
		if v, err := s.cipher.EncryptString(it.SessionToken); err == nil {
			stEnc = v
		}
	}

	if existing == nil {
		// 新建
		a := &Account{
			Email:            it.Email,
			AuthTokenEnc:     atEnc,
			ClientID:         clientID,
			ChatGPTAccountID: it.ChatGPTAccountID,
			AccountType:      accountType,
			PlanType:         "free",
			DailyImageQuota:  100,
			Status:           StatusHealthy,
		}
		if rtEnc != "" {
			a.RefreshTokenEnc.String = rtEnc
			a.RefreshTokenEnc.Valid = true
		}
		if stEnc != "" {
			a.SessionTokenEnc.String = stEnc
			a.SessionTokenEnc.Valid = true
		}
		if !expAt.IsZero() {
			a.TokenExpiresAt.Time = expAt
			a.TokenExpiresAt.Valid = true
		}
		id, err := s.dao.Create(ctx, a)
		if err != nil {
			out.Status = "failed"
			out.Reason = "入库失败:" + err.Error()
			return out
		}
		if opt.DefaultProxyID > 0 {
			_ = s.dao.SetBinding(ctx, id, opt.DefaultProxyID)
		}
		out.Status = "created"
		out.ID = id
		return out
	}

	// 已存在
	if !opt.UpdateExisting {
		out.Status = "skipped"
		out.Reason = "邮箱已存在"
		out.ID = existing.ID
		return out
	}
	// 更新 token 字段,其它字段保持
	existing.AuthTokenEnc = atEnc
	if rtEnc != "" {
		existing.RefreshTokenEnc.String = rtEnc
		existing.RefreshTokenEnc.Valid = true
	}
	if stEnc != "" {
		existing.SessionTokenEnc.String = stEnc
		existing.SessionTokenEnc.Valid = true
	}
	if clientID != "" {
		existing.ClientID = clientID
	}
	if it.ChatGPTAccountID != "" {
		existing.ChatGPTAccountID = it.ChatGPTAccountID
	}
	if accountType != "" {
		existing.AccountType = accountType
	}
	if !expAt.IsZero() {
		existing.TokenExpiresAt.Time = expAt
		existing.TokenExpiresAt.Valid = true
	}
	// 复活已死账号(导入新 token 视为重新投放)
	if existing.Status == StatusDead || existing.Status == StatusSuspicious {
		existing.Status = StatusHealthy
	}
	if err := s.dao.Update(ctx, existing); err != nil {
		out.Status = "failed"
		out.Reason = "更新失败:" + err.Error()
		return out
	}
	out.Status = "updated"
	out.ID = existing.ID
	return out
}
