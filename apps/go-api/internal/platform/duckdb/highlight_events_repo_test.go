//go:build integration

package duckdb

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// resetHighlightEventsTable nettoie la table avant fixtures.
func resetHighlightEventsTable(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	if _, err := pdb.Player.Exec(context.Background(),
		"DELETE FROM shared.highlight_events"); err != nil {
		t.Fatalf("reset highlight_events: %v", err)
	}
}

// seedHighlightEventsFixtures insere un jeu d'events couvrant les types
// principaux + 2 matchs distincts pour tester les filtres.
func seedHighlightEventsFixtures(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	resetHighlightEventsTable(t, pdb)
	resetPlayerMatchesTables(t, pdb)
	ctx := context.Background()

	// 2 matchs avec start_time differents pour le filtre Since.
	for _, m := range []struct {
		matchID, startTime string
	}{
		{"m_recent", "2026-04-26T10:00:00Z"},
		{"m_old", "2026-01-01T10:00:00Z"},
	} {
		if _, err := pdb.Player.Exec(ctx,
			`INSERT INTO shared.match_registry (match_id, start_time) VALUES (?, ?)`,
			m.matchID, m.startTime); err != nil {
			t.Fatalf("insert match_registry %s: %v", m.matchID, err)
		}
	}

	// Events fixtures : kill, death, assist, medal, first_kill, first_death.
	type ev struct {
		matchID, eventType, xuid string
		timeMS                   int64
	}
	events := []ev{
		// m_recent : sequence kills/deaths du joueur cible
		{"m_recent", "first_kill", "p1", 1500},
		{"m_recent", "kill", "p1", 5000},
		{"m_recent", "kill", "p1", 9000},
		{"m_recent", "death", "p1", 7000},
		{"m_recent", "kill", "p2", 3000},
		{"m_recent", "medal", "p1", 5500},
		{"m_recent", "assist", "p2", 5000},
		{"m_recent", "first_death", "p2", 1500},
		// m_old : 1 seul kill
		{"m_old", "kill", "p1", 2000},
	}
	for _, e := range events {
		if _, err := pdb.Player.Exec(ctx,
			`INSERT INTO shared.highlight_events (match_id, event_type, xuid, time_ms) VALUES (?, ?, ?, ?)`,
			e.matchID, e.eventType, e.xuid, e.timeMS); err != nil {
			t.Fatalf("insert event %s/%s: %v", e.matchID, e.eventType, err)
		}
	}
}

