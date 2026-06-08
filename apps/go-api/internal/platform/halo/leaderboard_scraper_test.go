package halo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// buildLeaderboardHTML fabrique une page Halo Waypoint synthétique avec un bloc
// __NEXT_DATA__ contenant `gamertags` (rank=0 dans le payload, comme en prod).
func buildLeaderboardHTML(pageSize int, gamertags []string, baseScore int) []byte {
	entries := make([]map[string]any, 0, len(gamertags))
	for i, gt := range gamertags {
		entries = append(entries, map[string]any{
			"player": map[string]any{"xuid": "x" + gt, "gamertag": gt},
			"rank":   0,
			"score":  baseScore - i,
		})
	}
	payload := map[string]any{
		"props": map[string]any{
			"pageProps": map[string]any{
				"pageSize":    pageSize,
				"leaderboard": entries,
			},
		},
	}
	b, _ := json.Marshal(payload)
	return []byte(`<html><body><script id="__NEXT_DATA__" type="application/json">` +
		string(b) + `</script></body></html>`)
}

// TestFetchCSRLeaderboard_PaginationAndRank valide la pagination, le calcul du
// rang inter-pages (le payload met rank=0) et l'arrêt sur page incomplète.
func TestFetchCSRLeaderboard_PaginationAndRank(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("page") {
		case "1":
			_, _ = w.Write(buildLeaderboardHTML(2, []string{"A", "B"}, 2000))
		case "2":
			_, _ = w.Write(buildLeaderboardHTML(2, []string{"C", "D"}, 1800))
		case "3":
			_, _ = w.Write(buildLeaderboardHTML(2, []string{"E"}, 1600)) // page courte → stop
		default:
			_, _ = w.Write(buildLeaderboardHTML(2, nil, 0))
		}
	}))
	defer srv.Close()

	scraper := NewLeaderboardScraper(0)
	scraper.host = srv.URL

	entries, err := scraper.FetchCSRLeaderboard(context.Background(), "csrseason13-2", "p", 0)
	if err != nil {
		t.Fatalf("FetchCSRLeaderboard: %v", err)
	}
	wantGT := []string{"A", "B", "C", "D", "E"}
	if len(entries) != len(wantGT) {
		t.Fatalf("len = %d, want %d", len(entries), len(wantGT))
	}
	for i, e := range entries {
		if e.Gamertag != wantGT[i] {
			t.Errorf("entry %d gamertag = %q, want %q", i, e.Gamertag, wantGT[i])
		}
		if e.Rank != i+1 {
			t.Errorf("entry %d rank = %d, want %d (rang inter-pages)", i, e.Rank, i+1)
		}
		if e.XUID == "" || e.CSRValue == 0 {
			t.Errorf("entry %d incomplète: %+v", i, e)
		}
	}

	// Limite : coupe à 3 entrées (A, B, C).
	limited, err := scraper.FetchCSRLeaderboard(context.Background(), "csrseason13-2", "p", 3)
	if err != nil {
		t.Fatalf("FetchCSRLeaderboard limit: %v", err)
	}
	if len(limited) != 3 || limited[2].Gamertag != "C" || limited[2].Rank != 3 {
		t.Fatalf("limit=3 incorrect: %+v", limited)
	}
}

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
