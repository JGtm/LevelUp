//go:build integration

package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis"
)

// TestMatchEventsSource_LoadAndTimeline valide la source DuckDB de la timeline
// canonique d'events (halo_infinite.EventsSource) bout-à-bout : lecture des
// highlight_events + calcul T0 (countdown) depuis match_registry, sur un match
// réaliste (paire kill/death + médaille + mode).
func TestMatchEventsSource_LoadAndTimeline(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	resetHighlightEventsTable(t, pdb)
	resetPlayerMatchesTables(t, pdb)

	// Match avec countdown de 5 s (real_start_time = start_time_utc + 5 s) et
	// durée totale 600 s.
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_registry
		   (match_id, start_time, start_time_utc, real_start_time, duration_seconds)
		 VALUES (?, ?, ?, ?, ?)`,
		"m_ev", "2026-04-26T10:00:00Z", "2026-04-26T10:00:00Z", "2026-04-26T10:00:05Z", 600)

	events := []struct {
		eventType, xuid string
		timeMS          int64
	}{
		{"kill", "killerA", 8000},
		{"death", "victimA", 8003},
		{"medal", "killerA", 8050},
		{"mode", "playerB", 9000},
	}
	for _, e := range events {
		execOnSharedDBs(t, pdb, ctx,
			`INSERT INTO shared.highlight_events (match_id, event_type, xuid, time_ms) VALUES (?, ?, ?, ?)`,
			"m_ev", e.eventType, e.xuid, e.timeMS)
	}

	src := NewMatchEventsSource(pdb)

	raw, err := src.LoadHighlightEvents(ctx, "m_ev")
	if err != nil {
		t.Fatalf("LoadHighlightEvents: %v", err)
	}
	if len(raw) != 4 {
		t.Fatalf("highlight events = %d, want 4", len(raw))
	}

	tl, err := src.GetMatchTimeline(ctx, "m_ev")
	if err != nil {
		t.Fatalf("GetMatchTimeline: %v", err)
	}
	if tl.T0Ms != 5000 {
		t.Errorf("T0Ms = %d, want 5000 (countdown 5 s)", tl.T0Ms)
	}
	if tl.DurationMs != 600_000 {
		t.Errorf("DurationMs = %d, want 600000", tl.DurationMs)
	}
	// CorrectEventTime sur un kill raw 8000 → 3000 (référentiel gameplay).
	if got := tl.CorrectEventTime(8000); got != 3000 {
		t.Errorf("CorrectEventTime(8000) = %d, want 3000", got)
	}

	// Match absent → timeline zéro-value, pas d'erreur (identité T0=0).
	empty, err := src.GetMatchTimeline(ctx, "nope")
	if err != nil {
		t.Fatalf("GetMatchTimeline(absent): %v", err)
	}
	if empty.T0Ms != 0 || empty.DurationMs != 0 {
		t.Errorf("match absent → timeline zéro-value attendue, got %+v", empty)
	}

	// Sanity : l'appariement killer/victim fonctionne sur ces rows (tolérance 5 ms).
	pairs := analysis.ComputeKillerVictimPairs([]analysis.RawEvent{
		{EventType: raw[0].EventType, XUID: raw[0].XUID, TimeMS: raw[0].TimeMS},
		{EventType: raw[1].EventType, XUID: raw[1].XUID, TimeMS: raw[1].TimeMS},
	}, 5)
	if len(pairs) != 1 || pairs[0].KillerXUID != "killerA" || pairs[0].VictimXUID != "victimA" {
		t.Errorf("appariement inattendu: %+v", pairs)
	}
}
