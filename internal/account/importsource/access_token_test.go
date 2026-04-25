package importsource

import "testing"

func TestParseAccessTokenTextHandlesCRLF(t *testing.T) {
	candidates := ParseAccessTokenText("tok-a\r\ntok-b\r\n\r\n")
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].SourceRef != "line:1" || candidates[1].SourceRef != "line:2" {
		t.Fatalf("unexpected source refs: %+v", candidates)
	}
}
