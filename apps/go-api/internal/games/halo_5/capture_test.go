package halo_5

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// --- fixtures -------------------------------------------------------------

// matchJSON construit un H5MatchResult minimal (self = JGtm vainqueur) au shape réel.
// GameMode 1 = arène (le mode collecté).
func matchJSON(id string) string { return matchJSONMode(id, 1) }

// matchJSONMode : variante de matchJSON avec GameMode paramétrable
// (1=arène, 2=campagne, 4=Warzone) — sert à tester l'exclusion à la collecte.
func matchJSONMode(id string, gameMode int) string {
	return fmt.Sprintf(`{"Id":{"MatchId":%q,"GameMode":%d},"HopperId":"h0","MapId":"m0",`+
		`"GameBaseVariantId":"g0","MatchDuration":"PT5M0S",`+
		`"MatchCompletedDate":{"ISO8601Date":"2023-04-05T00:00:00Z"},`+
		`"Teams":[{"Id":1,"Score":3,"Rank":1}],`+
		`"Players":[{"Player":{"Gamertag":"JGtm","Xuid":null},"TeamId":1,"Rank":1,"Result":3,`+
		`"TotalKills":5,"TotalDeaths":3,"TotalAssists":2}],"IsTeamGame":true,"SeasonId":""}`, id, gameMode)
}

func mustMatches(t *testing.T, ids ...string) *H5MatchesResponse {
	t.Helper()
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = matchJSON(id)
	}
	raw := fmt.Sprintf(`{"Start":0,"Count":%d,"ResultCount":%d,"Results":[%s]}`,
		len(ids), len(ids), strings.Join(parts, ","))
	var resp H5MatchesResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("mustMatches unmarshal: %v", err)
	}
	return &resp
}

// killEvents : une timeline avec 1 kill (arme) + 1 médaille.
func killEvents() *h5MatchEventsResponse {
	return &h5MatchEventsResponse{GameEvents: []h5GameEvent{
		{EventName: "Death", TimeSinceStart: "PT1M0S", IsWeapon: true,
			Killer: &h5EventPlayer{Gamertag: "JGtm"}, Victim: &h5EventPlayer{Gamertag: "Foe"},
			KillerWeaponStockId: 523953283},
		{EventName: "Medal", TimeSinceStart: "PT1M2S",
			Player: &h5EventPlayer{Gamertag: "JGtm"}, MedalId: 824733727},
	}}
}

// fakeH5Source — source live mockée (par page de matchs + par timeline de match).
type fakeH5Source struct {
	pages      map[int]*H5MatchesResponse        // start -> page
	events     map[string]*h5MatchEventsResponse // matchID -> timeline
	eventsErr  map[string]error                  // matchID -> erreur fetch events
	carnage    map[string]*H5CarnageResponse     // matchID -> carnage (roster)
	carnageErr map[string]error                  // matchID -> erreur fetch carnage
}

func (f *fakeH5Source) GetServiceRecords(context.Context, string, string) (*H5ServiceRecordResponse, error) {
	return nil, errors.New("unused")
}
func (f *fakeH5Source) GetPlayerMatches(_ context.Context, _ string, start, _ int) (*H5MatchesResponse, error) {
	if r, ok := f.pages[start]; ok {
		return r, nil
	}
	return &H5MatchesResponse{}, nil // page vide = fin
}
func (f *fakeH5Source) GetMatchEvents(_ context.Context, matchID string) (*h5MatchEventsResponse, error) {
	if err := f.eventsErr[matchID]; err != nil {
		return nil, err
	}
	if r, ok := f.events[matchID]; ok {
		return r, nil
	}
	return &h5MatchEventsResponse{}, nil
}
func (f *fakeH5Source) GetMatchCarnage(_ context.Context, matchID, _ string) (*H5CarnageResponse, error) {
	if err := f.carnageErr[matchID]; err != nil {
		return nil, err
	}
	if r, ok := f.carnage[matchID]; ok {
		return r, nil
	}
	return &H5CarnageResponse{}, nil // pas de roster = pas de participants (toléré)
}

var _ CaptureSource = (*fakeH5Source)(nil)

func jgtmViewer() canonical.PlayerIdentity {
	return canonical.PlayerIdentity{Gamertag: "JGtm", XUID: "xJG"}
}

func idResolver(gt string) string {
	if gt == "JGtm" {
		return "xJG"
	}
	return "" // adversaires non résolus -> "" toléré (identité reste dans le gamertag)
}

// --- tests ----------------------------------------------------------------

