//go:build integration

// Package sync — csr_backfill_from_shared_integration_test.go : tests
// d'intégration de BackfillCSRFromShared, qui projette le CSR par-match écrit en
// shared.match_csrs (chemin import OpenSpartan) vers player.match_skill_rank
// (lu par l'UI). DuckDB in-memory shared + player.
package sync

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openCSRProjectionDBs ouvre player (match_skill_rank) + shared (match_registry
// + match_csrs + vues latest) in-memory.
func openCSRProjectionDBs(t *testing.T) (playerDB, sharedDB *sql.DB) {
	t.Helper()
	pdb, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	pdb.SetMaxOpenConns(1)
	t.Cleanup(func() { pdb.Close() })
	if err := execScript(t.Context(), pdb, `
		CREATE SEQUENCE msr_seq START 1;
		CREATE TABLE match_skill_rank (
			id                            BIGINT DEFAULT nextval('msr_seq') PRIMARY KEY,
			match_id                      VARCHAR NOT NULL,
			rating_type                   VARCHAR NOT NULL,
			rating_value                  DOUBLE,
			rating_deviation              DOUBLE,
			tier                          VARCHAR,
			tier_fr                       VARCHAR,
			sub_tier                      SMALLINT DEFAULT 0,
			tier_label                    VARCHAR,
			rating_delta                  DOUBLE,
			playlist_group                VARCHAR,
			start_time                    TIMESTAMPTZ,
			measurement_matches_remaining INTEGER DEFAULT 0,
			written_at                    TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			created_at                    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at                    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE OR REPLACE VIEW match_skill_rank_latest AS
			SELECT * FROM match_skill_rank
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY match_id
				ORDER BY CASE rating_type WHEN 'CSR' THEN 0 ELSE 1 END, written_at DESC, id DESC
			) = 1;
	`); err != nil {
		t.Fatal(err)
	}

	sdb, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	sdb.SetMaxOpenConns(1)
	t.Cleanup(func() { sdb.Close() })
	if err := execScript(t.Context(), sdb, `
		CREATE TABLE match_registry (
			match_id   VARCHAR PRIMARY KEY,
			start_time TIMESTAMPTZ,
			is_ranked  BOOLEAN DEFAULT FALSE
		);
		CREATE SEQUENCE mcsrs_seq START 1;
		CREATE TABLE match_csrs (
			id                            BIGINT DEFAULT nextval('mcsrs_seq') PRIMARY KEY,
			match_id                      VARCHAR NOT NULL,
			xuid                          VARCHAR NOT NULL,
			rating_type                   VARCHAR NOT NULL DEFAULT 'CSR',
			rating_value                  DOUBLE,
			tier                          VARCHAR,
			sub_tier                      SMALLINT DEFAULT 0,
			tier_label                    VARCHAR,
			rating_delta                  DOUBLE,
			measurement_matches_remaining INTEGER DEFAULT 0,
			season_id                     VARCHAR,
			written_at                    TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			created_at                    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at                    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE OR REPLACE VIEW match_csrs_latest AS
			SELECT * FROM match_csrs
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, xuid ORDER BY written_at DESC, id DESC) = 1;
	`); err != nil {
		t.Fatal(err)
	}
	return pdb, sdb
}

// seedSharedCSR insère un match ranked + sa ligne CSR shared via le vrai chemin
// d'écriture (ExtractAllSharedCSRRows + UpsertSharedCSRs).
func seedSharedCSR(t *testing.T, sdb *sql.DB, matchID, xuid string, st time.Time, pre, post float64, tier string, subTier, measRem int) {
	t.Helper()
	if _, err := sdb.Exec(`INSERT INTO match_registry (match_id, start_time, is_ranked) VALUES (?, ?, TRUE)`, matchID, st); err != nil {
		t.Fatalf("seed registry %s: %v", matchID, err)
	}
	season := "CsrSeason13-1"
	reg := &MatchRegistryRow{MatchID: matchID, IsRanked: true, StartTime: st, SeasonID: &season}
	rows := ExtractAllSharedCSRRows(reg, makeRankRecapSkill(xuid, pre, post, tier, subTier, measRem))
	if err := UpsertSharedCSRs(t.Context(), sdb, rows); err != nil {
		t.Fatalf("UpsertSharedCSRs %s: %v", matchID, err)
	}
}

func TestBackfillCSRFromShared_ProjectsStableRank(t *testing.T) {
	pdb, sdb := openCSRProjectionDBs(t)
	xuid := "2533274823110022"
	seedSharedCSR(t, sdb, "m1", xuid, time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC), 1200, 1225, "Gold", 2, 0)

	n, err := BackfillCSRFromShared(t.Context(), sdb, pdb, xuid)
	if err != nil {
		t.Fatalf("BackfillCSRFromShared: %v", err)
	}
	if n != 1 {
		t.Fatalf("written: want 1, got %d", n)
	}

	var (
		ratingType string
		ratingVal  sql.NullFloat64
		tier       sql.NullString
		tierFR     sql.NullString
		subTier    sql.NullInt64
		group      sql.NullString
	)
	if err := pdb.QueryRow(`SELECT rating_type, rating_value, tier, tier_fr, sub_tier, playlist_group
		FROM match_skill_rank_latest WHERE match_id = ?`, "m1").
		Scan(&ratingType, &ratingVal, &tier, &tierFR, &subTier, &group); err != nil {
		t.Fatalf("select projected row: %v", err)
	}
	if ratingType != "CSR" {
		t.Errorf("rating_type: want CSR, got %q", ratingType)
	}
	if !ratingVal.Valid || ratingVal.Float64 != 1225 {
		t.Errorf("rating_value: want 1225, got %v", ratingVal)
	}
	if !tier.Valid || tier.String != "Gold" {
		t.Errorf("tier: want Gold, got %v", tier)
	}
	if !tierFR.Valid || tierFR.String != "Or" {
		t.Errorf("tier_fr: want Or (traduit), got %v", tierFR)
	}
	if !subTier.Valid || subTier.Int64 != 2 {
		t.Errorf("sub_tier: want 2, got %v", subTier)
	}
	if !group.Valid || group.String != PerfChainRanked {
		t.Errorf("playlist_group: want %q, got %v", PerfChainRanked, group)
	}
}

