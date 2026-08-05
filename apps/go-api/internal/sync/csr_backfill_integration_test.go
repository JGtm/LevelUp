//go:build integration

// Package sync — csr_backfill_integration_test.go : tests d'intégration de
// BackfillCSRFromAPI contre des DuckDB en mémoire + mockHaloClient.
//
// Le mock retourne pour chaque match_id la skill data configurée dans
// `skillBody[matchID][xuid]`. On simule donc le payload skill Halo y compris
// PostMatchCSR / PreMatchCSR.
package sync

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openCSRBackfillDBs ouvre les 2 DBs in-memory nécessaires : sharedDB avec
// match_registry, playerDB avec match_skill_rank (post-migration Phase B).
func openCSRBackfillDBs(t *testing.T) (playerDB, sharedDB *sql.DB) {
	t.Helper()

	pdb, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	pdb.SetMaxOpenConns(1)
	t.Cleanup(func() { pdb.Close() })
	// Schéma append-only (Phase 2.B/E) : id PK + msr_seq + written_at + vue
	// match_skill_rank_latest avec priorité CSR > LUSR.
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
			match_id      VARCHAR PRIMARY KEY,
			start_time    TIMESTAMPTZ,
			is_ranked     BOOLEAN DEFAULT FALSE,
			is_firefight  BOOLEAN DEFAULT FALSE,
			playlist_name VARCHAR,
			pair_name     VARCHAR
		);
	`); err != nil {
		t.Fatal(err)
	}
	return pdb, sdb
}

func seedRegistry(t *testing.T, sdb *sql.DB, matches []struct {
	ID        string
	StartTime time.Time
	IsRanked  bool
}) {
	t.Helper()
	for _, m := range matches {
		if _, err := sdb.Exec(`INSERT INTO match_registry (match_id, start_time, is_ranked) VALUES (?, ?, ?)`,
			m.ID, m.StartTime, m.IsRanked); err != nil {
			t.Fatalf("seed %s: %v", m.ID, err)
		}
	}
}

func makeRankRecapSkill(xuid string, pre, post float64, tier string, subTier, measurementRem int) map[string]*MatchSkillData {
	return map[string]*MatchSkillData{
		xuid: {
			XUID:         xuid,
			PreMatchCSR:  &CSRRankSnapshot{Value: pre, Tier: tier, SubTier: subTier},
			PostMatchCSR: &CSRRankSnapshot{Value: post, Tier: tier, SubTier: subTier, MeasurementMatchesRemaining: measurementRem},
		},
	}
}

func TestBackfillCSRFromAPI_InsertsCSRForRankedMatches(t *testing.T) {
	pdb, sdb := openCSRBackfillDBs(t)
	xuid := "xuid_test"

	seedRegistry(t, sdb, []struct {
		ID        string
		StartTime time.Time
		IsRanked  bool
	}{
		{"m1", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), true},
		{"m2", time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC), true},
		{"m_social", time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC), false}, // ne doit pas être traité
	})

	mock := &mockHaloClient{
		skillBody: map[string]map[string]*MatchSkillData{
			"m1": makeRankRecapSkill(xuid, 1200, 1212, "Gold", 1, 0),
			"m2": makeRankRecapSkill(xuid, 1212, 1225, "Gold", 2, 0),
		},
	}

	res, err := BackfillCSRFromAPI(context.Background(), mock, pdb, sdb, xuid, false)
	if err != nil {
		t.Fatalf("BackfillCSRFromAPI: %v", err)
	}
	if res.RankedMatches != 2 {
		t.Errorf("RankedMatches: want 2 (social exclu), got %d", res.RankedMatches)
	}
	if res.Inserted != 2 {
		t.Errorf("Inserted: want 2, got %d", res.Inserted)
	}
	if res.SkippedNoRankRecap != 0 || res.SkillErrors != 0 {
		t.Errorf("expected no skips/errors, got skipped=%d errors=%d", res.SkippedNoRankRecap, res.SkillErrors)
	}
	// Vérifier que m_social n'a PAS été traité.
	if mock.callsGetSkill.Load() != 2 {
		t.Errorf("GetMatchSkill calls: want 2 (only ranked), got %d", mock.callsGetSkill.Load())
	}

	var count int
	if err := pdb.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type = 'CSR'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("CSR rows in player DB: want 2, got %d", count)
	}
}

func TestBackfillCSRFromAPI_SkipsMatchesWithExistingCSR(t *testing.T) {
	pdb, sdb := openCSRBackfillDBs(t)
	xuid := "xuid_test"

	seedRegistry(t, sdb, []struct {
		ID        string
		StartTime time.Time
		IsRanked  bool
	}{
		{"m_existing", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), true},
		{"m_new", time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC), true},
	})

	// m_existing a déjà sa row CSR (Phase B inline).
	_, err := pdb.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, tier, tier_fr, sub_tier, tier_label, playlist_group, start_time)
		VALUES ('m_existing', 'CSR', 1500.0, 'Platinum', 'Platine', 1, 'Platine 1', 'ranked', ?)`,
		time.Now().UTC())
	if err != nil {
		t.Fatalf("seed existing CSR: %v", err)
	}

	mock := &mockHaloClient{
		skillBody: map[string]map[string]*MatchSkillData{
			"m_new": makeRankRecapSkill(xuid, 1500, 1525, "Platinum", 2, 0),
		},
	}

	res, err := BackfillCSRFromAPI(context.Background(), mock, pdb, sdb, xuid, false)
	if err != nil {
		t.Fatalf("BackfillCSRFromAPI: %v", err)
	}
	if res.AlreadyHadCSR != 1 {
		t.Errorf("AlreadyHadCSR: want 1, got %d", res.AlreadyHadCSR)
	}
	if res.Inserted != 1 {
		t.Errorf("Inserted: want 1, got %d", res.Inserted)
	}
	if mock.callsGetSkill.Load() != 1 {
		t.Errorf("GetMatchSkill calls: want 1 (only m_new), got %d", mock.callsGetSkill.Load())
	}
}

