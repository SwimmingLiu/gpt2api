package accountsource

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/432539/gpt2api/internal/account/importcore"
	"github.com/432539/gpt2api/pkg/crypto"
)

type Service struct {
	store    Store
	cipher   *crypto.AESGCM
	importer Importer
	hc       *http.Client
}

func NewService(store Store, cipher *crypto.AESGCM, importer Importer) *Service {
	return &Service{
		store:    store,
		cipher:   cipher,
		importer: importer,
		hc:       defaultHTTPClient(),
	}
}

func (s *Service) SetHTTPClient(hc *http.Client) {
	if hc != nil {
		s.hc = hc
	}
}

func (s *Service) List(ctx context.Context) ([]*SourceView, error) {
	items, err := s.store.ListSources(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*SourceView, 0, len(items))
	for _, item := range items {
		out = append(out, toView(item))
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, id uint64) (*SourceView, error) {
	item, err := s.store.GetSourceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toView(item), nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*SourceView, error) {
	src, err := s.buildNewSource(in)
	if err != nil {
		return nil, err
	}
	if err := s.store.CreateSource(ctx, src); err != nil {
		return nil, err
	}
	return toView(src), nil
}

func (s *Service) Update(ctx context.Context, id uint64, in UpdateInput) (*SourceView, error) {
	src, err := s.store.GetSourceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.applyUpdate(src, in); err != nil {
		return nil, err
	}
	if err := s.store.UpdateSource(ctx, src); err != nil {
		return nil, err
	}
	return toView(src), nil
}

func (s *Service) Delete(ctx context.Context, id uint64) error {
	return s.store.SoftDeleteSource(ctx, id)
}

func (s *Service) ListSub2APIGroups(ctx context.Context, sourceID uint64) ([]*Sub2APIGroup, error) {
	src, err := s.requireEnabledSource(ctx, sourceID, SourceTypeSub2API)
	if err != nil {
		return nil, err
	}
	headers, err := s.sub2apiHeaders(ctx, src)
	if err != nil {
		return nil, err
	}
	rawURL := buildURL(src.BaseURL, "/api/v1/admin/groups")
	u, _ := url.Parse(rawURL)
	q := u.Query()
	q.Set("page", "1")
	q.Set("page_size", "200")
	u.RawQuery = q.Encode()
	payload, err := doJSON(ctx, s.hc, http.MethodGet, u.String(), headers, nil)
	if err != nil {
		return nil, err
	}
	rows := mapSlice(envelopeData(payload))
	if len(rows) == 0 {
		if data := nestedMap(payload, "data"); data != nil {
			rows = mapSlice(data["items"])
			if len(rows) == 0 {
				rows = mapSlice(data["list"])
			}
		}
	}
	out := make([]*Sub2APIGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, &Sub2APIGroup{
			ID:                 strings.TrimSpace(asString(row["id"])),
			Name:               strings.TrimSpace(asString(row["name"])),
			Description:        strings.TrimSpace(asString(row["description"])),
			Platform:           strings.TrimSpace(asString(row["platform"])),
			Status:             strings.TrimSpace(asString(row["status"])),
			AccountCount:       asInt(row["account_count"]),
			ActiveAccountCount: asInt(row["active_account_count"]),
		})
	}
	return out, nil
}