func TestBackfillCSRFromShared_Idempotent(t *testing.T) {
	pdb, sdb := openCSRProjectionDBs(t)
	xuid := "2533274823110022"
	seedSharedCSR(t, sdb, "m1", xuid, time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC), 1200, 1225, "Gold", 2, 0)

	if _, err := BackfillCSRFromShared(t.Context(), sdb, pdb, xuid); err != nil {
		t.Fatalf("first run: %v", err)
	}
	n, err := BackfillCSRFromShared(t.Context(), sdb, pdb, xuid)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if n != 0 {
		t.Errorf("second run written: want 0 (match déjà pourvu d'un CSR), got %d", n)
	}
	var physical int
	if err := pdb.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE match_id = ?`, "m1").Scan(&physical); err != nil {
		t.Fatal(err)
	}
	if physical != 1 {
		t.Errorf("physical rows: want 1 (pas de doublon append-only), got %d", physical)
	}
}

// TestBackfillCSRFromShared_PlacementZeroValue : une row shared en placement
// (rating_value NULL) est projetée avec rating_value=0.0 côté player (invariant
// NOT NULL répliqué depuis ExtractCSRRowIfRanked).
func TestBackfillCSRFromShared_PlacementZeroValue(t *testing.T) {
	pdb, sdb := openCSRProjectionDBs(t)
	xuid := "2533274823110022"
	// post=0, tier="", measurementRem=3 → ExtractAllSharedCSRRows produit
	// tier="Placement", rating_value NULL.
	seedSharedCSR(t, sdb, "m_place", xuid, time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC), 0, 0, "", 0, 3)

	if _, err := BackfillCSRFromShared(t.Context(), sdb, pdb, xuid); err != nil {
		t.Fatalf("BackfillCSRFromShared: %v", err)
	}
	var (
		ratingVal sql.NullFloat64
		tier      sql.NullString
		measRem   sql.NullInt64
	)
	if err := pdb.QueryRow(`SELECT rating_value, tier, measurement_matches_remaining
		FROM match_skill_rank_latest WHERE match_id = ?`, "m_place").
		Scan(&ratingVal, &tier, &measRem); err != nil {
		t.Fatalf("select placement row: %v", err)
	}
	if !ratingVal.Valid || ratingVal.Float64 != 0.0 {
		t.Errorf("rating_value: want 0.0 (placement, NOT NULL safe), got valid=%v val=%v", ratingVal.Valid, ratingVal.Float64)
	}
	if !tier.Valid || tier.String != TierLabelPlacement {
		t.Errorf("tier: want %q, got %v", TierLabelPlacement, tier)
	}
	if !measRem.Valid || measRem.Int64 != 3 {
		t.Errorf("measurement_matches_remaining: want 3, got %v", measRem)
	}
}
