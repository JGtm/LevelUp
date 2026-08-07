//go:build integration

package sync

// citations_pve_test.go — couverture des bugs I7 sur le pipeline citations :
//   - BUG A : les stats PvE Firefight (shared_pve.pve_match_stats_latest) sont
//     désormais chargées et injectées dans CitationContext.Stats (clés pve_stat :
//     grunt_kills, elite_kills, boss_kills, total_enemy_kills, ...).
//   - BUG B : grenade_kills (match_participants) est désormais lu par loadMatchStats.
//   - Dégradation gracieuse : pveDB nil / match inconnu → aucune stat PvE, pas d'erreur.

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	pveTestMatchID = "match-pve-001"
	pveTestXUID    = "xuid-pve-player"
)

// buildPveTestDB crée une shared_pve in-memory (pve_match_stats append-only + vue
// pve_match_stats_latest, réplique de la migration shared_pve_append_only_v1).
func buildPveTestDB(t *testing.T) *sql.DB {
	t.Helper()
	return openFixtureDB(t, `
CREATE SEQUENCE pve_id_seq;
-- Fixture ALIGNÉE sur le schéma RÉEL de shared_pve.pve_match_stats (vérifié sur
-- pièces 2026-07-24 : la colonne est total_enemy_kills, PAS total_kills — la
-- première version de cette fixture divergeait et masquait un Binder Error prod).
CREATE TABLE pve_match_stats (
    id                INTEGER PRIMARY KEY DEFAULT nextval('pve_id_seq'),
    match_id          VARCHAR NOT NULL,
    xuid              VARCHAR NOT NULL,
    total_enemy_kills INTEGER DEFAULT 0,
    boss_kills        INTEGER DEFAULT 0,
    grunt_kills       INTEGER DEFAULT 0,
    elite_kills       INTEGER DEFAULT 0,
    jackal_kills      INTEGER DEFAULT 0,
    brute_kills       INTEGER DEFAULT 0,
    hunter_kills      INTEGER DEFAULT 0,
    skimmer_kills     INTEGER DEFAULT 0,
    sentinel_kills    INTEGER DEFAULT 0,
    marine_kills      INTEGER DEFAULT 0,
    pve_bits          BIGINT DEFAULT 0,
    created_at        TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    written_at        TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
);
CREATE OR REPLACE VIEW pve_match_stats_latest AS
    SELECT * FROM pve_match_stats
    QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, xuid ORDER BY written_at DESC, id DESC) = 1;
`)
}

// insertPveRow insère une ligne Firefight de référence pour (pveTestMatchID, pveTestXUID).
func insertPveRow(t *testing.T, db *sql.DB) {
	t.Helper()
	mustExec(t, db, `
INSERT INTO pve_match_stats (
    match_id, xuid, boss_kills,
    grunt_kills, elite_kills, jackal_kills, brute_kills,
    hunter_kills, skimmer_kills, sentinel_kills, marine_kills,
    total_enemy_kills, pve_bits
) VALUES (?, ?, 1, 12, 4, 3, 0, 2, 0, 0, 0, 21, 0)`,
		pveTestMatchID, pveTestXUID)
}

