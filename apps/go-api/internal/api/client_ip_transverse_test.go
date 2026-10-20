// client_ip_transverse_test.go — l'IP cliente vue par la pile transverse RÉELLE
// (applyTransverseMiddlewares, server.go), derrière le proxy de confiance.
//
// Contrat éprouvé : avec LEVELUP_TRUST_PROXY_HEADERS, l'IP cliente est l'en-tête
// X-Real-IP, que nginx écrase à chaque requête ; aucun autre en-tête fourni par le
// client (True-Client-IP, X-Forwarded-For) n'entre dans la clé du limiteur de débit
// ni dans ce que lisent les handlers. Sans confiance, l'IP cliente est le pair TCP.
// Les tests passent par la pile assemblée et non par le limiteur seul : c'est
// l'ordre des middlewares qui décide de la clé.
package api

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	session_platform "levelup/go-api/internal/platform/session"
)

// clientIPProbePath est une route applicative (sous /api/, donc soumise au limiteur)
// qui renvoie l'IP cliente lue par les handlers.
const clientIPProbePath = "/api/v1/probe/client-ip"

// nginxPeer est le pair TCP vu par Go derrière nginx sur le même hôte.
const nginxPeer = "127.0.0.1:40000"

// newTransverseProbeRouter assemble la pile transverse de production autour d'une
// route sonde.
func newTransverseProbeRouter(t *testing.T, trustProxyHeaders bool) http.Handler {
	t.Helper()
	cfg := &config.AppConfig{TrustProxyHeaders: trustProxyHeaders}
	store := session_platform.NewStore(t.TempDir(), session_platform.DefaultTTL,
		"secret-de-test-de-trente-deux-octets!") // pragma: allowlist secret
	r := chi.NewRouter()
	applyTransverseMiddlewares(r, cfg, store, middleware.SecureCookiePolicy{}, titlePkg.DefaultRegistry())
	r.Get(clientIPProbePath, func(w http.ResponseWriter, req *http.Request) {
		_, _ = io.WriteString(w, chimiddleware.GetClientIP(req.Context()))
	})
	return r
}

// probe émet une requête sur la sonde avec le pair TCP et les en-têtes donnés.
func probe(t *testing.T, h http.Handler, peer string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, clientIPProbePath, nil)
	req.RemoteAddr = peer
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// remaining lit le solde du seau du limiteur dans la réponse.
func remaining(t *testing.T, rec *httptest.ResponseRecorder) int {
	t.Helper()
	raw := rec.Header().Get("X-RateLimit-Remaining")
	n, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("X-RateLimit-Remaining illisible %q (status %d) : la route sonde n'est pas sous le limiteur", raw, rec.Code)
	}
	return n
}

