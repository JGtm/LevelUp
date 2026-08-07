//go:build integration

// Package sync — csr_writes_integration_test.go : tests d'intégration
// d'UpsertCSRRow contre une DuckDB en mémoire qui simule le schéma post-migration
// `add_msr_measurement_matches_remaining`.
package sync

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openCSRDB ouvre une DB DuckDB en mémoire avec match_skill_rank dans la
// structure post-Phase-2.B (append-only) + la vue match_skill_rank_latest
// avec priorité CSR > LUSR (post-Phase-2.E).
func openCSRDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
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
		CREATE INDEX idx_msr_match_lookup ON match_skill_rank(match_id, rating_type, written_at);
		CREATE OR REPLACE VIEW match_skill_rank_latest AS
			SELECT * FROM match_skill_rank
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY match_id
				ORDER BY
					CASE rating_type WHEN 'CSR' THEN 0 ELSE 1 END,
					written_at DESC,
					id DESC
			) = 1;
	`
	if err := execScript(t.Context(), db, ddl); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestUpsertCSRRow_InsertNew(t *testing.T) {
	db := openCSRDB(t)
	val, delta := 1259.0, 12.0
	row := &MatchCSRRow{
		MatchID:       "m_new_csr",
		RatingValue:   &val,
		Tier:          "Gold",
		TierFR:        "Or",
		SubTier:       5,
		TierLabel:     "Or 5",
		RatingDelta:   &delta,
		PlaylistGroup: "ranked",
		StartTime:     time.Date(2026, 5, 15, 14, 0, 0, 0, time.UTC),
	}
	if err := UpsertCSRRow(t.Context(), db, row); err != nil {
		t.Fatalf("UpsertCSRRow: %v", err)
	}

	var (
		ratingType  string
		ratingValue sql.NullFloat64
		tier        sql.NullString
		tierFR      sql.NullString
		subTier     sql.NullInt64
		tierLabel   sql.NullString
		ratingDelta sql.NullFloat64
		measurement sql.NullInt64
	)
	err := db.QueryRow(`SELECT rating_type, rating_value, tier, tier_fr, sub_tier, tier_label, rating_delta, measurement_matches_remaining FROM match_skill_rank_latest WHERE match_id = ?`, "m_new_csr").
		Scan(&ratingType, &ratingValue, &tier, &tierFR, &subTier, &tierLabel, &ratingDelta, &measurement)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if ratingType != "CSR" {
		t.Errorf("rating_type: want CSR, got %q", ratingType)
	}
	if !ratingValue.Valid || ratingValue.Float64 != 1259 {
		t.Errorf("rating_value: want 1259, got %v", ratingValue)
	}
	if !tier.Valid || tier.String != "Gold" {
		t.Errorf("tier: want Gold, got %v", tier)
	}
	if !tierFR.Valid || tierFR.String != "Or" {
		t.Errorf("tier_fr: want Or, got %v", tierFR)
	}
	if !subTier.Valid || subTier.Int64 != 5 {
		t.Errorf("sub_tier: want 5, got %v", subTier)
	}
	if !tierLabel.Valid || tierLabel.String != "Or 5" {
		t.Errorf("tier_label: want %q, got %v", "Or 5", tierLabel)
	}
	if !ratingDelta.Valid || ratingDelta.Float64 != 12 {
		t.Errorf("rating_delta: want 12, got %v", ratingDelta)
	}
	if measurement.Valid && measurement.Int64 != 0 {
		t.Errorf("measurement_matches_remaining: want 0, got %v", measurement)
	}
}

// TestUpsertCSRRow_PlacementInsertWithZeroRating cadenasse le fix
// 2026-05-20 : un match en placement a rating_value=0.0 (au lieu de NULL)
// pour ne pas violer la contrainte NOT NULL du schéma prod.
//
// Régression catchée : si le code revient à `row.RatingValue = nil` pour le
// placement, ce test failed avec "Constraint Error: NOT NULL constraint
// failed: match_skill_rank.rating_value" (observé en prod sur JGtm avec 7
// matchs placement perdus avant le fix).
//
// Note : la fixture openCSRDB n'a PAS NOT NULL sur rating_value, donc le
// driver DuckDB stocke 0.0 normalement. En prod le schéma applique NOT NULL
// implicitement. On vérifie ici que le row.RatingValue=&0.0 est BIEN passé
// (pas nil) à UpsertCSRRow, et que la row peut être lue + distinguée comme
// placement via measurement_matches_remaining > 0.
func TestUpsertCSRRow_PlacementInsertWithZeroRating(t *testing.T) {
	db := openCSRDB(t)

	// Reproduit ce que ExtractCSRRowIfRanked génère pour un placement :
	// rating_value=0.0 (post-fix), measurement_matches_remaining > 0.
	zero := 0.0
	row := &MatchCSRRow{
		MatchID:                     "m_placement_csr",
		RatingValue:                 &zero, // <-- 0.0, PAS nil
		Tier:                        "Placement",
		TierFR:                      "Placement",
		SubTier:                     0,
		TierLabel:                   "Placement (3 restants)",
		PlaylistGroup:               "ranked",
		StartTime:                   time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
		MeasurementMatchesRemaining: 3,
	}
	if err := UpsertCSRRow(t.Context(), db, row); err != nil {
		t.Fatalf("UpsertCSRRow placement: %v (régression : retour de rating_value=nil ?)", err)
	}

	var (
		ratingType  string
		ratingValue sql.NullFloat64
		tier        sql.NullString
		measurement sql.NullInt64
		tierLabel   sql.NullString
	)
	err := db.QueryRow(`SELECT rating_type, rating_value, tier, measurement_matches_remaining, tier_label FROM match_skill_rank_latest WHERE match_id = ?`, "m_placement_csr").
		Scan(&ratingType, &ratingValue, &tier, &measurement, &tierLabel)
	if err != nil {
		t.Fatalf("select placement row: %v", err)
	}
	if ratingType != "CSR" {
		t.Errorf("rating_type: want CSR, got %q", ratingType)
	}
	if !ratingValue.Valid || ratingValue.Float64 != 0.0 {
		t.Errorf("rating_value: want 0.0 (placement, NOT NULL safe), got valid=%v val=%v", ratingValue.Valid, ratingValue.Float64)
	}
	if !tier.Valid || tier.String != "Placement" {
		t.Errorf("tier: want Placement, got %v", tier)
	}
	if !measurement.Valid || measurement.Int64 != 3 {
		t.Errorf("measurement_matches_remaining: want 3, got %v", measurement)
	}
	if !tierLabel.Valid || tierLabel.String != "Placement (3 restants)" {
		t.Errorf("tier_label: want 'Placement (3 restants)', got %v", tierLabel)
	}
}

func TestUpsertCSRRow_ReplacesLUSR(t *testing.T) {
	// Scénario : un match classé avait reçu une row LUSR par erreur (bug
	// pré-existant, désormais corrigé). Lorsqu'on sync à nouveau et que le
	// payload skill ramène le RankRecap, UpsertCSRRow doit remplacer la
	// ligne LUSR par une ligne CSR sans erreur (l'UPSERT le fait via
	// ON CONFLICT (match_id) DO UPDATE).
	db := openCSRDB(t)
	_, err := db.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, tier, tier_fr, sub_tier, tier_label, playlist_group, start_time)
		VALUES (?, 'LUSR', ?, ?, ?, ?, ?, ?, ?)`,
		"m_was_lusr", 1500.0, "Gold", "Or", 3, "Gold 3", "open", time.Now().UTC())
	if err != nil {
		t.Fatalf("seed LUSR: %v", err)
	}

	val, delta := 1259.0, -25.0
	row := &MatchCSRRow{
		MatchID:       "m_was_lusr",
		RatingValue:   &val,
		Tier:          "Gold",
		TierFR:        "Or",
		SubTier:       5,
		TierLabel:     "Or 5",
		RatingDelta:   &delta,
		PlaylistGroup: "ranked",
		StartTime:     time.Date(2026, 5, 15, 14, 0, 0, 0, time.UTC),
	}
	if err := UpsertCSRRow(t.Context(), db, row); err != nil {
		t.Fatalf("UpsertCSRRow: %v", err)
	}

	var ratingType string
	var ratingValue sql.NullFloat64
	// Sémantique append-only : LUSR et CSR coexistent physiquement, mais
	// la vue match_skill_rank_latest priorise CSR (Phase 2.E).
	err = db.QueryRow(`SELECT rating_type, rating_value FROM match_skill_rank_latest WHERE match_id = ?`, "m_was_lusr").
		Scan(&ratingType, &ratingValue)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if ratingType != "CSR" {
		t.Errorf("rating_type after upsert: want CSR, got %q (LUSR should be replaced)", ratingType)
	}
	if !ratingValue.Valid || ratingValue.Float64 != 1259 {
		t.Errorf("rating_value after upsert: want 1259 (CSR value), got %v", ratingValue)
	}
}