// TestLoadPveStats vérifie le loader isolé : mapping colonnes → clés stat_name
// (total_enemy_kills lue telle quelle) et dégradation gracieuse (nil / no rows).
func TestLoadPveStats(t *testing.T) {
	ctx := context.Background()
	pve := buildPveTestDB(t)
	insertPveRow(t, pve)

	stats, err := loadPveStats(ctx, pve, pveTestMatchID, pveTestXUID)
	if err != nil {
		t.Fatalf("loadPveStats: %v", err)
	}
	want := map[string]float64{
		"grunt_kills":       12,
		"elite_kills":       4,
		"jackal_kills":      3,
		"hunter_kills":      2,
		"boss_kills":        1,
		"total_enemy_kills": 21,
	}
	for k, v := range want {
		if got := stats[k]; got != v {
			t.Errorf("loadPveStats[%q] = %v, want %v", k, got, v)
		}
	}

	// Match inconnu (non-Firefight) → aucune ligne PvE, pas d'erreur.
	empty, err := loadPveStats(ctx, pve, "match-unknown", pveTestXUID)
	if err != nil {
		t.Fatalf("loadPveStats(unknown): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("loadPveStats(unknown) = %v, want vide", empty)
	}

	// pveDB nil (titre sans Firefight / DB absente) → dégradation gracieuse.
	nilStats, err := loadPveStats(ctx, nil, pveTestMatchID, pveTestXUID)
	if err != nil {
		t.Fatalf("loadPveStats(nil): %v", err)
	}
	if nilStats != nil {
		t.Errorf("loadPveStats(nil) = %v, want nil", nilStats)
	}
}

// seedCtxSharedForPve insère un match Firefight + le participant fixXUID avec
// grenade_kills peuplé (BUG B) dans la shared DB de test.
func seedCtxSharedForPve(t *testing.T, shared *sql.DB) {
	t.Helper()
	mustExec(t, shared, `
INSERT INTO match_registry (match_id, start_time, playlist_name, game_variant_name, is_firefight)
VALUES (?, TIMESTAMP '2026-07-01 12:00:00', 'Firefight', 'Firefight: King', TRUE)`,
		pveTestMatchID)
	mustExec(t, shared, `
INSERT INTO match_participants (
    match_id, xuid, gamertag, outcome, kills, deaths, assists,
    headshot_kills, melee_kills, power_weapon_kills, grenade_kills
) VALUES (?, ?, 'PvePlayer', 2, 7, 4, 1, 2, 1, 0, 3)`,
		pveTestMatchID, pveTestXUID)
}

// TestBuildCitationContext_MergesPveAndGrenade vérifie l'intégration :
// buildCitationContext injecte les stats PvE (BUG A) ET grenade_kills (BUG B)
// dans Stats, et que la non-régression des stats match_participants tient.
func TestBuildCitationContext_MergesPveAndGrenade(t *testing.T) {
	ctx := context.Background()
	shared := openFixtureDB(t, buildSharedDDL())
	player := openFixtureDB(t, buildPlayerDDL())
	seedCtxSharedForPve(t, shared)

	pve := buildPveTestDB(t)
	insertPveRow(t, pve)

	cc, err := buildCitationContext(ctx, shared, player, pve, map[uint64]string{}, pveTestXUID, pveTestMatchID)
	if err != nil {
		t.Fatalf("buildCitationContext: %v", err)
	}

	// BUG A — stats PvE présentes.
	pveWant := map[string]float64{
		"grunt_kills":       12,
		"elite_kills":       4,
		"boss_kills":        1,
		"total_enemy_kills": 21,
	}
	for k, v := range pveWant {
		if got := cc.Stats[k]; got != v {
			t.Errorf("Stats[%q] = %v, want %v (BUG A)", k, got, v)
		}
	}

	// BUG B — grenade_kills présent.
	if got := cc.Stats["grenade_kills"]; got != 3 {
		t.Errorf("Stats[grenade_kills] = %v, want 3 (BUG B)", got)
	}

	// Non-régression — stats match_participants historiques préservées.
	for k, v := range map[string]float64{
		"kills": 7, "deaths": 4, "assists": 1,
		"headshot_kills": 2, "melee_kills": 1,
	} {
		if got := cc.Stats[k]; got != v {
			t.Errorf("Stats[%q] = %v, want %v (non-régression)", k, got, v)
		}
	}
	if !cc.IsFirefight {
		t.Errorf("IsFirefight = false, want true")
	}
}

// TestBuildCitationContext_NilPveGraceful vérifie que pveDB nil ne charge aucune
// stat PvE mais n'échoue pas, et que grenade_kills (source match_participants)
// reste renseigné.
func TestBuildCitationContext_NilPveGraceful(t *testing.T) {
	ctx := context.Background()
	shared := openFixtureDB(t, buildSharedDDL())
	player := openFixtureDB(t, buildPlayerDDL())
	seedCtxSharedForPve(t, shared)

	cc, err := buildCitationContext(ctx, shared, player, nil, map[uint64]string{}, pveTestXUID, pveTestMatchID)
	if err != nil {
		t.Fatalf("buildCitationContext(nil pve): %v", err)
	}
	if _, ok := cc.Stats["grunt_kills"]; ok {
		t.Errorf("Stats ne devrait contenir aucune stat PvE avec pveDB nil, got grunt_kills=%v", cc.Stats["grunt_kills"])
	}
	if got := cc.Stats["grenade_kills"]; got != 3 {
		t.Errorf("Stats[grenade_kills] = %v, want 3 (source match_participants, indépendant du PvE)", got)
	}
}
