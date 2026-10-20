// Package handlers_test — session_context_validation_test.go : validation de forme
// du corps de POST /session/context.
//
// Contrat : `locale` appartient à la liste fermée {fr, en} ; `player_slug` tient en
// 64 caractères d'un alphabet de slug (lettres, chiffres, espace, _ - . #). Sinon 400
// et la session n'est ni modifiée ni écrite. Le contrôle d'ACCÈS au joueur est un
// autre sujet (autorisation, hors de ce fichier).
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/platform/session"
)

// newSessionContextRouterWithDir expose le répertoire des sessions pour vérifier
// qu'un refus n'écrit rien.
func newSessionContextRouterWithDir(t *testing.T) (*chi.Mux, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "sessions")
	store := session.NewStore(dir, time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	r := chi.NewRouter()
	r.Use(middleware.WithSession(store, middleware.SecureCookiePolicy{}))
	handlers.NewSessionHandler(store).Mount(r)
	return r, dir
}

func postSessionContext(r http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/session/context", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func sessionFileCount(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture du répertoire de sessions : %v", err)
	}
	return len(entries)
}

func TestSessionContext_RejectsMalformedValues(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"locale inconnue", `{"locale":"xx"}`},
		{"locale composée", `{"locale":"fr-FR"}`},
		{"locale vide", `{"locale":""}`},
		{"locale volumineuse", `{"locale":"` + strings.Repeat("a", 5000) + `"}`},
		{"slug vide", `{"player_slug":""}`},
		{"slug de 65 caractères", `{"player_slug":"` + strings.Repeat("a", 65) + `"}`},
		{"slug volumineux", `{"player_slug":"` + strings.Repeat("a", 100000) + `"}`},
		{"slug avec traversée", `{"player_slug":"../etc"}`},
		{"slug avec séparateur", `{"player_slug":"a/b"}`},
		{"slug avec balise", `{"player_slug":"<script>"}`},
		{"slug avec contrôle", `{"player_slug":"a\u0000b"}`},
		{"slug de points", `{"player_slug":".."}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, dir := newSessionContextRouterWithDir(t)
			w := postSessionContext(r, tc.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("statut %d, attendu 400 (%.150s)", w.Code, w.Body.String())
			}
			if n := sessionFileCount(t, dir); n != 0 {
				t.Errorf("une requête refusée a écrit %d fichier(s) de session", n)
			}
		})
	}
}

func TestSessionContext_AcceptsRealValues(t *testing.T) {
	for _, body := range []string{
		`{"locale":"fr"}`,
		`{"locale":"en"}`,
		`{"player_slug":"JGtm"}`,
		`{"player_slug":"test-player"}`,
		`{"player_slug":"Xx Daemon_Gamer-1"}`,
		`{"player_slug":"Chocoboflor"}`,
		`{"player_slug":"Élodie#1234"}`,
		`{"player_slug":"` + strings.Repeat("a", 64) + `"}`,
		`{"title_slug":"halo_infinite"}`,
		`{}`,
	} {
		r, _ := newSessionContextRouterWithDir(t)
		if w := postSessionContext(r, body); w.Code != http.StatusOK {
			t.Errorf("%s : statut %d, attendu 200 (%.150s)", body, w.Code, w.Body.String())
		}
	}
}
