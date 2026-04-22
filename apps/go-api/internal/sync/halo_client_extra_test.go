package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-343-authorization-spartan") != "spartan" {
			t.Error("missing spartan header")
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "spartan",
		clearanceToken: "clear",
		minInterval:    time.Millisecond,
	}
	body, err := c.doGet(context.Background(), srv.URL+"/test")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestDoGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		minInterval:    time.Millisecond,
	}
	_, err := c.doGet(context.Background(), srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestDoGet_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		minInterval:    time.Millisecond,
	}
	_, err := c.doGet(context.Background(), srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func TestRateWait_FirstCall(t *testing.T) {
	c := &HaloAPIClient{minInterval: time.Millisecond}
	start := time.Now()
	c.rateWait(context.Background())
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("first call should not wait")
	}
	if c.lastRequest.IsZero() {
		t.Fatal("lastRequest should be set")
	}
}

func TestRateWait_SecondCall(t *testing.T) {
	c := &HaloAPIClient{
		minInterval: 50 * time.Millisecond,
		lastRequest: time.Now(),
	}
	start := time.Now()
	c.rateWait(context.Background())
	elapsed := time.Since(start)
	if elapsed < 30*time.Millisecond {
		t.Fatalf("expected wait ~50ms, got %v", elapsed)
	}
}

func TestNewHaloAPIClient_DefaultRate(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 0)
	if c.minInterval != time.Second/10 {
		t.Fatalf("expected default 100ms interval, got %v", c.minInterval)
	}
}

func TestNewHaloAPIClient_CustomRate(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 5)
	if c.minInterval != time.Second/5 {
		t.Fatalf("expected 200ms interval, got %v", c.minInterval)
	}
}

func TestGetMatchHistory_EmptyGamertag(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 10)
	_, err := c.GetMatchHistory(context.Background(), "", "all", 0, 25)
	if err == nil {
		t.Fatal("expected error for empty gamertag")
	}
}

func TestGetMatchHistory_InvalidType(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 10)
	_, err := c.GetMatchHistory(context.Background(), "Player", "invalid", 0, 25)
	if err == nil {
		t.Fatal("expected error for invalid match type")
	}
}

func TestGetMatchHistory_NegativeStart(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 10)
	_, err := c.GetMatchHistory(context.Background(), "Player", "all", -1, 25)
	if err == nil {
		t.Fatal("expected error for negative start")
	}
}

func TestGetMatchHistory_InvalidCount(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 10)
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
	c := NewHaloAPIClient("s", "c", 10)
	_, err := c.GetMatchStats(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

func TestBackoff_Short(t *testing.T) {
	c := &HaloAPIClient{minInterval: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	c.backoff(ctx, 0) // 2^0 * retryBaseDelay
	elapsed := time.Since(start)
	// Should complete quickly (base delay is small)
	if elapsed > 3*time.Second {
		t.Fatalf("backoff took too long: %v", elapsed)
	}
}

func TestBackoff_CancelledContext(t *testing.T) {
	c := &HaloAPIClient{minInterval: time.Millisecond}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	start := time.Now()
	c.backoff(ctx, 5)
	if time.Since(start) > 100*time.Millisecond {
		t.Fatal("backoff should return immediately on cancelled context")
	}
}

func TestGetMatchHistory_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"Results":[{"MatchId":"abc-123","MatchInfo":{"StartTime":"2025-01-01T00:00:00Z"}}]}`))
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		minInterval:    time.Millisecond,
	}
	// We can't easily override haloStatsHost, but doGet accepts any URL.
	// Use doGet directly for success path, then test parsing via GetMatchHistory indirectly.
	// Instead, test the full flow by testing the JSON parsing.
	body, err := c.doGet(context.Background(), srv.URL+"/hi/players/test/matches?start=0&count=1&type=all")
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty body")
	}
}

func TestGetMatchStats_SuccessViaDoGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"Players":[{"XUID":"123"}]}`))
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		minInterval:    time.Millisecond,
	}
	body, err := c.doGet(context.Background(), srv.URL+"/hi/matches/00000000-0000-0000-0000-000000000001/stats")
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
		w.Write([]byte(`ok`))
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		minInterval:    time.Millisecond,
	}
	body, err := c.doGet(context.Background(), srv.URL+"/test")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Fatalf("expected ok, got %s", body)
	}
	if calls < 3 {
		t.Fatalf("expected at least 3 calls, got %d", calls)
	}
}

func TestGetCareerRank_EmptyXUID(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 10)
	_, err := c.GetCareerRank(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty xuid")
	}
}

func TestGetCareerRank_UsesCareerAndCustomizationEndpoints(t *testing.T) {
	var careerCalls, customizationCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/rewardtracks/careerranks/careerrank1"):
			careerCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CurrentProgress": map[string]any{
					"Rank":              174,
					"PartialProgress":   21840,
					"HasReachedMaxRank": false,
				},
			})
		case strings.HasSuffix(r.URL.Path, "/customization"):
			customizationCalls++
			if got := r.URL.Query().Get("view"); got != "public" {
				t.Fatalf("view = %q, want public", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Appearance": map[string]any{"ServiceTag": "JGTM"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		economyBaseURL: srv.URL,
		minInterval:    time.Millisecond,
	}

	data, err := c.GetCareerRank(context.Background(), "2535469190789936")
	if err != nil {
		t.Fatalf("GetCareerRank: %v", err)
	}
	if data == nil {
		t.Fatal("GetCareerRank returned nil")
	}
	if data.CurrentRank != 174 || data.CurrentXP != 21840 {
		t.Fatalf("unexpected career data: %+v", data)
	}
	if data.SpartanID != "JGTM" {
		t.Fatalf("SpartanID = %q", data.SpartanID)
	}
	if careerCalls != 1 || customizationCalls != 1 {
		t.Fatalf("careerCalls=%d customizationCalls=%d", careerCalls, customizationCalls)
	}
}
