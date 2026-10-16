// Package auth — xbox_device_code_netguard_test.go : en mode démo, le device code flow
// n'émet aucune requête vers login.live.com (coupe-circuit netguard, lot L8-10 du plan
// sécurité du 2026-10-15). Le flux n'est pas « CLI uniquement » : le web le démarre
// (POST /auth/device-flow/start, /watcher/auth/start). La garde démo en écriture refuse déjà
// ces routes ; ceci est la défense en profondeur à la frontière sortante.
package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/platform/netguard"
)

func serveurCompteur(hits *atomic.Int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
}

func TestStartXboxDeviceCode_DemoOffline_NEmetRien(t *testing.T) {
	var hits atomic.Int32
	srv := serveurCompteur(&hits)
	defer srv.Close()
	netguard.SetOffline(true)
	t.Cleanup(func() { netguard.SetOffline(false) })

	_, err := startXboxDeviceCodeWithURL(context.Background(), srv.Client(), "cid", srv.URL)
	if !errors.Is(err, netguard.ErrOffline) {
		t.Fatalf("erreur = %v, attendu netguard.ErrOffline", err)
	}
	if hits.Load() != 0 {
		t.Fatalf("%d requête(s) émise(s) en démo, attendu 0", hits.Load())
	}
}

func TestPollXboxDeviceCode_DemoOffline_NEmetRien(t *testing.T) {
	var hits atomic.Int32
	srv := serveurCompteur(&hits)
	defer srv.Close()
	netguard.SetOffline(true)
	t.Cleanup(func() { netguard.SetOffline(false) })

	_, _, err := pollXboxDeviceCodeWithURL(context.Background(), srv.Client(), "cid", "dc", 0, srv.URL)
	if !errors.Is(err, netguard.ErrOffline) {
		t.Fatalf("erreur = %v, attendu netguard.ErrOffline", err)
	}
	if hits.Load() != 0 {
		t.Fatalf("%d requête(s) émise(s) en démo, attendu 0", hits.Load())
	}
}
