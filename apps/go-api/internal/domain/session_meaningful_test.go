package domain

import "testing"

// TestSessionData_IsMeaningful vérifie qu'une session anonyme vierge n'est pas
// "significative" (donc non persistée), mais que tout état interactif la rend telle.
func TestSessionData_IsMeaningful(t *testing.T) {
	// Session vierge telle que produite par Store.New (locale fr, hints visibles).
	blank := func() *SessionData {
		return &SessionData{Locale: "fr", HintsVisible: true}
	}

	if blank().IsMeaningful() {
		t.Error("session anonyme vierge ne doit PAS être significative")
	}

	str := "x"
	cases := map[string]func(*SessionData){
		"username":             func(s *SessionData) { s.Username = &str },
		"role":                 func(s *SessionData) { s.Role = &str },
		"oauth_state":          func(s *SessionData) { s.OAuthState = "state" },
		"linked_halo_identity": func(s *SessionData) { s.LinkedHaloIdentity = &HaloIdentity{XUID: "1"} },
		"halo_tokens":          func(s *SessionData) { s.HaloTokens = &HaloTokens{} },
		"current_player_slug":  func(s *SessionData) { s.CurrentPlayerSlug = &str },
		"active_sync_job":      func(s *SessionData) { s.ActiveSyncJobID = &str },
		"auth_ready":           func(s *SessionData) { s.AuthReady = true },
		"current_title":        func(s *SessionData) { s.CurrentTitleSlug = "halo_infinite" },
		"locale_changed":       func(s *SessionData) { s.Locale = "en" },
		"hints_hidden":         func(s *SessionData) { s.HintsVisible = false },
	}
	for name, mutate := range cases {
		s := blank()
		mutate(s)
		if !s.IsMeaningful() {
			t.Errorf("session avec %s doit être significative", name)
		}
	}
}
