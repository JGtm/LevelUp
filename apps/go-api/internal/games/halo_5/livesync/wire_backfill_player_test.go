package livesync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
)

// TestEnrichmentBackfillPlayerFor_FriendsOfThePlayer : le backfill post-score Halo 5 découpe
// les sessions avec la liste d'amis DU joueur (player_friends.json, celle du recalcul admin et
// de la page Sessions), jamais avec l'ensemble des autres profils déclarés.
func TestEnrichmentBackfillPlayerFor_FriendsOfThePlayer(t *testing.T) {
	cfg := writeProfilesV3(t, `{
		"version": "3.0",
		"profiles": {
			"halo_5": {"JGtm": {"xuid": "1"}, "Madina": {"xuid": "2"}},
			"halo_infinite": {"Choco": {"xuid": "3"}}
		}
	}`)
	friendsPath := cfg.RuntimePaths().PlayerFriendsPath()
	if err := os.MkdirAll(filepath.Dir(friendsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(friendsPath, []byte(`{"version":"1.0","friends":{
		"1": {"gamertags": ["AmiA", "AmiB"]},
		"2": {"gamertags": ["AutreAmi"]}
	}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg.AppSettingsPath = filepath.Join(cfg.RepoRoot, "app_settings.json")
	if err := os.WriteFile(cfg.AppSettingsPath, []byte(`{"session_team_change_mode":"group","session_gap_minutes":45}`), 0o600); err != nil {
		t.Fatal(err)
	}

	got := enrichmentBackfillPlayerFor(context.Background(), cfg, "1")

	if got.XUID != "1" {
		t.Errorf("xuid = %q, attendu 1", got.XUID)
	}
	if len(got.FriendGamertags) != 2 || !contains(got.FriendGamertags, "AmiA") || !contains(got.FriendGamertags, "AmiB") {
		t.Errorf("amis = %v, attendu la liste du joueur [AmiA AmiB]", got.FriendGamertags)
	}
	for _, autre := range []string{"Madina", "Choco", "AutreAmi"} {
		if contains(got.FriendGamertags, autre) {
			t.Errorf("%q n'est pas un ami du joueur : %v", autre, got.FriendGamertags)
		}
	}
	if got.SessionOptions.TeamChangeMode != domain.TeamChangeModeGroup || got.SessionOptions.GapMinutes != 45 {
		t.Errorf("règle de session = %+v, attendu le réglage (group, 45 min)", got.SessionOptions)
	}
}

// TestEnrichmentBackfillPlayerFor_NoFriendsFile : sans fichier d'amis, aucun ami ; la règle
// reste posée (jamais la valeur zéro).
func TestEnrichmentBackfillPlayerFor_NoFriendsFile(t *testing.T) {
	cfg := writeProfilesV3(t, `{"version": "3.0", "profiles": {"halo_5": {"JGtm": {"xuid": "1"}}}}`)
	got := enrichmentBackfillPlayerFor(context.Background(), cfg, "1")
	if len(got.FriendGamertags) != 0 {
		t.Errorf("amis = %v, attendu aucun", got.FriendGamertags)
	}
	if got.SessionOptions.GapMinutes <= 0 {
		t.Errorf("règle de session absente : %+v", got.SessionOptions)
	}
}
