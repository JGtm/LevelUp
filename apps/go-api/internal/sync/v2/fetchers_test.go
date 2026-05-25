// Package v2 — fetchers_test.go : tests des 3 adapters V2-native
// (MatchListProvider, SharedMatchFetcher, PlayerEnrichmentFetcher).
package v2

import (
	"context"
	"errors"
	"fmt"
	"strings"
	gosync "sync"
	"sync/atomic"
	"testing"

	syncpkg "levelup/go-api/internal/sync"
)

// ─── Mock HaloClient narrow ───────────────────────────────────────────

type mockNarrowClient struct {
	historyByArg        map[string][]syncpkg.MatchHistoryEntry // arg = "xuid(NNN)"
	historyErr          error
	statsByMatch        map[string]map[string]any
	statsErr            error
	skillByMatch        map[string]map[string]*syncpkg.MatchSkillData
	skillErr            error
	highlightByMatch    map[string][]byte // matchID → chunk bytes
	highlightVerByMatch map[string]int    // matchID → film major ver
	highlightErr        error
	// historyArgSeenMu protège historyArgSeen contre les appels concurrents
	// de RunDiscovery (errgroup N joueurs).
	historyArgSeenMu   gosync.Mutex
	historyArgSeen     []string
	statsCallCount     atomic.Int32
	skillCallCount     atomic.Int32
	highlightCallCount atomic.Int32
}

func (m *mockNarrowClient) GetMatchHistory(ctx context.Context, arg, matchType string, start, count int) ([]syncpkg.MatchHistoryEntry, error) {
	m.historyArgSeenMu.Lock()
	m.historyArgSeen = append(m.historyArgSeen, arg)
	m.historyArgSeenMu.Unlock()
	if m.historyErr != nil {
		return nil, m.historyErr
	}
	full := m.historyByArg[arg]
	end := start + count
	if start >= len(full) {
		return nil, nil
	}
	if end > len(full) {
		end = len(full)
	}
	return full[start:end], nil
}

func (m *mockNarrowClient) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	m.statsCallCount.Add(1)
	if m.statsErr != nil {
		return nil, m.statsErr
	}
	return m.statsByMatch[matchID], nil
}

func (m *mockNarrowClient) GetMatchSkill(ctx context.Context, matchID string, xuids []string) (map[string]*syncpkg.MatchSkillData, error) {
	m.skillCallCount.Add(1)
	if m.skillErr != nil {
		return nil, m.skillErr
	}
	return m.skillByMatch[matchID], nil
}

func (m *mockNarrowClient) GetHighlightEventsChunk(ctx context.Context, matchID string) ([]byte, int, bool, error) {
	m.highlightCallCount.Add(1)
	if m.highlightErr != nil {
		return nil, 0, false, m.highlightErr
	}
	chunk, ok := m.highlightByMatch[matchID]
	if !ok {
		return nil, 0, false, nil // 404 simulé (film absent)
	}
	ver := m.highlightVerByMatch[matchID]
	return chunk, ver, true, nil
}

// ─── MatchListProvider tests ──────────────────────────────────────────

func TestMatchListProviderV2_StopsAtFirstKnown(t *testing.T) {
	// API returns [m1, m2, m3, m_known, m4]. Known = {m_known}.
	// Expected unknown : [m1, m2, m3]. m4 jamais demandé.
	client := &mockNarrowClient{
		historyByArg: map[string][]syncpkg.MatchHistoryEntry{
			"xuid(999)": {
				{MatchID: "m1"}, {MatchID: "m2"}, {MatchID: "m3"},
				{MatchID: "m_known"}, {MatchID: "m4"},
			},
		},
	}
	factory := func(gt, xuid string) HaloClient { return client }
	provider := NewMatchListProvider(factory, "matchmaking", 25, 20)

	known := map[string]bool{"m_known": true}
	unknown, err := provider.ListUnknownMatches(context.Background(), PlayerProfile{
		Gamertag: "alice", XUID: "999",
	}, known)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !equalSlice(unknown, []string{"m1", "m2", "m3"}) {
		t.Errorf("unknown = %v, want [m1 m2 m3]", unknown)
	}
}