// TestTransverseClientIP_HandlersReadXRealIP : derrière le proxy, le handler lit
// l'IP de X-Real-IP, jamais celle des en-têtes forgés par le client.
func TestTransverseClientIP_HandlersReadXRealIP(t *testing.T) {
	h := newTransverseProbeRouter(t, true)
	rec := probe(t, h, nginxPeer, map[string]string{
		"True-Client-IP":  "198.51.100.66",
		"X-Forwarded-For": "192.0.2.77",
		"X-Real-IP":       "203.0.113.10",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != "203.0.113.10" {
		t.Errorf("IP cliente vue par le handler = %q, want %q (X-Real-IP)", got, "203.0.113.10")
	}
}

// TestTransverseRateLimit_KeyIsXRealIP : deux requêtes au même X-Real-IP et aux
// en-têtes forgés différents puisent dans le même seau ; un autre X-Real-IP a son
// propre seau.
func TestTransverseRateLimit_KeyIsXRealIP(t *testing.T) {
	h := newTransverseProbeRouter(t, true)
	first := remaining(t, probe(t, h, nginxPeer, map[string]string{
		"True-Client-IP": "198.51.100.1", "X-Forwarded-For": "192.0.2.1", "X-Real-IP": "203.0.113.20",
	}))
	second := remaining(t, probe(t, h, nginxPeer, map[string]string{
		"True-Client-IP": "198.51.100.2", "X-Forwarded-For": "192.0.2.2", "X-Real-IP": "203.0.113.20",
	}))
	if second != first-1 {
		t.Errorf("même X-Real-IP, en-têtes forgés différents : solde %d puis %d, want %d (un seul seau)",
			first, second, first-1)
	}
	other := remaining(t, probe(t, h, nginxPeer, map[string]string{
		"True-Client-IP": "198.51.100.1", "X-Forwarded-For": "192.0.2.1", "X-Real-IP": "203.0.113.21",
	}))
	if other != first {
		t.Errorf("autre X-Real-IP : solde %d, want %d (seau distinct)", other, first)
	}
}

// TestTransverseRateLimit_ForgedTrueClientIPDoesNotEscape : 301 requêtes au même
// X-Real-IP, chacune avec un True-Client-IP différent : la 301e reçoit 429.
func TestTransverseRateLimit_ForgedTrueClientIPDoesNotEscape(t *testing.T) {
	h := newTransverseProbeRouter(t, true)
	limit := middleware.RateLimitRequests
	for i := 1; i <= limit+1; i++ {
		rec := probe(t, h, nginxPeer, map[string]string{
			"True-Client-IP":  fmt.Sprintf("10.%d.%d.1", i/256, i%256),
			"X-Forwarded-For": fmt.Sprintf("172.16.%d.%d", i/256, i%256),
			"X-Real-IP":       "203.0.113.30",
		})
		if i <= limit && rec.Code != http.StatusOK {
			t.Fatalf("requête %d/%d : status %d, want 200", i, limit, rec.Code)
		}
		if i == limit+1 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("requête %d : status %d, want 429 (True-Client-IP variable ne doit pas ouvrir de seau neuf)",
				i, rec.Code)
		}
	}
}

// TestTransverseClientIP_UntrustedUsesTCPPeer : sans confiance dans le proxy,
// X-Real-IP est ignoré et l'IP cliente est le pair TCP.
func TestTransverseClientIP_UntrustedUsesTCPPeer(t *testing.T) {
	h := newTransverseProbeRouter(t, false)
	first := probe(t, h, "198.51.100.40:5000", map[string]string{
		"X-Real-IP": "203.0.113.40", "True-Client-IP": "203.0.113.41",
	})
	if got := first.Body.String(); got != "198.51.100.40" {
		t.Errorf("IP cliente sans confiance = %q, want le pair TCP %q", got, "198.51.100.40")
	}
	second := probe(t, h, "198.51.100.40:5001", map[string]string{"X-Real-IP": "203.0.113.42"})
	if remaining(t, second) != remaining(t, first)-1 {
		t.Errorf("sans confiance, un X-Real-IP différent ne doit pas ouvrir de seau neuf (soldes %d puis %d)",
			remaining(t, first), remaining(t, second))
	}
}

// TestTransverseClientIP_TrustedWithoutHeaderUsesTCPPeer : derrière le proxy, une
// requête qui n'est pas passée par lui (sonde locale, X-Real-IP absent ou
// illisible) garde le pair TCP comme IP cliente.
func TestTransverseClientIP_TrustedWithoutHeaderUsesTCPPeer(t *testing.T) {
	h := newTransverseProbeRouter(t, true)
	for _, headers := range []map[string]string{
		{},
		{"X-Real-IP": "pas-une-ip", "True-Client-IP": "203.0.113.50"},
	} {
		rec := probe(t, h, "172.17.0.1:6000", headers)
		if got := rec.Body.String(); got != "172.17.0.1" {
			t.Errorf("en-têtes %v : IP cliente = %q, want le pair TCP %q", headers, got, "172.17.0.1")
		}
	}
}
