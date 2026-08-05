//go:build integration

package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// applyNamedMig trouve une migration par nom dans le registre global et applique
// son ApplySchema sur db. Échoue si introuvable.
func applyNamedMig(t *testing.T, db *sql.DB, name string) {
	t.Helper()
	for _, m := range All() {
		if m.Name == name {
			if err := m.ApplySchema(db); err != nil {
				t.Fatalf("ApplySchema %s: %v", name, err)
			}
			return
		}
	}
	t.Fatalf("migration %q introuvable dans le registre", name)
}

// TestPlayerSkillStateV2ResetMarker_AppendOnlyReset vérifie le mécanisme de reset
// watermark append-only (#23046 Phase 2) sur une table LEGACY (sans is_reset) :
//  1. la migration ALTER ajoute is_reset + recrée la vue filtrée ;
//  2. une row sentinelle is_reset=TRUE masque le groupe dans _latest (LoadState→nil) ;
//  3. un état frais (written_at postérieur) réapparaît.
func TestPlayerSkillStateV2ResetMarker_AppendOnlyReset(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	// Table LEGACY (pré-migration) : pas de colonne is_reset, ancienne vue SELECT s.*.
	legacy := `
		CREATE SEQUENCE player_skill_state_v2_seq START 1;
		CREATE TABLE player_skill_state_v2 (
			id BIGINT DEFAULT nextval('player_skill_state_v2_seq') PRIMARY KEY,
			xuid VARCHAR NOT NULL, playlist_group VARCHAR NOT NULL,
			mu DOUBLE NOT NULL, sigma DOUBLE NOT NULL,
			experience INTEGER NOT NULL DEFAULT 0,
			last_match_id VARCHAR, last_match_at TIMESTAMP,
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP));
		CREATE VIEW player_skill_state_v2_latest AS
			SELECT s.* FROM player_skill_state_v2 s
			JOIN (SELECT xuid, playlist_group, MAX(written_at) AS mw
			      FROM player_skill_state_v2 GROUP BY xuid, playlist_group) m
			  ON s.xuid=m.xuid AND s.playlist_group=m.playlist_group AND s.written_at=m.mw;`
	if err := execScript(db, legacy); err != nil {
		t.Fatalf("legacy schema: %v", err)
	}
	// État initial.
	if _, err := db.Exec(`INSERT INTO player_skill_state_v2
		(xuid, playlist_group, mu, sigma, written_at)
		VALUES ('x1', 'ranked', 25.0, 8.0, TIMESTAMP '2026-01-01 00:00:00')`); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	// Applique la migration reset_marker (ALTER + vue filtrée).
	applyNamedMig(t, db, "player_skill_state_v2_reset_marker_v1")

	latestCount := func() int {
		var n int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM player_skill_state_v2_latest WHERE xuid='x1' AND playlist_group='ranked'`).Scan(&n); err != nil {
			t.Fatalf("count latest: %v", err)
		}
		return n
	}

	// Post-migration : l'état initial reste visible (is_reset=FALSE par défaut).
	if got := latestCount(); got != 1 {
		t.Fatalf("post-migration: _latest = %d, want 1 (état initial visible)", got)
	}

	// Reset : sentinelle is_reset=TRUE, written_at postérieur → groupe masqué.
	if _, err := db.Exec(`INSERT INTO player_skill_state_v2
		(xuid, playlist_group, mu, sigma, experience, written_at, is_reset)
		SELECT xuid, playlist_group, 0, 0, 0, TIMESTAMP '2026-02-01 00:00:00', TRUE
		FROM player_skill_state_v2_latest WHERE xuid='x1'`); err != nil {
		t.Fatalf("sentinel insert: %v", err)
	}
	if got := latestCount(); got != 0 {
		t.Fatalf("post-reset: _latest = %d, want 0 (sentinelle masque le groupe → LoadState nil)", got)
	}

	// Re-seed : état frais (written_at encore postérieur) → réapparaît.
	if _, err := db.Exec(`INSERT INTO player_skill_state_v2
		(xuid, playlist_group, mu, sigma, written_at)
		VALUES ('x1', 'ranked', 30.0, 6.0, TIMESTAMP '2026-03-01 00:00:00')`); err != nil {
		t.Fatalf("reseed: %v", err)
	}
	if got := latestCount(); got != 1 {
		t.Fatalf("post-reseed: _latest = %d, want 1 (état frais visible)", got)
	}
	var mu float64
	if err := db.QueryRow(
		`SELECT mu FROM player_skill_state_v2_latest WHERE xuid='x1'`).Scan(&mu); err != nil {
		t.Fatalf("read mu: %v", err)
	}
	if mu != 30.0 {
		t.Errorf("mu = %v, want 30.0 (état re-seedé, pas l'ancien 25.0 ni la sentinelle 0)", mu)
	}
}
