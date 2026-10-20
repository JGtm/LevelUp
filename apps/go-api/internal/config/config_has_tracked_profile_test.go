// Package config — config_has_tracked_profile_test.go : porte « profil suivi »
// (ADR 0035 D3). Recherche par xuid, filtre SyncablePlayers.
package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTrackedProfiles écrit un db_profiles.json v3 portant quatre profils :
// un normal, un auth_only, un en pause et un sur un autre titre.
func writeTrackedProfiles(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "db_profiles.json")
	const content = `{
  "version": "3.0",
  "admin": "JoueurNormal",
  "profiles": {
    "halo_infinite": {
      "JoueurNormal": {"xuid": "1000000000000001", "gamertag": "JoueurNormal"},
      "CompteAuthSeule": {"xuid": "1000000000000002", "gamertag": "CompteAuthSeule", "auth_only": true},
      "JoueurEnPause": {"xuid": "1000000000000003", "gamertag": "JoueurEnPause", "sync_enabled": false}
    },
    "halo_5": {
      "JoueurAutreTitre": {"xuid": "1000000000000004", "gamertag": "JoueurAutreTitre"}
    }
  }
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestHasTrackedProfile(t *testing.T) {
	cfg := &AppConfig{DBProfilesPath: writeTrackedProfiles(t)}

	cases := []struct {
		name      string
		titleSlug string
		xuid      string
		want      bool
	}{
		{"profil normal", "halo_infinite", "1000000000000001", true},
		{"profil auth_only exclu", "halo_infinite", "1000000000000002", false},
		{"profil en pause exclu", "halo_infinite", "1000000000000003", false},
		{"xuid inconnu", "halo_infinite", "9999999999999999", false},
		{"xuid vide", "halo_infinite", "", false},
		{"profil d'un autre titre", "halo_infinite", "1000000000000004", false},
		{"le même profil sur son titre", "halo_5", "1000000000000004", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := cfg.HasTrackedProfile(tc.titleSlug, tc.xuid)
			if err != nil {
				t.Fatalf("HasTrackedProfile: %v", err)
			}
			if got != tc.want {
				t.Fatalf("HasTrackedProfile(%q, %q) = %v, want %v", tc.titleSlug, tc.xuid, got, tc.want)
			}
		})
	}
}

// TestHasTrackedProfile_GamertagNeverMatches — la recherche est par xuid :
// passer un gamertag (même exact) ne doit JAMAIS ouvrir la porte.
func TestHasTrackedProfile_GamertagNeverMatches(t *testing.T) {
	cfg := &AppConfig{DBProfilesPath: writeTrackedProfiles(t)}

	got, err := cfg.HasTrackedProfile("halo_infinite", "JoueurNormal")
	if err != nil {
		t.Fatalf("HasTrackedProfile: %v", err)
	}
	if got {
		t.Fatal("un gamertag ne doit pas être accepté comme clé d'identité (ADR 0035 D1)")
	}
}

// TestHasTrackedProfile_FichierAbsent — pas de profils = personne n'est suivi,
// et ce n'est pas une erreur (instance neuve).
func TestHasTrackedProfile_FichierAbsent(t *testing.T) {
	cfg := &AppConfig{DBProfilesPath: filepath.Join(t.TempDir(), "absent.json")}

	got, err := cfg.HasTrackedProfile("halo_infinite", "1000000000000001")
	if err != nil {
		t.Fatalf("fichier absent ne doit pas être une erreur : %v", err)
	}
	if got {
		t.Fatal("aucun profil déclaré ⇒ aucun joueur suivi")
	}
}

// TestHasTrackedProfile_FichierIllisible — l'erreur REMONTE (le caller décide
// de sa dégradation ; les portes ADR 0035 refusent en journalisant).
func TestHasTrackedProfile_FichierIllisible(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db_profiles.json")
	if err := os.WriteFile(path, []byte("{ ceci n'est pas du JSON"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &AppConfig{DBProfilesPath: path}

	got, err := cfg.HasTrackedProfile("halo_infinite", "1000000000000001")
	if err == nil {
		t.Fatal("un db_profiles.json illisible doit remonter une erreur, pas un false silencieux")
	}
	if got {
		t.Fatal("erreur de lecture ⇒ false")
	}
}
