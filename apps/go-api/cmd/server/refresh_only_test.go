// refresh_only_test.go — le mode -mode refresh-only (ADR 0045 D-10) : une racine temporaire, un
// magasin de jetons et un faux serveur OAuth (httptest) qui fait tourner le jeton à usage unique,
// comme Microsoft. Le mode prend le verrou d'instance, ne touche à aucune base, sort en 0 dans tous
// les cas d'issue des jetons, et ne fait rien quand l'application tient déjà le verrou.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/instancelock"
)

const refreshOnlyXUID = "2533274800000001"

// fakeOAuth : l'endpoint /token de Microsoft, réduit à l'essentiel. Un refresh token vaut une fois :
// l'accepter en rend un nouveau et invalide l'ancien (invalid_grant si on le présente de nouveau).
type fakeOAuth struct {
	srv *httptest.Server

	mu    sync.Mutex
	valid string // le seul refresh token accepté
	next  string // celui qu'on rend à la place
	hits  int
	dead  bool // tout refresh token est refusé (invalid_grant)
}

func newFakeOAuth(t *testing.T, valid, next string) *fakeOAuth {
	t.Helper()
	f := &fakeOAuth{valid: valid, next: next}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeOAuth) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hits++
	_ = r.ParseForm()
	w.Header().Set("Content-Type", "application/json")
	if f.dead || r.PostForm.Get("refresh_token") != f.valid {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant", "error_description": "AADSTS70000"})
		return
	}
	f.valid = f.next
	_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "refresh_token": f.next, "expires_in": 3600})
}

func (f *fakeOAuth) hitCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.hits
}

// httpProvider : un fournisseur dont le rafraîchissement OAuth interroge le faux serveur.
type httpProvider struct{ url string }

func (p httpProvider) InitDeviceFlow(context.Context) (auth.DeviceFlow, error) { return nil, nil }
func (p httpProvider) TryOAuthRefresh(context.Context, string) (string, error) { return "", nil }
func (p httpProvider) Exchange(context.Context, string) (*auth.ExchangeResult, error) {
	return nil, errors.New("pas d'échange Halo en refresh-only")
}

func (p httpProvider) TryOAuthRefreshWithRotation(ctx context.Context, rt string) (string, string, error) {
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {rt}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err // réseau absent : erreur de transport non typée
	}
	defer resp.Body.Close()
	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error"`
		Description  string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", "", err
	}
	if body.Error != "" {
		return "", "", &auth.OAuthExchangeError{ErrorCode: body.Error, Description: body.Description}
	}
	return body.AccessToken, body.RefreshToken, nil
}

// refreshOnlyRoot prépare une racine neuve en -mode refresh-only : un profil suivi, un magasin de
// jetons à rt, une configuration chargée comme le fait le binaire. Rend la configuration et le magasin.
func refreshOnlyRoot(t *testing.T, rt string) (*config.AppConfig, *auth.MultiUserTokenStore) {
	t.Helper()
	root := t.TempDir()
	for _, k := range envCheminsDeBoot {
		t.Setenv(k, "")
	}
	t.Setenv("LEVELUP_ENV", "")
	t.Setenv("LEVELUP_AUTH_MODE", "")
	t.Setenv("LEVELUP_REPO_ROOT", root)
	cfg, err := config.LoadServer(config.LaunchModeRefreshOnly)
	if err != nil {
		t.Fatalf("config.LoadServer : %v", err)
	}
	if !cfg.Capabilities.WakeSequence {
		t.Fatal("refresh-only doit résoudre le profil desktop")
	}
	profiles := `{"version":"3.0","admin":"Alice","profiles":{"halo_infinite":{"Alice":{` +
		`"db_path":"data/titles/halo_infinite/players/Alice/stats.duckdb","xuid":"` + refreshOnlyXUID + `","waypoint_player":"Alice"}}}}`
	if err := os.WriteFile(cfg.DBProfilesPath, []byte(profiles), 0o600); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(cfg.WatcherTokensDir())
	if err := store.Upsert(&auth.UserTokens{XUID: refreshOnlyXUID, Gamertag: "Alice", OAuthRefreshToken: rt}); err != nil {
		t.Fatal(err)
	}
	return cfg, store
}

func storedRT(t *testing.T, store *auth.MultiUserTokenStore) string {
	t.Helper()
	u, err := store.Load(refreshOnlyXUID)
	if err != nil {
		t.Fatal(err)
	}
	return u.OAuthRefreshToken
}