func (s *Service) ListSub2APIAccounts(ctx context.Context, sourceID uint64) ([]*Sub2APIAccount, error) {
	src, err := s.requireEnabledSource(ctx, sourceID, SourceTypeSub2API)
	if err != nil {
		return nil, err
	}
	headers, err := s.sub2apiHeaders(ctx, src)
	if err != nil {
		return nil, err
	}
	rawURL := buildURL(src.BaseURL, "/api/v1/admin/accounts")
	u, _ := url.Parse(rawURL)
	q := u.Query()
	q.Set("platform", "openai")
	q.Set("type", "oauth")
	q.Set("page", "1")
	q.Set("page_size", "200")
	if src.GroupID != "" {
		q.Set("group", src.GroupID)
	}
	u.RawQuery = q.Encode()
	payload, err := doJSON(ctx, s.hc, http.MethodGet, u.String(), headers, nil)
	if err != nil {
		return nil, err
	}
	rows := extractRowList(payload)
	out := make([]*Sub2APIAccount, 0, len(rows))
	for _, row := range rows {
		email := firstNonEmpty(
			strings.TrimSpace(asString(row["email"])),
			strings.TrimSpace(asString(mapValue(nestedMap(row, "extra"), "email"))),
			emailFromName(strings.TrimSpace(asString(row["name"]))),
		)
		out = append(out, &Sub2APIAccount{
			ID:              strings.TrimSpace(asString(row["id"])),
			Name:            strings.TrimSpace(asString(row["name"])),
			Email:           email,
			PlanType:        strings.TrimSpace(asString(firstNonEmptyAny(row["plan_type"], row["plan"]))),
			Status:          strings.TrimSpace(asString(row["status"])),
			ExpiresAt:       strings.TrimSpace(asString(firstNonEmptyAny(row["expires_at"], row["expired_at"]))),
			HasRefreshToken: hasRefreshToken(row),
		})
	}
	return out, nil
}

func (s *Service) ListCPAFiles(ctx context.Context, sourceID uint64) ([]*CPAFile, error) {
	src, err := s.requireEnabledSource(ctx, sourceID, SourceTypeCPA)
	if err != nil {
		return nil, err
	}
	headers, err := s.cpaHeaders(src)
	if err != nil {
		return nil, err
	}
	payload, err := doJSON(ctx, s.hc, http.MethodGet, buildURL(src.BaseURL, "/v0/management/auth-files"), headers, nil)
	if err != nil {
		return nil, err
	}
	data := envelopeData(payload)
	rows := mapSlice(data)
	files := []string(nil)
	if len(rows) == 0 {
		files = stringSlice(data)
	}
	if len(rows) == 0 && len(files) == 0 {
		if mapped, ok := data.(map[string]any); ok {
			rows = mapSlice(mapped["files"])
			if len(rows) == 0 {
				files = stringSlice(mapped["files"])
			}
		}
	}
	out := make([]*CPAFile, 0, len(rows)+len(files))
	for _, name := range files {
		out = append(out, &CPAFile{Name: name})
	}
	for _, row := range rows {
		out = append(out, &CPAFile{
			Name:  strings.TrimSpace(asString(row["name"])),
			Email: firstNonEmpty(strings.TrimSpace(asString(row["email"])), strings.TrimSpace(asString(row["account"]))),
		})
	}
	return out, nil
}

func (s *Service) ImportSelected(ctx context.Context, sourceID uint64, in ImportSelectedInput) (*ImportSummary, error) {
	if s.importer == nil {
		return nil, errors.New("accountsource: importer is not configured")
	}
	src, err := s.requireEnabledSource(ctx, sourceID, "")
	if err != nil {
		return nil, err
	}
	var candidates []importcore.ImportCandidate
	switch src.SourceType {
	case SourceTypeSub2API:
		candidates, err = s.importFromSub2API(ctx, src, in.AccountIDs)
	case SourceTypeCPA:
		names := in.FileNames
		if len(names) == 0 {
			names = in.AccountIDs
		}
		candidates, err = s.importFromCPA(ctx, src, names)
	default:
		err = badRequestErr(fmt.Errorf("unsupported source_type: %s", src.SourceType))
	}
	if err != nil {
		return nil, err
	}
	result, err := s.importer.Import(ctx, candidates, mergeImportOptions(src, in))
	if err != nil {
		return nil, err
	}
	return mapImportSummary(result), nil
}

