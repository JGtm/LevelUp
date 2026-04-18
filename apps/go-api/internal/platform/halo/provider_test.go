// Package halo — provider_test.go : tests unitaires HaloProvider (boîte blanche).
package halo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// newTestProvider crée un provider avec des base URLs injectées depuis les serveurs de test.
func newTestProvider(battlePassURL, challengesURL string) *HaloProvider {
	p := NewHaloProvider()
	p.battlePassBaseURL = battlePassURL
	p.challengesBaseURL = challengesURL
	p.limiter = newRateLimiter(600)
	p.maxRetries = 3
	return p
}

func testTokens() *domain.HaloTokens {
	return &domain.HaloTokens{SpartanToken: "spartan_test", ClearanceToken: "clear_test"}
}

func ctxWithAuth(tokens *domain.HaloTokens, xuid string) context.Context {
	return ctxkeys.WithHaloAuth(context.Background(), tokens, xuid)
}

// ---------------------------------------------------------------------------
// GetBattlePass — cas sans auth
// ---------------------------------------------------------------------------

func TestGetBattlePass_NoTokens(t *testing.T) {
	p := newTestProvider("", "")
	resp := p.GetBattlePass(context.Background())

	if resp.Available {
		t.Error("expected available=false when no tokens")
	}
	if resp.ErrorHint == nil || *resp.ErrorHint != "auth_required" {
		t.Errorf("expected error_hint=auth_required, got %v", resp.ErrorHint)
	}
}

func TestGetBattlePass_NoXUID(t *testing.T) {
	ctx := ctxkeys.WithHaloAuth(context.Background(), testTokens(), "")
	p := newTestProvider("", "")
	resp := p.GetBattlePass(ctx)

	if resp.Available {
		t.Error("expected available=false when xuid empty")
	}
	if resp.ErrorHint == nil || *resp.ErrorHint != "auth_required" {
		t.Errorf("expected error_hint=auth_required, got %v", resp.ErrorHint)
	}
}

// ---------------------------------------------------------------------------
// GetBattlePass — appel live OK
// ---------------------------------------------------------------------------

func TestGetBattlePass_LiveOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-343-authorization-spartan") == "" {
			http.Error(w, "missing spartan", http.StatusUnauthorized)
			return
		}
		payload := map[string]any{
			"ActiveOperationRewardTrackPath": "RewardTracks/Operations/Season6-Op",
			"OperationRewardTracks": []any{
				map[string]any{
					"RewardTrackPath": "RewardTracks/Operations/Season6-Op",
					"CurrentProgress": map[string]any{
						"Rank":            45,
						"PartialProgress": 2500,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	p := newTestProvider(srv.URL, "")
	ctx := ctxWithAuth(testTokens(), "xuid-test")
	resp := p.GetBattlePass(ctx)

	if !resp.Available {
		t.Errorf("expected available=true, error_hint=%v", resp.ErrorHint)
	}
	if resp.Rank == nil || *resp.Rank != 45 {
		t.Errorf("expected rank=45, got %v", resp.Rank)
	}
	if resp.Progress == nil || *resp.Progress != 2500 {
		t.Errorf("expected progress=2500, got %v", resp.Progress)
	}
	if resp.RewardTrack == nil || *resp.RewardTrack != "RewardTracks/Operations/Season6-Op" {
		t.Errorf("expected reward_track, got %v", resp.RewardTrack)
	}
}

// ---------------------------------------------------------------------------
// GetBattlePass — erreurs HTTP
// ---------------------------------------------------------------------------

func TestGetBattlePass_HTTP401(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := newTestProvider(srv.URL, "")
	resp := p.GetBattlePass(ctxWithAuth(testTokens(), "xuid-test"))

	if resp.Available {
		t.Error("expected available=false on 401")
	}
	if resp.ErrorHint == nil || *resp.ErrorHint != "fetch_error" {
		t.Errorf("expected error_hint=fetch_error, got %v", resp.ErrorHint)
	}
	// 401 : pas de retry — 1 seul appel.
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call (no retry on 401), got %d", callCount)
	}
}

func TestGetBattlePass_HTTP500ThenOK(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		if n == 1 {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		payload := map[string]any{
			"ActiveOperationRewardTrackPath": "RewardTracks/Operations/S7",
			"OperationRewardTracks": []any{
				map[string]any{
					"RewardTrackPath": "RewardTracks/Operations/S7",
					"CurrentProgress": map[string]any{"Rank": 10, "PartialProgress": 0},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	p := newTestProvider(srv.URL, "")
	resp := p.GetBattlePass(ctxWithAuth(testTokens(), "xuid-test"))

	if !resp.Available {
		t.Errorf("expected available=true after retry, error_hint=%v", resp.ErrorHint)
	}
	if resp.Rank == nil || *resp.Rank != 10 {
		t.Errorf("expected rank=10, got %v", resp.Rank)
	}
	if atomic.LoadInt32(&callCount) < 2 {
		t.Errorf("expected at least 2 calls (retry), got %d", callCount)
	}
}

func TestGetBattlePass_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{invalid json"))
	}))
	defer srv.Close()

	p := newTestProvider(srv.URL, "")
	resp := p.GetBattlePass(ctxWithAuth(testTokens(), "xuid-test"))

	if resp.Available {
		t.Error("expected available=false on malformed JSON")
	}
	if resp.ErrorHint == nil || *resp.ErrorHint != "fetch_error" {
		t.Errorf("expected error_hint=fetch_error, got %v", resp.ErrorHint)
	}
}

// ---------------------------------------------------------------------------
// GetChallenges — cas sans auth
// ---------------------------------------------------------------------------

func TestGetChallenges_NoTokens(t *testing.T) {
	p := newTestProvider("", "")
	resp := p.GetChallenges(context.Background())

	if resp.Available {
		t.Error("expected available=false when no tokens")
	}
	if resp.ErrorHint == nil || *resp.ErrorHint != "auth_required" {
		t.Errorf("expected error_hint=auth_required, got %v", resp.ErrorHint)
	}
}

// ---------------------------------------------------------------------------
// GetChallenges — appel live OK
// ---------------------------------------------------------------------------

func TestGetChallenges_LiveOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := map[string]any{
			"AssignedDecks": []any{
				map[string]any{
					"Expiration": map[string]string{"ISO8601Date": "2024-11-19T17:00:00Z"},
					"ActiveChallenges": []any{
						map[string]any{"XPReward": 1000},
						map[string]any{"XPReward": 2000},
						map[string]any{"XPReward": 1500},
					},
					"CompletedChallenges": []any{
						map[string]any{"XPReward": 500},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	p := newTestProvider("", srv.URL)
	resp := p.GetChallenges(ctxWithAuth(testTokens(), "xuid-test"))

	if !resp.Available {
		t.Errorf("expected available=true, error_hint=%v", resp.ErrorHint)
	}
	if resp.Total == nil || *resp.Total != 3 {
		t.Errorf("expected total=3, got %v", resp.Total)
	}
	if resp.Completed == nil || *resp.Completed != 1 {
		t.Errorf("expected completed=1, got %v", resp.Completed)
	}
	if resp.XPAvailable == nil || *resp.XPAvailable != 4500 {
		t.Errorf("expected xp_available=4500, got %v", resp.XPAvailable)
	}
	if resp.NextExpiry == nil || *resp.NextExpiry != "2024-11-19T17:00:00Z" {
		t.Errorf("expected next_expiry, got %v", resp.NextExpiry)
	}
}

func TestGetChallenges_EmptyDecks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"AssignedDecks": []any{}})
	}))
	defer srv.Close()

	p := newTestProvider("", srv.URL)
	resp := p.GetChallenges(ctxWithAuth(testTokens(), "xuid-test"))

	if !resp.Available {
		t.Errorf("expected available=true even with empty decks")
	}
	if resp.Total == nil || *resp.Total != 0 {
		t.Errorf("expected total=0, got %v", resp.Total)
	}
	if resp.Completed == nil || *resp.Completed != 0 {
		t.Errorf("expected completed=0, got %v", resp.Completed)
	}
	if resp.NextExpiry != nil {
		t.Errorf("expected nil next_expiry for empty decks, got %v", resp.NextExpiry)
	}
}

