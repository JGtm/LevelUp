//go:build integration

package sync

// citations_objective_test.go — plomberie citations objective_stat (v7.2) :
//   - loadObjectiveStats mappe les colonnes match_objective_stats_latest → clés Stats
//     (zone_captures, flag_returns, zone_secures, flag_carriers_killed, ...).
//   - buildCitationContext injecte ces stats dans CitationContext.Stats.
//   - Dégradation gracieuse : sharedDB nil / match non-objectif → aucune stat, pas d'erreur.

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	objTestMatchID = "match-obj-001"
	objTestXUID    = "xuid-obj-player"
)

// insertObjectiveRow insère une ligne objectif de référence (CTF + Zones peuplés)
// pour (objTestMatchID, objTestXUID).
func insertObjectiveRow(t *testing.T, db *sql.DB) {
	t.Helper()
	mustExec(t, db, `
INSERT INTO match_objective_stats (
    match_id, xuid,
    flag_returns, flag_captures, flag_carriers_killed,
    zone_captures, zone_secures,
    time_as_flag_carrier_seconds
) VALUES (?, ?, 5, 2, 3, 7, 4, 42.9)`,
		objTestMatchID, objTestXUID)
}

// TestLoadObjectiveStats vérifie le loader isolé : mapping colonnes → clés stat_name,
// troncature des *_seconds à l'entier côté moteur (valeur float ici) et dégradation
// gracieuse (nil / no rows).
func TestLoadObjectiveStats(t *testing.T) {
	ctx := context.Background()
	shared := openFixtureDB(t, buildSharedDDL())
	insertObjectiveRow(t, shared)

	stats := loadObjectiveStats(ctx, shared, objTestMatchID, objTestXUID)
	want := map[string]float64{
		"flag_returns":                 5,
		"flag_captures":                2,
		"flag_carriers_killed":         3,
		"zone_captures":                7,
		"zone_secures":                 4,
		"time_as_flag_carrier_seconds": 42.9,
	}
	for k, v := range want {
		if got := stats[k]; got != v {
			t.Errorf("loadObjectiveStats[%q] = %v, want %v", k, got, v)
		}
	}
	// Colonne non peuplée → 0 propre (COALESCE), pas d'absence de clé.
	if got, ok := stats["skull_grabs"]; !ok || got != 0 {
		t.Errorf("loadObjectiveStats[skull_grabs] = %v (ok=%v), want 0 présent", got, ok)
	}

	// Match sans ligne objectif (Slayer) → aucune stat, pas d'erreur.
	empty := loadObjectiveStats(ctx, shared, "match-slayer", objTestXUID)
	if len(empty) != 0 {
		t.Errorf("loadObjectiveStats(slayer) = %v, want vide", empty)
	}

	// sharedDB nil → dégradation gracieuse.
	nilStats := loadObjectiveStats(ctx, nil, objTestMatchID, objTestXUID)
	if nilStats != nil {
		t.Errorf("loadObjectiveStats(nil) = %v, want nil", nilStats)
	}
}

// seedCtxSharedForObjective insère un match Zones + le participant avec la ligne objectif.
func seedCtxSharedForObjective(t *testing.T, shared *sql.DB) {
	t.Helper()
	mustExec(t, shared, `
INSERT INTO match_registry (match_id, start_time, playlist_name, game_variant_name, is_firefight)
VALUES (?, TIMESTAMP '2026-07-01 12:00:00', 'Ranked Arena', 'Strongholds', FALSE)`,
		objTestMatchID)
	mustExec(t, shared, `
INSERT INTO match_participants (
    match_id, xuid, gamertag, outcome, kills, deaths, assists
) VALUES (?, ?, 'ObjPlayer', 2, 9, 5, 2)`,
		objTestMatchID, objTestXUID)
	insertObjectiveRow(t, shared)
}

// TestBuildCitationContext_MergesObjective vérifie l'intégration : buildCitationContext
// injecte les stats objectifs dans Stats, sans régression des stats match_participants.
func TestBuildCitationContext_MergesObjective(t *testing.T) {
	ctx := context.Background()
	shared := openFixtureDB(t, buildSharedDDL())
	player := openFixtureDB(t, buildPlayerDDL())
	seedCtxSharedForObjective(t, shared)

	cc, err := buildCitationContext(ctx, shared, player, nil, map[uint64]string{}, citationWeaponSource{}, objTestXUID, objTestMatchID)
	if err != nil {
		t.Fatalf("buildCitationContext: %v", err)
	}

	objWant := map[string]float64{
		"zone_captures":        7,
		"zone_secures":         4,
		"flag_returns":         5,
		"flag_carriers_killed": 3,
	}
	for k, v := range objWant {
		if got := cc.Stats[k]; got != v {
			t.Errorf("Stats[%q] = %v, want %v (objective_stat)", k, got, v)
		}
	}

	// Non-régression — stats match_participants historiques préservées.
	for k, v := range map[string]float64{"kills": 9, "deaths": 5, "assists": 2} {
		if got := cc.Stats[k]; got != v {
			t.Errorf("Stats[%q] = %v, want %v (non-régression)", k, got, v)
		}
	}
}