func (s *Service) buildNewSource(in CreateInput) (*Source, error) {
	src := &Source{
		SourceType:     normalizeSourceType(in.SourceType),
		Name:           strings.TrimSpace(in.Name),
		BaseURL:        normalizeBaseURL(in.BaseURL),
		Enabled:        true,
		AuthMode:       normalizeAuthMode(in.AuthMode),
		Email:          strings.TrimSpace(in.Email),
		GroupID:        strings.TrimSpace(in.GroupID),
		DefaultProxyID: in.DefaultProxyID,
		TargetPoolID:   in.TargetPoolID,
	}
	if in.Enabled != nil {
		src.Enabled = *in.Enabled
	}
	if err := validateSource(src); err != nil {
		return nil, badRequestErr(err)
	}
	if err := s.applySecrets(src, in.APIKey, in.Password, in.SecretKey, false); err != nil {
		return nil, badRequestErr(err)
	}
	return src, nil
}

func (s *Service) applyUpdate(src *Source, in UpdateInput) error {
	if in.Name != nil {
		if name := strings.TrimSpace(*in.Name); name != "" {
			src.Name = name
		}
	}
	if in.BaseURL != nil {
		if baseURL := normalizeBaseURL(*in.BaseURL); baseURL != "" {
			src.BaseURL = baseURL
		}
	}
	if in.Enabled != nil {
		src.Enabled = *in.Enabled
	}
	if in.AuthMode != nil {
		if authMode := normalizeAuthMode(*in.AuthMode); authMode != "" {
			src.AuthMode = authMode
		}
	}
	if in.Email != nil {
		src.Email = strings.TrimSpace(*in.Email)
	}
	if in.GroupID != nil {
		src.GroupID = strings.TrimSpace(*in.GroupID)
	}
	if in.DefaultProxyID != nil {
		src.DefaultProxyID = *in.DefaultProxyID
	}
	if in.TargetPoolID != nil {
		src.TargetPoolID = *in.TargetPoolID
	}
	if err := validateSource(src); err != nil {
		return badRequestErr(err)
	}
	return s.applySecrets(
		src,
		stringValue(in.APIKey),
		stringValue(in.Password),
		stringValue(in.SecretKey),
		true,
	)
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func badRequestErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", ErrBadRequest, err)
}

func (s *Service) applySecrets(src *Source, apiKey, password, secretKey string, keepExisting bool) error {
	var err error
	if apiKey = strings.TrimSpace(apiKey); apiKey != "" {
		src.APIKeyEnc, err = s.cipher.EncryptString(apiKey)
		if err != nil {
			return err
		}
	} else if !keepExisting {
		src.APIKeyEnc = ""
	}
	if password = strings.TrimSpace(password); password != "" {
		src.PasswordEnc, err = s.cipher.EncryptString(password)
		if err != nil {
			return err
		}
	} else if !keepExisting {
		src.PasswordEnc = ""
	}
	if secretKey = strings.TrimSpace(secretKey); secretKey != "" {
		src.SecretKeyEnc, err = s.cipher.EncryptString(secretKey)
		if err != nil {
			return err
		}
	} else if !keepExisting {
		src.SecretKeyEnc = ""
	}
	return nil
}

