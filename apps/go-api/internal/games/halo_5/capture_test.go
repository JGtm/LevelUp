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
func matchJSON(id string) string {
	return fmt.Sprintf(`{"Id":{"MatchId":%q,"GameMode":1},"HopperId":"h0","MapId":"m0",`+
		`"GameBaseVariantId":"g0","MatchDuration":"PT5M0S",`+
		`"MatchCompletedDate":{"ISO8601Date":"2023-04-05T00:00:00Z"},`+
		`"Teams":[{"Id":1,"Score":3,"Rank":1}],`+
		`"Players":[{"Player":{"Gamertag":"JGtm","Xuid":null},"TeamId":1,"Rank":1,"Result":3,`+
		`"TotalKills":5,"TotalDeaths":3,"TotalAssists":2}],"IsTeamGame":true,"SeasonId":""}`, id)
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

var _ h5CaptureSource = (*fakeH5Source)(nil)

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
	if len(parts) != 2 {
		t.Fatalf("participants = %d, want 2 (roster carnage)", len(parts))
	}
	var jg *domain.MatchParticipantRow
	for i := range parts {
		if parts[i].XUID == "xJG" {
			jg = &parts[i]
		}
	}
	if jg == nil {
		t.Fatal("JGtm (xuid résolu xJG) absent du roster")
	}
	// Équipe gagnante (Rank 1) → Win ; comptes bruts repris ; KDA JAMAIS fabriqué.
	if jg.Outcome == nil || *jg.Outcome != domain.OutcomeWin {
		t.Errorf("outcome JGtm = %v, want win(2)", jg.Outcome)
	}
	if jg.Kills == nil || *jg.Kills != 10 {
		t.Errorf("kills JGtm = %v, want 10", jg.Kills)
	}
	if jg.KDA != nil {
		t.Errorf("KDA h5 doit rester nil (jamais fabriqué), got %v", *jg.KDA)
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
