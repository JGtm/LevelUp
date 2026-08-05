package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// replay_local_gate_test.go — le garde doit refuser un client DISTANT, et ne pas se laisser
// contourner par un en-tete.

func TestReplayGate_AllowsLoopback(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:5555", "[::1]:5555", "127.0.0.1"} {
		r := httptest.NewRequest(http.MethodGet, "/players/p/matches/m/replay", nil)
		r.RemoteAddr = addr
		if !allowReplay(r) {
			t.Errorf("adresse locale refusee : %q", addr)
		}
	}
}

func TestReplayGate_RejectsRemote(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/players/p/matches/m/replay", nil)
	r.RemoteAddr = "192.168.1.42:5555"
	if allowReplay(r) {
		t.Error("une adresse distante ne doit pas passer le garde")
	}
}

func TestReplayGate_IgnoresForwardedHeader(t *testing.T) {
	// UN EN-TETE EST FOURNI PAR LE CLIENT. S'en servir transformerait le garde en suggestion :
	// n'importe qui pourrait se declarer local. Seule l'adresse de la connexion compte.
	r := httptest.NewRequest(http.MethodGet, "/players/p/matches/m/replay", nil)
	r.RemoteAddr = "203.0.113.7:5555"
	r.Header.Set("X-Forwarded-For", "127.0.0.1")
	r.Header.Set("X-Real-IP", "127.0.0.1")
	if allowReplay(r) {
		t.Error("un en-tete ne doit pas suffire a passer le garde")
	}
}
