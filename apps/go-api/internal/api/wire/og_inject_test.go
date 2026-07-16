package wire

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ogmeta"
)

const testIndexHTML = `<!doctype html>
<html lang="en">
  <head>
    <title>LevelUp</title>
    <!-- og:start -->
    <meta property="og:title" content="OLD" />
    <meta property="og:url" content="https://lvelup.info/" />
    <meta property="og:image" content="https://lvelup.info/og-default.png" />
    <!-- og:end -->
  </head>
  <body><div id="root"></div></body>
</html>`

func writeTempIndex(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "index.html")
	if err := os.WriteFile(p, []byte(testIndexHTML), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPlayerSlugFromPath(t *testing.T) {
	cases := []struct {
		path string
		slug string
		ok   bool
	}{
		{"/players/demo-player/home", "demo-player", true},
		{"/players/demo-player/matches/abc", "demo-player", true},
		{"/players/foo", "foo", true},
		{"/players/", "", false},
		{"/players", "", false},
		{"/career", "", false},
		{"/", "", false},
	}
	for _, c := range cases {
		slug, ok := playerSlugFromPath(c.path)
		if ok != c.ok || slug != c.slug {
			t.Errorf("playerSlugFromPath(%q) = (%q,%v), want (%q,%v)", c.path, slug, ok, c.slug, c.ok)
		}
	}
}

func TestRequestOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/players/demo-player/home", nil)
	req.Host = "demo.lvelup.info"
	req.Header.Set("X-Forwarded-Proto", "https")
	if got := requestOrigin(req); got != "https://demo.lvelup.info" {
		t.Errorf("origin = %q", got)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Host = "localhost:8000"
	if got := requestOrigin(req2); got != "http://localhost:8000" {
		t.Errorf("origin default = %q", got)
	}

	// En-tetes proxy multi-valeurs → premier token.
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.Host = "internal"
	req3.Header.Set("X-Forwarded-Proto", "https, http")
	req3.Header.Set("X-Forwarded-Host", "demo.lvelup.info, proxy")
	if got := requestOrigin(req3); got != "https://demo.lvelup.info" {
		t.Errorf("origin fwd = %q", got)
	}
}

// DemoMode off (prod) : meme un crawler sur une route joueur reçoit la carte
// generique — jamais d'enrichissement (donc jamais de fuite de KPIs prives).
func TestServeIndexWithOG_DefaultCard(t *testing.T) {
	indexPath := writeTempIndex(t)
	reg := &ServiceRegistry{cfg: &config.AppConfig{}}

	req := httptest.NewRequest(http.MethodGet, "/players/demo-player/home", nil)
	req.Host = "demo.lvelup.info"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("User-Agent", "facebookexternalhit/1.1")
	rec := httptest.NewRecorder()

	reg.serveIndexWithOG(rec, req, indexPath)

	if cc := rec.Result().Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q", cc)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `content="https://demo.lvelup.info/players/demo-player/home"`) {
		t.Errorf("og:url non reecrit a l'origine:\n%s", body)
	}
	if !strings.Contains(body, `content="https://demo.lvelup.info/og-default.png"`) {
		t.Errorf("og:image non reecrit a l'origine:\n%s", body)
	}
	if !strings.Contains(body, "Stats Halo") {
		t.Errorf("carte generique attendue:\n%s", body)
	}
	if strings.Contains(body, `content="OLD"`) {
		t.Errorf("bloc OG non remplace:\n%s", body)
	}
}

// Route non-joueur en demo + crawler : carte generique, sans acces DB (le nil
// resolver de reg ferait paniquer si l'enrichissement etait tente).
func TestServeIndexWithOG_NonPlayerRouteNoDB(t *testing.T) {
	indexPath := writeTempIndex(t)
	reg := &ServiceRegistry{cfg: &config.AppConfig{DemoMode: true}}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "demo.lvelup.info"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("User-Agent", "facebookexternalhit/1.1")
	rec := httptest.NewRecorder()

	reg.serveIndexWithOG(rec, req, indexPath)

	if body := rec.Body.String(); !strings.Contains(body, `content="https://demo.lvelup.info/"`) {
		t.Errorf("og:url racine attendu:\n%s", body)
	}
}

