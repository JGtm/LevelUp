package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/games/canonical"
)

func mkPMRowContrib(
	gt string,
	startedAt time.Time,
	durationSec int,
	kills, deaths, assists, headshots, powerWeapons, maxSpree int,
	perfScore float64,
) canonical.PlayerMatchRow {
	d := durationSec
	k, dth, a := kills, deaths, assists
	hs, pk, ms := headshots, powerWeapons, maxSpree
	score := perfScore
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			StartedAtUTC:    startedAt,
			DurationSeconds: &d,
		},
		Self: canonical.MatchParticipant{
			Identity:         canonical.PlayerIdentity{Gamertag: gt},
			Kills:            &k,
			Deaths:           &dth,
			Assists:          &a,
			HeadshotKills:    &hs,
			PowerWeaponKills: &pk,
			MaxKillingSpree:  &ms,
		},
		Enrichment: canonical.PlayerMatchEnrichment{
			PerformanceScore: &score,
		},
	}
}

func TestBuildPerMinuteStats_AggregatesByPlayer(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	// p1 : 10 kills + 5 deaths + 8 assists sur 600s (10 min) -> 1.0/0.5/0.8 par min
	// p2 : 6 kills + 3 deaths + 0 assists sur 300s (5 min)   -> 1.2/0.6/0.0 par min
	rows := map[string][]canonical.PlayerMatchRow{
		"main": {mkPMRowContrib("main", t0, 600, 10, 5, 8, 0, 0, 0, 0)},
		"f1":   {mkPMRowContrib("f1", t0, 300, 6, 3, 0, 0, 0, 0, 0)},
	}
	chart := BuildPerMinuteStats(rows)
	if chart.Key != "squad.contrib.per_minute" {
		t.Errorf("Key want squad.contrib.per_minute, got %s", chart.Key)
	}
	if len(chart.Datapoints) != 2 {
		t.Fatalf("want 2 datapoints, got %d", len(chart.Datapoints))
	}
	// Tri alpha : f1 puis main
	if chart.Datapoints[0].Category != "f1" || chart.Datapoints[1].Category != "main" {
		t.Errorf("categories order want [f1, main], got [%s, %s]",
			chart.Datapoints[0].Category, chart.Datapoints[1].Category)
	}
	// f1 : 6/5=1.2 kills/min
	if got := chart.Datapoints[0].Components["kills_per_min"]; got != 1.2 {
		t.Errorf("f1 kills_per_min want 1.2, got %f", got)
	}
	// main : 10/10=1.0 kills/min
	if got := chart.Datapoints[1].Components["kills_per_min"]; got != 1.0 {
		t.Errorf("main kills_per_min want 1.0, got %f", got)
	}
}

func TestBuildPerMinuteStats_SkipsRowsWithoutDuration(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	row := mkPMRowContrib("main", t0, 600, 10, 0, 0, 0, 0, 0, 0)
	row.Summary.DurationSeconds = nil // pas de duree
	rows := map[string][]canonical.PlayerMatchRow{"main": {row}}
	chart := BuildPerMinuteStats(rows)
	if len(chart.Datapoints) != 0 {
		t.Errorf("want 0 datapoints (row sans duration), got %d", len(chart.Datapoints))
	}
}

func TestBuildFragsDeathsCombined(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	rows := map[string][]canonical.PlayerMatchRow{
		"main": {
			mkPMRowContrib("main", t0, 600, 10, 5, 0, 0, 0, 0, 0),
			mkPMRowContrib("main", t0.Add(time.Hour), 600, 8, 7, 0, 0, 0, 0, 0),
		},
	}
	chart := BuildFragsDeathsCombined(rows)
	if len(chart.Datapoints) != 1 {
		t.Fatalf("want 1 datapoint, got %d", len(chart.Datapoints))
	}
	if got := chart.Datapoints[0].Components["kills"]; got != 18 {
		t.Errorf("kills want 18, got %f", got)
	}
	if got := chart.Datapoints[0].Components["deaths"]; got != 12 {
		t.Errorf("deaths want 12, got %f", got)
	}
}

