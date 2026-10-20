// Package handlers_test — auth_device_flow_limits_test.go : plafonds de tentatives
// en attente de POST /auth/device-flow/start.
//
// Contrat : au plus auth.MaxPendingPerIP tentatives en attente par IP cliente et
// auth.MaxPendingAttempts au total ; au-delà, 429 avec Retry-After et AUCUN appel
// sortant (InitDeviceFlow n'est pas appelé). Un rafraîchissement de la page de
// connexion (même cookie) reprend la tentative en attente, hors plafond.
package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
)

// blockingFlow reste en attente d'autorisation jusqu'à l'annulation du contexte :
// la tentative demeure pending, comme un code Microsoft non saisi.
type blockingFlow struct{ auth_platform.DeviceFlow }

func (blockingFlow) AcquireToken(ctx context.Context) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

// countingProvider compte les appels sortants (InitDeviceFlow) et rend un flux bloquant.
type countingProvider struct {
	stubTokenProvider
	inits atomic.Int32
}

func (p *countingProvider) InitDeviceFlow(context.Context) (auth_platform.DeviceFlow, error) {
	p.inits.Add(1)
	return blockingFlow{auth_platform.NewStubDeviceFlow("CODE1234", "https://microsoft.com/link", "msg", 900, "sisu")}, nil
}

func newDeviceFlowLimitRouter(t *testing.T, p auth_platform.TokenProvider) http.Handler {
	t.Helper()
	sessStore := session.NewStore(filepath.Join(t.TempDir(), "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	h := handlers.NewAuthHandler(sessStore, auth_platform.NewAttemptStore(), p)
	r := chi.NewRouter()
	r.Use(chimiddleware.ClientIPFromRemoteAddr)
	r.Use(chimiddleware.ClientIPFromHeader("X-Real-IP"))
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	h.MountDeviceFlow(r)
	return r
}

func startDeviceFlowFrom(r http.Handler, ip string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/auth/device-flow/start", nil)
	req.RemoteAddr = "127.0.0.1:40000"
	req.Header.Set("X-Real-IP", ip)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestStartDeviceFlow_PerIPCap(t *testing.T) {
	p := &countingProvider{}
	r := newDeviceFlowLimitRouter(t, p)
	const ip = "203.0.113.7"

	var first *httptest.ResponseRecorder
	for i := range auth_platform.MaxPendingPerIP {
		w := startDeviceFlowFrom(r, ip)
		if w.Code != http.StatusOK {
			t.Fatalf("démarrage %d: statut %d, attendu 200: %s", i+1, w.Code, w.Body.String())
		}
		if i == 0 {
			first = w
		}
	}

	w := startDeviceFlowFrom(r, ip)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("4e démarrage de la même IP: statut %d, attendu 429: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("le refus 429 doit porter Retry-After")
	}
	if got := p.inits.Load(); got != int32(auth_platform.MaxPendingPerIP) {
		t.Errorf("appels sortants = %d, attendu %d (le refus ne doit rien appeler)", got, auth_platform.MaxPendingPerIP)
	}

	// Rafraîchissement de la page : même cookie, même tentative, hors plafond.
	w = startDeviceFlowFrom(r, ip, first.Result().Cookies()...)
	if w.Code != http.StatusOK {
		t.Fatalf("reprise avec le cookie: statut %d, attendu 200: %s", w.Code, w.Body.String())
	}
	if got := p.inits.Load(); got != int32(auth_platform.MaxPendingPerIP) {
		t.Errorf("la reprise a rappelé l'extérieur: appels = %d", got)
	}

	if w := startDeviceFlowFrom(r, "198.51.100.2"); w.Code != http.StatusOK {
		t.Fatalf("une autre IP doit rester servie: statut %d", w.Code)
	}
}

func TestStartDeviceFlow_GlobalCap(t *testing.T) {
	p := &countingProvider{}
	r := newDeviceFlowLimitRouter(t, p)

	for i := range auth_platform.MaxPendingAttempts {
		w := startDeviceFlowFrom(r, fmt.Sprintf("198.51.100.%d", i+1))
		if w.Code != http.StatusOK {
			t.Fatalf("démarrage %d: statut %d, attendu 200: %s", i+1, w.Code, w.Body.String())
		}
	}
	w := startDeviceFlowFrom(r, "192.0.2.99")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("21e démarrage: statut %d, attendu 429: %s", w.Code, w.Body.String())
	}
	if got := p.inits.Load(); got != int32(auth_platform.MaxPendingAttempts) {
		t.Errorf("appels sortants = %d, attendu %d", got, auth_platform.MaxPendingAttempts)
	}
}
