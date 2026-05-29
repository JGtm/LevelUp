package timeline_test

// Golden test d'intégration de la couche MatchTimeline sur données réelles.
//
// Rejoue la chaîne service → CorrectEvents → narrative sur les events des 8
// matchs de référence (testdata/t0_fixtures/events_golden.json, généré par
// cmd/export_t0_fixtures). Capture FirstEvents + IntensityProfiles par match.
//
// Phase 1 (T0=0) : le golden EST la baseline du comportement historique. Le
// refacto étant une identité, ce golden ne doit pas bouger.
// Phase 3 (T0 réel) : régénérer avec UPDATE_GOLDEN=1 et reviewer le diff — il
// montre exactement comment les chronologies se recalent par match.
//
// Régénérer : UPDATE_GOLDEN=1 go test ./internal/analysis/timeline/ -run TestGolden

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

type fixtureEvent struct {
	MatchID   string `json:"match_id"`
	EventType string `json:"event_type"`
	TimeMS    int64  `json:"time_ms"`
	XUID      string `json:"xuid"`
}

type fixtureMatch struct {
	MatchID         string `json:"match_id"`
	DurationSeconds *int   `json:"duration_seconds"`
	TopKillerXUID   string `json:"top_killer_xuid"`
}

type fixtureFile struct {
	Matches []fixtureMatch `json:"matches"`
	Events  []fixtureEvent `json:"events"`
}

type matchGoldenOutput struct {
	MatchID    string                       `json:"match_id"`
	T0Ms       int64                        `json:"t0_ms"`
	FirstKill  *int64                       `json:"first_kill_ms"`
	FirstDeath *int64                       `json:"first_death_ms"`
	Intensity  []narrative.IntensityProfile `json:"intensity_profiles"`
}

// eventsFromFixture reconstruit les canonical.HighlightEvent en répliquant le
// mapping XUID → Killer/Victim/Player de HighlightEventsRepo.scanHighlightEvent.
func eventsFromFixture(raw []fixtureEvent) []canonical.HighlightEvent {
	out := make([]canonical.HighlightEvent, 0, len(raw))
	for _, e := range raw {
		ev := canonical.HighlightEvent{MatchID: e.MatchID, EventType: e.EventType, TimeMS: e.TimeMS}
		if e.XUID != "" {
			ev.XUID = e.XUID
			x := e.XUID
			switch e.EventType {
			case string(canonical.EventKill), string(canonical.EventFirstKill),
				string(canonical.EventFinisher), string(canonical.EventClutch):
				ev.KillerXUID = &x
			case string(canonical.EventDeath), string(canonical.EventFirstDeath):
				ev.VictimXUID = &x
			case string(canonical.EventMedal), string(canonical.EventAssist):
				ev.PlayerXUID = &x
			}
		}
		out = append(out, ev)
	}
	return out
}

func TestGolden_T0Pipeline_RealMatches(t *testing.T) {
	inPath := filepath.Join("..", "..", "testdata", "t0_fixtures", "events_golden.json")
	rawData, err := os.ReadFile(inPath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fx fixtureFile
	if err := json.Unmarshal(rawData, &fx); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	if len(fx.Matches) == 0 || len(fx.Events) == 0 {
		t.Fatalf("empty fixture: %d matches, %d events", len(fx.Matches), len(fx.Events))
	}

	allEvents := eventsFromFixture(fx.Events)

	outputs := make([]matchGoldenOutput, 0, len(fx.Matches))
	for _, m := range fx.Matches {
		// Events de ce match uniquement.
		var matchEvents []canonical.HighlightEvent
		for _, ev := range allEvents {
			if ev.MatchID == m.MatchID {
				matchEvents = append(matchEvents, ev)
			}
		}

		durMs := int64(0)
		if m.DurationSeconds != nil {
			durMs = int64(*m.DurationSeconds) * 1000
		}
		tl := domain.NewMatchTimeline(durMs, 0) // Phase 1: T0=0
		timelines := map[string]domain.MatchTimeline{m.MatchID: tl}

		corrected := timeline.CorrectEvents(matchEvents, timelines)

		fe := narrative.ComputeFirstEventsPerMatch(corrected, m.TopKillerXUID, []string{m.MatchID})
		intens := narrative.ComputeMatchIntensityProfiles(corrected, 10)

		out := matchGoldenOutput{MatchID: m.MatchID, T0Ms: tl.T0Ms, Intensity: intens}
		if len(fe) == 1 {
			out.FirstKill = fe[0].FirstKillMS
			out.FirstDeath = fe[0].FirstDeathMS
		}
		outputs = append(outputs, out)
	}

	got, err := json.MarshalIndent(outputs, "", "  ")
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}

	goldenPath := filepath.Join("..", "..", "testdata", "t0_fixtures", "golden_output.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("golden updated: %s (%d matches)", goldenPath, len(outputs))
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run with UPDATE_GOLDEN=1 first): %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("golden mismatch — Phase 1 must be identity.\nRun UPDATE_GOLDEN=1 to inspect diff if change is intended.")
	}
}
