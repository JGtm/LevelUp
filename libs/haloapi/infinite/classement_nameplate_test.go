package infinite

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// TestGetEmblemMapping : chemin de la table, jetons posés, DTO ; statut non-200 et JSON illisible
// en erreur.
func TestGetEmblemMapping(t *testing.T) {
	srv, cap := serveurCorps(t, []byte(`{"104-001-x":{"651339664":{"nameplateCmsPath":"/images/nameplates/104-001-x_651339664.png","emblemCmsPath":"e","textColor":"#fff"}}}`))
	m, err := clientSur(srv).GetEmblemMapping(context.Background(), authTest)
	if err != nil || m["104-001-x"]["651339664"].TextColor != "#fff" {
		t.Fatalf("mapping = %+v, %v", m, err)
	}
	u, h := cap.derniere(t)
	if u.Path != "/hi/Waypoint/file/images/emblems/mapping.json" || h.Get(enteteSpartan) != "sp" {
		t.Errorf("URL = %s, en-têtes = %v", u, h)
	}
	ko, n := serveurScripte(t, "", http.StatusServiceUnavailable)
	if _, err := clientSur(ko).GetEmblemMapping(context.Background(), authTest); err == nil || n.Load() != 1 {
		t.Errorf("503 : err=%v requêtes=%d (une seule tentative)", err, n.Load())
	}
	illisible, _ := serveurCorps(t, []byte(`[`))
	if _, err := clientSur(illisible).GetEmblemMapping(context.Background(), authTest); err == nil {
		t.Error("JSON illisible : erreur attendue")
	}
}

// TestGetEmblemPositiveCfg : première configuration positive, absence durable (0, nil), échec.
func TestGetEmblemPositiveCfg(t *testing.T) {
	srv, cap := serveurCorps(t, []byte(`{"AvailableConfigurations":[{"ConfigurationId":-5},{"ConfigurationId":651339664}]}`))
	cfg, err := clientSur(srv).GetEmblemPositiveCfg(context.Background(), authTest, "/Inventory/Spartan/Emblems/e.json")
	if err != nil || cfg != 651339664 {
		t.Fatalf("cfg = %d, %v", cfg, err)
	}
	if u, _ := cap.derniere(t); u.Path != "/hi/progression/file/Inventory/Spartan/Emblems/e.json" {
		t.Errorf("URL = %s", u)
	}
	sans, _ := serveurCorps(t, []byte(`{"AvailableConfigurations":[{"ConfigurationId":-809699482}]}`))
	if cfg, err := clientSur(sans).GetEmblemPositiveCfg(context.Background(), authTest, "e"); cfg != 0 || err != nil {
		t.Errorf("sans cfg positive : %d, %v (veut 0, nil)", cfg, err)
	}
	illisible, _ := serveurCorps(t, []byte(`{`))
	if _, err := clientSur(illisible).GetEmblemPositiveCfg(context.Background(), authTest, "e"); err == nil {
		t.Error("JSON illisible : erreur attendue (échec indéterminé)")
	}
	if ExtractEmblemStem("Inventory/Spartan/Emblems/104-001-olympus-campa-2ddbe23b.json") != "104-001-olympus-campa-2ddbe23b" ||
		ExtractEmblemStem("Inventory/Autre/x.json") != "" {
		t.Error("ExtractEmblemStem")
	}
	e := DefaultEndpoints().Avec(CleNameplate, "https://n")
	if NameplateURLDepuisMapping(e, "/images/x.png") != "https://n/hi/Waypoint/file/images/x.png" ||
		NameplateURLDepuisCfg(e, "s", 7) != "https://n/hi/Waypoint/file/images/nameplates/s_7.png" {
		t.Error("gabarits d'URL nameplate")
	}
}

// classementSur rend un lecteur de classement dont le site vise le serveur de test.
func classementSur(srv *httptest.Server, sansRedirection *http.Client) *Classement {
	return NewClassement(srv.Client(), sansRedirection).WithEndpoints(DefaultEndpoints().Avec(CleWaypointWeb, srv.URL))
}