// assertNoDatabase : aucune base (ni *.duckdb ni *.wal) n'existe sous la racine de données.
func assertNoDatabase(t *testing.T, cfg *config.AppConfig) {
	t.Helper()
	root := cfg.Roots().Data
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			name := strings.ToLower(d.Name())
			if strings.HasSuffix(name, ".duckdb") || strings.HasSuffix(name, ".wal") {
				t.Errorf("refresh-only a ouvert une base : %s", p)
			}
		}
		return nil
	})
	if _, err := os.Stat(filepath.Join(root, "titles")); err == nil {
		t.Errorf("refresh-only a créé %s", filepath.Join(root, "titles"))
	}
}

// TestRefreshOnly_RoteLeJeton : le jeton vivant est présenté une fois, le jeton tourné est écrit au
// magasin, le code de sortie est 0, aucune base n'est ouverte et le verrou est rendu.
func TestRefreshOnly_RoteLeJeton(t *testing.T) {
	cfg, store := refreshOnlyRoot(t, "rt-1")
	oauth := newFakeOAuth(t, "rt-1", "rt-2")

	if code := refreshOnly(context.Background(), cfg, httpProvider{url: oauth.srv.URL}); code != 0 {
		t.Fatalf("code de sortie = %d, attendu 0", code)
	}

	if got := storedRT(t, store); got != "rt-2" {
		t.Errorf("refresh token du magasin = %q, attendu le jeton tourné rt-2", got)
	}
	if oauth.hitCount() != 1 {
		t.Errorf("appels OAuth = %d, attendu 1", oauth.hitCount())
	}
	assertNoDatabase(t, cfg)
	lock, err := instancelock.Acquire(cfg.PathResolver().InstanceLockPath())
	if err != nil {
		t.Fatalf("verrou non rendu à la sortie : %v", err)
	}
	_ = lock.Release()
}

// TestRefreshOnly_VerrouTenu : l'application est ouverte (elle tient le verrou) — sortie 0, sans
// rotation : le jeton n'est présenté à Microsoft par aucun second processus.
func TestRefreshOnly_VerrouTenu(t *testing.T) {
	cfg, store := refreshOnlyRoot(t, "rt-1")
	oauth := newFakeOAuth(t, "rt-1", "rt-2")
	held, err := instancelock.Acquire(cfg.PathResolver().InstanceLockPath())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = held.Release() })

	if code := refreshOnly(context.Background(), cfg, httpProvider{url: oauth.srv.URL}); code != 0 {
		t.Fatalf("code de sortie = %d, attendu 0", code)
	}

	if got := storedRT(t, store); got != "rt-1" {
		t.Errorf("refresh token du magasin = %q, attendu rt-1 inchangé", got)
	}
	if oauth.hitCount() != 0 {
		t.Errorf("appels OAuth = %d, attendu 0 (une seule rotation à la fois)", oauth.hitCount())
	}
	assertNoDatabase(t, cfg)
}

// TestRefreshOnly_JetonMort : Microsoft refuse le jeton — reauth_required est posé (la bannière de
// reconnexion), le jeton n'est pas remplacé, la sortie reste 0.
func TestRefreshOnly_JetonMort(t *testing.T) {
	cfg, store := refreshOnlyRoot(t, "rt-1")
	oauth := newFakeOAuth(t, "rt-1", "rt-2")
	oauth.dead = true

	if code := refreshOnly(context.Background(), cfg, httpProvider{url: oauth.srv.URL}); code != 0 {
		t.Fatalf("code de sortie = %d, attendu 0", code)
	}

	if !store.IsReauthRequired(refreshOnlyXUID) {
		t.Error("reauth_required n'est pas posé pour un jeton mort")
	}
	if got := storedRT(t, store); got != "rt-1" {
		t.Errorf("refresh token du magasin = %q, attendu rt-1 inchangé", got)
	}
}

// TestRefreshOnly_ReseauAbsent : aucun serveur ne répond — rien n'est marqué, le jeton reste, la
// sortie est 0 (la tâche planifiée repassera).
func TestRefreshOnly_ReseauAbsent(t *testing.T) {
	cfg, store := refreshOnlyRoot(t, "rt-1")
	oauth := newFakeOAuth(t, "rt-1", "rt-2")
	oauth.srv.Close() // le port ne répond plus

	if code := refreshOnly(context.Background(), cfg, httpProvider{url: oauth.srv.URL}); code != 0 {
		t.Fatalf("code de sortie = %d, attendu 0", code)
	}

	u, err := store.Load(refreshOnlyXUID)
	if err != nil {
		t.Fatal(err)
	}
	if u.ReauthRequired || u.LastAuthErrorClass != "" || u.OAuthRefreshToken != "rt-1" {
		t.Errorf("le magasin a été touché sans réponse du réseau : reauth=%v dernier échec=%q jeton=%q",
			u.ReauthRequired, u.LastAuthErrorClass, u.OAuthRefreshToken)
	}
}
