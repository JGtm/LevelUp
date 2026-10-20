// Package auth — sisu_exchange_flow_test.go : complétion du device-flow de connexion
// Xbox face à des serveurs Microsoft factices qui reproduisent le comportement réel.
package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"levelup/go-api/internal/domain/title"
)

// Jetons factices rendus par les serveurs simulés.
const (
	fakeUserToken      = "user-token-xbl"
	fakeXSTSHalo       = "xsts-halo"
	fakeSpartanToken   = "spartan-token"
	fakeClearanceToken = "clearance-id"
)

// fakeXboxLive simule les serveurs Microsoft/Halo atteints par la complétion du
// device-flow, routés par hôte + chemin, et reproduit les réponses observées :
//   - sisu.xboxlive.com/authorize : 400 corps vide au ticket du device-flow ;
//   - user.auth.xboxlive.com : n'accepte un ticket MSA du client Xbox natif QUE
//     préfixé « t= » (401 corps vide sinon) ;
//   - xsts.auth.xboxlive.com : XSTS de l'audience Halo (DisplayClaims sans gtg/xid) ;
//   - settings.svc.halowaypoint.com : Spartan puis Clearance.
type fakeXboxLive struct {
	mu    sync.Mutex
	calls []string // "hôte+chemin" dans l'ordre d'appel
	rps   []string // préfixes RpsTicket reçus par user/authenticate
}

func (f *fakeXboxLive) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.calls = append(f.calls, r.URL.Host+r.URL.Path)
	f.mu.Unlock()

	rec := httptest.NewRecorder()
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}
	switch r.URL.Host + r.URL.Path {
	case "sisu.xboxlive.com/authorize":
		rec.WriteHeader(http.StatusBadRequest)
	case "user.auth.xboxlive.com/user/authenticate":
		f.serveUserToken(rec, body)
	case "xsts.auth.xboxlive.com/xsts/authorize":
		writeFakeJSON(rec, map[string]any{
			"Token":         fakeXSTSHalo,
			"NotAfter":      "2099-01-01T00:00:00Z",
			"DisplayClaims": map[string]any{"xui": []any{map[string]any{"uhs": "uhs-1"}}},
		})
	case "settings.svc.halowaypoint.com/spartan-token":
		writeFakeJSON(rec, map[string]any{
			"SpartanToken": fakeSpartanToken,
			"ExpiresUtc":   map[string]any{"ISO8601Date": "2099-01-01T00:00:00Z"},
		})
	case "settings.svc.halowaypoint.com/oban/flight-configurations/titles/hi/audiences/RETAIL/active":
		writeFakeJSON(rec, map[string]any{"FlightConfigurationId": fakeClearanceToken})
	default:
		rec.WriteHeader(http.StatusNotFound)
	}
	return rec.Result(), nil
}

func (f *fakeXboxLive) serveUserToken(rec *httptest.ResponseRecorder, body []byte) {
	var req struct {
		Properties struct {
			RpsTicket string `json:"RpsTicket"`
		} `json:"Properties"`
	}
	_ = json.Unmarshal(body, &req)
	prefix, _, _ := strings.Cut(req.Properties.RpsTicket, "=")
	f.mu.Lock()
	f.rps = append(f.rps, prefix+"=")
	f.mu.Unlock()
	if prefix != "t" {
		rec.WriteHeader(http.StatusUnauthorized)
		return
	}
	writeFakeJSON(rec, map[string]any{"Token": fakeUserToken})
}

func writeFakeJSON(rec *httptest.ResponseRecorder, v any) {
	rec.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(rec).Encode(v)
}

// TestSISUDeviceFlow_ExchangeFlow_ClassicChain : le ticket MSA rendu par le
// device-flow (client Xbox natif, scope MBI_SSL) s'échange contre les jetons Halo
// par la chaîne XBL classique, la même que le pool emploie à chaque
// rafraîchissement. Le service SISU /authorize, qui répond 400 corps vide à ce
// ticket, n'est jamais appelé, et le préfixe « t= » est présenté d'emblée (aucun
// 401 de tâtonnement).
func TestSISUDeviceFlow_ExchangeFlow_ClassicChain(t *testing.T) {
	fake := &fakeXboxLive{}
	flow := newTestSISUDeviceFlow(t, &http.Client{Transport: fake})

	res, err := flow.ExchangeFlow(context.Background(), "EwA-ticket-msa")
	if err != nil {
		t.Fatalf("ExchangeFlow : %v (appels : %v)", err, fake.calls)
	}
	if res.Tokens.SpartanToken != fakeSpartanToken || res.Tokens.ClearanceToken != fakeClearanceToken {
		t.Errorf("jetons Halo = %+v, attendu spartan=%q clearance=%q", res.Tokens, fakeSpartanToken, fakeClearanceToken)
	}
	if res.Tokens.SpartanExpiresAt.IsZero() {
		t.Error("expiration Spartan non reportée")
	}
	for _, c := range fake.calls {
		if strings.HasPrefix(c, "sisu.xboxlive.com") {
			t.Errorf("SISU /authorize appelé (%s) : la complétion ne doit plus en dépendre", c)
		}
	}
	if len(fake.rps) != 1 || fake.rps[0] != rpsPrefixXboxNative {
		t.Errorf("préfixes RpsTicket présentés = %v, attendu un seul %q", fake.rps, rpsPrefixXboxNative)
	}
	want := []string{
		"user.auth.xboxlive.com/user/authenticate",
		"xsts.auth.xboxlive.com/xsts/authorize",
		"settings.svc.halowaypoint.com/spartan-token",
	}
	for i, w := range want {
		if i >= len(fake.calls) || fake.calls[i] != w {
			t.Fatalf("séquence d'appels = %v, attendu en tête %v", fake.calls, want)
		}
	}
}

// TestSISUDeviceFlow_ExchangeFlow_PropagatesXboxError : un refus du service Xbox
// remonte avec l'URL et le statut (diagnostic), jamais avalé.
func TestSISUDeviceFlow_ExchangeFlow_PropagatesXboxError(t *testing.T) {
	refuse := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		rec.WriteHeader(http.StatusForbidden)
		return rec.Result(), nil
	})
	flow := newTestSISUDeviceFlow(t, &http.Client{Transport: refuse})

	_, err := flow.ExchangeFlow(context.Background(), "EwA-ticket-msa")
	if err == nil {
		t.Fatal("erreur attendue sur un refus 403")
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Errorf("erreur sans statut HTTP : %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// newTestSISUDeviceFlow construit un device-flow tel qu'InitDeviceFlow le rend,
// branché sur le client HTTP factice.
func newTestSISUDeviceFlow(t *testing.T, client *http.Client) *sisuDeviceFlow {
	t.Helper()
	calls := 0
	srv := xboxDeviceCodeServer(t, "UC", &calls)
	p := NewSISUProviderWithIDs(title.DefaultHaloAuthDescriptor().SISUAppID, SISUDefaultTitleID)
	flow, err := p.initDeviceFlowWithURL(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("InitDeviceFlow : %v", err)
	}
	sf := flow.(*sisuDeviceFlow)
	sf.httpClient = client
	return sf
}
