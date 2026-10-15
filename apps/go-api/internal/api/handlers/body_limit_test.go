// Package handlers_test — body_limit_test.go : plafond de corps des routes Huma
// à entrée RawBody (login, inscription, mot de passe, contexte de session).
//
// Contrat : une opération Huma qui lit son corps via RawBody est bornée par le
// plafond par défaut posé dans humacore (1 Mio) ; un corps de 2 Mio est refusé
// en 413 sans que le handler ne s'exécute. Un corps ordinaire reste servi.
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// oversizedJSON retourne un objet JSON valide de plus de 2 Mio.
func oversizedJSON() string {
	return `{"username":"` + strings.Repeat("a", 2<<20) + `"}`
}

func TestBodyLimit_RawBodyRoutesRefuse2MiB(t *testing.T) {
	auth := newLoginLimitRouter(t)
	session := newSessionContextRouter(t)

	cases := []struct {
		name   string
		router http.Handler
		path   string
	}{
		{"login", auth, "/auth/login"},
		{"register", auth, "/auth/register"},
		{"password", auth, "/auth/password"},
		{"session_context", session, "/session/context"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(oversizedJSON()))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = "127.0.0.1:40000"
			w := httptest.NewRecorder()
			tc.router.ServeHTTP(w, req)
			if w.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("POST %s avec un corps de 2 Mio : statut %d, attendu 413 (corps %.120s)",
					tc.path, w.Code, w.Body.String())
			}
		})
	}
}

func TestBodyLimit_OrdinaryBodyStillServed(t *testing.T) {
	r := newSessionContextRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/session/context", strings.NewReader(`{"locale":"fr"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("corps ordinaire : statut %d, attendu 200 (%s)", w.Code, w.Body.String())
	}
}
