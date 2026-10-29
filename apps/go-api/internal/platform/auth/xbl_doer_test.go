// Package auth — xbl_doer_test.go : la garde `netguard` passée à la brique `haloapi/xblauth` par
// un Doer (plan D3 lot 2, D3-K) coupe la sortie AVANT émission, sans reprise, avec l'erreur et le
// message d'avant le déplacement.
package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/haloapi"
	"levelup/go-api/internal/platform/netguard"
)

// TestNetguardDoer_DemoOffline : en mode démo, Do n'émet rien et rend ErrOffline marquée
// définitive ; hors démo, il émet.
func TestNetguardDoer_DemoOffline(t *testing.T) {
	var hits atomic.Int32
	srv := serveurCompteur(&hits)
	defer srv.Close()
	d := netguardDoer{surface: "test.surface", next: srv.Client()}

	netguard.SetOffline(true)
	t.Cleanup(func() { netguard.SetOffline(false) })
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	_, err := d.Do(req)
	if !errors.Is(err, netguard.ErrOffline) || !errors.Is(err, haloapi.ErrDefinitif) {
		t.Fatalf("erreur = %v, attendu ErrOffline définitive", err)
	}
	if hits.Load() != 0 {
		t.Fatalf("%d requête(s) émise(s) en démo, attendu 0", hits.Load())
	}

	netguard.SetOffline(false)
	resp, err := d.Do(req.Clone(context.Background()))
	if err != nil {
		t.Fatalf("hors démo : %v", err)
	}
	_ = resp.Body.Close()
	if hits.Load() != 1 {
		t.Errorf("hors démo : %d requête(s), 1 attendue", hits.Load())
	}
}

// TestExchangeRefreshToken_DemoOffline_NEmetRien : le rafraîchissement OAuth (surface
// « microsoft_oauth.refresh ») ne part pas en mode démo ; l'erreur est ErrOffline, au message
// inchangé, et aucun repli (MSA natif) n'est tenté.
func TestExchangeRefreshToken_DemoOffline_NEmetRien(t *testing.T) {
	var hits atomic.Int32
	withMockTokenEndpoint(t, func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`{"access_token":"at"}`))
	}, "custom-app-id", "")
	srvMSA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`{"access_token":"at"}`))
	}))
	defer srvMSA.Close()
	prev := msaTokenURL
	msaTokenURL = srvMSA.URL
	t.Cleanup(func() { msaTokenURL = prev })

	netguard.SetOffline(true)
	t.Cleanup(func() { netguard.SetOffline(false) })

	_, _, _, err := ExchangeRefreshTokenWithRotation(context.Background(), "rt-1")
	if !errors.Is(err, netguard.ErrOffline) {
		t.Fatalf("erreur = %v, attendu netguard.ErrOffline", err)
	}
	if err.Error() != netguard.ErrOffline.Error() {
		t.Errorf("message = %q, attendu %q (inchangé depuis le déplacement)", err.Error(), netguard.ErrOffline.Error())
	}
	if hits.Load() != 0 {
		t.Fatalf("%d requête(s) émise(s) en démo, attendu 0", hits.Load())
	}
}
