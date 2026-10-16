package handlers

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/api/middleware"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/service"
)

// oauthFinishRig : la fin du callback OAuth (après les échanges avec Microsoft, sans
// réseau), derrière le vrai middleware de session.
func oauthFinishRig(t *testing.T, locked bool) (http.Handler, *session.Store, *userstore.Store) {
	t.Helper()
	dir := t.TempDir()
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	h := NewXboxOAuthHandler(sessStore, nil, false, "http://localhost/cb").
		WithLinkStrategy(service.NewXboxSSOLinkStrategy(users).WithInstanceLock(func() bool { return locked }))
	attempt := &auth_platform.Attempt{
		Status: auth_platform.AttemptStatusAuthorized, Gamertag: "Oauther", XUID: "2535400000000099",
		SpartanToken: "spartan",
	}
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.finishOAuthLogin(w, r, middleware.GetSession(r.Context()), attempt) {
			http.Redirect(w, r, "/", http.StatusFound)
		}
	})
	return middleware.WithSession(sessStore, middleware.SecureCookiePolicy{})(final), sessStore, users
}

func plantedOAuthSession(t *testing.T, sessStore *session.Store) (string, *http.Cookie) {
	t.Helper()
	sess := sessStore.New()
	sess.CurrentTitleSlug = "halo_5"
	if err := sessStore.Save(sess); err != nil {
		t.Fatalf("save : %v", err)
	}
	return sess.SessionID, &http.Cookie{Name: session.CookieName, Value: sessStore.SignCookie(sess.SessionID)}
}

// TestOAuthFinish_RotatesSessionID : la connexion OAuth réussie renouvelle
// l'identifiant ; l'ancien ne désigne plus rien ; l'état de navigation suit.
func TestOAuthFinish_RotatesSessionID(t *testing.T) {
	handler, sessStore, _ := oauthFinishRig(t, false)
	oldID, cookie := plantedOAuthSession(t, sessStore)

	req := httptest.NewRequest(http.MethodGet, "/auth/xbox/callback", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("statut %d (%s)", w.Code, w.Body.String())
	}
	newID := ""
	for _, c := range w.Result().Cookies() {
		if c.Name == session.CookieName {
			newID = sessStore.UnsignCookie(c.Value)
		}
	}
	if newID == "" || newID == oldID {
		t.Fatalf("identifiant après connexion OAuth = %q, attendu nouveau (avant %q)", newID, oldID)
	}
	if sessStore.Load(t.Context(), oldID) != nil {
		t.Error("ancienne session encore présente")
	}
	fresh := sessStore.Load(t.Context(), newID)
	if fresh == nil || !fresh.AuthReady || fresh.Username == nil || fresh.CurrentTitleSlug != "halo_5" {
		t.Errorf("nouvelle session incomplète : %+v", fresh)
	}
}

// TestOAuthFinish_LockedLeavesSessionUntouched : identité inconnue, instance
// verrouillée : retour à l'écran de connexion avec le code instance_locked, session
// inchangée, aucune rotation.
func TestOAuthFinish_LockedLeavesSessionUntouched(t *testing.T) {
	handler, sessStore, users := oauthFinishRig(t, true)
	oldID, cookie := plantedOAuthSession(t, sessStore)

	req := httptest.NewRequest(http.MethodGet, "/auth/xbox/callback", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/login?oauth_error=instance_locked" {
		t.Fatalf("statut %d vers %q, attendu 302 vers /login?oauth_error=instance_locked (%s)",
			w.Code, w.Header().Get("Location"), w.Body.String())
	}
	sess := sessStore.Load(t.Context(), oldID)
	if sess == nil || sess.AuthReady || sess.LinkedHaloIdentity != nil || sess.HaloTokens != nil {
		t.Errorf("session touchée par une connexion refusée : %+v", sess)
	}
	if _, err := users.GetByXUID("2535400000000099"); err == nil {
		t.Error("compte créé malgré le verrou")
	}
}

// TestOAuthFinish_InvalidInviteOnLockedInstance : identité inconnue, instance verrouillée,
// la session porte une invitation qui ne vaut plus rien : le refus le dit
// (invite_invalid) au lieu d'annoncer une instance fermée ; aucun compte créé.
func TestOAuthFinish_InvalidInviteOnLockedInstance(t *testing.T) {
	dir := t.TempDir()
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	invites := userstore.NewInviteStore(filepath.Join(dir, "invites.json"), users)
	h := NewXboxOAuthHandler(sessStore, nil, false, "http://localhost/cb").
		WithLinkStrategy(service.NewXboxSSOLinkStrategy(users).
			WithInstanceLock(func() bool { return true }).
			WithInviteStore(invites))
	attempt := &auth_platform.Attempt{
		Status: auth_platform.AttemptStatusAuthorized, Gamertag: "Invite", XUID: "2535400000000077",
	}
	handler := middleware.WithSession(sessStore, middleware.SecureCookiePolicy{})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if h.finishOAuthLogin(w, r, middleware.GetSession(r.Context()), attempt) {
				http.Redirect(w, r, "/", http.StatusFound)
			}
		}))
	sess := sessStore.New()
	sess.PendingInviteCode = "EXPIRE00"
	if err := sessStore.Save(sess); err != nil {
		t.Fatalf("save : %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/auth/xbox/callback", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: sessStore.SignCookie(sess.SessionID)})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/login?oauth_error=invite_invalid" {
		t.Fatalf("statut %d vers %q, attendu 302 vers /login?oauth_error=invite_invalid (%s)",
			w.Code, w.Header().Get("Location"), w.Body.String())
	}
	if _, err := users.GetByXUID("2535400000000077"); err == nil {
		t.Error("compte créé malgré le verrou et une invitation invalide")
	}
}
