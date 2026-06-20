package halo_5

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

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
	pages     map[int]*H5MatchesResponse        // start -> page
	events    map[string]*h5MatchEventsResponse // matchID -> timeline
	eventsErr map[string]error                  // matchID -> erreur fetch events
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

var _ h5Source = (*fakeH5Source)(nil)

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
