// Package handlers_test — auth_browser_login_test.go : la connexion Microsoft par le navigateur (L21.00)
// de bout en bout, sur la pile réelle du garde de la coquille, avec un Azure simulé.
//
// Le « navigateur externe » est un client SANS cookie qui présente Host localhost:<port> ; la
// « coquille » est un client avec son cookie de coquille et sa session. Azure simulé : il vérifie
// le PKCE S256 (le verifier présenté à l'échange redonne le challenge de l'URL d'autorisation) et
// l'adresse de retour.
package handlers_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/service"
)

const (
	browserLoginXUID   = "2535400000000077"
	browserLoginSecret = "secret-du-lanceur-de-test" // pragma: allowlist secret
	browserLoginKey    = "cle-de-cookie-de-test-de-trente-deux-octets"
)

// fakeAzure joue Microsoft : il retient le challenge de l'URL d'autorisation, refuse un code ou un
// verifier qui ne correspond pas, et compte les échanges.
type fakeAzure struct {
	mu          sync.Mutex
	redirectURI string
	challenge   string
	exchanges   int
	failCode    bool
}

func (f *fakeAzure) AuthorizeURL(redirectURI, state, challenge string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.redirectURI, f.challenge = redirectURI, challenge
	q := url.Values{"redirect_uri": {redirectURI}, "state": {state}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}}
	return "https://login.test/authorize?" + q.Encode()
}

func (f *fakeAzure) ExchangeCode(_ context.Context, code, redirectURI, verifier string) (*auth_platform.AuthCodeResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.exchanges++
	sum := sha256.Sum256([]byte(verifier))
	switch {
	case f.failCode:
		return nil, errors.New("azure: invalid_grant")
	case code != "code-ok":
		return nil, errors.New("azure: code inconnu")
	case redirectURI != f.redirectURI:
		return nil, errors.New("azure: redirect_uri différente de celle de l'autorisation")
	case base64.RawURLEncoding.EncodeToString(sum[:]) != f.challenge:
		return nil, errors.New("azure: verifier ne redonne pas le challenge")
	}
	return &auth_platform.AuthCodeResult{AccessToken: "ms-access", RefreshToken: "ms-refresh", ExpiresIn: 3600}, nil
}

func (f *fakeAzure) AcquireRTA(context.Context, string) (*auth_platform.XSTSResult, error) {
	return nil, errors.New("xsts rta indisponible (non bloquant)")
}

// countingStrategy compte les arrivées dans le point d'arrivée commun de la connexion.
type countingStrategy struct {
	mu    sync.Mutex
	calls int
	inner auth_platform.LinkStrategy
}

func (c *countingStrategy) OnAuthSuccess(ctx context.Context, a *auth_platform.Attempt, s *domain.SessionData) error {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return c.inner.OnAuthSuccess(ctx, a, s)
}

// browserRig : le serveur httptest, le client de la coquille, le client du navigateur externe.
type browserRig struct {
	srv      *httptest.Server
	port     int
	azure    *fakeAzure
	store    *auth_platform.MultiUserTokenStore
	attempts *auth_platform.AttemptStore
	strategy *countingStrategy
	shell    *http.Client
	external *http.Client
}

func newBrowserRig(t *testing.T) *browserRig {
	t.Helper()
	dir := t.TempDir()
	srv := httptest.NewUnstartedServer(nil)
	portNum := srv.Listener.Addr().(*net.TCPAddr).Port
	portStr := strconv.Itoa(portNum)
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, browserLoginKey)
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	attempts := auth_platform.NewAttemptStore()
	store := auth_platform.NewMultiUserTokenStore(filepath.Join(dir, "watcher_tokens"))
	gate := func(_ context.Context, _, xuid string) bool { return xuid == browserLoginXUID }
	strategy := &countingStrategy{inner: service.NewAuthLinkStrategy(
		auth_platform.NewPasswordLinkStrategy(users), service.NewAuthCompletion(store, gate))}
	provider := &stubTokenProvider{exchangeResult: &auth_platform.ExchangeResult{
		Tokens: &domain.HaloTokens{SpartanToken: "spartan"}, Gamertag: "PosteSeul", XUID: browserLoginXUID,
	}}
	azure := &fakeAzure{}
	login := handlers.NewBrowserLoginHandler(sessStore, attempts, provider, portNum).WithBackend(azure)

	authority := "127.0.0.1:" + portStr
	desktopGate := middleware.NewDesktopGate(authority, portNum, browserLoginKey, browserLoginSecret).AllowBrowserCallback()
	r := chi.NewRouter()
	r.Use(middleware.PreserveRemoteAddr)
	r.Use(desktopGate.Middleware)
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	r.Use(middleware.TitleExtractor(titlePkg.DefaultRegistry()))
	r.Get(middleware.DesktopEnterPath, desktopGate.EnterHandler)
	r.Route("/api/v1", func(r chi.Router) {
		handlers.NewAuthHandler(sessStore, attempts, provider).WithLinkStrategy(strategy).MountDeviceFlow(r)
		login.MountStart(r)
	})
	r.With(middleware.LoopbackOnly).Get(handlers.BrowserLoginCallbackPath, login.Callback)
	srv.Config.Handler = r
	srv.Start()
	t.Cleanup(srv.Close)

	jar, _ := cookiejar.New(nil)
	rig := &browserRig{
		srv: srv, port: portNum, azure: azure, store: store, attempts: attempts, strategy: strategy,
		shell:    &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		external: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}
	// La coquille passe son entrée une fois : son cookie vit dans son jar.
	resp := rig.shellGet(t, middleware.DesktopEnterPath+"?secret="+browserLoginSecret)
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("entrée de la coquille : %d", resp.StatusCode)
	}
	return rig
}