func TestBuildKillingSpreeMax_SmoothsValues(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	rows := map[string][]canonical.PlayerMatchRow{}
	// 10 matchs avec spree croissant 1..10
	for i := 0; i < 10; i++ {
		rows["main"] = append(rows["main"],
			mkPMRowContrib("main", base.Add(time.Duration(i)*time.Hour), 600, 0, 0, 0, 0, 0, i+1, 0),
		)
	}
	out := BuildKillingSpreeMax(rows, nil, nil)
	if len(out) != 1 {
		t.Fatalf("want 1 series, got %d", len(out))
	}
	if len(out[0].Datapoints) == 0 {
		t.Errorf("want >0 datapoints from RollingMeanAdaptive")
	}
	// La moyenne lissee doit etre croissante (la fenetre roule sur des valeurs croissantes).
	for i := 1; i < len(out[0].Datapoints); i++ {
		if out[0].Datapoints[i].Y < out[0].Datapoints[i-1].Y {
			t.Errorf("smoothed should be monotonically non-decreasing on monotonic input, [%d]=%f < [%d]=%f",
				i, out[0].Datapoints[i].Y, i-1, out[0].Datapoints[i-1].Y)
		}
	}
}

func TestBuildKillingSpreeMax_SkipsNilSpree(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	row := mkPMRowContrib("main", t0, 600, 0, 0, 0, 0, 0, 0, 0)
	row.Self.MaxKillingSpree = nil
	rows := map[string][]canonical.PlayerMatchRow{"main": {row}}
	// Sans events ni xuid map → la row sans native est skippée (pas de calcul possible).
	out := BuildKillingSpreeMax(rows, nil, nil)
	if len(out) != 0 {
		t.Errorf("want 0 series (toutes rows sans spree, pas d'events), got %d", len(out))
	}
}

// TestBuildKillingSpreeMax_ComputesFromEventsWhenNativeAbsent — quand la valeur native
// est absente (Halo 5), la spree est CALCULÉE depuis les events kill/death du match
// (analysis.ComputeMaxKillingSpree). La série n'est donc PAS masquée. 6 matchs pour
// dépasser la fenêtre minimale du lissage (SpreeRollingMinWindow=5).
func TestBuildKillingSpreeMax_ComputesFromEventsWhenNativeAbsent(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	const xuid = "x_main"

	var playerRows []canonical.PlayerMatchRow
	var events []canonical.HighlightEvent
	for i := 0; i < 6; i++ {
		mid := "m" + string(rune('1'+i))
		row := mkPMRowContrib("main", base.Add(time.Duration(i)*time.Hour), 600, 0, 0, 0, 0, 0, 0, 0)
		row.Self.MaxKillingSpree = nil // native absente → doit être calculée
		row.Summary.MatchID = mid
		playerRows = append(playerRows, row)
		// 3 kills d'affilée puis death → spree max = 3 sur ce match.
		events = append(events,
			canonical.HighlightEvent{MatchID: mid, EventType: string(canonical.EventKill), TimeMS: 1000, XUID: xuid},
			canonical.HighlightEvent{MatchID: mid, EventType: string(canonical.EventKill), TimeMS: 2000, XUID: xuid},
			canonical.HighlightEvent{MatchID: mid, EventType: string(canonical.EventKill), TimeMS: 3000, XUID: xuid},
			canonical.HighlightEvent{MatchID: mid, EventType: string(canonical.EventDeath), TimeMS: 4000, XUID: xuid},
		)
	}
	rows := map[string][]canonical.PlayerMatchRow{"main": playerRows}
	squadXUIDs := map[string]string{"main": xuid}

	out := BuildKillingSpreeMax(rows, events, squadXUIDs)
	if len(out) != 1 {
		t.Fatalf("want 1 series (spree calculée depuis events), got %d", len(out))
	}
	if len(out[0].Datapoints) == 0 {
		t.Fatalf("want >0 datapoints")
	}
	// Toutes les valeurs valent 3 → la moyenne lissée vaut 3.
	for _, dp := range out[0].Datapoints {
		if dp.Y != 3 {
			t.Errorf("spree calculée want 3, got %f", dp.Y)
		}
	}
}