// ── Invariant identité de PAGE sur le chemin OG ──────────────────────────────
//
// serveIndexWithOG passe par HomeCtx (variante NON authentifiée) : le contexte
// crawler ne porte aucun xuid. ogMetaFromHome doit forcer le xuid du joueur de la
// PAGE avant GetHomePage, sinon la résolution d'identité Spartan
// (GetSpartanIdentity, scopée sur ctxkeys.HaloXUID) cible "" (identité vide).
// Même invariant que HomeCtxWithAuth (forcePageIdentityXUID).

// ctxCapturingHomeService enregistre le xuid porté par le contexte reçu par
// GetHomePage, pour vérifier l'identité posée dessus.
type ctxCapturingHomeService struct {
	seenXUID string
	page     *domain.HomePageResponse
}

func (f *ctxCapturingHomeService) GetHomePage(ctx context.Context, _, _ string) (*domain.HomePageResponse, error) {
	f.seenXUID = ctxkeys.HaloXUID(ctx)
	return f.page, nil
}

func (f *ctxCapturingHomeService) GetBattlePass(context.Context) domain.BattlePassResponse {
	return domain.BattlePassResponse{}
}

func (f *ctxCapturingHomeService) GetChallenges(context.Context) domain.ChallengesResponse {
	return domain.ChallengesResponse{}
}

func (f *ctxCapturingHomeService) RefreshTrack(context.Context, string) {}

// Crawler anonyme (aucun xuid en contexte) : ogMetaFromHome impose le xuid du
// joueur de la page avant GetHomePage. Verrouille que la carte OG résout
// l'identité sur le bon joueur et non sur "".
func TestOGMetaFromHome_NoContextXUID_PinsPageXUID(t *testing.T) {
	const pageXUID = "xuid-og-page"
	svc := &ctxCapturingHomeService{page: &domain.HomePageResponse{Hero: domain.HomeHeroCard{PlayerName: "JGtm"}}}

	meta, ok := ogMetaFromHome(context.Background(), svc, pageXUID, "JGtm",
		"https://lvelup.info", "/players/jgtm/home", ogmeta.LocaleFR)

	if !ok {
		t.Fatal("ogMetaFromHome doit réussir avec un HomeService qui rend une page")
	}
	if svc.seenXUID != pageXUID {
		t.Errorf("GetHomePage doit voir le xuid de la PAGE %q, got %q (identité résolue sur \"\")", pageXUID, svc.seenXUID)
	}
	if meta.URL != "https://lvelup.info/players/jgtm/home" {
		t.Errorf("meta.URL = %q", meta.URL)
	}
}

// xuid ambiant étranger (défense en profondeur) : écrasé par celui de la page —
// jamais l'identité d'un autre compte sur la carte OG.
func TestOGMetaFromHome_ForeignContextXUID_OverriddenByPage(t *testing.T) {
	const pageXUID = "xuid-og-page"
	const ambientXUID = "xuid-autre-compte"
	svc := &ctxCapturingHomeService{page: &domain.HomePageResponse{Hero: domain.HomeHeroCard{PlayerName: "JGtm"}}}
	ctx := ctxkeys.WithHaloXUID(context.Background(), ambientXUID)

	if _, ok := ogMetaFromHome(ctx, svc, pageXUID, "JGtm",
		"https://lvelup.info", "/players/jgtm/home", ogmeta.LocaleFR); !ok {
		t.Fatal("ogMetaFromHome doit réussir")
	}
	if svc.seenXUID != pageXUID {
		t.Errorf("xuid ambiant %q doit être écrasé par le xuid de la page %q, got %q", ambientXUID, pageXUID, svc.seenXUID)
	}
}
