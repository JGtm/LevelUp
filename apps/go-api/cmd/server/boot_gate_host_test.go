// boot_gate_host_test.go — la porte de démarrage du mode desktop applique le Host strict (ADR 0045
// D-3), sans exiger le cookie ; hors desktop elle est inchangée.
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/config"
)

func desktopBootConfig() *config.AppConfig {
	cfg := &config.AppConfig{APIHost: "127.0.0.1", APIPort: 49555}
	cfg.Capabilities.ShellEntryGate = true
	return cfg
}

func gateGetHost(h http.Handler, path, host string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Host = host
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

// Porte en cours d'initialisation, sous desktop : un Host étranger reçoit 421 sur toutes les routes
// (page, API, sonde), le Host exact reçoit le 503 server_starting, sans cookie.
func TestBootHandler_DesktopHostStrictSurLaPorteDeDemarrage(t *testing.T) {
	cfg := desktopBootConfig()
	gate := newBootGate(cfg)
	h := bootHandler(cfg, gate)

	for _, host := range []string{"localhost:49555", "127.0.0.1:49556", "127.0.0.1", "evil.example", "[::1]:49555"} {
		for _, path := range []string{"/", "/api/v1/bootstrap", "/health", "/assets/x.js"} {
			if w := gateGetHost(h, path, host); w.Code != http.StatusMisdirectedRequest {
				t.Errorf("Host %q sur %s : %d, attendu 421", host, path, w.Code)
			}
		}
	}
	w := gateGetHost(h, "/api/v1/bootstrap", "127.0.0.1:49555")
	assertStarting(t, w, bootStepMigrations)

	// Une fois le routeur ouvert, le contrôle reste devant lui (le garde du routeur le refait).
	gate.open(t.Context(), routerStub())
	if w := gateGetHost(h, "/", "evil.example"); w.Code != http.StatusMisdirectedRequest {
		t.Errorf("routeur ouvert, Host étranger : %d, attendu 421", w.Code)
	}
	if w := gateGetHost(h, "/", "127.0.0.1:49555"); w.Code != http.StatusOK || w.Body.String() != "router" {
		t.Errorf("routeur ouvert, Host exact : %d %q", w.Code, w.Body.String())
	}
}

// Hors desktop, la porte est servie telle quelle, quel que soit le Host.
func TestBootHandler_HorsDesktopInchange(t *testing.T) {
	cfg := &config.AppConfig{APIHost: "127.0.0.1", APIPort: 8000}
	gate := newBootGate(cfg)
	h := bootHandler(cfg, gate)
	if h != http.Handler(gate) {
		t.Fatal("hors desktop, le handler de démarrage doit être la porte elle-même")
	}
	assertStarting(t, gateGetHost(h, "/api/v1/bootstrap", "example.org"), bootStepMigrations)
}
