//go:build cgo

// desktop_entry_cgo_test.go — l'entrée de la coquille desktop SUR la pile réelle (ADR 0045 D-3 et
// D-7, lot L13.4) : le routeur d'api.NewRouter assemblé sous la capacité ShellEntryGate, requêtes
// httptest. Chaque refus (sans cookie, Host étranger, secret absent ou faux, pair hors boucle
// locale), le succès (cookie du bon nom et des bons attributs, 302), /health ouvert sans cookie,
// les libellés en fr et en en, CORS et CSRF limités à l'origine du port lié, et la preuve que
// hors du profil desktop rien de tout cela n'est monté.
//
// Driver DuckDB requis (tag cgo, dépendance transitive du routeur assemblé).

package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/api/openapigen"
)

const (
	desktopTestPort   = 49555
	desktopTestHost   = "127.0.0.1:49555"
	desktopTestOrigin = "http://127.0.0.1:49555"
	// desktopTestLauncherSecret : le secret que la coquille présente à /desktop/enter.
	desktopTestLauncherSecret = "secret-du-lanceur-de-test" // pragma: allowlist secret
	// desktopTestCookieKey : le secret de session (clé du cookie), de 32 caractères au moins.
	desktopTestCookieKey = "cle-de-cookie-de-test-de-trente-deux-octets" // pragma: allowlist secret
	loopbackPeer         = "127.0.0.1:50321"
)

// buildDesktopRouter assemble le vrai routeur sous l'entrée de la coquille, port desktopTestPort.
func buildDesktopRouter(t *testing.T) http.Handler {
	t.Helper()
	return buildDesktopRouterWith(t, nil)
}

// buildDesktopRouterWith est buildDesktopRouter avec la demande d'arrêt branchée sur requester.
func buildDesktopRouterWith(t *testing.T, requester func(string)) http.Handler {
	t.Helper()
	router, _, err := openapigen.BuildDemoRouter(t.Context(), openapigen.Options{
		Root:              t.TempDir(),
		GroupStorePath:    filepath.Join(t.TempDir(), "groups.json"),
		Prestige:          true,
		DesktopSecret:     desktopTestLauncherSecret,
		ShellEntryPort:    desktopTestPort,
		ShellCookieKey:    desktopTestCookieKey,
		ShutdownRequester: requester,
	})
	if err != nil {
		t.Fatalf("BuildDemoRouter(desktop) : %v", err)
	}
	return router
}

// desktopReq décrit une requête vers le routeur de la coquille.
type desktopReq struct {
	method, target, host, peer, origin, locale string
	cookie                                     *http.Cookie
}

