// Package middleware_test — client_ip_logs_test.go : l'attribut `ip` des journaux
// de refus (CSRF, authentification, propriété, requête HTTP) est l'IP cliente de la
// pile (chimiddleware.GetClientIP), jamais le pair TCP ni un en-tête forgé.
//
// Derrière nginx, le pair TCP vaut l'adresse du proxy pour tous les clients : un
// journal qui le lirait perdrait l'IP réelle ; un journal qui lirait True-Client-IP
// écrirait celle que le client a choisie.
package middleware_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
)

const (
	logsClientIP  = "203.0.113.90"
	logsForgedIP  = "198.51.100.91"
	logsProxyPeer = "127.0.0.1:40000"
)

// behindProxy monte la résolution d'IP cliente de la pile transverse devant h.
func behindProxy(h http.Handler) http.Handler {
	return chimiddleware.ClientIPFromRemoteAddr(chimiddleware.ClientIPFromHeader("X-Real-IP")(h))
}

// noUsers est un magasin d'utilisateurs vide.
type noUsers struct{}

func (noUsers) Get(string) (*domain.User, error)       { return nil, nil }
func (noUsers) GetByXUID(string) (*domain.User, error) { return nil, nil }

func ok() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func newOwnershipRouter(t *testing.T) http.Handler {
	t.Helper()
	store := session.NewStore(filepath.Join(t.TempDir(), "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	unknownSlug := func(context.Context, string) (string, bool) { return "", false }
	r := chi.NewRouter()
	r.Use(middleware.WithSession(store, middleware.SecureCookiePolicy{}))
	r.With(middleware.RequirePlayerOwnership(middleware.NewPlayerAccess(false, "password", unknownSlug, noUsers{}, nil))).
		Get("/players/{player_slug}/x", ok().ServeHTTP)
	return r
}

// TestClientIPLogs_RejectionsCarryClientIP : chaque refus journalise `ip` = X-Real-IP.
func TestClientIPLogs_RejectionsCarryClientIP(t *testing.T) {
	cases := []struct {
		name, method, path, msg string
		handler                 func(t *testing.T) http.Handler
	}{
		{"csrf", http.MethodPost, "/api/v1/x", "middleware.csrf: origin non autorisée",
			func(*testing.T) http.Handler { return middleware.CSRF(nil)(ok()) }},
		{"admin", http.MethodGet, "/api/v1/admin/x", "admin: rejet 403",
			func(*testing.T) http.Handler { return middleware.RequireAdmin(false, "password")(ok()) }},
		{"auth", http.MethodGet, "/api/v1/x", "auth: rejet 401 — session non authentifiée",
			func(*testing.T) http.Handler { return middleware.RequireAuth(false, "password")(ok()) }},
		{"auth-mutations", http.MethodPost, "/api/v1/players/p/sync", "authz: écriture refusée — aucune session authentifiée",
			func(*testing.T) http.Handler { return middleware.RequireAuthForMutations(false, "password")(ok()) }},
		{"ownership", http.MethodGet, "/players/inconnu/x", "authz: slug joueur inconnu refusé", newOwnershipRouter},
		{"http", http.MethodGet, "/api/v1/x", "http",
			func(*testing.T) http.Handler {
				return middleware.SlogLogger(middleware.RequireAdmin(false, "password")(ok()))
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := captureLogs(t, slog.LevelDebug)
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
			req.RemoteAddr = logsProxyPeer
			req.Header.Set("X-Real-IP", logsClientIP)
			req.Header.Set("True-Client-IP", logsForgedIP)
			req.Header.Set("X-Forwarded-For", logsForgedIP)
			rec := httptest.NewRecorder()
			behindProxy(tc.handler(t)).ServeHTTP(rec, req)
			if rec.Code < http.StatusBadRequest {
				t.Fatalf("la requête devait être refusée, status %d", rec.Code)
			}
			recs := logRecords(t, buf, tc.msg)
			if len(recs) == 0 {
				t.Fatalf("aucun journal %q (journaux : %s)", tc.msg, buf.String())
			}
			for _, rec := range recs {
				if got := rec["ip"]; got != logsClientIP {
					t.Errorf("journal %q : ip = %v, want %q (record %v)", tc.msg, got, logsClientIP, rec)
				}
			}
			if strings.Contains(buf.String(), logsForgedIP) {
				t.Errorf("l'IP forgée %s apparaît dans les journaux : %s", logsForgedIP, buf.String())
			}
		})
	}
}
