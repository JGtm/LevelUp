package auth

// xbl_exchange_facade_test.go — la façade de l'échange XBL de bout en bout, sur un faux Xbox Live
// (`httptest`) : ce que la façade ajoute au protocole de la brique (identité de la réponse XSTS
// rangée dans les bons champs, provenance mesurée remontée à l'observateur du `ctx`, créneau XBL
// process-wide pris pour la jambe User Token) est tenu ici, par `chaineDepuisCtx`.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
)

// fauxXboxLive sert les quatre jambes de l'échange et mesure la concurrence sur la jambe User Token.
type fauxXboxLive struct {
	enCours, maxVu atomic.Int32
	attente        time.Duration
}

func (f *fauxXboxLive) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch {
	case strings.HasSuffix(r.URL.Path, "/user/authenticate"):
		n := f.enCours.Add(1)
		for {
			m := f.maxVu.Load()
			if n <= m || f.maxVu.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(f.attente)
		f.enCours.Add(-1)
		_, _ = w.Write([]byte(`{"Token":"jeton-utilisateur"}`))
	case strings.HasSuffix(r.URL.Path, "/xsts/authorize"):
		_, _ = w.Write([]byte(`{"Token":"jeton-xsts","DisplayClaims":{"xui":[{"gtg":"Gamertag Test","xid":"2533274800000001","uhs":"h"}]}}`))
	case strings.HasSuffix(r.URL.Path, "/spartan-token"):
		_, _ = w.Write([]byte(`{"SpartanToken":"jeton-spartan","ExpiresUtc":{"ISO8601Date":"2030-01-02T03:04:05Z"}}`))
	default:
		_ = json.NewEncoder(w).Encode(map[string]string{"FlightConfigurationId": "jeton-clearance"})
	}
}

// doerVers : un client dont toute requête aboutit à srv (chemin conservé).
func doerVers(srv *httptest.Server) *http.Client {
	hote := strings.TrimPrefix(srv.URL, "http://")
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r2 := r.Clone(r.Context())
		r2.URL.Scheme, r2.URL.Host = "http", hote
		return http.DefaultTransport.RoundTrip(r2)
	})}
}

// TestFacadeEchangeXBL_IdentiteEtProvenance : gamertag et xuid de la réponse XSTS dans leurs
// champs, jetons convertis, provenance mesurée (« d= » accepté : famille Azure) écrite dans
// l'observateur du ctx.
func TestFacadeEchangeXBL_IdentiteEtProvenance(t *testing.T) {
	srv := httptest.NewServer(&fauxXboxLive{})
	defer srv.Close()

	ctx, obs := WithTokenFamilyObserver(context.Background())
	res, err := exchangeAccessTokenWithClient(ctx, doerVers(srv), "jeton-acces", title.DefaultHaloAuthDescriptor())
	if err != nil {
		t.Fatalf("échange : %v", err)
	}
	if res.Gamertag != "Gamertag Test" || res.XUID != "2533274800000001" {
		t.Errorf("identité = (%q, %q), attendu (\"Gamertag Test\", \"2533274800000001\")", res.Gamertag, res.XUID)
	}
	if res.Tokens == nil || res.Tokens.SpartanToken != "jeton-spartan" || res.Tokens.ClearanceToken != "jeton-clearance" ||
		!res.Tokens.SpartanExpiresAt.Equal(time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Errorf("jetons = %+v", res.Tokens)
	}
	if got := obs.Observed(); got != TokenFamilyAzure {
		t.Errorf("provenance observée = %q, attendu %q", got, TokenFamilyAzure)
	}
}

// TestFacadeEchangeXBL_CreneauBorneLaConcurrence : six échanges simultanés ne portent jamais plus
// de `xblUserAuthMaxConcurrent` requêtes User Token en même temps chez Xbox Live.
func TestFacadeEchangeXBL_CreneauBorneLaConcurrence(t *testing.T) {
	faux := &fauxXboxLive{attente: 40 * time.Millisecond}
	srv := httptest.NewServer(faux)
	defer srv.Close()

	const echanges = 6
	var wg sync.WaitGroup
	for i := 0; i < echanges; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := exchangeAccessTokenWithClient(context.Background(), doerVers(srv), "jeton-acces",
				title.DefaultHaloAuthDescriptor()); err != nil {
				t.Errorf("échange : %v", err)
			}
		}()
	}
	wg.Wait()
	if m := faux.maxVu.Load(); m > xblUserAuthMaxConcurrent || m == 0 {
		t.Errorf("concurrence User Token observée = %d, attendu 1..%d", m, xblUserAuthMaxConcurrent)
	}
}
