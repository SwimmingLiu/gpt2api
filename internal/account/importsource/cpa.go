package importsource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/432539/gpt2api/internal/account/importcore"
)

func ParseCPAFile(name string, raw []byte) ([]importcore.ImportCandidate, error) {
	var obj map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(raw), &obj); err != nil {
		return nil, err
	}
	token, _ := obj["access_token"].(string)
	if token == "" {
		token, _ = obj["accessToken"].(string)
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("%s missing access token", name)
	}

	candidate := importcore.ImportCandidate{
		SourceType:  "cpa_file",
		SourceRef:   "file:" + name,
		AccessToken: strings.TrimSpace(token),
	}
	if email, _ := obj["email"].(string); strings.TrimSpace(email) != "" {
		candidate.Email = strings.TrimSpace(email)
	}
	if rt, _ := obj["refresh_token"].(string); strings.TrimSpace(rt) != "" {
		candidate.RefreshToken = strings.TrimSpace(rt)
	}
	if rt, _ := obj["refreshToken"].(string); candidate.RefreshToken == "" && strings.TrimSpace(rt) != "" {
		candidate.RefreshToken = strings.TrimSpace(rt)
	}
	if clientID, _ := obj["client_id"].(string); strings.TrimSpace(clientID) != "" {
		candidate.ClientID = strings.TrimSpace(clientID)
	}
	if clientID, _ := obj["clientId"].(string); candidate.ClientID == "" && strings.TrimSpace(clientID) != "" {
		candidate.ClientID = strings.TrimSpace(clientID)
	}
	return []importcore.ImportCandidate{candidate}, nil
}
