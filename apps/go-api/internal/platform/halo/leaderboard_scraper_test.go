package halo

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseLeaderboardPage valide le parsing du bloc __NEXT_DATA__ sur un
// échantillon réel capturé depuis Halo Waypoint (Ranked Snipers, csrseason13-2).
func TestParseLeaderboardPage(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "leaderboard_sample.html"))
	if err != nil {
		t.Skipf("fixture absente (%v) — la déposer via Invoke-WebRequest", err)
	}
	pp, err := parseLeaderboardPage(body)
	if err != nil {
		t.Fatalf("parseLeaderboardPage: %v", err)
	}
	if pp.PageSize != 100 {
		t.Errorf("PageSize = %d, attendu 100", pp.PageSize)
	}
	if len(pp.Leaderboard) != 100 {
		t.Fatalf("len(Leaderboard) = %d, attendu 100", len(pp.Leaderboard))
	}
	first := pp.Leaderboard[0]
	if first.Player.Gamertag != "Twissted Mindss" {
		t.Errorf("premier gamertag = %q, attendu \"Twissted Mindss\"", first.Player.Gamertag)
	}
	if first.Score != 2180 {
		t.Errorf("premier score = %d, attendu 2180", first.Score)
	}
	if first.Player.XUID == "" {
		t.Error("xuid manquant sur la première entrée")
	}
	// Les scores doivent être décroissants (classement).
	for i := 1; i < len(pp.Leaderboard); i++ {
		if pp.Leaderboard[i].Score > pp.Leaderboard[i-1].Score {
			t.Errorf("scores non décroissants à l'index %d (%d > %d)",
				i, pp.Leaderboard[i].Score, pp.Leaderboard[i-1].Score)
			break
		}
	}
	// Les listes de sélecteurs doivent être présentes.
	if len(pp.Seasons) == 0 {
		t.Error("aucune saison dans pageProps.seasons")
	}
	if len(pp.Playlists) == 0 {
		t.Error("aucune playlist dans pageProps.playlists")
	}
}
