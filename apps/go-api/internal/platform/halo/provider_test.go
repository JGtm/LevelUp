// Package halo — provider_test.go : tests unitaires HaloProvider (boîte blanche).
package halo

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

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// newTestProvider crée un provider avec des base URLs injectées depuis les serveurs de test.
func newTestProvider(battlePassURL, challengesURL string) *HaloProvider {
	p := NewHaloProvider()
	p.battlePassBaseURL = battlePassURL
	p.challengesBaseURL = challengesURL
	if challengesURL != "" {
		p.gameCMSBaseURL = challengesURL
	}
	p.limiter = newRateLimiter(600)
	p.maxRetries = 3
	return p
}

func testTokens() *domain.HaloTokens {
	return &domain.HaloTokens{SpartanToken: "spartan_test", ClearanceToken: "clear_test"}
}

// ctxWithAuth construit un contexte avec auth Halo (tokens + xuid).
//
//nolint:unparam // xuid est gardé pour l'extension future à d'autres tests
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

// challengeDefsByPath retourne un mockResolver qui sert des définitions de challenges JSON.
func challengeDefsByPath(defs map[string]map[string]any) *mockResolver {
	return &mockResolver{
		getFunc: func(_ context.Context, ref assets.Ref) (assets.Resolved, error) {
			if ref.Kind != assets.KindChallengeDefinition {
				return assets.Resolved{}, assets.ErrNotFound
			}
			data, ok := defs[ref.ID]
			if !ok {
				return assets.Resolved{}, assets.ErrNotFound
			}
			b, _ := json.Marshal(data)
			return assets.Resolved{Payload: assets.JSONPayload{RawJSON: b}}, nil
		},
	}
}

func TestGetChallenges_LiveOK(t *testing.T) {
	// definitions servies via mockResolver (P4/P5)
	resolver := challengeDefsByPath(map[string]map[string]any{
		"ChallengeContent/ClientChallengeDefinitions/WeeklyChallenges/Action/ch1.json": {
			"Title":               map[string]any{"translations": map[string]any{"fr-FR": "Défi avancé"}, "value": "Advanced Challenge"},
			"Description":         map[string]any{"translations": map[string]any{"fr-FR": "Gagne des points rapidement"}, "value": "Gain points quickly"},
			"Category":            "Weekly",
			"Difficulty":          "Heroic",
			"ThresholdForSuccess": 10,
		},
		"ChallengeContent/ClientChallengeDefinitions/DailyChallenges/ch2.json": {
			"Title":               map[string]any{"translations": map[string]any{"fr-FR": "Défi en cours"}, "value": "In Progress Challenge"},
			"Description":         map[string]any{"translations": map[string]any{"fr-FR": "Continue la progression"}, "value": "Keep going"},
			"Category":            "Daily",
			"Difficulty":          "Normal",
			"ThresholdForSuccess": 3,
		},
		"ChallengeContent/ClientChallengeDefinitions/WeeklyChallenges/Weapon/ch3.json": {
			"Title":               map[string]any{"translations": map[string]any{"fr-FR": "Défi pas commencé"}, "value": "Not Started Challenge"},
			"Description":         map[string]any{"translations": map[string]any{"fr-FR": "Commence ce défi"}, "value": "Start this challenge"},
			"Category":            "Weekly",
			"Difficulty":          "Legendary",
			"ThresholdForSuccess": 5,
		},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/hi/players/xuid(xuid-test)/decks":
			if r.Header.Get("x-343-authorization-spartan") == "" {
				http.Error(w, "missing spartan", http.StatusUnauthorized)
				return
			}
			payload := map[string]any{
				"AssignedDecks": []any{
					map[string]any{
						"Expiration": map[string]string{"ISO8601Date": "2024-11-19T17:00:00Z"},
						"ActiveChallenges": []any{
							map[string]any{"Path": "ChallengeContent/ClientChallengeDefinitions/WeeklyChallenges/Action/ch1.json", "XPReward": 1000, "Progress": 7, "Threshold": 10, "TrackingId": "track-1"},
							map[string]any{"Path": "ChallengeContent/ClientChallengeDefinitions/DailyChallenges/ch2.json", "XPReward": 2000, "Progress": 1, "Threshold": 3, "TrackingId": "track-2"},
							map[string]any{"Path": "ChallengeContent/ClientChallengeDefinitions/WeeklyChallenges/Weapon/ch3.json", "XPReward": 1500, "Progress": 0, "Threshold": 5, "TrackingId": "track-3"},
						},
						"CompletedChallenges": []any{
							map[string]any{"XPReward": 500},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p := newTestProvider("", srv.URL).WithAssetResolver(resolver)
	resp := p.GetChallenges(ctxWithAuth(testTokens(), "xuid-test"))

	if !resp.Available {
		t.Errorf("expected available=true, error_hint=%v", resp.ErrorHint)
	}
	if resp.Total == nil || *resp.Total != 4 {
		t.Errorf("expected total=4, got %v", resp.Total)
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
	if len(resp.Items) != 3 {
		t.Fatalf("expected 3 active items, got %d", len(resp.Items))
	}
	if resp.Items[0].Title != "Défi avancé" || resp.Items[1].Title != "Défi en cours" || resp.Items[2].Title != "Défi pas commencé" {
		t.Fatalf("unexpected item order: %#v", resp.Items)
	}
	if resp.Items[0].ProgressPercent == nil || int(*resp.Items[0].ProgressPercent) != 70 {
		t.Fatalf("expected first item progress 70%%, got %#v", resp.Items[0].ProgressPercent)
	}
	if resp.Items[2].ProgressPercent == nil || int(*resp.Items[2].ProgressPercent) != 0 {
		t.Fatalf("expected third item progress 0%%, got %#v", resp.Items[2].ProgressPercent)
	}
	if resp.Items[0].Description == nil || *resp.Items[0].Description == "" {
		t.Fatal("expected description on enriched challenge")
	}
	if resp.Items[0].ImageURL == nil || *resp.Items[0].ImageURL == "" {
		t.Fatal("expected image_url on enriched challenge")
	}
	if !strings.HasPrefix(*resp.Items[0].ImageURL, "/api/v1/assets/challenge-badge/") {
		t.Fatalf("expected challenge badge API URL, got %v", resp.Items[0].ImageURL)
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

func TestGetChallengesWithRaw_ConcurrentCallsShareSingleflight(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hi/players/xuid(xuid-test)/decks" {
			http.NotFound(w, r)
			return
		}
		atomic.AddInt32(&callCount, 1)
		time.Sleep(75 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"AssignedDecks": []any{},
		})
	}))
	defer srv.Close()

	p := newTestProvider("", srv.URL)
	ctx := ctxWithAuth(testTokens(), "xuid-test")

	const concurrentCalls = 6
	results := make(chan domain.ChallengesResponse, concurrentCalls)
	errs := make(chan error, concurrentCalls)
	var wg sync.WaitGroup
	for range concurrentCalls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, _ := p.GetChallengesWithRaw(ctx)
			results <- resp
			errs <- nil
		}()
	}
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	for resp := range results {
		if !resp.Available {
			t.Fatalf("expected available=true, got error_hint=%v", resp.ErrorHint)
		}
		if resp.Total == nil || *resp.Total != 0 {
			t.Fatalf("expected total=0, got %v", resp.Total)
		}
	}
	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Fatalf("expected a single live challenges call, got %d", got)
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
