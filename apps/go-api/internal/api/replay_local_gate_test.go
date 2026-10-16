// replay_local_gate_test.go — le garde « rejeu servi en local seulement »
// (handlers.LocalOnlyReplay) éprouvé derrière la pile transverse RÉELLE
// (applyTransverseMiddlewares), avec et sans confiance dans le proxy.
//
// Le garde décide sur le pair TCP de la connexion. Le tester sans la pile ne
// prouverait rien : c'est un middleware amont qui, en réécrivant l'adresse depuis un
// en-tête, rendait le garde contournable.
package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	session_platform "levelup/go-api/internal/platform/session"
)

const replayGatePath = "/api/v1/players/p/matches/m/replay"

// newReplayGateRouter monte le garde de rejeu hors démo (ReplayGate(nil)) sous la
// pile transverse de production.
func newReplayGateRouter(t *testing.T, trustProxyHeaders bool) http.Handler {
	t.Helper()
	if os.Getenv("LEVELUP_REPLAY_PUBLIC") == "1" {
		t.Fatal("LEVELUP_REPLAY_PUBLIC=1 lève le garde : ce test exige l'environnement par défaut")
	}
	cfg := &config.AppConfig{TrustProxyHeaders: trustProxyHeaders}
	store := session_platform.NewStore(t.TempDir(), session_platform.DefaultTTL,
		"secret-de-test-de-trente-deux-octets!") // pragma: allowlist secret
	r := chi.NewRouter()
	applyTransverseMiddlewares(r, cfg, store, middleware.SecureCookiePolicy{}, titlePkg.DefaultRegistry())
	r.With(handlers.ReplayGate(nil)).Get("/api/v1/players/{player_slug}/matches/{match_id}/replay",
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	return r
}

func replayRequest(h http.Handler, peer string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, replayGatePath, nil)
	req.RemoteAddr = peer
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// forgedLoopbackHeaders : tout ce qu'un client distant peut écrire pour se dire local.
var forgedLoopbackHeaders = map[string]string{
	"True-Client-IP":  "127.0.0.1",
	"X-Real-IP":       "127.0.0.1",
	"X-Forwarded-For": "127.0.0.1",
}

// TestReplayLocalGate_AllowsLoopbackPeer : un pair TCP local obtient le rejeu, avec
// ou sans confiance dans le proxy.
func TestReplayLocalGate_AllowsLoopbackPeer(t *testing.T) {
	for _, trust := range []bool{false, true} {
		h := newReplayGateRouter(t, trust)
		for _, peer := range []string{"127.0.0.1:5555", "[::1]:5555"} {
			if rec := replayRequest(h, peer, nil); rec.Code != http.StatusOK {
				t.Errorf("trust=%v pair %q : status %d, want 200 (body=%s)", trust, peer, rec.Code, rec.Body.String())
			}
		}
	}
}

// TestReplayLocalGate_RejectsUnreadablePeer : une adresse de pair vide ou illisible n'est
// pas locale (garde fermée par défaut), même accompagnée d'en-têtes qui se disent locaux.
func TestReplayLocalGate_RejectsUnreadablePeer(t *testing.T) {
	for _, trust := range []bool{false, true} {
		h := newReplayGateRouter(t, trust)
		for _, peer := range []string{"", ":5555", "[]:5555", "localhost:5555", "pas-une-adresse"} {
			rec := replayRequest(h, peer, forgedLoopbackHeaders)
			if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "replay_not_available") {
				t.Errorf("trust=%v pair %q : status %d body %s, want 404 replay_not_available",
					trust, peer, rec.Code, rec.Body.String())
			}
		}
	}
}

// TestReplayLocalGate_RejectsRemotePeerWithForgedHeaders : un pair distant reste
// distant, quels que soient les en-têtes d'IP qu'il envoie.
func TestReplayLocalGate_RejectsRemotePeerWithForgedHeaders(t *testing.T) {
	for _, trust := range []bool{false, true} {
		h := newReplayGateRouter(t, trust)
		for _, headers := range []map[string]string{nil, forgedLoopbackHeaders} {
			rec := replayRequest(h, "203.0.113.7:5555", headers)
			if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "replay_not_available") {
				t.Errorf("trust=%v en-têtes %v : status %d body %s, want 404 replay_not_available",
					trust, headers, rec.Code, rec.Body.String())
			}
		}
	}
}