func TestMatchListProviderV2_XUIDFormat(t *testing.T) {
	// Anti-régression : doit appeler GetMatchHistory avec arg = "xuid(NNN)",
	// pas avec gamertag brut. Incident mai 2026 (14 jours).
	client := &mockNarrowClient{
		historyByArg: map[string][]syncpkg.MatchHistoryEntry{
			"xuid(12345)": {{MatchID: "m1"}},
		},
	}
	factory := func(gt, xuid string) HaloClient { return client }
	provider := NewMatchListProvider(factory, "matchmaking", 25, 20)

	_, err := provider.ListUnknownMatches(context.Background(), PlayerProfile{
		Gamertag: "Alice_GT", XUID: "12345",
	}, map[string]bool{})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(client.historyArgSeen) == 0 {
		t.Fatal("GetMatchHistory pas appelé")
	}
	arg := client.historyArgSeen[0]
	if !strings.HasPrefix(arg, "xuid(") || !strings.HasSuffix(arg, ")") {
		t.Errorf("arg = %q, want xuid(NNN) format", arg)
	}
	if !strings.Contains(arg, "12345") {
		t.Errorf("arg = %q should contain xuid 12345", arg)
	}
	if arg == "Alice_GT" {
		t.Errorf("arg = gamertag brut → incident mai 2026 reproduit")
	}
}

func TestMatchListProviderV2_PaginatesUntilKnown(t *testing.T) {
	// pageSize=2, 4 pages dispo, known dans page 3.
	page1 := []syncpkg.MatchHistoryEntry{{MatchID: "m1"}, {MatchID: "m2"}}
	page2 := []syncpkg.MatchHistoryEntry{{MatchID: "m3"}, {MatchID: "m4"}}
	page3 := []syncpkg.MatchHistoryEntry{{MatchID: "m_known"}, {MatchID: "m5"}}
	full := append(append(page1, page2...), page3...)
	client := &mockNarrowClient{
		historyByArg: map[string][]syncpkg.MatchHistoryEntry{
			"xuid(999)": full,
		},
	}
	provider := NewMatchListProvider(func(gt, xuid string) HaloClient { return client }, "matchmaking", 2, 20)

	unknown, err := provider.ListUnknownMatches(context.Background(),
		PlayerProfile{XUID: "999"},
		map[string]bool{"m_known": true},
	)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !equalSlice(unknown, []string{"m1", "m2", "m3", "m4"}) {
		t.Errorf("unknown = %v, want [m1..m4]", unknown)
	}
}

func TestMatchListProviderV2_MaxPagesSafetyStop(t *testing.T) {
	// pageSize=2, maxPages=3, aucun match connu → stop après 3*2=6 matchs.
	full := make([]syncpkg.MatchHistoryEntry, 20)
	for i := 0; i < 20; i++ {
		full[i] = syncpkg.MatchHistoryEntry{MatchID: fmt.Sprintf("m%02d", i)}
	}
	client := &mockNarrowClient{
		historyByArg: map[string][]syncpkg.MatchHistoryEntry{
			"xuid(999)": full,
		},
	}
	provider := NewMatchListProvider(func(gt, xuid string) HaloClient { return client }, "matchmaking", 2, 3)
	unknown, _ := provider.ListUnknownMatches(context.Background(),
		PlayerProfile{XUID: "999"}, map[string]bool{})
	if len(unknown) != 6 {
		t.Errorf("len(unknown) = %d, want 6 (maxPages=3 × pageSize=2)", len(unknown))
	}
}