func TestBackfillCSRFromAPI_ForceRefetchesAll(t *testing.T) {
	pdb, sdb := openCSRBackfillDBs(t)
	xuid := "xuid_test"

	seedRegistry(t, sdb, []struct {
		ID        string
		StartTime time.Time
		IsRanked  bool
	}{
		{"m1", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), true},
	})
	// Row CSR pré-existante.
	_, err := pdb.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, tier, playlist_group, start_time)
		VALUES ('m1', 'CSR', 1200.0, 'Gold', 'ranked', ?)`, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// L'API renvoie une valeur différente (simule un drift / nouveau snapshot).
	mock := &mockHaloClient{
		skillBody: map[string]map[string]*MatchSkillData{
			"m1": makeRankRecapSkill(xuid, 1200, 1700, "Diamond", 4, 0),
		},
	}

	res, err := BackfillCSRFromAPI(context.Background(), mock, pdb, sdb, xuid, true /* force */)
	if err != nil {
		t.Fatalf("BackfillCSRFromAPI: %v", err)
	}
	if res.Inserted != 1 {
		t.Errorf("Inserted: want 1 (force re-fetch), got %d", res.Inserted)
	}
	if res.AlreadyHadCSR != 0 {
		t.Errorf("AlreadyHadCSR: want 0 in force mode, got %d", res.AlreadyHadCSR)
	}

	// Vérifier que la nouvelle valeur a été écrite via la vue latest
	// (append-only : 2 rows physiques, la dernière gagne).
	var val float64
	var tier string
	err = pdb.QueryRow(`SELECT rating_value, tier FROM match_skill_rank_latest WHERE match_id = 'm1'`).Scan(&val, &tier)
	if err != nil {
		t.Fatal(err)
	}
	if val != 1700 || tier != "Diamond" {
		t.Errorf("after force: want 1700/Diamond, got %.0f/%q", val, tier)
	}
}

func TestBackfillCSRFromAPI_SkipsNoRankRecap(t *testing.T) {
	pdb, sdb := openCSRBackfillDBs(t)
	xuid := "xuid_test"

	seedRegistry(t, sdb, []struct {
		ID        string
		StartTime time.Time
		IsRanked  bool
	}{
		{"m_no_recap", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), true},
	})

	// L'API répond mais sans RankRecap (skill data sans PostMatchCSR).
	tm := 1500.0
	mock := &mockHaloClient{
		skillBody: map[string]map[string]*MatchSkillData{
			"m_no_recap": {xuid: {XUID: xuid, TeamMMR: &tm}},
		},
	}

	res, err := BackfillCSRFromAPI(context.Background(), mock, pdb, sdb, xuid, false)
	if err != nil {
		t.Fatalf("BackfillCSRFromAPI: %v", err)
	}
	if res.Fetched != 1 {
		t.Errorf("Fetched: want 1, got %d", res.Fetched)
	}
	if res.Inserted != 0 {
		t.Errorf("Inserted: want 0 (no RankRecap), got %d", res.Inserted)
	}
	if res.SkippedNoRankRecap != 1 {
		t.Errorf("SkippedNoRankRecap: want 1, got %d", res.SkippedNoRankRecap)
	}
}

func TestBackfillCSRFromAPI_HandlesPlacement(t *testing.T) {
	pdb, sdb := openCSRBackfillDBs(t)
	xuid := "xuid_test"

	seedRegistry(t, sdb, []struct {
		ID        string
		StartTime time.Time
		IsRanked  bool
	}{
		{"m_placement", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), true},
	})

	mock := &mockHaloClient{
		skillBody: map[string]map[string]*MatchSkillData{
			"m_placement": makeRankRecapSkill(xuid, 0, 0, "", 0, 3),
		},
	}

	res, err := BackfillCSRFromAPI(context.Background(), mock, pdb, sdb, xuid, false)
	if err != nil {
		t.Fatalf("BackfillCSRFromAPI: %v", err)
	}
	if res.Inserted != 1 {
		t.Errorf("Inserted: want 1 (placement counts), got %d", res.Inserted)
	}

	var ratingValue sql.NullFloat64
	var tier, tierLabel string
	var measurement int
	err = pdb.QueryRow(`SELECT rating_value, tier, tier_label, measurement_matches_remaining FROM match_skill_rank WHERE match_id = 'm_placement'`).
		Scan(&ratingValue, &tier, &tierLabel, &measurement)
	if err != nil {
		t.Fatal(err)
	}
	// Note : depuis commit ae0edbd0, rating_value est stocke 0.0 (au lieu de
	// NULL) pour respecter la contrainte NOT NULL sur match_skill_rank.
	// Le caller distingue placement vs rating reel via measurement_matches_remaining > 0.
	if !ratingValue.Valid || ratingValue.Float64 != 0 {
		t.Errorf("rating_value: want 0.0 for placement (NOT NULL contraint), got valid=%v float=%v", ratingValue.Valid, ratingValue.Float64)
	}
	if tier != "Placement" {
		t.Errorf("tier: want Placement, got %q", tier)
	}
	if tierLabel != "Placement (3 restants)" {
		t.Errorf("tier_label: want %q, got %q", "Placement (3 restants)", tierLabel)
	}
	if measurement != 3 {
		t.Errorf("measurement_matches_remaining: want 3, got %d", measurement)
	}
}

func TestBackfillCSRFromAPI_ContinuesOnSkillError(t *testing.T) {
	pdb, sdb := openCSRBackfillDBs(t)
	xuid := "xuid_test"

	seedRegistry(t, sdb, []struct {
		ID        string
		StartTime time.Time
		IsRanked  bool
	}{
		{"m_err", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), true},
		{"m_ok", time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC), true},
	})

	// getSkillErr s'applique à TOUS les appels — donc on simule un échec global.
	// On vérifie surtout que res.SkillErrors compte et qu'aucune erreur n'est
	// propagée (la fonction continue le batch).
	mock := &mockHaloClient{getSkillErr: errors.New("simulated network error")}

	res, err := BackfillCSRFromAPI(context.Background(), mock, pdb, sdb, xuid, false)
	if err != nil {
		t.Fatalf("BackfillCSRFromAPI should not propagate per-match errors, got: %v", err)
	}
	if res.SkillErrors != 2 {
		t.Errorf("SkillErrors: want 2, got %d", res.SkillErrors)
	}
	if res.Inserted != 0 {
		t.Errorf("Inserted: want 0 on all-error, got %d", res.Inserted)
	}
}

func TestBackfillCSRFromAPI_EmptyRegistry(t *testing.T) {
	pdb, sdb := openCSRBackfillDBs(t)
	mock := &mockHaloClient{}

	res, err := BackfillCSRFromAPI(context.Background(), mock, pdb, sdb, "xuid_test", false)
	if err != nil {
		t.Fatalf("BackfillCSRFromAPI: %v", err)
	}
	if res.RankedMatches != 0 || res.Inserted != 0 {
		t.Errorf("empty registry: want all zeros, got %+v", res)
	}
	if mock.callsGetSkill.Load() != 0 {
		t.Errorf("no API calls expected on empty registry, got %d", mock.callsGetSkill.Load())
	}
}

func TestBackfillCSRFromAPI_ContextCancellation(t *testing.T) {
	pdb, sdb := openCSRBackfillDBs(t)
	xuid := "xuid_test"

	seedRegistry(t, sdb, []struct {
		ID        string
		StartTime time.Time
		IsRanked  bool
	}{
		{"m1", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), true},
		{"m2", time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC), true},
	})
	mock := &mockHaloClient{
		skillBody: map[string]map[string]*MatchSkillData{
			"m1": makeRankRecapSkill(xuid, 1200, 1212, "Gold", 1, 0),
			"m2": makeRankRecapSkill(xuid, 1212, 1225, "Gold", 2, 0),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // annulé immédiatement avant l'appel

	_, err := BackfillCSRFromAPI(ctx, mock, pdb, sdb, xuid, false)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
