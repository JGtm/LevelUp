package halo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/observability"
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

// TestFetchCSRLeaderboard_EmptyPage1Counter valide le canari de changement de
// markup : une page 1 vide incrémente le compteur expvar leaderboard_empty_page1.
func TestFetchCSRLeaderboard_EmptyPage1Counter(t *testing.T) {
	before := observability.LoadCounter("leaderboard_empty_page1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(buildLeaderboardHTML(2, nil, 0)) // leaderboard vide
	}))
	defer srv.Close()

	scraper := NewLeaderboardScraper(0)
	scraper.host = srv.URL

	entries, err := scraper.FetchCSRLeaderboard(context.Background(), "csrseason13-2", "p", 0)
	if err != nil {
		t.Fatalf("FetchCSRLeaderboard: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("attendu 0 entrée, got %d", len(entries))
	}
	if got := observability.LoadCounter("leaderboard_empty_page1"); got != before+1 {
		t.Errorf("compteur leaderboard_empty_page1 = %d, attendu %d", got, before+1)
	}
}

// TestFetchCSRLeaderboard_404Sentinel : un 404 (playlist non classée cette saison)
// renvoie une erreur qui matche ErrLeaderboardPageNotFound via errors.Is — l'appelant
// (backfill multi-saisons) skippe alors en silence au lieu de logger une ERROR.
func TestFetchCSRLeaderboard_404Sentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	scraper := NewLeaderboardScraper(0)
	scraper.host = srv.URL

	_, err := scraper.FetchCSRLeaderboard(context.Background(), "csrseason3-1", "absente", 0)
	if err == nil {
		t.Fatal("attendu une erreur sur 404")
	}
	if !errors.Is(err, ErrLeaderboardPageNotFound) {
		t.Errorf("erreur = %v, attendu errors.Is(ErrLeaderboardPageNotFound)", err)
	}
}

// buildCatalogHTML fabrique une page avec menus saisons/playlists (ordre
// décroissant pour les saisons, comme en prod : seasons[0] = active).
func buildCatalogHTML(seasons, playlists [][2]string) []byte {
	toRefs := func(pairs [][2]string, idKey string) []map[string]any {
		out := make([]map[string]any, 0, len(pairs))
		for _, p := range pairs {
			out = append(out, map[string]any{idKey: p[0], "displayName": p[1]})
		}
		return out
	}
	payload := map[string]any{
		"props": map[string]any{
			"pageProps": map[string]any{
				"pageSize":    100,
				"leaderboard": []map[string]any{},
				"seasons":     toRefs(seasons, "seasonId"),
				"playlists":   toRefs(playlists, "playlistId"),
			},
		},
	}
	b, _ := json.Marshal(payload)
	return []byte(`<html><body><script id="__NEXT_DATA__" type="application/json">` +
		string(b) + `</script></body></html>`)
}

// TestFetchActiveSeasonAndCatalog valide la découverte autonome de la saison
// active (seasons[0]) et la remontée des deux menus.
func TestFetchActiveSeasonAndCatalog(t *testing.T) {
	seasons := [][2]string{{"csrseason13-2", "Infinite"}, {"csrseason12-1", "Shadows"}, {"csrseason11-1", "Combined Arms"}}
	playlists := [][2]string{{"pl-arena", "Ranked Arena"}, {"pl-snipers", "Ranked Snipers"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(buildCatalogHTML(seasons, playlists))
	}))
	defer srv.Close()

	scraper := NewLeaderboardScraper(0)
	scraper.host = srv.URL

	active, err := scraper.FetchActiveSeason(context.Background(), "pl-arena")
	if err != nil {
		t.Fatalf("FetchActiveSeason: %v", err)
	}
	if active != "csrseason13-2" {
		t.Errorf("saison active = %q, attendu csrseason13-2 (seasons[0])", active)
	}

	gotSeasons, gotPlaylists, err := scraper.FetchCatalog(context.Background(), "pl-arena")
	if err != nil {
		t.Fatalf("FetchCatalog: %v", err)
	}
	if len(gotSeasons) != 3 || gotSeasons[0].ID != "csrseason13-2" || gotSeasons[0].DisplayName != "Infinite" {
		t.Errorf("seasons incorrect: %+v", gotSeasons)
	}
	if len(gotPlaylists) != 2 || gotPlaylists[1].ID != "pl-snipers" {
		t.Errorf("playlists incorrect: %+v", gotPlaylists)
	}

	// refPlaylistID vide → erreur claire.
	if _, err := scraper.FetchActiveSeason(context.Background(), ""); err == nil {
		t.Error("FetchActiveSeason avec refPlaylistID vide devrait échouer")
	}
}