func TestUpsertCSRRow_PlacementWithNullValue(t *testing.T) {
	db := openCSRDB(t)
	row := &MatchCSRRow{
		MatchID:                     "m_placement",
		RatingValue:                 nil, // NULL
		Tier:                        "Placement",
		TierFR:                      "Placement",
		SubTier:                     0,
		TierLabel:                   "Placement (3 restants)",
		RatingDelta:                 nil, // NULL
		PlaylistGroup:               "ranked",
		StartTime:                   time.Date(2026, 5, 15, 14, 0, 0, 0, time.UTC),
		MeasurementMatchesRemaining: 3,
	}
	if err := UpsertCSRRow(t.Context(), db, row); err != nil {
		t.Fatalf("UpsertCSRRow: %v", err)
	}

	var ratingValue sql.NullFloat64
	var tierLabel sql.NullString
	var measurement sql.NullInt64
	err := db.QueryRow(`SELECT rating_value, tier_label, measurement_matches_remaining FROM match_skill_rank_latest WHERE match_id = ?`, "m_placement").
		Scan(&ratingValue, &tierLabel, &measurement)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if ratingValue.Valid {
		t.Errorf("rating_value should be NULL for placement, got %v", ratingValue.Float64)
	}
	if !tierLabel.Valid || tierLabel.String != "Placement (3 restants)" {
		t.Errorf("tier_label: got %v", tierLabel)
	}
	if !measurement.Valid || measurement.Int64 != 3 {
		t.Errorf("measurement_matches_remaining: want 3, got %v", measurement)
	}
}

