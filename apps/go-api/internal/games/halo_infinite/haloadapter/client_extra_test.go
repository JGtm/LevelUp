package haloadapter

// client_extra_test.go — transport, validation et carrière/apparence du client de synchronisation
// (tests du client d'avant la brique, portés sous leur nom ; l'ancien GET générique `doGet`
// n'existe plus : ses tests passent par l'appel le plus simple qui traverse le transport, les
// stats d'un match).

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// matchTest : un identifiant de match valide.
const matchTest = "00000000-0000-0000-0000-000000000001"

// clientVers : un client dont toute requête aboutit à srv (adresses de production réécrites).
func clientVers(srv *httptest.Server, spartan string) *Client {
	return &Client{
		http:           &http.Client{Transport: &redirectTransport{host: strings.TrimPrefix(srv.URL, "http://")}},
		spartanToken:   spartan,
		clearanceToken: "clear",
		limiter:        fastLimiter(),
	}
}

func TestDoGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-343-authorization-spartan") != "spartan" {
			t.Error("missing spartan header")
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	body, err := clientVers(srv, "spartan").GetMatchStats(context.Background(), matchTest)
	if err != nil {
		t.Fatal(err)
	}
	if body["ok"] != true {
		t.Fatalf("unexpected body: %v", body)
	}
}

func TestDoGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	if _, err := clientVers(srv, "s").GetMatchStats(context.Background(), matchTest); err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestDoGet_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()

	_, err := clientVers(srv, "s").GetMatchStats(context.Background(), matchTest)
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !IsAuthError(err) {
		t.Fatalf("401 doit être un refus d'authentification typé, got %v", err)
	}
}

func TestNewHaloAPIClient_DefaultRate(t *testing.T) {
	c := NewClient("s", "c", 0)
	if c.limiter == nil {
		t.Fatal("expected limiter to be set")
	}
	if got := c.limiter.Limit(); got != rate.Limit(10) {
		t.Fatalf("expected default 10 RPS, got %v", got)
	}
}

func TestNewHaloAPIClient_CustomRate(t *testing.T) {
	c := NewClient("s", "c", 5)
	if got := c.limiter.Limit(); got != rate.Limit(5) {
		t.Fatalf("expected 5 RPS, got %v", got)
	}
}

// TestRateWait_ConcurrentRespectsRPS valide que rate-limiting tient sous
// concurrence (régression du bug data-race lastRequest pré-rate.Limiter).
func TestGetMatchHistory_EmptyGamertag(t *testing.T) {
	c := NewClient("s", "c", 10)
	_, err := c.GetMatchHistory(context.Background(), "", "all", 0, 25)
	if err == nil {
		t.Fatal("expected error for empty gamertag")
	}
}

func TestGetMatchHistory_InvalidType(t *testing.T) {
	c := NewClient("s", "c", 10)
	_, err := c.GetMatchHistory(context.Background(), "Player", "invalid", 0, 25)
	if err == nil {
		t.Fatal("expected error for invalid match type")
	}
}

func TestGetMatchHistory_NegativeStart(t *testing.T) {
	c := NewClient("s", "c", 10)
	_, err := c.GetMatchHistory(context.Background(), "Player", "all", -1, 25)
	if err == nil {
		t.Fatal("expected error for negative start")
	}
}

func TestGetMatchHistory_InvalidCount(t *testing.T) {
	c := NewClient("s", "c", 10)
	_, err := c.GetMatchHistory(context.Background(), "Player", "all", 0, 0)
	if err == nil {
		t.Fatal("expected error for count=0")
	}
	_, err = c.GetMatchHistory(context.Background(), "Player", "all", 0, 26)
	if err == nil {
		t.Fatal("expected error for count>25")
	}
}

func TestGetMatchStats_InvalidUUID(t *testing.T) {
	c := NewClient("s", "c", 10)
	_, err := c.GetMatchStats(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}
func TestGetMatchHistory_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"Results":[{"MatchId":"abc-123","MatchInfo":{"StartTime":"2025-01-01T00:00:00Z"}}]}`))
	}))
	defer srv.Close()

	entries, err := clientVers(srv, "s").GetMatchHistory(context.Background(), "xuid(1)", "all", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].MatchID != "abc-123" || entries[0].StartTime != "2025-01-01T00:00:00Z" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestGetMatchStats_SuccessViaDoGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"Players":[{"XUID":"123"}]}`))
	}))
	defer srv.Close()

	body, err := clientVers(srv, "s").GetMatchStats(context.Background(), matchTest)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty body")
	}
}

func TestDoGet_ServerError_Retry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":"ok"}`))
	}))
	defer srv.Close()

	body, err := clientVers(srv, "s").GetMatchStats(context.Background(), matchTest)
	if err != nil {
		t.Fatal(err)
	}
	if body["ok"] != "ok" {
		t.Fatalf("expected ok, got %v", body)
	}
	if calls < 3 {
		t.Fatalf("expected at least 3 calls, got %d", calls)
	}
}

// TestDoGet_429_NoInClientRetry vérifie qu'un 429 (rate limit) n'est PAS retenté en interne :
// l'erreur typée (avec Retry-After parsé) est rendue après UN seul appel, pour laisser le
// refroidissement global du pool temporiser.
func TestDoGet_429_NoInClientRetry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := clientVers(srv, "s").GetMatchStats(context.Background(), matchTest)
	if err == nil {
		t.Fatal("expected error on 429")
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call (no in-client retry on 429), got %d", calls)
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected *HTTPError, got %T", err)
	}
	if httpErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", httpErr.StatusCode)
	}
	if httpErr.RetryAfter != 30*time.Second {
		t.Fatalf("expected RetryAfter 30s (parsé), got %v", httpErr.RetryAfter)
	}
}
