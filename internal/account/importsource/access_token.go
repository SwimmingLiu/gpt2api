package importsource

import (
	"fmt"
	"strings"

	"github.com/432539/gpt2api/internal/account/importcore"
)

func ParseAccessTokenText(raw string) []importcore.ImportCandidate {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	out := make([]importcore.ImportCandidate, 0, len(lines))
	for i, line := range lines {
		token := strings.TrimSpace(line)
		if token == "" {
			continue
		}
		out = append(out, importcore.ImportCandidate{
			SourceType:  "access_token_text",
			SourceRef:   fmt.Sprintf("line:%d", i+1),
			AccessToken: token,
		})
	}
	return out
}