func TestMatchListProviderV2_APIError(t *testing.T) {
	client := &mockNarrowClient{historyErr: errors.New("api 500")}
	provider := NewMatchListProvider(func(gt, xuid string) HaloClient { return client }, "matchmaking", 25, 20)
	_, err := provider.ListUnknownMatches(context.Background(), PlayerProfile{XUID: "999"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMatchListProviderV2_NilClientFromFactory(t *testing.T) {
	provider := NewMatchListProvider(func(gt, xuid string) HaloClient { return nil }, "matchmaking", 25, 20)
	_, err := provider.ListUnknownMatches(context.Background(), PlayerProfile{Gamertag: "alice", XUID: "999"}, nil)
	if err == nil {
		t.Fatal("expected error when factory returns nil")
	}
}

func TestMatchListProviderV2_AllMatchesUnknownNoStop(t *testing.T) {
	// page complete avec aucun connu, pagination continue jusqu'à page partielle.
	client := &mockNarrowClient{
		historyByArg: map[string][]syncpkg.MatchHistoryEntry{
			"xuid(999)": {{MatchID: "m1"}, {MatchID: "m2"}, {MatchID: "m3"}}, // 3 matchs, page partielle de 2 puis 1
		},
	}
	provider := NewMatchListProvider(func(gt, xuid string) HaloClient { return client }, "matchmaking", 2, 20)
	unknown, _ := provider.ListUnknownMatches(context.Background(),
		PlayerProfile{XUID: "999"}, map[string]bool{})
	if !equalSlice(unknown, []string{"m1", "m2", "m3"}) {
		t.Errorf("unknown = %v, want all 3", unknown)
	}
}

// ─── SharedMatchFetcher tests ─────────────────────────────────────────

func TestSharedMatchFetcherV2_StatsAndSkillCombined(t *testing.T) {
	client := &mockNarrowClient{
		statsByMatch: map[string]map[string]any{
			"m1": {"score": 100},
		},
		skillByMatch: map[string]map[string]*syncpkg.MatchSkillData{
			"m1": {
				"999": {XUID: "999"},
				"888": {XUID: "888"},
			},
		},
	}
	fetcher := NewSharedMatchFetcher(func(gt, xuid string) HaloClient { return client })
	data, err := fetcher.FetchSharedMatch(context.Background(), "m1",
		PlayerProfile{Gamertag: "alice", XUID: "999", PlayerSlug: "alice"},
		[]PlayerProfile{
			{Gamertag: "alice", XUID: "999"},
			{Gamertag: "bob", XUID: "888"},
		},
	)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if data.Stats["score"] != 100 {
		t.Errorf("Stats[score] = %v, want 100", data.Stats["score"])
	}
	if data.Skill == nil {
		t.Error("Skill should not be nil")
	}
	if data.Skill["999"] == nil || data.Skill["888"] == nil {
		t.Errorf("Skill missing xuids: %v", data.Skill)
	}
	if client.skillCallCount.Load() != 1 {
		t.Errorf("skillCallCount = %d, want 1", client.skillCallCount.Load())
	}
}

func TestSharedMatchFetcherV2_StatsErrorIsFatal(t *testing.T) {
	client := &mockNarrowClient{statsErr: errors.New("404")}
	fetcher := NewSharedMatchFetcher(func(gt, xuid string) HaloClient { return client })
	_, err := fetcher.FetchSharedMatch(context.Background(), "m1",
		PlayerProfile{XUID: "999"}, nil)
	if err == nil {
		t.Fatal("expected stats error to propagate")
	}
}

func TestSharedMatchFetcherV2_SkillErrorIsTolerated(t *testing.T) {
	// Skill API échec → Stats OK, Skill nil, pas d'erreur retournée.
	client := &mockNarrowClient{
		statsByMatch: map[string]map[string]any{"m1": {"ok": true}},
		skillErr:     errors.New("skill 410 expired"),
	}
	fetcher := NewSharedMatchFetcher(func(gt, xuid string) HaloClient { return client })
	data, err := fetcher.FetchSharedMatch(context.Background(), "m1",
		PlayerProfile{XUID: "999"},
		[]PlayerProfile{{XUID: "999"}},
	)
	if err != nil {
		t.Fatalf("Skill error should not be fatal: %v", err)
	}
	if data.Stats["ok"] != true {
		t.Error("Stats should still be present")
	}
	if data.Skill != nil {
		t.Errorf("Skill should be nil when GetMatchSkill fails, got %v", data.Skill)
	}
}

func TestSharedMatchFetcherV2_NoParticipantsSkipsSkillCall(t *testing.T) {
	// 0 participants tracked → pas d'appel skill (économise API).
	client := &mockNarrowClient{
		statsByMatch: map[string]map[string]any{"m1": {}},
	}
	fetcher := NewSharedMatchFetcher(func(gt, xuid string) HaloClient { return client })
	_, _ = fetcher.FetchSharedMatch(context.Background(), "m1",
		PlayerProfile{XUID: "999"}, nil)
	if client.skillCallCount.Load() != 0 {
		t.Errorf("skill should not be called (no participants), got %d calls", client.skillCallCount.Load())
	}
}

func TestSharedMatchFetcherV2_NilClient(t *testing.T) {
	fetcher := NewSharedMatchFetcher(func(gt, xuid string) HaloClient { return nil })
	_, err := fetcher.FetchSharedMatch(context.Background(), "m1",
		PlayerProfile{Gamertag: "alice"}, nil)
	if err == nil {
		t.Fatal("expected error when factory returns nil")
	}
}

func TestSharedMatchFetcherV2_HighlightsFetchedInPhase3(t *testing.T) {
	// T2 (parité V1) : vérifier que les highlights sont fetchés inline
	// en Phase 3 et propagés dans SharedMatchData.
	client := &mockNarrowClient{
		statsByMatch: map[string]map[string]any{"m1": {"k": 1}},
		highlightByMatch: map[string][]byte{
			"m1": []byte("FAKE_CHUNK_DATA"),
		},
		highlightVerByMatch: map[string]int{"m1": 42},
	}
	fetcher := NewSharedMatchFetcher(func(gt, xuid string) HaloClient { return client })
	data, err := fetcher.FetchSharedMatch(context.Background(), "m1",
		PlayerProfile{XUID: "999"}, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !data.HasHighlights {
		t.Error("HasHighlights should be true")
	}
	if string(data.HighlightChunk) != "FAKE_CHUNK_DATA" {
		t.Errorf("HighlightChunk = %q, want FAKE_CHUNK_DATA", string(data.HighlightChunk))
	}
	if data.FilmMajorVer != 42 {
		t.Errorf("FilmMajorVer = %d, want 42", data.FilmMajorVer)
	}
	if client.highlightCallCount.Load() != 1 {
		t.Errorf("highlightCallCount = %d, want 1", client.highlightCallCount.Load())
	}
}

func TestSharedMatchFetcherV2_HighlightsAbsentToleratedAsFalse(t *testing.T) {
	// Film 404/410 → highlightByMatch ne contient pas m1 → found=false,
	// pas d'erreur. V1-compatible.
	client := &mockNarrowClient{
		statsByMatch:     map[string]map[string]any{"m1": {"k": 1}},
		highlightByMatch: map[string][]byte{}, // vide → 404 simulé
	}
	fetcher := NewSharedMatchFetcher(func(gt, xuid string) HaloClient { return client })
	data, err := fetcher.FetchSharedMatch(context.Background(), "m1",
		PlayerProfile{XUID: "999"}, nil)
	if err != nil {
		t.Fatalf("Highlights absent should not be fatal: %v", err)
	}
	if data.HasHighlights {
		t.Error("HasHighlights should be false when film absent")
	}
	if data.HighlightChunk != nil {
		t.Errorf("HighlightChunk should be nil, got %v", data.HighlightChunk)
	}
}

func TestSharedMatchFetcherV2_HighlightsErrorToleratedAsFalse(t *testing.T) {
	// Erreur réseau sur GetHighlightEventsChunk → continue avec HasHighlights=false.
	client := &mockNarrowClient{
		statsByMatch: map[string]map[string]any{"m1": {"k": 1}},
		highlightErr: errors.New("network error"),
	}
	fetcher := NewSharedMatchFetcher(func(gt, xuid string) HaloClient { return client })
	data, err := fetcher.FetchSharedMatch(context.Background(), "m1",
		PlayerProfile{XUID: "999"}, nil)
	if err != nil {
		t.Fatalf("Highlights error should not be fatal: %v", err)
	}
	if data.HasHighlights {
		t.Error("HasHighlights should be false on error")
	}
}

// ─── PlayerEnrichmentFetcher tests ────────────────────────────────────

func TestPlayerEnrichmentFetcherV2_NoOpReturnsEmptyData(t *testing.T) {
	// D6.3 : no-op pour main sync (V1 ne fait pas d'appel API per-player).
	fetcher := NewPlayerEnrichmentFetcher()
	data, err := fetcher.FetchPlayerEnrichment(context.Background(),
		PlayerProfile{PlayerSlug: "alice", Gamertag: "alice_GT"},
		"m1",
	)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if data.PlayerSlug != "alice" || data.MatchID != "m1" {
		t.Errorf("PlayerSlug/MatchID not set : %+v", data)
	}
	if data.Data != nil {
		t.Errorf("Data should be nil (no-op), got %v", data.Data)
	}
}
