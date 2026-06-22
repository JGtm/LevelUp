// halo_exchange_descriptor_test.go — oracle PMT-2 Contract (MT-02) : parité
// byte-identique PAR LEG du chemin d'échange après threading de title.AuthDescriptor.
//
// Le client *http.Client est construit en interne par les entrées publiques
// (ExchangeAccessToken…), donc l'interception se fait au niveau des fonctions de
// leg (qui prennent le client). On y prouve : (a) le défaut Halo produit des
// requêtes byte-identiques aux littéraux actuels ; (b) un descripteur synthétique
// route vers ses propres valeurs.

package auth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
)

// recordingRT capture la requête sortante et renvoie une réponse canned.
type recordingRT struct {
	req  *http.Request
	body string
	resp string
}

func (r *recordingRT) RoundTrip(req *http.Request) (*http.Response, error) {
	r.req = req.Clone(req.Context())
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		r.body = string(b)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(r.resp)),
		Header:     make(http.Header),
	}, nil
}

func TestRequestSpartanTokenWith_GoldenAndRouting(t *testing.T) {
	ctx := context.Background()

	t.Run("défaut Halo = byte-identique", func(t *testing.T) {
		rt := &recordingRT{resp: `{"SpartanToken":"sp"}`}
		client := &http.Client{Transport: rt}
		d := title.DefaultHaloAuthDescriptor()
		if _, _, err := requestSpartanTokenWith(ctx, client, "XSTS_TOK", d.SpartanAudience, d.SpartanTokenURL); err != nil {
			t.Fatalf("requestSpartanTokenWith: %v", err)
		}
		if got := rt.req.URL.String(); got != "https://settings.svc.halowaypoint.com/spartan-token" {
			t.Errorf("URL = %q, want spartan-token (parité)", got)
		}
		// Corps byte-identique : encoding/json trie les clés map alphabétiquement.
		wantBody := `{"Audience":"urn:343:s3:services","MinVersion":"4","Proof":[{"Token":"XSTS_TOK","TokenType":"Xbox_XSTSv3"}]}`
		if rt.body != wantBody {
			t.Errorf("body =\n  %q\nwant\n  %q", rt.body, wantBody)
		}
		if rt.req.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept header manquant")
		}
	})

	t.Run("descripteur synthétique route", func(t *testing.T) {
		rt := &recordingRT{resp: `{"SpartanToken":"sp"}`}
		client := &http.Client{Transport: rt}
		if _, _, err := requestSpartanTokenWith(ctx, client, "X", "urn:example:services", "https://settings.example.test/spartan-token"); err != nil {
			t.Fatalf("requestSpartanTokenWith: %v", err)
		}
		if got := rt.req.URL.String(); got != "https://settings.example.test/spartan-token" {
			t.Errorf("URL = %q, want example.test (routing réel)", got)
		}
		if !strings.Contains(rt.body, `"Audience":"urn:example:services"`) {
			t.Errorf("body ne porte pas l'audience synthétique : %q", rt.body)
		}
	})
}

func TestRequestClearanceTokenWith_GoldenAndRouting(t *testing.T) {
	ctx := context.Background()

	t.Run("défaut Halo = byte-identique", func(t *testing.T) {
		rt := &recordingRT{resp: `{"FlightConfigurationId":"fc"}`}
		client := &http.Client{Transport: rt}
		d := title.DefaultHaloAuthDescriptor()
		if _, err := requestClearanceTokenWith(ctx, client, "SPARTAN", d.ClearanceURL); err != nil {
			t.Fatalf("requestClearanceTokenWith: %v", err)
		}
		want := "https://settings.svc.halowaypoint.com/oban/flight-configurations/titles/hi/audiences/RETAIL/active"
		if got := rt.req.URL.String(); got != want {
			t.Errorf("URL = %q, want %q (parité)", got, want)
		}
		if rt.req.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", rt.req.Method)
		}
		if rt.req.Header.Get("x-343-authorization-spartan") != "SPARTAN" {
			t.Errorf("header spartan manquant/incorrect")
		}
	})

	t.Run("descripteur synthétique route", func(t *testing.T) {
		rt := &recordingRT{resp: `{"FlightConfigurationId":"fc"}`}
		client := &http.Client{Transport: rt}
		if _, err := requestClearanceTokenWith(ctx, client, "S", "https://settings.example.test/clearance/titles/syn/active"); err != nil {
			t.Fatalf("requestClearanceTokenWith: %v", err)
		}
		if got := rt.req.URL.String(); got != "https://settings.example.test/clearance/titles/syn/active" {
			t.Errorf("URL = %q, want example.test", got)
		}
	})
}

// TestScopes_DescriptorDerivation_GoldenParity prouve que les scopes dérivés du
// descripteur (MT-02 leg 4) sont byte-identiques aux littéraux historiques.
func TestScopes_DescriptorDerivation_GoldenParity(t *testing.T) {
	if len(XboxScopes) != 2 || XboxScopes[0] != "Xboxlive.signin" || XboxScopes[1] != "Xboxlive.offline_access" {
		t.Errorf("XboxScopes = %v, want [Xboxlive.signin Xboxlive.offline_access]", XboxScopes)
	}
	if xboxScopes != "Xboxlive.signin Xboxlive.offline_access" {
		t.Errorf("xboxScopes = %q, want space-joined", xboxScopes)
	}
}

// TestRequestXSTSToken_DescriptorAudience prouve que l'audience du descripteur
// arrive dans le corps RelyingParty de la requête XSTS.
func TestRequestXSTSToken_DescriptorAudience(t *testing.T) {
	ctx := context.Background()
	rt := &recordingRT{resp: `{"Token":"x","DisplayClaims":{"xui":[{"gtg":"GT","xid":"123","uhs":"h"}]}}`}
	client := &http.Client{Transport: rt}

	d := title.DefaultHaloAuthDescriptor()
	if _, _, _, err := requestXSTSToken(ctx, client, "USER_TOK", d.XSTSAudience); err != nil {
		t.Fatalf("requestXSTSToken: %v", err)
	}
	if !strings.Contains(rt.body, `"RelyingParty":"https://prod.xsts.halowaypoint.com/"`) {
		t.Errorf("body ne porte pas l'audience Halo en RelyingParty : %q", rt.body)
	}

	// Synthétique : une autre audience route dans le corps.
	rt2 := &recordingRT{resp: `{"Token":"x","DisplayClaims":{"xui":[{"gtg":"GT","xid":"123","uhs":"h"}]}}`}
	client2 := &http.Client{Transport: rt2}
	if _, _, _, err := requestXSTSToken(ctx, client2, "USER_TOK", "https://xsts.example.test/"); err != nil {
		t.Fatalf("requestXSTSToken synthetic: %v", err)
	}
	if !strings.Contains(rt2.body, `"RelyingParty":"https://xsts.example.test/"`) {
		t.Errorf("body ne route pas l'audience synthétique : %q", rt2.body)
	}
}
