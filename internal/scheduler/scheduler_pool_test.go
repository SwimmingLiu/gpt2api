package scheduler

import (
	"testing"

	"github.com/432539/gpt2api/internal/account"
)

func TestFilterByAllowedAccountsKeepsPoolMembersOnly(t *testing.T) {
	candidates := []*account.Account{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}
	allowed := map[uint64]struct{}{
		2: {},
		3: {},
	}

	got := filterByAllowedAccounts(candidates, allowed)

	if len(got) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(got))
	}
	if got[0].ID != 2 || got[1].ID != 3 {
		t.Fatalf("unexpected filtered order: %+v", got)
	}
}