func TestGetChallenges_HTTP401(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := newTestProvider("", srv.URL)
	resp := p.GetChallenges(ctxWithAuth(testTokens(), "xuid-test"))

	if resp.Available {
		t.Error("expected available=false on 401")
	}
	if resp.ErrorHint == nil || *resp.ErrorHint != "fetch_error" {
		t.Errorf("expected error_hint=fetch_error")
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call (no retry on 401), got %d", callCount)
	}
}

// ---------------------------------------------------------------------------
// parseBattlePassTrack — unité pure
// ---------------------------------------------------------------------------

func TestParseBattlePassTrack_MatchFound(t *testing.T) {
	tracks := []battlePassTrack{
		{RewardTrackPath: "op/Season5", CurrentProgress: battlePassProgress{Rank: 20, PartialProgress: 500}},
		{RewardTrackPath: "op/Season6", CurrentProgress: battlePassProgress{Rank: 45, PartialProgress: 2500}},
	}
	rank, prog, path := parseBattlePassTrack("op/Season6", tracks)
	if rank != 45 || prog != 2500 || path != "op/Season6" {
		t.Errorf("unexpected result: rank=%d prog=%d path=%q", rank, prog, path)
	}
}

func TestParseBattlePassTrack_NoMatch_FallbackFirst(t *testing.T) {
	tracks := []battlePassTrack{
		{RewardTrackPath: "op/Season5", CurrentProgress: battlePassProgress{Rank: 10, PartialProgress: 100}},
	}
	rank, prog, path := parseBattlePassTrack("op/UnknownSeason", tracks)
	if rank != 10 || prog != 100 || path != "op/Season5" {
		t.Errorf("unexpected fallback: rank=%d prog=%d path=%q", rank, prog, path)
	}
}

func TestParseBattlePassTrack_EmptyTracks(t *testing.T) {
	rank, prog, path := parseBattlePassTrack("op/Season6", nil)
	if rank != 0 || prog != 0 || path != "op/Season6" {
		t.Errorf("unexpected result on empty tracks: rank=%d prog=%d path=%q", rank, prog, path)
	}
}