func (rig *browserRig) shellGet(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := rig.shell.Get(rig.srv.URL + path)
	if err != nil {
		t.Fatalf("GET %s : %v", path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// browserCallback joue le navigateur externe : aucun cookie, Host localhost:<port>.
func (rig *browserRig) browserCallback(t *testing.T, query url.Values) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, rig.srv.URL+handlers.BrowserLoginCallbackPath+"?"+query.Encode(), nil)
	req.Host = "localhost:" + strconv.Itoa(rig.port)
	resp, err := rig.external.Do(req)
	if err != nil {
		t.Fatalf("callback : %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

// start joue le clic du bouton : rend l'identifiant de tentative et le state de l'URL d'autorisation.
func (rig *browserRig) start(t *testing.T, locale string) (attemptID, state string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, rig.srv.URL+"/api/v1/auth/xbox/login", nil)
	if locale != "" {
		req.Header.Set("X-LevelUp-Locale", locale)
	}
	resp, err := rig.shell.Do(req)
	if err != nil {
		t.Fatalf("start : %v", err)
	}
	defer resp.Body.Close()
	var out struct {
		AttemptID    string `json:"attempt_id"`
		AuthorizeURL string `json:"authorize_url"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&out) != nil {
		t.Fatalf("start : statut %d", resp.StatusCode)
	}
	u, err := url.Parse(out.AuthorizeURL)
	if err != nil {
		t.Fatalf("authorize_url illisible : %v", err)
	}
	return out.AttemptID, u.Query().Get("state")
}

// poll joue le sondage de la page : l'état de la tentative.
func (rig *browserRig) poll(t *testing.T, attemptID string) (status, errorCode string) {
	t.Helper()
	resp := rig.shellGet(t, "/api/v1/auth/device-flow/"+attemptID)
	var out struct {
		Status    string  `json:"status"`
		ErrorCode *string `json:"error_code"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&out) != nil {
		t.Fatalf("sondage : statut %d", resp.StatusCode)
	}
	if out.ErrorCode != nil {
		errorCode = *out.ErrorCode
	}
	return out.Status, errorCode
}

func TestBrowserLogin_FluxComplet_AboutitAuPointDArriveeCommun(t *testing.T) {
	rig := newBrowserRig(t)
	attemptID, state := rig.start(t, "en")

	// L'URL d'autorisation porte l'adresse de retour loopback du port lié, S256 et le state.
	if want := "http://localhost:" + strconv.Itoa(rig.port) + "/auth/xbox/callback"; rig.azure.redirectURI != want {
		t.Errorf("redirect_uri = %q, attendu %q", rig.azure.redirectURI, want)
	}
	if status, _ := rig.poll(t, attemptID); status != "pending" {
		t.Fatalf("avant le retour : %q, attendu pending", status)
	}

	code, page := rig.browserCallback(t, url.Values{"code": {"code-ok"}, "state": {state}})
	if code != http.StatusOK || !strings.Contains(page, "Signed in") || !strings.Contains(page, `lang="en"`) {
		t.Fatalf("page de retour : %d %q (langue de la coquille attendue : en)", code, page)
	}
	if strings.Contains(page, "code-ok") || strings.Contains(page, state) {
		t.Error("la page de retour ne doit contenir ni le code ni le state")
	}

	// Le sondage de la coquille remet la tentative : session connectée, point d'arrivée appelé UNE fois.
	if status, _ := rig.poll(t, attemptID); status != "provisioned" {
		t.Fatalf("après le retour : %q, attendu provisioned", status)
	}
	rig.poll(t, attemptID)
	if rig.strategy.calls != 1 {
		t.Errorf("point d'arrivée appelé %d fois, attendu 1", rig.strategy.calls)
	}
	stored, err := rig.store.Load(browserLoginXUID)
	if err != nil || stored.OAuthRefreshToken != "ms-refresh" {
		t.Fatalf("jeton de rafraîchissement non persisté pour le profil suivi : %+v, %v", stored, err)
	}
}

func TestBrowserLogin_StateMauvaisInconnuOuRejoue(t *testing.T) {
	rig := newBrowserRig(t)
	attemptID, state := rig.start(t, "")

	for _, bad := range []string{"", "inconnu"} {
		code, page := rig.browserCallback(t, url.Values{"code": {"code-ok"}, "state": {bad}})
		if code != http.StatusBadRequest || !strings.Contains(page, "state_invalid") {
			t.Errorf("state %q : %d, attendu 400 state_invalid", bad, code)
		}
	}
	if rig.azure.exchanges != 0 {
		t.Fatalf("un state refusé ne doit jamais atteindre Microsoft (%d échanges)", rig.azure.exchanges)
	}
	if status, _ := rig.poll(t, attemptID); status != "pending" {
		t.Errorf("la tentative légitime doit survivre aux essais ratés : %q", status)
	}

	if code, _ := rig.browserCallback(t, url.Values{"code": {"code-ok"}, "state": {state}}); code != http.StatusOK {
		t.Fatalf("retour légitime : %d", code)
	}
	// Rejeu du même state : usage unique.
	code, page := rig.browserCallback(t, url.Values{"code": {"code-ok"}, "state": {state}})
	if code != http.StatusBadRequest || !strings.Contains(page, "state_invalid") || rig.azure.exchanges != 1 {
		t.Errorf("rejeu : %d, %d échanges ; attendu 400 state_invalid et un seul échange", code, rig.azure.exchanges)
	}
}

func TestBrowserLogin_StateExpireApresDixMinutes(t *testing.T) {
	rig := newBrowserRig(t)
	attemptID, state := rig.start(t, "")
	rig.attempts.Update(attemptID, func(a *auth_platform.Attempt) {
		a.StartedAt = time.Now().Add(-auth_platform.BrowserStateTTL - time.Minute)
	})

	code, page := rig.browserCallback(t, url.Values{"code": {"code-ok"}, "state": {state}})
	if code != http.StatusBadRequest || !strings.Contains(page, "state_expired") || rig.azure.exchanges != 0 {
		t.Fatalf("state expiré : %d, %d échanges ; attendu 400 state_expired sans échange", code, rig.azure.exchanges)
	}
	if status, _ := rig.poll(t, attemptID); status != "expired" {
		t.Errorf("sondage après expiration = %q, attendu expired", status)
	}
}

func TestBrowserLogin_RefusDeMicrosoftEtEchecDEchange(t *testing.T) {
	rig := newBrowserRig(t)
	attemptID, state := rig.start(t, "")
	code, page := rig.browserCallback(t, url.Values{"error": {"access_denied"}, "error_description": {"secret détail"}, "state": {state}})
	if code != http.StatusBadRequest || !strings.Contains(page, "oauth_denied") || strings.Contains(page, "secret détail") {
		t.Errorf("refus Microsoft : %d %q", code, page)
	}
	if status, errCode := rig.poll(t, attemptID); status != "failed" || errCode != "oauth_denied" {
		t.Errorf("sondage = %q/%q, attendu failed/oauth_denied", status, errCode)
	}

	rig.azure.failCode = true
	attemptID, state = rig.start(t, "")
	code, page = rig.browserCallback(t, url.Values{"code": {"code-ok"}, "state": {state}})
	if code != http.StatusBadRequest || !strings.Contains(page, "code_exchange_failed") || strings.Contains(page, "invalid_grant") {
		t.Errorf("échec d'échange : %d %q (le détail de Microsoft ne doit pas fuiter)", code, page)
	}
	if status, errCode := rig.poll(t, attemptID); status != "failed" || errCode != "code_exchange_failed" {
		t.Errorf("sondage = %q/%q, attendu failed/code_exchange_failed", status, errCode)
	}
	if rig.strategy.calls != 0 {
		t.Errorf("un échec n'atteint jamais le point d'arrivée (%d appels)", rig.strategy.calls)
	}
}

func TestBrowserLogin_UnSecondClicRemplaceLaTentative(t *testing.T) {
	rig := newBrowserRig(t)
	firstID, firstState := rig.start(t, "")
	secondID, secondState := rig.start(t, "")
	if firstID == secondID || firstState == secondState {
		t.Fatal("un second clic doit ouvrir une tentative neuve")
	}
	if code, page := rig.browserCallback(t, url.Values{"code": {"code-ok"}, "state": {firstState}}); code != http.StatusBadRequest || !strings.Contains(page, "state_invalid") {
		t.Errorf("l'ancien state ne vaut plus rien : %d", code)
	}
	if code, _ := rig.browserCallback(t, url.Values{"code": {"code-ok"}, "state": {secondState}}); code != http.StatusOK {
		t.Errorf("le state neuf aboutit : %d", code)
	}
}
