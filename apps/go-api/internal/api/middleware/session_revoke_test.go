package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/platform/session"
)

// TestWithSession_RevokedDuringRequestExpiresCookieAndSkipsTouch : une session
// révoquée pendant la requête (déconnexion concurrente) reçoit un cookie expiré et
// n'est pas réécrite en fin de requête.
func TestWithSession_RevokedDuringRequestExpiresCookieAndSkipsTouch(t *testing.T) {
	store := newSessionStore(t)
	sess := store.New()
	u := "jgtm"
	sess.Username = &u
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save : %v", err)
	}
	handler := middleware.WithSession(store, middleware.SecureCookiePolicy{})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := store.Revoke(middleware.GetSession(r.Context()).SessionID); err != nil {
				t.Errorf("Revoke : %v", err)
			}
			w.WriteHeader(http.StatusOK)
		}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: store.SignCookie(sess.SessionID)})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	c := sessionCookie(rec)
	if c == nil || c.MaxAge >= 0 || c.Value != "" {
		t.Errorf("cookie de réponse = %+v, attendu un cookie vide et expiré", c)
	}
	if store.Load(t.Context(), sess.SessionID) != nil {
		t.Error("session réécrite en fin de requête après sa révocation")
	}
}

// TestWithSession_ConcurrentRequestFollowsRotatedSuccessor : une requête qui a chargé
// la session AVANT qu'une connexion concurrente n'en renouvelle l'identifiant répond
// par le cookie du successeur (le navigateur ne retombe pas sur l'ancien) et ne
// ressuscite pas l'ancienne session.
func TestWithSession_ConcurrentRequestFollowsRotatedSuccessor(t *testing.T) {
	store := newSessionStore(t)
	sess := store.New()
	sess.CurrentTitleSlug = "halo_5"
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save : %v", err)
	}
	oldID := sess.SessionID
	var newID string
	handler := middleware.WithSession(store, middleware.SecureCookiePolicy{})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			login := store.Load(t.Context(), oldID) // la connexion concurrente
			if _, err := store.Rotate(login); err != nil {
				t.Errorf("Rotate : %v", err)
			}
			newID = login.SessionID
			w.WriteHeader(http.StatusOK)
		}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: store.SignCookie(oldID)})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	c := sessionCookie(rec)
	if c == nil || store.UnsignCookie(c.Value) != newID || newID == oldID {
		t.Errorf("cookie de la requête concurrente = %+v, attendu celui du successeur %q", c, newID)
	}
	if store.Load(t.Context(), oldID) != nil {
		t.Error("ancienne session ressuscitée par la requête concurrente")
	}
}
