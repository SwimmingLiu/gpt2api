package importsource

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/432539/gpt2api/internal/account/importcore"
)

func ParseSub2APIJSON(raw []byte) ([]importcore.ImportCandidate, error) {
	var payload struct {
		Accounts []sub2apiAccount `json:"accounts"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(raw), &payload); err != nil {
		return nil, err
	}
	out := make([]importcore.ImportCandidate, 0, len(payload.Accounts))
	for i, account := range payload.Accounts {
		candidate, ok := account.toCandidate()
		if !ok {
			continue
		}
		if candidate.SourceRef == "" {
			candidate.SourceRef = fmt.Sprintf("accounts[%d]", i)
		}
		out = append(out, candidate)
	}
	if len(out) == 0 {
		return nil, errors.New("sub2api json missing importable accounts")
	}
	return out, nil
}

func ParseAutoJSON(raw []byte) ([]importcore.ImportCandidate, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, errors.New("empty input")
	}
	if candidates, err := parseAutoJSONSingle(trimmed); err == nil && len(candidates) > 0 {
		return candidates, nil
	}

	dec := json.NewDecoder(bytes.NewReader(trimmed))
	var all []importcore.ImportCandidate
	var firstErr error
	for {
		var one json.RawMessage
		if err := dec.Decode(&one); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if firstErr == nil {
				firstErr = err
			}
			break
		}
		candidates, err := parseAutoJSONSingle(bytes.TrimSpace(one))
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		all = append(all, candidates...)
	}
	if len(all) > 0 {
		return all, nil
	}
	if firstErr == nil {
		firstErr = errors.New("unrecognized json format")
	}
	return nil, firstErr
}

func parseAutoJSONSingle(raw []byte) ([]importcore.ImportCandidate, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, errors.New("empty json")
	}
	switch raw[0] {
	case '[':
		var arr []json.RawMessage
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil, fmt.Errorf("parse json array: %w", err)
		}
		out := make([]importcore.ImportCandidate, 0, len(arr))
		for _, item := range arr {
			candidates, err := parseAutoJSONSingle(item)
			if err != nil {
				continue
			}
			out = append(out, candidates...)
		}
		if len(out) == 0 {
			return nil, errors.New("json array has no importable accounts")
		}
		return out, nil
	case '{':
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil, fmt.Errorf("parse json object: %w", err)
		}
		if _, ok := obj["accounts"]; ok {
			return ParseSub2APIJSON(raw)
		}
		if tokenJSONObjectHasAccessToken(obj) {
			candidates, err := parseTokenJSONCandidates("inline", raw, tokenJSONOptions{
				sourceType:   "cpa_file",
				sourceRef:    "inline",
				requireEmail: true,
			})
			if err != nil {
				return nil, err
			}
			return candidates, nil
		}
		return nil, errors.New("unrecognized json object")
	default:
		return nil, errors.New("unrecognized json payload")
	}
}

type sub2apiAccount struct {
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Type        string `json:"type"`
	Credentials struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		SessionToken     string `json:"session_token"`
		ClientID         string `json:"client_id"`
		ChatGPTAccountID string `json:"chatgpt_account_id"`
	} `json:"credentials"`
	Extra struct {
		Email string `json:"email"`
	} `json:"extra"`
}

func (a sub2apiAccount) toCandidate() (importcore.ImportCandidate, bool) {
	email := strings.TrimSpace(a.Extra.Email)
	if email == "" {
		email = emailFromName(a.Name)
	}
	accessToken := strings.TrimSpace(a.Credentials.AccessToken)
	if accessToken == "" || email == "" {
		return importcore.ImportCandidate{}, false
	}
	return importcore.ImportCandidate{
		SourceType:       "sub2api_json",
		SourceRef:        "name:" + strings.TrimSpace(a.Name),
		AccessToken:      accessToken,
		RefreshToken:     strings.TrimSpace(a.Credentials.RefreshToken),
		SessionToken:     strings.TrimSpace(a.Credentials.SessionToken),
		Email:            email,
		ClientID:         strings.TrimSpace(a.Credentials.ClientID),
		ChatGPTAccountID: strings.TrimSpace(a.Credentials.ChatGPTAccountID),
		AccountType:      normalizeType(a.Name, a.Platform, a.Type),
		Notes:            strings.TrimSpace(a.Name),
	}, true
}

func emailFromName(name string) string {
	if name == "" {
		return ""
	}
	n := name
	for _, prefix := range []string{"codex-", "chatgpt-", "openai-"} {
		if strings.HasPrefix(n, prefix) {
			n = strings.TrimPrefix(n, prefix)
			break
		}
	}
	idx := strings.LastIndex(n, "_")
	if idx > 0 && idx < len(n)-1 {
		return n[:idx] + "@" + n[idx+1:]
	}
	return ""
}

func normalizeType(name, platform, declaredType string) string {
	if t := strings.ToLower(strings.TrimSpace(declaredType)); t != "" {
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
