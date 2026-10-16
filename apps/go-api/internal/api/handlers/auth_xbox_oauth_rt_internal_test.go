package handlers

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/service"
)

// recordingTokenStore enregistre les persistances de refresh token, et dit si le
// compte du xuid existait au moment de chacune.
type recordingTokenStore struct {
	users   *userstore.Store
	written []string
	before  []bool
}

func (s *recordingTokenStore) Load(string) (*auth_platform.UserTokens, error) { return nil, nil }
func (s *recordingTokenStore) ClearReauthRequired(string) error               { return nil }
func (s *recordingTokenStore) UpdateOAuthRefreshToken(xuid, _ string) error {
	_, err := s.users.GetByXUID(xuid)
	s.written = append(s.written, xuid)
	s.before = append(s.before, err == nil)
	return nil
}

func runOAuthCompletion(t *testing.T, locked bool) (*httptest.ResponseRecorder, *recordingTokenStore) {
	t.Helper()
	dir := t.TempDir()
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	tokens := &recordingTokenStore{users: users}
	h := NewXboxOAuthHandler(sessStore, nil, false, "http://localhost/cb").
		WithLinkStrategy(service.NewXboxSSOLinkStrategy(users).WithInstanceLock(func() bool { return locked })).
		WithAuthStore(tokens)
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok := h.completeOAuthCallback(w, r, middleware.GetSession(r.Context()),
			&auth_platform.AuthCodeResult{AccessToken: "at", RefreshToken: "fresh-rt"},
			&auth_platform.ExchangeResult{Tokens: &domain.HaloTokens{SpartanToken: "sp"}, Gamertag: "Stranger", XUID: "2535400000000123"},
			nil)
		if ok {
			w.WriteHeader(http.StatusFound)
		}
	})
	w := httptest.NewRecorder()
	middleware.WithSession(sessStore, middleware.SecureCookiePolicy{})(final).
		ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/xbox/callback", nil))
	return w, tokens
}

// TestOAuthCallback_LockedPersistsNoRefreshToken : identité inconnue refusée par le
// verrou : aucun fichier de jetons n'est écrit pour elle (ADR 0023, registre C5-6).
func TestOAuthCallback_LockedPersistsNoRefreshToken(t *testing.T) {
	w, tokens := runOAuthCompletion(t, true)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/login?oauth_error=instance_locked" {
		t.Fatalf("statut %d vers %q, attendu 302 vers /login?oauth_error=instance_locked", w.Code, w.Header().Get("Location"))
	}
	if len(tokens.written) != 0 {
		t.Errorf("refresh token persisté pour une identité refusée : %v", tokens.written)
	}
}

// TestOAuthCallback_AcceptedPersistsRefreshTokenAfterDecision : connexion acceptée :
// le refresh token est persisté une fois, le compte existant déjà.
func TestOAuthCallback_AcceptedPersistsRefreshTokenAfterDecision(t *testing.T) {
	w, tokens := runOAuthCompletion(t, false)
	if w.Code != http.StatusFound {
		t.Fatalf("statut %d, attendu 302", w.Code)
	}
	if len(tokens.written) != 1 || tokens.written[0] != "2535400000000123" {
		t.Fatalf("persistances = %v, attendu une pour le xuid connecté", tokens.written)
	}
	if !tokens.before[0] {
		t.Error("refresh token persisté AVANT la décision de connexion (compte encore absent)")
	}
}
