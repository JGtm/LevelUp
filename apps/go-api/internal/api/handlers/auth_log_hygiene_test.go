// Package handlers_test — auth_log_hygiene_test.go : les journaux d'authentification
// ne portent ni un identifiant de connexion non borné (fourni par le client), ni un
// code d'invitation (il vaut accès).
package handlers_test

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
)

// captureLogs redirige le logger par défaut le temps du test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// TestAuthLogs_LongIdentifierTruncated : un identifiant de 100 000 caractères n'est
// journalisé que sur 64 (connexion et inscription).
func TestAuthLogs_LongIdentifierTruncated(t *testing.T) {
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	if _, err := users.Create("boss", testUserAuthPass, domain.RoleAdmin); err != nil {
		t.Fatalf("Create : %v", err)
	}
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	h := handlers.NewUserAuthHandler(users, userstore.NewInviteStore(filepath.Join(dir, "invites.json"), users), sessStore, "closed").
		WithAuthMode("password")
	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	h.Mount(r)

	logs := captureLogs(t)
	huge := strings.Repeat("z", 100_000)
	postJSON(t, r, "/auth/login", domain.LoginRequest{Username: huge, Password: "wrong-password"})
	postJSON(t, r, "/auth/register", domain.RegisterRequest{Username: huge, Password: testUserAuthPass})

	if strings.Contains(logs.String(), strings.Repeat("z", 65)) {
		t.Errorf("identifiant journalisé au-delà de 64 caractères (%d octets de journal)", logs.Len())
	}
	if !strings.Contains(logs.String(), strings.Repeat("z", 64)) {
		t.Error("identifiant absent des journaux (attendu tronqué, pas retiré)")
	}
}

// TestAdminLogs_InviteCodeNotLogged : générer puis révoquer une invitation ne
// journalise jamais son code.
func TestAdminLogs_InviteCodeNotLogged(t *testing.T) {
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	h := handlers.NewAdminHandler(users, userstore.NewInviteStore(filepath.Join(dir, "invites.json"), users))
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) { h.Mount(r) })

	logs := captureLogs(t)
	w := serveJSON(r, "POST", "/admin/invites", `{"expires_in_days":3}`)
	if w.Code != 201 {
		t.Fatalf("génération : %d (%s)", w.Code, w.Body.String())
	}
	code := between(w.Body.String(), `"code":"`, `"`)
	if code == "" {
		t.Fatalf("code absent de la réponse : %s", w.Body.String())
	}
	serveJSON(r, "DELETE", "/admin/invites/"+code, "")
	if strings.Contains(logs.String(), code) {
		t.Errorf("code d'invitation journalisé : %s", logs.String())
	}
}

func between(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	rest := s[i+len(start):]
	j := strings.Index(rest, end)
	if j < 0 {
		return ""
	}
	return rest[:j]
}