func TestUpsertCSRRow_Idempotent(t *testing.T) {
	// Appel répété → même résultat (pas de doublon, pas d'erreur).
	db := openCSRDB(t)
	val := 1500.0
	row := &MatchCSRRow{
		MatchID: "m_idem", RatingValue: &val,
		Tier: "Platinum", TierFR: "Platine", SubTier: 1, TierLabel: "Platine 1",
		PlaylistGroup: "ranked", StartTime: time.Now().UTC(),
	}
	if err := UpsertCSRRow(t.Context(), db, row); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if err := UpsertCSRRow(t.Context(), db, row); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	// Append-only : 2 rows physiques (chaque appel INSERT), 1 row côté
	// vue latest (la version la plus récente). L'idempotence fonctionnelle
	// est portée par la vue, pas par la table physique.
	var physicalCount, latestCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE match_id = ?`, "m_idem").Scan(&physicalCount); err != nil {
		t.Fatalf("count physical: %v", err)
	}
	if physicalCount != 2 {
		t.Errorf("table physique : want 2 rows après 2 upsert (append-only), got %d", physicalCount)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank_latest WHERE match_id = ?`, "m_idem").Scan(&latestCount); err != nil {
		t.Fatalf("count latest: %v", err)
	}
	if latestCount != 1 {
		t.Errorf("vue latest : want exactly 1 row, got %d", latestCount)
	}
}

func TestUpsertCSRRow_NilRow_NoOp(t *testing.T) {
	db := openCSRDB(t)
	if err := UpsertCSRRow(t.Context(), db, nil); err != nil {
		t.Fatalf("UpsertCSRRow(nil) should be no-op, got %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected empty table, got %d rows", count)
	}
}