// TestFetchSeasons_TranslationsFR valide, sur la fixture réelle, que FetchSeasons
// résout le nom FR depuis translations["fr-FR"] (csrseason12-1 → "Ombres") et
// retombe sur le DisplayName EN quand aucune traduction FR n'existe (csrseason13-2
// "Infinite" n'a que des locales qps-ploc).
func TestFetchSeasons_TranslationsFR(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "leaderboard_sample.html"))
	if err != nil {
		t.Skipf("fixture absente (%v)", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	scraper := NewLeaderboardScraper(0)
	scraper.host = srv.URL

	seasons, err := scraper.FetchSeasons(context.Background(), "pl-arena")
	if err != nil {
		t.Fatalf("FetchSeasons: %v", err)
	}
	byID := make(map[string]struct{ en, fr string }, len(seasons))
	for _, s := range seasons {
		byID[s.SeasonID] = struct{ en, fr string }{s.DisplayName, s.NameFR}
	}
	if got := byID["csrseason12-1"]; got.en != "Shadows" || got.fr != "Ombres" {
		t.Errorf("csrseason12-1 = %+v, attendu {Shadows, Ombres}", got)
	}
	if got := byID["csrseason11-1"]; got.fr != "Dernier bastion" {
		t.Errorf("csrseason11-1 FR = %q, attendu \"Dernier bastion\"", got.fr)
	}
	// Pas de fr-FR pour csrseason13-2 → fallback EN.
	if got := byID["csrseason13-2"]; got.en != "Infinite" || got.fr != "Infinite" {
		t.Errorf("csrseason13-2 = %+v, attendu fallback {Infinite, Infinite}", got)
	}
}

// TestFetchActiveSeasonByRedirect_ExtractsSeasonFromLocation valide le repli de
// découverte SANS page-graine : la racine des classements répond 307 et le header
// Location porte la saison active. La redirection ne doit PAS être suivie (une seule
// requête au serveur) — la suivre mènerait à une page playlist par défaut qui peut
// rendre 500, et perdrait l'information cherchée.
func TestFetchActiveSeasonByRedirect_ExtractsSeasonFromLocation(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != "/halo-infinite/leaderboards" {
			t.Errorf("chemin sondé = %q, attendu /halo-infinite/leaderboards (racine sans saison ni playlist)", r.URL.Path)
		}
		w.Header().Set("Location", "/halo-infinite/leaderboards/csrseason13-3")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer srv.Close()

	scraper := NewLeaderboardScraper(0)
	scraper.host = srv.URL

	season, err := scraper.FetchActiveSeasonByRedirect(context.Background())
	if err != nil {
		t.Fatalf("FetchActiveSeasonByRedirect: %v", err)
	}
	if season != "csrseason13-3" {
		t.Errorf("saison = %q, attendu csrseason13-3 (dernier segment du Location)", season)
	}
	if got := hits.Load(); got != 1 {
		t.Errorf("%d requêtes au serveur, attendu 1 (redirection suivie ?)", got)
	}
}

// TestFetchActiveSeasonByRedirect_RejectsInvalidResponses : toute réponse qui ne
// désigne pas une saison donne une erreur EXPLICITE, jamais une saison vide ou un
// segment arbitraire — le cron ne doit pas scraper (ni persister) sous une
// pseudo-saison inventée par une page d'erreur ou une redirection inattendue.
func TestFetchActiveSeasonByRedirect_RejectsInvalidResponses(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"200 sans redirection", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<html></html>"))
		}},
		{"307 sans header Location", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTemporaryRedirect)
		}},
		{"Location sans segment de saison", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "/halo-infinite/leaderboards")
			w.WriteHeader(http.StatusTemporaryRedirect)
		}},
		{"Location vers une autre page", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "/halo-infinite/news")
			w.WriteHeader(http.StatusTemporaryRedirect)
		}},
		{"500 page en erreur", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()

			scraper := NewLeaderboardScraper(0)
			scraper.host = srv.URL

			season, err := scraper.FetchActiveSeasonByRedirect(context.Background())
			if err == nil {
				t.Fatalf("attendu une erreur, got saison %q", season)
			}
			if season != "" {
				t.Errorf("saison = %q sur erreur, attendu \"\"", season)
			}
		})
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