func TestCollectRecentMatches_HappyPath(t *testing.T) {
	src := &fakeH5Source{
		pages:  map[int]*H5MatchesResponse{0: mustMatches(t, "m1", "m2")},
		events: map[string]*h5MatchEventsResponse{"m1": killEvents(), "m2": killEvents()},
	}
	batches, stats, err := CollectRecentMatches(context.Background(), src, jgtmViewer(), idResolver, nil,
		CaptureOptions{PageSize: 25})
	if err != nil {
		t.Fatalf("CollectRecentMatches: %v", err)
	}
	if len(batches) != 2 {
		t.Fatalf("batches = %d, want 2", len(batches))
	}
	if stats.MatchesSeen != 2 || stats.MatchesCollected != 2 || stats.StoppedOnKnown || stats.EventsFailed != 0 {
		t.Errorf("stats = %+v, want seen2/collected2/no-known/no-failed", stats)
	}
	for i, b := range batches {
		if b == nil {
			t.Errorf("batch %d nil", i)
		}
	}
}

// TestCollectRecentMatches_ExcludesCampaignAndWarzone : les matchs Campagne (GameMode 2)
// et Warzone (GameMode 4) sont écartés AVANT carnage/events ; seul l'arène (1) est collecté.
// Compteurs ExcludedCampaign / ExcludedWarzone incrémentés (cf. isExcludedH5GameMode).
func TestCollectRecentMatches_ExcludesCampaignAndWarzone(t *testing.T) {
	raw := fmt.Sprintf(`{"Start":0,"Count":3,"ResultCount":3,"Results":[%s,%s,%s]}`,
		matchJSONMode("arena1", 1), matchJSONMode("camp1", 2), matchJSONMode("wz1", 4))
	var page H5MatchesResponse
	if err := json.Unmarshal([]byte(raw), &page); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	src := &fakeH5Source{
		pages:  map[int]*H5MatchesResponse{0: &page},
		events: map[string]*h5MatchEventsResponse{"arena1": killEvents()},
	}
	batches, stats, err := CollectRecentMatches(context.Background(), src, jgtmViewer(), idResolver, nil,
		CaptureOptions{PageSize: 25})
	if err != nil {
		t.Fatalf("CollectRecentMatches: %v", err)
	}
	if len(batches) != 1 || stats.MatchesCollected != 1 {
		t.Fatalf("batches=%d collected=%d, want 1/1 (arène seule)", len(batches), stats.MatchesCollected)
	}
	if stats.ExcludedCampaign != 1 {
		t.Errorf("ExcludedCampaign=%d, want 1", stats.ExcludedCampaign)
	}
	if stats.ExcludedWarzone != 1 {
		t.Errorf("ExcludedWarzone=%d, want 1", stats.ExcludedWarzone)
	}
	// 3 résumés parcourus, mais campagne + warzone écartés AVANT toute collecte coûteuse.
	if stats.MatchesSeen != 3 {
		t.Errorf("MatchesSeen=%d, want 3", stats.MatchesSeen)
	}
}

func TestCollectRecentMatches_DeltaStopOnKnown(t *testing.T) {
	src := &fakeH5Source{
		pages:  map[int]*H5MatchesResponse{0: mustMatches(t, "m1", "m2", "m3")},
		events: map[string]*h5MatchEventsResponse{"m1": killEvents()},
	}
	// m2 déjà connu -> on collecte m1 puis on s'arrête (delta).
	isKnown := func(id string) bool { return id == "m2" }
	batches, stats, err := CollectRecentMatches(context.Background(), src, jgtmViewer(), idResolver, isKnown,
		CaptureOptions{PageSize: 25})
	if err != nil {
		t.Fatalf("CollectRecentMatches: %v", err)
	}
	if len(batches) != 1 || !stats.StoppedOnKnown {
		t.Errorf("len=%d stoppedOnKnown=%v, want 1 + true", len(batches), stats.StoppedOnKnown)
	}
	if stats.MatchesCollected != 1 {
		t.Errorf("collected = %d, want 1 (m1 avant le connu m2)", stats.MatchesCollected)
	}
}

func TestCollectRecentMatches_EventsFailureStillCollects(t *testing.T) {
	src := &fakeH5Source{
		pages:     map[int]*H5MatchesResponse{0: mustMatches(t, "m1")},
		eventsErr: map[string]error{"m1": errors.New("403 token expiré")},
	}
	batches, stats, err := CollectRecentMatches(context.Background(), src, jgtmViewer(), idResolver, nil,
		CaptureOptions{PageSize: 25})
	if err != nil {
		t.Fatalf("CollectRecentMatches: %v", err)
	}
	// events KO -> match tout de même collecté (registry seul) + EventsFailed compté.
	if len(batches) != 1 || stats.EventsFailed != 1 {
		t.Errorf("len=%d eventsFailed=%d, want 1 + 1 (registry-only)", len(batches), stats.EventsFailed)
	}
}

