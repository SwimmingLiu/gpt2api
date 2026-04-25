package importcore

import "time"

func ClassifyCredentialState(candidate ImportCandidate, now time.Time, refreshAheadSec int) CredentialState {
	hasRT := candidate.RefreshToken != ""
	hasST := candidate.SessionToken != ""
	refreshable := hasRT || hasST

	state := CredentialState{Capability: "at_only", Lifecycle: "unknown"}
	switch {
	case hasRT && hasST:
		state.Capability = "refreshable_full"
	case hasRT:
		state.Capability = "refreshable_rt"
	case hasST:
		state.Capability = "refreshable_st"
	}

	nowUTC := now.UTC()
	if candidate.TokenExpiresAt == nil {
		if refreshable {
			state.Warning = "token_expiry_unknown"
		}
		return state
	}

	exp := candidate.TokenExpiresAt.UTC()
	if !exp.After(nowUTC) {
		state.Lifecycle = "expired"
		if !refreshable {
			state.SkipImport = true
			return state
		}
		state.Warning = "access_token_expired_but_refreshable"
		return state
	}

	if exp.Before(nowUTC.Add(time.Duration(refreshAheadSec) * time.Second)) {
		state.Lifecycle = "expiring_soon"
		if refreshable {
			state.Warning = "access_token_expiring_soon"
		} else {
			state.Warning = "access_token_expiring_soon_unrefreshable"
		}
		return state
	}

	state.Lifecycle = "active"
	return state
}
