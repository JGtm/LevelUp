// halo_client_url_dryrun_test.go — dry-run de validation du fix 2026-05-20.
//
// Vérifie que l'URL effective construite pour /matches utilise bien le format
// xuid(NNN) attendu par l'API Halo (cf. Grunt StatsModule + SPNKr), sans
// taper l'API réelle ni écrire en DB. Réutilise le pattern redirectTransport
// déjà en place dans halo_client_extra_test.go.

package haloclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// captureTransport enregistre la dernière requête HTTP traversée et la
// renvoie vers srv (même comportement que redirectTransport, mais avec capture).
type captureTransport struct {
	mu       sync.Mutex
	host     string
	captured *http.Request
}

func (ct *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	ct.mu.Lock()
	ct.captured = req2.Clone(req.Context())
	ct.mu.Unlock()
	req2.URL.Scheme = "http"
	req2.URL.Host = ct.host
	return http.DefaultTransport.RoundTrip(req2)
}

func (ct *captureTransport) last() *http.Request {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.captured
}

func newDryRunClient(srv *httptest.Server) (*HaloAPIClient, *captureTransport) {
	ct := &captureTransport{host: strings.TrimPrefix(srv.URL, "http://")}
	return &HaloAPIClient{
		http:           &http.Client{Transport: ct},
		spartanToken:   "spartan-token-test",
		clearanceToken: "clearance-token-test",
		limiter:        rate.NewLimiter(rate.Every(time.Millisecond), 1),
	}, ct
}

// TestGetMatchHistory_URLFormat_DryRun valide que le format xuid(NNN) sort
// bien sur la wire. Reproduit le call site de engine.go:803 après fix.
func TestGetMatchHistory_URLFormat_DryRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"Results": []any{}})
	}))
	defer srv.Close()

	client, ct := newDryRunClient(srv)

	const (
		xuid      = "2535469190789936" // Chocoboflor
		matchType = "matchmaking"
		start     = 0
		count     = 25
	)
	playerID := fmt.Sprintf("xuid(%s)", xuid) // EXACTEMENT comme engine.go:803 après fix

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.GetMatchHistory(ctx, playerID, matchType, start, count); err != nil {
		t.Fatalf("GetMatchHistory dry-run: %v", err)
	}
	got := ct.last()
	if got == nil {
		t.Fatal("aucune requête capturée")
	}

	decodedPath, err := url.PathUnescape(got.URL.Path)
	if err != nil {
		t.Fatalf("PathUnescape: %v", err)
	}

	wantPath := fmt.Sprintf("/hi/players/xuid(%s)/matches", xuid)
	if decodedPath != wantPath {
		t.Errorf("path effectif incorrect\n  got  : %s\n  want : %s", decodedPath, wantPath)
	}

	q := got.URL.Query()
	if q.Get("type") != matchType {
		t.Errorf("query type incorrect : got=%q want=%q", q.Get("type"), matchType)
	}
	if q.Get("start") != "0" || q.Get("count") != "25" {
		t.Errorf("query start/count incorrect : start=%q count=%q",
			q.Get("start"), q.Get("count"))
	}
	if got.Header.Get("x-343-authorization-spartan") == "" {
		t.Errorf("header x-343-authorization-spartan manquant")
	}

	t.Logf("URL effective (host redirigé) : %s%s?%s",
		srv.URL, got.URL.Path, got.URL.RawQuery)
	t.Logf("URL équivalente Halo : https://halostats.svc.halowaypoint.com%s?%s",
		decodedPath, got.URL.RawQuery)
}

// TestGetMatchHistory_OldGamertagFormat_DryRun documente le BUG d'avant fix :
// passer le gamertag brut produit /hi/players/Chocoboflor/matches au lieu de
// /hi/players/xuid(...)/matches. Garde-fou contre une régression du call site.
func TestGetMatchHistory_OldGamertagFormat_DryRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"Results": []any{}})
	}))
	defer srv.Close()

	client, ct := newDryRunClient(srv)

	if _, err := client.GetMatchHistory(context.Background(), "Chocoboflor", "matchmaking", 0, 25); err != nil {
		t.Fatalf("GetMatchHistory: %v", err)
	}
	got := ct.last()
	decoded, _ := url.PathUnescape(got.URL.Path)
	if !strings.Contains(decoded, "Chocoboflor") {
		t.Fatalf("dry-run setup cassé : path attendu contient 'Chocoboflor', got %q", decoded)
	}
	if strings.Contains(decoded, "xuid(") {
		t.Errorf("regression : gamertag brut auto-wrappé en xuid() — ce n'est pas le contrat actuel\n  got : %s", decoded)
	}
	t.Logf("ancien comportement (BUG si en prod) : path=%s", decoded)
}
