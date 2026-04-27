package importsource

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/432539/gpt2api/internal/account/importcore"
)

func ParseCPAFile(name string, raw []byte) ([]importcore.ImportCandidate, error) {
	return parseTokenJSONCandidates(name, raw, tokenJSONOptions{
		sourceType:   "cpa_file",
		sourceRef:    "file:" + name,
		requireEmail: false,
	})
}

type tokenJSONOptions struct {
	sourceType   string
	sourceRef    string
	requireEmail bool
}

type tokenJSONFile struct {
	AccessToken   string `json:"access_token"`
	AccessToken2  string `json:"accessToken"`
	RefreshToken  string `json:"refresh_token"`
	RefreshToken2 string `json:"refreshToken"`
	AccountID     string `json:"account_id"`
	Email         string `json:"email"`
	Type          string `json:"type"`
	ClientID      string `json:"client_id"`
	ClientID2     string `json:"clientId"`
	Expired       string `json:"expired"`
	Expires       string `json:"expires"`
}

func parseTokenJSONCandidates(name string, raw []byte, opts tokenJSONOptions) ([]importcore.ImportCandidate, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimSpace(raw), &obj); err != nil {
		return nil, err
	}

	var file tokenJSONFile
	if err := json.Unmarshal(bytes.TrimSpace(raw), &file); err != nil {
		return nil, err
	}

	accessToken := strings.TrimSpace(file.AccessToken)
	if accessToken == "" {
		accessToken = strings.TrimSpace(file.AccessToken2)
	}
	if accessToken == "" {
		return nil, fmt.Errorf("%s missing access token", name)
	}

	email := strings.TrimSpace(file.Email)
	if opts.requireEmail && email == "" {
		return nil, fmt.Errorf("%s missing email", name)
	}

	refreshToken := strings.TrimSpace(file.RefreshToken)
	if refreshToken == "" {
		refreshToken = strings.TrimSpace(file.RefreshToken2)
	}

	clientID := strings.TrimSpace(file.ClientID)
	if clientID == "" {
		clientID = strings.TrimSpace(file.ClientID2)
	}

	candidate := importcore.ImportCandidate{
		SourceType:       opts.sourceType,
		SourceRef:        opts.sourceRef,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		Email:            email,
		ClientID:         clientID,
		ChatGPTAccountID: strings.TrimSpace(file.AccountID),
		AccountType:      strings.ToLower(strings.TrimSpace(file.Type)),
	}
	if candidate.AccountType == "" {
		candidate.AccountType = "codex"
	}
	if ts := firstNonEmpty(strings.TrimSpace(file.Expired), strings.TrimSpace(file.Expires)); ts != "" {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			candidate.TokenExpiresAt = &t
		}
	}

	if !tokenJSONObjectHasAccessToken(obj) {
		return nil, errors.New("missing access token")
	}
	return []importcore.ImportCandidate{candidate}, nil
}

func tokenJSONObjectHasAccessToken(obj map[string]json.RawMessage) bool {
	_, okSnake := obj["access_token"]
	_, okCamel := obj["accessToken"]
	return okSnake || okCamel
}