func TestCollectRecentMatches_MaxMatchesCap(t *testing.T) {
	src := &fakeH5Source{pages: map[int]*H5MatchesResponse{0: mustMatches(t, "m1", "m2", "m3")}}
	batches, stats, err := CollectRecentMatches(context.Background(), src, jgtmViewer(), idResolver, nil,
		CaptureOptions{PageSize: 25, MaxMatches: 2})
	if err != nil {
		t.Fatalf("CollectRecentMatches: %v", err)
	}
	if len(batches) != 2 || stats.MatchesCollected != 2 {
		t.Errorf("len=%d collected=%d, want 2 (cap MaxMatches)", len(batches), stats.MatchesCollected)
	}
}

func TestCollectRecentMatches_Paginates(t *testing.T) {
	src := &fakeH5Source{pages: map[int]*H5MatchesResponse{
		0: mustMatches(t, "m1", "m2"), // page pleine -> on continue
		2: mustMatches(t, "m3"),       // page incomplète -> dernière
	}}
	batches, stats, err := CollectRecentMatches(context.Background(), src, jgtmViewer(), idResolver, nil,
		CaptureOptions{PageSize: 2})
	if err != nil {
		t.Fatalf("CollectRecentMatches: %v", err)
	}
	if len(batches) != 3 || stats.MatchesSeen != 3 {
		t.Errorf("len=%d seen=%d, want 3 (2 pages)", len(batches), stats.MatchesSeen)
	}
}

// TestCapturePageAt_SkipKnownNoStop — la capture par page (backfill) SAUTE les matchs
// connus (MatchesSkipped) mais NE s'arrête PAS dessus (contraste avec le delta-stop de
// CollectRecentMatches) : le batch des matchs nouveaux de la page est tout de même
// produit, et hasMore reflète la complétude de la page.
func TestCapturePageAt_SkipKnownNoStop(t *testing.T) {
	src := &fakeH5Source{
		pages:  map[int]*H5MatchesResponse{0: mustMatches(t, "m1", "m2", "m3")},
		events: map[string]*h5MatchEventsResponse{"m3": killEvents()},
	}
	isKnown := func(id string) bool { return id == "m1" || id == "m2" } // 2 connus, m3 nouveau
	var stats CaptureStats
	seen := make(map[string]struct{})
	batches, hasMore, err := CapturePageAt(context.Background(), src, jgtmViewer(), idResolver, isKnown,
		CaptureOptions{}, 0, 3, seen, &stats)
	if err != nil {
		t.Fatalf("CapturePageAt: %v", err)
	}
	// m1/m2 sautés, m3 collecté — PAS de delta-stop (StoppedOnKnown reste false).
	if len(batches) != 1 || stats.MatchesSkipped != 2 || stats.MatchesCollected != 1 {
		t.Errorf("batches=%d skipped=%d collected=%d, want 1/2/1", len(batches), stats.MatchesSkipped, stats.MatchesCollected)
	}
	if stats.StoppedOnKnown {
		t.Error("StoppedOnKnown doit rester false (backfill, pas de delta-stop)")
	}
	// Page pleine (3 == pageSize) → hasMore=true (il reste probablement des pages).
	if !hasMore {
		t.Error("hasMore = false, want true (page pleine → continuer à paginer)")
	}
}

// TestCapturePageAt_EmptyPageStops — une page vide (fin d'historique) → aucun batch +
// hasMore=false (signal d'arrêt du backfill).
func TestCapturePageAt_EmptyPageStops(t *testing.T) {
	src := &fakeH5Source{pages: map[int]*H5MatchesResponse{}} // tout start -> page vide
	var stats CaptureStats
	batches, hasMore, err := CapturePageAt(context.Background(), src, jgtmViewer(), idResolver, nil,
		CaptureOptions{}, 100, 25, nil, &stats)
	if err != nil {
		t.Fatalf("CapturePageAt: %v", err)
	}
	if len(batches) != 0 || hasMore {
		t.Errorf("batches=%d hasMore=%v, want 0 + false (page vide arrête)", len(batches), hasMore)
	}
}

func TestCollectRecentMatches_GetMatchesError(t *testing.T) {
	src := &errSource{}
	if _, _, err := CollectRecentMatches(context.Background(), src, jgtmViewer(), idResolver, nil, CaptureOptions{}); err == nil {
		t.Error("erreur GetPlayerMatches doit remonter")
	}
}