// TestClassement_Fixture : la page enregistrée décode le DTO ; agent, Accept et langue de
// l'appelant ; menus saisons et playlists.
func TestClassement_Fixture(t *testing.T) {
	srv, cap := serveurCorps(t, lireFixture(t, "leaderboard_sample.html"))
	s := classementSur(srv, nil)
	page, err := s.FetchPage(context.Background(), "csrseason13-2", "pl-arena", 2, "xx-YY,xx;q=0.9")
	if err != nil {
		t.Fatal(err)
	}
	if page.PageSize != 100 || len(page.Leaderboard) != 100 || page.Leaderboard[0].Player.Gamertag != "Twissted Mindss" ||
		page.Leaderboard[0].Score != 2180 || page.Leaderboard[0].Player.XUID == "" {
		t.Errorf("page = taille %d, %d entrées, première %+v", page.PageSize, len(page.Leaderboard), page.Leaderboard[0])
	}
	u, h := cap.derniere(t)
	if u.Path != "/halo-infinite/leaderboards/csrseason13-2/pl-arena" || u.Query().Get("page") != "2" ||
		h.Get(enteteLangue) != "xx-YY,xx;q=0.9" || h.Get("User-Agent") != ScraperUserAgent || h.Get(enteteAccept) != acceptHTML ||
		h.Get(enteteSpartan) != "" {
		t.Errorf("URL = %s, en-têtes = %v", u, h)
	}
	seasons, playlists, err := s.FetchCatalog(context.Background(), "csrseason13-2", "pl-arena", "")
	if err != nil || len(seasons) == 0 || len(playlists) == 0 {
		t.Fatalf("catalogue : %d saisons, %d playlists, %v", len(seasons), len(playlists), err)
	}
	trouve := false
	for _, se := range seasons {
		if se.ID == "csrseason12-1" && se.DisplayName == "Shadows" && len(se.Translations) > 0 {
			trouve = true
		}
	}
	if !trouve {
		t.Error("csrseason12-1 (Shadows, avec traductions) absente du menu")
	}
	if _, _, err := s.FetchCatalog(context.Background(), "s", " ", ""); err == nil {
		t.Error("playlist de référence vide : erreur attendue")
	}
	if _, err := ParseLeaderboardPage([]byte("<html></html>")); err == nil {
		t.Error("page sans __NEXT_DATA__ : erreur attendue")
	}
}

// TestClassement_404 : sentinelle de page absente.
func TestClassement_404(t *testing.T) {
	srv, _ := serveurScripte(t, "", http.StatusNotFound)
	_, err := classementSur(srv, nil).FetchPageBytes(context.Background(), "s", "p", 1, "")
	if !errors.Is(err, ErrLeaderboardPageNotFound) {
		t.Errorf("err = %v", err)
	}
	srv2, _ := serveurScripte(t, "", http.StatusInternalServerError)
	if _, err := classementSur(srv2, nil).FetchPageBytes(context.Background(), "s", "p", 1, ""); err == nil || errors.Is(err, ErrLeaderboardPageNotFound) {
		t.Errorf("500 : err = %v", err)
	}
}

// TestClassement_Redirection : la saison active est lue dans Location, sans suivre la redirection ;
// une réponse sans redirection ou vers autre chose qu'une saison est refusée.
func TestClassement_Redirection(t *testing.T) {
	var n atomic.Int32
	var cible atomic.Value
	cible.Store("/halo-infinite/leaderboards/csrseason13-3")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		if r.URL.Path == "/halo-infinite/leaderboards" {
			http.Redirect(w, r, cible.Load().(string), http.StatusTemporaryRedirect)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	sansRedir := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	s := classementSur(srv, sansRedir)
	season, err := s.FetchActiveSeasonByRedirect(context.Background(), "")
	if err != nil || season != "csrseason13-3" || n.Load() != 1 {
		t.Fatalf("saison = %q, err = %v, requêtes = %d", season, err, n.Load())
	}
	cible.Store("/ailleurs")
	if _, err := s.FetchActiveSeasonByRedirect(context.Background(), ""); err == nil {
		t.Error("Location hors saison : erreur attendue")
	}
	ok, _ := serveurCorps(t, []byte("page"))
	if _, err := classementSur(ok, sansRedir).FetchActiveSeasonByRedirect(context.Background(), ""); err == nil {
		t.Error("200 sans redirection : erreur attendue")
	}
}
