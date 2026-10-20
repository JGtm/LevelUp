// Package handlers_test — session_context_access_test.go : contrôle d'accès du joueur
// courant (POST /session/context).
//
// Contrat : un `player_slug` que la session ne peut pas LIRE (ni propriétaire, ni
// co-membre de groupe, ni admin) est refusé en 403 player_forbidden, et la session
// n'est pas modifiée. Un slug accessible (son profil, celui d'un ami) est accepté.
package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/platform/session"
)

// readableSlugs : le contrôle câblé par le routeur, réduit à un ensemble fixe.
func readableSlugs(slugs ...string) handlers.PlayerSelectionCheck {
	set := map[string]bool{}
	for _, s := range slugs {
		set[s] = true
	}
	return func(_ context.Context, slug string) bool { return set[slug] }
}

func newSessionContextRouterWithAccess(t *testing.T, check handlers.PlayerSelectionCheck) (*chi.Mux, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "sessions")
	store := session.NewStore(dir, time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	r := chi.NewRouter()
	r.Use(middleware.WithSession(store, middleware.SecureCookiePolicy{}))
	handlers.NewSessionHandler(store).WithPlayerAccess(check).Mount(r)
	return r, dir
}

func TestSessionContext_ForeignPlayerSlugIs403(t *testing.T) {
	r, dir := newSessionContextRouterWithAccess(t, readableSlugs("alice", "bob"))
	w := postSessionContext(r, `{"player_slug":"mallory"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("statut %d, attendu 403 (%s)", w.Code, w.Body.String())
	}
	var body struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Code != "player_forbidden" {
		t.Fatalf("code %q, attendu player_forbidden", body.Code)
	}
	if n := sessionFileCount(t, dir); n != 0 {
		t.Fatalf("%d fichier(s) de session écrit(s) sur un refus", n)
	}
}

func TestSessionContext_ReadablePlayerSlugAccepted(t *testing.T) {
	for _, slug := range []string{"alice", "bob"} {
		r, _ := newSessionContextRouterWithAccess(t, readableSlugs("alice", "bob"))
		w := postSessionContext(r, `{"player_slug":"`+slug+`"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("%s : statut %d, attendu 200 (%s)", slug, w.Code, w.Body.String())
		}
	}
}
