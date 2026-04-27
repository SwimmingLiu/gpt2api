package settings

import (
	"strings"
	"testing"
)

func TestDefsDoNotExposeLegacyBillingOrAPIKeySettings(t *testing.T) {
	for _, def := range Defs {
		if def.Category == "billing" {
			t.Fatalf("unexpected legacy billing setting in defs: %s", def.Key)
		}
		if strings.HasPrefix(def.Key, "billing.") {
			t.Fatalf("unexpected billing key in defs: %s", def.Key)
		}
		if strings.HasPrefix(def.Key, "recharge.") {
			t.Fatalf("unexpected recharge key in defs: %s", def.Key)
		}
		if strings.HasPrefix(def.Key, "key.") {
			t.Fatalf("unexpected legacy api key setting in defs: %s", def.Key)
		}
	}
}