// TestBuildKillingSpreeMax_NativeWinsNoRecompute — la valeur native (Infinite) fait foi
// même quand des events sont présents : pas de recalcul (NO-OP Infinite garanti).
func TestBuildKillingSpreeMax_NativeWinsNoRecompute(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	const xuid = "x_main"

	var playerRows []canonical.PlayerMatchRow
	var events []canonical.HighlightEvent
	for i := 0; i < 6; i++ {
		mid := "m" + string(rune('1'+i))
		// native = 7, mais les events suggèrent 3 → on doit retenir 7 (native fait foi).
		row := mkPMRowContrib("main", base.Add(time.Duration(i)*time.Hour), 600, 0, 0, 0, 0, 0, 7, 0)
		row.Summary.MatchID = mid
		playerRows = append(playerRows, row)
		events = append(events,
			canonical.HighlightEvent{MatchID: mid, EventType: string(canonical.EventKill), TimeMS: 1000, XUID: xuid},
			canonical.HighlightEvent{MatchID: mid, EventType: string(canonical.EventKill), TimeMS: 2000, XUID: xuid},
			canonical.HighlightEvent{MatchID: mid, EventType: string(canonical.EventKill), TimeMS: 3000, XUID: xuid},
		)
	}
	rows := map[string][]canonical.PlayerMatchRow{"main": playerRows}
	squadXUIDs := map[string]string{"main": xuid}

	out := BuildKillingSpreeMax(rows, events, squadXUIDs)
	if len(out) != 1 || len(out[0].Datapoints) == 0 {
		t.Fatalf("want 1 series avec datapoints")
	}
	for _, dp := range out[0].Datapoints {
		if dp.Y != 7 {
			t.Errorf("native doit faire foi (7), got %f (recalcul indu ?)", dp.Y)
		}
	}
}

func TestBuildHsPkStacked(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	rows := map[string][]canonical.PlayerMatchRow{
		"main": {mkPMRowContrib("main", t0, 600, 0, 0, 0, 7, 3, 0, 0)},
		"f1":   {mkPMRowContrib("f1", t0, 600, 0, 0, 0, 4, 2, 0, 0)},
	}
	chart := BuildHsPkStacked(rows)
	if len(chart.Datapoints) != 2 {
		t.Fatalf("want 2 datapoints, got %d", len(chart.Datapoints))
	}
	// Tri alpha : f1 first
	if chart.Datapoints[0].Category != "f1" {
		t.Errorf("first category want f1, got %s", chart.Datapoints[0].Category)
	}
	if chart.Datapoints[0].Components["headshots"] != 4 || chart.Datapoints[0].Components["power_weapons"] != 2 {
		t.Errorf("f1 components want hs=4 pk=2, got %v", chart.Datapoints[0].Components)
	}
}

func TestBuildContributions_EmptyInputs(t *testing.T) {
	t.Parallel()
	if got := BuildPerMinuteStats(nil); len(got.Datapoints) != 0 {
		t.Errorf("PerMinuteStats nil: want empty, got %d datapoints", len(got.Datapoints))
	}
	if got := BuildFragsDeathsCombined(nil); len(got.Datapoints) != 0 {
		t.Errorf("FragsDeathsCombined nil: want empty, got %d datapoints", len(got.Datapoints))
	}
	if got := BuildKillingSpreeMax(nil, nil, nil); got != nil {
		t.Errorf("KillingSpreeMax nil: want nil, got %v", got)
	}
	if got := BuildHsPkStacked(nil); len(got.Datapoints) != 0 {
		t.Errorf("HsPkStacked nil: want empty, got %d datapoints", len(got.Datapoints))
	}
}