type errSource struct{ fakeH5Source }

func (errSource) GetPlayerMatches(context.Context, string, int, int) (*H5MatchesResponse, error) {
	return nil, errors.New("réseau KO")
}

// roster2 : carnage d'équipe 2 joueurs (JGtm équipe gagnante Rank 1, Foe perdante).
func roster2() *H5CarnageResponse {
	return &H5CarnageResponse{
		IsTeamGame: true,
		TeamStats:  []H5CarnageTeam{{TeamId: 0, Rank: 2}, {TeamId: 1, Rank: 1}},
		PlayerStats: []H5CarnagePlayer{
			{Player: H5PlayerRef{Gamertag: "JGtm"}, TeamId: 1, Rank: 1,
				TotalKills: 10, TotalDeaths: 14, TotalAssists: 11,
				TotalTimePlayed: "PT5M0S", AvgLifeTimeOfPlayer: "PT16S"},
			{Player: H5PlayerRef{Gamertag: "Foe"}, TeamId: 0, Rank: 5,
				TotalKills: 8, TotalDeaths: 13, TotalAssists: 6},
		},
	}
}

func TestCollectRecentMatches_Participants(t *testing.T) {
	src := &fakeH5Source{
		pages:   map[int]*H5MatchesResponse{0: mustMatches(t, "m1")},
		carnage: map[string]*H5CarnageResponse{"m1": roster2()},
	}
	batches, stats, err := CollectRecentMatches(context.Background(), src, jgtmViewer(), idResolver, nil,
		CaptureOptions{PageSize: 25})
	if err != nil {
		t.Fatalf("CollectRecentMatches: %v", err)
	}
	if len(batches) != 1 {
		t.Fatalf("batches = %d, want 1", len(batches))
	}
	parts := batches[0].Shared.Participants
	// Foe (xuid "" via idResolver) est SAUTÉ (resolve-or-skip) → seul JGtm (xJG).
	if len(parts) != 1 {
		t.Fatalf("participants = %d, want 1 (Foe non résolu → skip)", len(parts))
	}
	jg := &parts[0]
	if jg.XUID != "xJG" {
		t.Fatalf("participant = %q, want JGtm (xJG)", jg.XUID)
	}
	// Équipe gagnante (Rank 1) → Win ; comptes bruts repris ; KDA JAMAIS fabriqué.
	if jg.Outcome == nil || *jg.Outcome != domain.OutcomeWin {
		t.Errorf("outcome JGtm = %v, want win(2)", jg.Outcome)
	}
	if jg.Kills == nil || *jg.Kills != 10 {
		t.Errorf("kills JGtm = %v, want 10", jg.Kills)
	}
	// KDA : calculé à l'ingestion (FDA NET h5), stocké — non nil.
	if jg.KDA == nil {
		t.Error("KDA h5 doit être calculé à l'ingestion (FDA NET), got nil")
	} else if jg.Kills != nil && jg.Assists != nil && jg.Deaths != nil {
		want := float64(*jg.Kills) + float64(*jg.Assists)/3.0 - float64(*jg.Deaths)
		if *jg.KDA != want {
			t.Errorf("KDA h5 = %v, want FDA NET %v", *jg.KDA, want)
		}
	}
	if jg.DamageTaken != nil {
		t.Errorf("DamageTaken h5 doit rester nil (absent de l'API), got %v", *jg.DamageTaken)
	}
	if stats.CarnageFailed != 0 {
		t.Errorf("CarnageFailed = %d, want 0", stats.CarnageFailed)
	}
}

func TestCollectRecentMatches_CarnageFailureStillCollects(t *testing.T) {
	src := &fakeH5Source{
		pages:      map[int]*H5MatchesResponse{0: mustMatches(t, "m1")},
		carnageErr: map[string]error{"m1": errors.New("403 token expiré")},
	}
	batches, stats, err := CollectRecentMatches(context.Background(), src, jgtmViewer(), idResolver, nil,
		CaptureOptions{PageSize: 25})
	if err != nil {
		t.Fatalf("CollectRecentMatches: %v", err)
	}
	// carnage KO → match collecté sans participants + CarnageFailed compté.
	if len(batches) != 1 || stats.CarnageFailed != 1 {
		t.Errorf("len=%d carnageFailed=%d, want 1 + 1", len(batches), stats.CarnageFailed)
	}
	if len(batches[0].Shared.Participants) != 0 {
		t.Errorf("carnage KO → 0 participants, got %d", len(batches[0].Shared.Participants))
	}
}