func TestHighlightEventsRepo_Load_FilterByMatchIDs(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedHighlightEventsFixtures(t, pdb)
	repo := NewHighlightEventsRepo(pdb)

	events, err := repo.Load(context.Background(), port.HighlightEventFilters{
		MatchIDs: []string{"m_recent"},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(events) != 8 {
		t.Errorf("want 8 events for m_recent, got %d", len(events))
	}
	for _, e := range events {
		if e.MatchID != "m_recent" {
			t.Errorf("unexpected match_id: %s", e.MatchID)
		}
	}
}

func TestHighlightEventsRepo_Load_FilterByPlayerXUID(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedHighlightEventsFixtures(t, pdb)
	repo := NewHighlightEventsRepo(pdb)

	xuid := "p1"
	events, err := repo.Load(context.Background(), port.HighlightEventFilters{
		MatchIDs:   []string{"m_recent", "m_old"},
		PlayerXUID: &xuid,
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, e := range events {
		if e.XUID != "p1" {
			t.Errorf("expected xuid p1, got %s", e.XUID)
		}
	}
	// p1 events : 1 first_kill + 2 kills + 1 death + 1 medal sur m_recent +
	// 1 kill sur m_old = 6 events
	if len(events) != 6 {
		t.Errorf("want 6 events for p1, got %d", len(events))
	}
}

func TestHighlightEventsRepo_Load_FilterByEventTypes(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedHighlightEventsFixtures(t, pdb)
	repo := NewHighlightEventsRepo(pdb)

	events, err := repo.Load(context.Background(), port.HighlightEventFilters{
		MatchIDs:   []string{"m_recent", "m_old"},
		EventTypes: []canonical.HighlightEventType{canonical.EventKill},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, e := range events {
		if e.EventType != string(canonical.EventKill) {
			t.Errorf("expected only kill events, got %s", e.EventType)
		}
	}
	// 2 kills p1 sur m_recent + 1 kill p2 sur m_recent + 1 kill p1 sur m_old = 4
	if len(events) != 4 {
		t.Errorf("want 4 kill events, got %d", len(events))
	}
}

func TestHighlightEventsRepo_Load_FilterBySince(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedHighlightEventsFixtures(t, pdb)
	repo := NewHighlightEventsRepo(pdb)

	xuid := "p1"
	since := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	events, err := repo.Load(context.Background(), port.HighlightEventFilters{
		PlayerXUID: &xuid,
		Since:      &since,
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// m_old (2026-01-01) avant since, exclu. Reste m_recent (2026-04-26).
	for _, e := range events {
		if e.MatchID != "m_recent" {
			t.Errorf("event from %s should be excluded by since=%v", e.MatchID, since)
		}
	}
	if len(events) != 5 {
		t.Errorf("want 5 events (p1 sur m_recent), got %d", len(events))
	}
}

func TestHighlightEventsRepo_Load_KillerXUIDPopulatedForKill(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedHighlightEventsFixtures(t, pdb)
	repo := NewHighlightEventsRepo(pdb)

	events, err := repo.Load(context.Background(), port.HighlightEventFilters{
		MatchIDs:   []string{"m_recent"},
		EventTypes: []canonical.HighlightEventType{canonical.EventKill, canonical.EventFirstKill},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, e := range events {
		if e.KillerXUID == nil {
			t.Errorf("kill event should have KillerXUID, got nil for %s", e.EventType)
		}
		if e.VictimXUID != nil {
			t.Errorf("kill event should NOT have VictimXUID, got %v", *e.VictimXUID)
		}
		// XUID legacy egal a KillerXUID
		if e.XUID != *e.KillerXUID {
			t.Errorf("legacy XUID mismatch with KillerXUID: %s vs %s", e.XUID, *e.KillerXUID)
		}
	}
}

func TestHighlightEventsRepo_Load_VictimXUIDPopulatedForDeath(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedHighlightEventsFixtures(t, pdb)
	repo := NewHighlightEventsRepo(pdb)

	events, err := repo.Load(context.Background(), port.HighlightEventFilters{
		MatchIDs:   []string{"m_recent"},
		EventTypes: []canonical.HighlightEventType{canonical.EventDeath, canonical.EventFirstDeath},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, e := range events {
		if e.VictimXUID == nil {
			t.Errorf("death event should have VictimXUID, got nil for %s", e.EventType)
		}
		if e.KillerXUID != nil {
			t.Errorf("death event should NOT have KillerXUID, got %v", *e.KillerXUID)
		}
	}
}

func TestHighlightEventsRepo_Load_PlayerXUIDPopulatedForMedal(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedHighlightEventsFixtures(t, pdb)
	repo := NewHighlightEventsRepo(pdb)

	events, err := repo.Load(context.Background(), port.HighlightEventFilters{
		MatchIDs:   []string{"m_recent"},
		EventTypes: []canonical.HighlightEventType{canonical.EventMedal, canonical.EventAssist},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, e := range events {
		if e.PlayerXUID == nil {
			t.Errorf("%s event should have PlayerXUID populated", e.EventType)
		}
		if e.KillerXUID != nil || e.VictimXUID != nil {
			t.Errorf("%s event should not have killer/victim", e.EventType)
		}
	}
}

func TestHighlightEventsRepo_Load_LimitApplied(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedHighlightEventsFixtures(t, pdb)
	repo := NewHighlightEventsRepo(pdb)

	events, err := repo.Load(context.Background(), port.HighlightEventFilters{
		MatchIDs: []string{"m_recent"},
		Limit:    3,
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("limit 3, got %d", len(events))
	}
}

func TestHighlightEventsRepo_Load_OrderByTimeMSDesc(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedHighlightEventsFixtures(t, pdb)
	repo := NewHighlightEventsRepo(pdb)

	events, err := repo.Load(context.Background(), port.HighlightEventFilters{
		MatchIDs: []string{"m_recent"},
		OrderBy:  "time_ms DESC",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for i := 1; i < len(events); i++ {
		if events[i-1].TimeMS < events[i].TimeMS {
			t.Errorf("not sorted DESC at i=%d: %d < %d", i, events[i-1].TimeMS, events[i].TimeMS)
		}
	}
}

func TestHighlightEventsRepo_Load_UnknownOrderBy(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedHighlightEventsFixtures(t, pdb)
	repo := NewHighlightEventsRepo(pdb)

	_, err := repo.Load(context.Background(), port.HighlightEventFilters{
		MatchIDs: []string{"m_recent"},
		OrderBy:  "drop_table",
	})
	if err == nil {
		t.Fatal("expected error for unknown OrderBy")
	}
}

func TestHighlightEventsRepo_Load_RejectTooBroadFilters(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedHighlightEventsFixtures(t, pdb)
	repo := NewHighlightEventsRepo(pdb)

	// Aucun MatchIDs et pas de PlayerXUID+Since -> rejet par Validate()
	_, err := repo.Load(context.Background(), port.HighlightEventFilters{})
	if err == nil {
		t.Fatal("expected error for too-broad filters (no MatchIDs, no PlayerXUID+Since)")
	}
}

func TestHighlightEventsRepo_Load_MatchIDPropagated(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedHighlightEventsFixtures(t, pdb)
	repo := NewHighlightEventsRepo(pdb)

	events, err := repo.Load(context.Background(), port.HighlightEventFilters{
		MatchIDs: []string{"m_old"},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(events) != 1 || events[0].MatchID != "m_old" {
		t.Errorf("expected 1 event with match_id=m_old, got %v", events)
	}
}