func (s *Service) requireEnabledSource(ctx context.Context, sourceID uint64, wantType string) (*Source, error) {
	src, err := s.store.GetSourceByID(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	if !src.Enabled {
		return nil, badRequestErr(errors.New("source is disabled"))
	}
	if wantType != "" && src.SourceType != wantType {
		return nil, badRequestErr(fmt.Errorf("source_type mismatch: %s", src.SourceType))
	}
	return src, nil
}

func (s *Service) sub2apiHeaders(ctx context.Context, src *Source) (map[string]string, error) {
	switch src.AuthMode {
	case AuthModeAPIKey:
		token, err := s.decrypt(src.APIKeyEnc)
		if err != nil {
			return nil, err
		}
		return map[string]string{"X-API-Key": token}, nil
	case AuthModePassword:
		password, err := s.decrypt(src.PasswordEnc)
		if err != nil {
			return nil, err
		}
		payload, err := doJSON(ctx, s.hc, http.MethodPost, buildURL(src.BaseURL, "/api/v1/auth/login"), nil, map[string]string{
			"email":    src.Email,
			"password": password,
		})
		if err != nil {
			return nil, err
		}
		token := strings.TrimSpace(asString(firstNonEmptyAny(
			mapValue(nestedMap(payload, "data"), "access_token"),
			payload["access_token"],
		)))
		if token == "" {
			return nil, errors.New("sub2api login returned empty access_token")
		}
		return map[string]string{"Authorization": "Bearer " + token}, nil
	default:
		return nil, badRequestErr(fmt.Errorf("unsupported sub2api auth_mode: %s", src.AuthMode))
	}
}

func (s *Service) cpaHeaders(src *Source) (map[string]string, error) {
	switch src.AuthMode {
	case AuthModeAPIKey:
		key, err := s.decrypt(src.APIKeyEnc)
		if err != nil {
			return nil, err
		}
		return map[string]string{"X-API-Key": key}, nil
	case AuthModeBearer:
		key, err := s.decrypt(src.SecretKeyEnc)
		if err != nil {
			return nil, err
		}
		return map[string]string{"Authorization": "Bearer " + key}, nil
	default:
		return nil, badRequestErr(fmt.Errorf("unsupported cpa auth_mode: %s", src.AuthMode))
	}
}

func (s *Service) importFromSub2API(ctx context.Context, src *Source, accountIDs []string) ([]importcore.ImportCandidate, error) {
	if len(accountIDs) == 0 {
		return nil, badRequestErr(errors.New("account_ids 不能为空"))
	}
	headers, err := s.sub2apiHeaders(ctx, src)
	if err != nil {
		return nil, err
	}
	out := make([]importcore.ImportCandidate, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		accountID = strings.TrimSpace(accountID)
		if accountID == "" {
			continue
		}
		payload, err := doJSON(ctx, s.hc, http.MethodGet, buildURL(src.BaseURL, "/api/v1/admin/accounts", accountID), headers, nil)
		if err != nil {
			return nil, err
		}
		row := nestedMap(payload, "data")
		if row == nil {
			row = payload
		}
		credentials := nestedMap(row, "credentials")
		candidate := importcore.ImportCandidate{
			SourceType:       "sub2api_remote",
			SourceRef:        "remote:" + accountID,
			AccessToken:      strings.TrimSpace(asString(credentials["access_token"])),
			RefreshToken:     strings.TrimSpace(asString(credentials["refresh_token"])),
			SessionToken:     strings.TrimSpace(asString(credentials["session_token"])),
			Email:            firstNonEmpty(strings.TrimSpace(asString(row["email"])), strings.TrimSpace(asString(mapValue(nestedMap(row, "extra"), "email"))), emailFromName(strings.TrimSpace(asString(row["name"])))),
			ClientID:         strings.TrimSpace(asString(credentials["client_id"])),
			ChatGPTAccountID: strings.TrimSpace(asString(firstNonEmptyAny(credentials["chatgpt_account_id"], credentials["account_id"]))),
			AccountType:      normalizeAccountType(strings.TrimSpace(asString(row["name"])), strings.TrimSpace(asString(row["platform"])), strings.TrimSpace(asString(row["type"]))),
			PlanType:         strings.TrimSpace(asString(firstNonEmptyAny(row["plan_type"], row["plan"]))),
			Notes:            strings.TrimSpace(asString(row["name"])),
		}
		if candidate.AccessToken == "" {
			return nil, badRequestErr(fmt.Errorf("remote account %s missing access_token", accountID))
		}
		out = append(out, candidate)
	}
	return out, nil
}

func (s *Service) importFromCPA(ctx context.Context, src *Source, names []string) ([]importcore.ImportCandidate, error) {
	if len(names) == 0 {
		return nil, badRequestErr(errors.New("file_names 不能为空"))
	}
	headers, err := s.cpaHeaders(src)
	if err != nil {
		return nil, err
	}
	out := make([]importcore.ImportCandidate, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		u, _ := url.Parse(buildURL(src.BaseURL, "/v0/management/auth-files/download"))
		q := u.Query()
		q.Set("name", name)
		u.RawQuery = q.Encode()
		payload, err := doJSON(ctx, s.hc, http.MethodGet, u.String(), headers, nil)
		if err != nil {
			return nil, err
		}
		data := nestedMap(payload, "data")
		if data == nil {
			data = payload
		}
		candidate := importcore.ImportCandidate{
			SourceType:  "cpa_remote",
			SourceRef:   "file:" + name,
			AccessToken: strings.TrimSpace(asString(data["access_token"])),
			Email:       firstNonEmpty(strings.TrimSpace(asString(data["email"])), strings.TrimSpace(asString(data["account"]))),
		}
		if candidate.AccessToken == "" {
			return nil, badRequestErr(fmt.Errorf("remote cpa file %s missing access_token", name))
		}
		out = append(out, candidate)
	}
	return out, nil
}

func mergeImportOptions(src *Source, in ImportSelectedInput) importcore.ImportOptions {
	opt := importcore.DefaultOptions()
	opt.SkipExpiredATOnly = true
	opt.UpdateExisting = true
	if in.UpdateExisting != nil {
		opt.UpdateExisting = *in.UpdateExisting
	}
	if in.DefaultProxyID != nil {
		opt.DefaultProxyID = *in.DefaultProxyID
	} else {
		opt.DefaultProxyID = src.DefaultProxyID
	}
	if in.TargetPoolID != nil {
		opt.TargetPoolID = *in.TargetPoolID
	} else {
		opt.TargetPoolID = src.TargetPoolID
	}
	if in.ResolveIdentity != nil {
		opt.ResolveIdentity = *in.ResolveIdentity
	}
	if in.KickRefresh != nil {
		opt.KickRefresh = *in.KickRefresh
	} else {
		opt.KickRefresh = true
	}
	if in.KickQuotaProbe != nil {
		opt.KickQuotaProbe = *in.KickQuotaProbe
	} else {
		opt.KickQuotaProbe = true
	}
	return opt
}

func mapImportSummary(result *importcore.ImportResult) *ImportSummary {
	if result == nil {
		return &ImportSummary{}
	}
	out := &ImportSummary{
		Total:   result.Total,
		Created: result.Created,
		Updated: result.Updated,
		Skipped: result.Skipped,
		Failed:  result.Failed,
		Results: make([]ImportSummaryResultRow, 0, len(result.Results)),
	}
	for i, row := range result.Results {
		item := ImportSummaryResultRow{
			Index:     i + 1,
			Email:     row.Email,
			Status:    row.Status,
			Reason:    row.Reason,
			ID:        row.ID,
			SourceRef: row.Source,
		}
		if strings.HasPrefix(row.Source, "remote:") {
			item.SourceType = "sub2api_remote"
		} else if strings.HasPrefix(row.Source, "file:") {
			item.SourceType = "cpa_remote"
		}
		if row.Warning != "" {
			item.Warnings = []string{row.Warning}
		}
		out.Results = append(out.Results, item)
	}
	return out
}

func toView(src *Source) *SourceView {
	if src == nil {
		return nil
	}
	return &SourceView{
		ID:             src.ID,
		SourceType:     src.SourceType,
		Name:           src.Name,
		BaseURL:        src.BaseURL,
		Enabled:        src.Enabled,
		AuthMode:       src.AuthMode,
		Email:          src.Email,
		GroupID:        src.GroupID,
		DefaultProxyID: src.DefaultProxyID,
		TargetPoolID:   src.TargetPoolID,
		HasAPIKey:      src.APIKeyEnc != "",
		HasPassword:    src.PasswordEnc != "",
		HasSecretKey:   src.SecretKeyEnc != "",
		CreatedAt:      src.CreatedAt,
		UpdatedAt:      src.UpdatedAt,
		DeletedAt:      src.DeletedAt,
	}
}

func validateSource(src *Source) error {
	if src == nil {
		return errors.New("source 不能为空")
	}
	if src.SourceType != SourceTypeSub2API && src.SourceType != SourceTypeCPA {
		return errors.New("source_type 仅支持 sub2api / cpa")
	}
	if src.Name == "" {
		return errors.New("name 不能为空")
	}
	if src.BaseURL == "" {
		return errors.New("base_url 不能为空")
	}
	switch src.SourceType {
	case SourceTypeSub2API:
		if src.AuthMode != AuthModePassword && src.AuthMode != AuthModeAPIKey {
			return errors.New("sub2api auth_mode 仅支持 password / api_key")
		}
		if src.AuthMode == AuthModePassword && src.Email == "" {
			return errors.New("sub2api password 模式需要 email")
		}
	case SourceTypeCPA:
		if src.AuthMode != AuthModeBearer && src.AuthMode != AuthModeAPIKey {
			return errors.New("cpa auth_mode 仅支持 bearer / api_key")
		}
	}
	return nil
}

func normalizeSourceType(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
func normalizeAuthMode(value string) string   { return strings.ToLower(strings.TrimSpace(value)) }

func normalizeBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	return strings.TrimRight(trimmed, "/")
}

func (s *Service) decrypt(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	if s.cipher == nil {
		return "", errors.New("accountsource: cipher is not configured")
	}
	return s.cipher.DecryptString(value)
}

func extractRowList(payload map[string]any) []map[string]any {
	data := envelopeData(payload)
	rows := mapSlice(data)
	if len(rows) > 0 {
		return rows
	}
	if mapped, ok := data.(map[string]any); ok {
		for _, key := range []string{"items", "list", "data"} {
			if rows := mapSlice(mapped[key]); len(rows) > 0 {
				return rows
			}
		}
	}
	for _, key := range []string{"items", "list", "data"} {
		if rows := mapSlice(payload[key]); len(rows) > 0 {
			return rows
		}
	}
	return nil
}

func mapValue(mapped map[string]any, key string) any {
	if mapped == nil {
		return nil
	}
	return mapped[key]
}

func hasRefreshToken(row map[string]any) bool {
	credentials := nestedMap(row, "credentials")
	if strings.TrimSpace(asString(credentials["refresh_token"])) != "" {
		return true
	}
	switch raw := row["has_refresh_token"].(type) {
	case bool:
		return raw
	default:
		return false
	}
}

func normalizeAccountType(name, platform, declaredType string) string {
	if t := strings.ToLower(strings.TrimSpace(declaredType)); t != "" && t != "oauth" {
		return t
	}
	lower := strings.ToLower(name + " " + platform)
	switch {
	case strings.Contains(lower, "chatgpt"):
		return "chatgpt"
	case strings.Contains(lower, "openai"), strings.Contains(lower, "codex"):
		return "codex"
	default:
		return "codex"
	}
}

func emailFromName(name string) string {
	if name == "" {
		return ""
	}
	for _, prefix := range []string{"codex-", "chatgpt-", "openai-"} {
		if strings.HasPrefix(name, prefix) {
			name = strings.TrimPrefix(name, prefix)
			break
		}
	}
	idx := strings.LastIndex(name, "_")
	if idx <= 0 || idx >= len(name)-1 {
		return ""
	}
	return name[:idx] + "@" + name[idx+1:]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyAny(values ...any) any {
	for _, value := range values {
		if strings.TrimSpace(asString(value)) != "" {
			return value
		}
	}
	return nil
}

func asInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}