func (q desktopReq) do(h http.Handler) *httptest.ResponseRecorder {
	method := q.method
	if method == "" {
		method = http.MethodGet
	}
	req := httptest.NewRequest(method, q.target, nil)
	req.Host = desktopTestHost
	if q.host != "" {
		req.Host = q.host
	}
	req.RemoteAddr = loopbackPeer
	if q.peer != "" {
		req.RemoteAddr = q.peer
	}
	if q.origin != "" {
		req.Header.Set("Origin", q.origin)
	}
	if q.locale != "" {
		req.Header.Set("X-LevelUp-Locale", q.locale)
	}
	if q.cookie != nil {
		req.AddCookie(q.cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// enterDesktop passe l'entrée avec le bon secret et rend le cookie posé.
func enterDesktop(t *testing.T, h http.Handler) *http.Cookie {
	t.Helper()
	rec := desktopReq{target: middleware.DesktopEnterPath + "?secret=" + desktopTestLauncherSecret}.do(h)
	if rec.Code != http.StatusFound {
		t.Fatalf("entrée : statut %d, attendu 302 (%s)", rec.Code, rec.Body.String())
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == middleware.DesktopCookieName(desktopTestPort) {
			return c
		}
	}
	t.Fatalf("entrée : aucun cookie %q posé (%v)", middleware.DesktopCookieName(desktopTestPort), rec.Result().Cookies())
	return nil
}

func TestDesktopEntry_SansCookieToutEstRefuseSaufSondeEtEntree(t *testing.T) {
	h := buildDesktopRouter(t)
	for _, target := range []string{"/", "/api/v1/bootstrap", "/api/v1/desktop/strings", "/static/x", "/healthz", "/health/"} {
		if rec := (desktopReq{target: target}).do(h); rec.Code != http.StatusUnauthorized {
			t.Errorf("GET %s sans cookie : %d, attendu 401", target, rec.Code)
		}
	}
	// Le refus a la forme d'erreur de l'API : l'interface le reconnaît à son code et n'insiste pas.
	rec := (desktopReq{target: "/api/v1/bootstrap"}).do(h)
	var refusal struct {
		Code      string `json:"code"`
		Retryable bool   `json:"retryable"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &refusal); err != nil || refusal.Code != "desktop_unauthorized" ||
		refusal.Retryable || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Errorf("corps du 401 = %q (%v), type %q ; attendu {code:desktop_unauthorized, retryable:false} en JSON",
			rec.Body.String(), err, rec.Header().Get("Content-Type"))
	}
	if rec := (desktopReq{target: "/health"}).do(h); rec.Code != http.StatusOK {
		t.Errorf("GET /health sans cookie : %d, attendu 200 (la sonde du lanceur reste ouverte)", rec.Code)
	}
	// Un chemin ambigu qui se lirait « /health » après nettoyage ne contourne pas le cookie.
	for _, target := range []string{"/health/../api/v1/bootstrap", "//health", "/health/%2e%2e/api/v1/bootstrap"} {
		if rec := (desktopReq{target: target}).do(h); rec.Code == http.StatusOK {
			t.Errorf("GET %s sans cookie : 200, un chemin ambigu ne doit pas ouvrir la sonde", target)
		}
	}
}

func TestDesktopEntry_HostEtrangerRefuse(t *testing.T) {
	h := buildDesktopRouter(t)
	cookie := enterDesktop(t, h)
	hosts := []string{"localhost:49555", "127.0.0.1:49556", "127.0.0.1", "evil.example", "evil.example:49555",
		"[::1]:49555", "127.0.0.1:49555.evil.example"}
	for _, host := range hosts {
		for _, target := range []string{"/api/v1/bootstrap", "/health", middleware.DesktopEnterPath + "?secret=" + desktopTestLauncherSecret} {
			rec := desktopReq{target: target, host: host, cookie: cookie}.do(h)
			if rec.Code != http.StatusMisdirectedRequest {
				t.Errorf("Host %q sur %s : %d, attendu 421", host, target, rec.Code)
			}
			if len(rec.Result().Cookies()) != 0 {
				t.Errorf("Host %q sur %s : un cookie a été posé malgré le refus", host, target)
			}
		}
	}
}

func TestDesktopEntry_RefusDeLEntree(t *testing.T) {
	h := buildDesktopRouter(t)
	good := middleware.DesktopEnterPath + "?secret=" + desktopTestLauncherSecret
	for name, q := range map[string]desktopReq{
		"secret absent":         {target: middleware.DesktopEnterPath},
		"secret vide":           {target: middleware.DesktopEnterPath + "?secret="},
		"secret faux":           {target: middleware.DesktopEnterPath + "?secret=mauvais"},
		"secret tronqué":        {target: middleware.DesktopEnterPath + "?secret=" + desktopTestLauncherSecret[:5]},
		"secret trop long":      {target: good + "x"},
		"pair hors boucle":      {target: good, peer: "192.0.2.10:4000"},
		"pair hors boucle IPv6": {target: good, peer: "[2001:db8::1]:4000"},
	} {
		rec := q.do(h)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s : %d, attendu 403", name, rec.Code)
		}
		if len(rec.Result().Cookies()) != 0 {
			t.Errorf("%s : un cookie a été posé malgré le refus", name)
		}
	}
}

func TestDesktopEntry_SuccesPoseLeCookieEtRedirige(t *testing.T) {
	h := buildDesktopRouter(t)
	rec := desktopReq{target: middleware.DesktopEnterPath + "?secret=" + desktopTestLauncherSecret}.do(h)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/" {
		t.Fatalf("entrée : %d vers %q, attendu 302 vers /", rec.Code, rec.Header().Get("Location"))
	}
	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == middleware.DesktopCookieName(desktopTestPort) {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatalf("entrée : aucun cookie posé (%v)", rec.Result().Cookies())
	}
	if cookie.Name != "levelup_session_49555" {
		t.Errorf("nom du cookie = %q, attendu levelup_session_49555 (nom lié au port)", cookie.Name)
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Secure ||
		cookie.Path != "/" || cookie.MaxAge != 30*24*3600 {
		t.Errorf("attributs du cookie = %+v, attendu HttpOnly, SameSite=Lax (l'entrée est lancée depuis l'origine embarquée de la coquille), sans Secure, Path=/, MaxAge=30 jours", cookie)
	}
	if strings.Contains(cookie.Value, desktopTestLauncherSecret) || strings.Contains(cookie.Value, desktopTestCookieKey) {
		t.Error("la valeur du cookie contient un secret en clair")
	}
	// Avec le cookie, le reste du routeur répond (le contrat de la coquille : bootstrap = 200).
	for _, target := range []string{"/api/v1/bootstrap", "/health"} {
		if got := (desktopReq{target: target, cookie: cookie}).do(h); got.Code != http.StatusOK {
			t.Errorf("GET %s avec cookie : %d, attendu 200", target, got.Code)
		}
	}
	// Un cookie du bon nom mais de signature fausse, ou le cookie d'un autre port, ne passe pas.
	forged := &http.Cookie{Name: cookie.Name, Value: "9999999999.deadbeef"}
	other := &http.Cookie{Name: middleware.DesktopCookieName(49556), Value: cookie.Value}
	for name, c := range map[string]*http.Cookie{"signature fausse": forged, "cookie d'un autre port": other} {
		if got := (desktopReq{target: "/api/v1/bootstrap", cookie: c}).do(h); got.Code != http.StatusUnauthorized {
			t.Errorf("%s : %d, attendu 401", name, got.Code)
		}
	}
}

func TestDesktopEntry_LibellesDeLaCoquilleDansLaLangueDeLaRequete(t *testing.T) {
	h := buildDesktopRouter(t)
	cookie := enterDesktop(t, h)
	get := func(locale string) (string, map[string]string) {
		t.Helper()
		rec := desktopReq{target: "/api/v1/desktop/strings", locale: locale, cookie: cookie}.do(h)
		if rec.Code != http.StatusOK {
			t.Fatalf("strings (%q) : %d (%s)", locale, rec.Code, rec.Body.String())
		}
		var body struct {
			Locale  string            `json:"locale"`
			Strings map[string]string `json:"strings"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("strings (%q) : JSON illisible : %v", locale, err)
		}
		return body.Locale, body.Strings
	}
	wantKeys := []string{"starting", "failed_locked", "failed_migration_failed", "failed_schema_newer",
		"failed_port", "failed_timeout", "stopping", "stopped", "update_available"}
	frLoc, fr := get("fr")
	enLoc, en := get("en")
	_, def := get("")
	if frLoc != "fr" || enLoc != "en" {
		t.Errorf("langues rendues = %q et %q, attendu fr et en", frLoc, enLoc)
	}
	for _, k := range wantKeys {
		if fr[k] == "" || en[k] == "" {
			t.Errorf("clé %q : fr=%q en=%q, un libellé est vide", k, fr[k], en[k])
		}
		if fr[k] == en[k] {
			t.Errorf("clé %q : le libellé fr et le libellé en sont identiques (%q)", k, fr[k])
		}
		if def[k] != fr[k] {
			t.Errorf("clé %q : sans en-tête de langue, attendu la langue par défaut (fr)", k)
		}
	}
	if len(fr) != len(wantKeys) {
		t.Errorf("%d libellés rendus, attendu %d", len(fr), len(wantKeys))
	}
	if fr["starting"] != "LevelUp démarre…" || en["starting"] != "LevelUp is starting…" {
		t.Errorf("écran de démarrage : fr=%q en=%q", fr["starting"], en["starting"])
	}
	for _, m := range []map[string]string{fr, en} {
		if !strings.Contains(m["update_available"], "{version}") {
			t.Errorf("update_available = %q : le gabarit {version} doit voyager tel quel", m["update_available"])
		}
	}
}

// CORS : l'origine exacte du port lié est la seule acceptée ; CSRF suit la même liste.
func TestDesktopEntry_CORSEtCSRFLimitesALOrigineDuPortLie(t *testing.T) {
	h := buildDesktopRouter(t)
	cookie := enterDesktop(t, h)
	acao := func(origin string) string {
		return desktopReq{target: "/api/v1/bootstrap", origin: origin, cookie: cookie}.do(h).Header().Get("Access-Control-Allow-Origin")
	}
	if got := acao(desktopTestOrigin); got != desktopTestOrigin {
		t.Errorf("Access-Control-Allow-Origin pour l'origine du port = %q, attendu %q", got, desktopTestOrigin)
	}
	for _, foreign := range []string{"http://localhost:49555", "http://127.0.0.1:5173", "http://127.0.0.1:49556",
		"https://evil.example", "http://localhost:5173"} {
		if got := acao(foreign); got != "" {
			t.Errorf("origine %q : Access-Control-Allow-Origin = %q, attendu aucun", foreign, got)
		}
	}
	post := func(origin string) int {
		return desktopReq{method: http.MethodPost, target: "/api/v1/session/context", origin: origin, cookie: cookie}.do(h).Code
	}
	if got := post("https://evil.example"); got != http.StatusForbidden {
		t.Errorf("POST avec une origine étrangère : %d, attendu 403 (csrf_rejected)", got)
	}
	if got := post(desktopTestOrigin); got == http.StatusForbidden || got == http.StatusUnauthorized {
		t.Errorf("POST avec l'origine du port : %d, le contrôle d'origine ne doit pas le refuser", got)
	}
}

// Hors du profil desktop, rien de tout cela n'est monté : pas de route d'entrée, pas de libellés,
// pas de Host strict ni de cookie exigé.
func TestDesktopEntry_AbsenteHorsDuProfilDesktop(t *testing.T) {
	h := buildTestRouter(t)
	get := func(target, host string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		if host != "" {
			req.Host = host
		}
		req.RemoteAddr = loopbackPeer
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	if rec := get("/api/v1/desktop/strings", ""); rec.Code == http.StatusOK || rec.Code == http.StatusUnauthorized {
		t.Errorf("GET /api/v1/desktop/strings hors desktop : %d, la route ne doit pas exister", rec.Code)
	}
	if rec := get(middleware.DesktopEnterPath+"?secret="+desktopTestLauncherSecret, ""); rec.Code == http.StatusFound ||
		len(rec.Result().Cookies()) != 0 && rec.Result().Cookies()[0].Name == middleware.DesktopCookieName(8000) {
		t.Errorf("GET %s hors desktop : %d, la route ne doit pas poser de cookie de la coquille", middleware.DesktopEnterPath, rec.Code)
	}
	if rec := get("/api/v1/bootstrap", "example.org"); rec.Code != http.StatusOK {
		t.Errorf("GET /api/v1/bootstrap hors desktop : %d, attendu 200 sans cookie ni Host strict", rec.Code)
	}
}

// La route d'arrêt de secours (ADR 0045 D-4) n'exige pas le cookie de la coquille, que celle-ci ne peut
// pas fabriquer : elle garde LoopbackOnly et le secret d'en-tête. Le Host strict s'y applique.
func TestDesktopEntry_RouteDArretSansCookie(t *testing.T) {
	requested := make(chan string, 4)
	h := buildDesktopRouterWith(t, func(reason string) { requested <- reason })
	shutdown := func(secret, host, peer string, method string) int {
		req := httptest.NewRequest(method, middleware.DesktopShutdownPath, nil)
		req.Host = desktopTestHost
		if host != "" {
			req.Host = host
		}
		req.RemoteAddr = loopbackPeer
		if peer != "" {
			req.RemoteAddr = peer
		}
		if secret != "" {
			req.Header.Set("X-LevelUp-Shutdown-Secret", secret)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if got := shutdown(desktopTestLauncherSecret, "", "", http.MethodPost); got != http.StatusAccepted {
		t.Errorf("arrêt sans cookie, bon secret : %d, attendu 202", got)
	}
	select {
	case <-requested:
	case <-time.After(5 * time.Second):
		t.Error("202 rendu mais aucune demande d'arrêt transmise")
	}
	for name, got := range map[string]int{
		"mauvais secret":   shutdown("mauvais", "", "", http.MethodPost),
		"secret absent":    shutdown("", "", "", http.MethodPost),
		"pair hors boucle": shutdown(desktopTestLauncherSecret, "", "192.0.2.10:4000", http.MethodPost),
	} {
		if got != http.StatusForbidden {
			t.Errorf("arrêt sans cookie, %s : %d, attendu 403", name, got)
		}
	}
	if got := shutdown(desktopTestLauncherSecret, "evil.example", "", http.MethodPost); got != http.StatusMisdirectedRequest {
		t.Errorf("arrêt avec un Host étranger : %d, attendu 421", got)
	}
	// L'exemption est celle du POST de cette route seule.
	if got := shutdown(desktopTestLauncherSecret, "", "", http.MethodGet); got != http.StatusUnauthorized {
		t.Errorf("GET sur la route d'arrêt sans cookie : %d, attendu 401", got)
	}
}

// Le secret d'entrée est à usage unique : après l'entrée de la coquille, le rejouer (l'URL reste dans
// l'historique de la WebView) est refusé sans cookie, et le cookie déjà posé continue de servir.
func TestDesktopEntry_SecretAUsageUnique(t *testing.T) {
	h := buildDesktopRouter(t)
	cookie := enterDesktop(t, h)
	rec := desktopReq{target: middleware.DesktopEnterPath + "?secret=" + desktopTestLauncherSecret}.do(h)
	if rec.Code != http.StatusForbidden || len(rec.Result().Cookies()) != 0 {
		t.Errorf("seconde entrée : %d (%d cookie), attendu 403 sans cookie", rec.Code, len(rec.Result().Cookies()))
	}
	if rec := (desktopReq{target: "/api/v1/bootstrap", cookie: cookie}).do(h); rec.Code != http.StatusOK {
		t.Errorf("cookie de la première entrée : %d, attendu 200", rec.Code)
	}
}
