package config

import "testing"

// TestHasDeclaredProfile — la porte des jetons accepte tout profil déclaré (synchro en pause,
// auth_only, autre titre), contrairement à la porte du suivi live (HasTrackedProfile).
func TestHasDeclaredProfile(t *testing.T) {
	cfg := &AppConfig{DBProfilesPath: writeTrackedProfiles(t)}
	cases := []struct {
		name string
		xuid string
		want bool
	}{
		{"profil normal", "1000000000000001", true},
		{"profil auth_only accepté", "1000000000000002", true},
		{"profil en pause accepté", "1000000000000003", true},
		{"profil d'un autre titre accepté", "1000000000000004", true},
		{"xuid inconnu", "9999999999999999", false},
		{"xuid vide", "", false},
		{"gamertag refusé", "JoueurNormal", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := cfg.HasDeclaredProfile(tc.xuid)
			if err != nil {
				t.Fatalf("HasDeclaredProfile: %v", err)
			}
			if got != tc.want {
				t.Fatalf("HasDeclaredProfile(%q) = %v, want %v", tc.xuid, got, tc.want)
			}
		})
	}
}
