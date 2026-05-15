//go:build integration

package sync

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openLUSRDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMPTZ,
			playlist_name VARCHAR,
			pair_name VARCHAR,
			is_ranked BOOLEAN DEFAULT FALSE,
			is_firefight BOOLEAN DEFAULT FALSE,
			duration_seconds INTEGER
		);
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			outcome INTEGER,
			kills INTEGER, deaths INTEGER, assists INTEGER,
			kills_expected DOUBLE, deaths_expected DOUBLE,
			damage_dealt DOUBLE, damage_taken DOUBLE,
			accuracy DOUBLE,
			team_id INTEGER
		);
		CREATE TABLE match_skill_rank (
			match_id         VARCHAR PRIMARY KEY,
			rating_type      VARCHAR NOT NULL,
			rating_value     DOUBLE,
			rating_deviation DOUBLE,
			tier             VARCHAR,
			tier_fr          VARCHAR,
			sub_tier         SMALLINT DEFAULT 0,
			tier_label       VARCHAR,
			rating_delta     DOUBLE,
			playlist_group   VARCHAR,
			start_time       TIMESTAMPTZ,
			created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	if err := execScript(db, ddl); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestLoadLUSRMatchData_Empty(t *testing.T) {
	db := openLUSRDB(t)
	data, err := loadLUSRMatchData(db, "xuid_none")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("expected 0, got %d", len(data))
	}
}

func TestLoadLUSRMatchData_WithData(t *testing.T) {
	db := openLUSRDB(t)
	db.Exec(`INSERT INTO match_registry VALUES
		('m1', '2025-01-01 10:00:00'::TIMESTAMPTZ, 'Ranked Arena', 'Slayer', FALSE, FALSE, 600)`)
	db.Exec(`INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, assists, kills_expected, deaths_expected, damage_dealt, damage_taken, accuracy, team_id) VALUES
		('m1', 'xuid1', 2, 15, 5, 3, 12.0, 6.0, 3000.0, 1500.0, 0.55, 0)`)

	data, err := loadLUSRMatchData(db, "xuid1")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 1 {
		t.Fatalf("expected 1, got %d", len(data))
	}
	if data[0].Kills != 15 {
		t.Fatalf("expected kills=15, got %v", data[0].Kills)
	}
}

func TestLoadLUSRMatchData_FiltersRanked(t *testing.T) {
	db := openLUSRDB(t)
	db.Exec(`INSERT INTO match_registry VALUES
		('m1', '2025-01-01 10:00:00'::TIMESTAMPTZ, 'Ranked', 'Slayer', TRUE, FALSE, 600)`)
	db.Exec(`INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, assists, kills_expected, deaths_expected, damage_dealt, damage_taken, accuracy, team_id) VALUES
		('m1', 'xuid1', 2, 10, 5, 2, 10.0, 5.0, 2000.0, 1000.0, 0.5, 0)`)

	data, err := loadLUSRMatchData(db, "xuid1")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("expected 0 (ranked filtered), got %d", len(data))
	}
}

func TestLoadLUSRParticipants_Empty(t *testing.T) {
	db := openLUSRDB(t)
	result, err := loadLUSRParticipants(db, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestLoadLUSRParticipants_WithData(t *testing.T) {
	db := openLUSRDB(t)
	db.Exec(`INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, assists, kills_expected, deaths_expected, damage_dealt, damage_taken, accuracy, team_id) VALUES
		('m1', 'xuid1', 2, 10, 5, 2, 10.0, 5.0, 2000.0, 1000.0, 0.5, 0),
		('m1', 'xuid2', 3, 8, 7, 1, 9.0, 6.0, 1800.0, 1200.0, 0.4, 1)`)

	result, err := loadLUSRParticipants(db, []string{"m1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result["m1"]) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(result["m1"]))
	}
}

func TestLoadExistingRatingIDs_Empty(t *testing.T) {
	db := openLUSRDB(t)
	result, err := loadExistingRatingIDs(db, "LUSR")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestLoadExistingRatingIDs_WithData(t *testing.T) {
	db := openLUSRDB(t)
	db.Exec(`INSERT INTO match_skill_rank (match_id, rating_type, rating_value, rating_deviation, playlist_group, start_time)
		VALUES ('m1', 'LUSR', 25.0, 8.33, 'social', '2025-01-01'::TIMESTAMPTZ)`)

	result, err := loadExistingRatingIDs(db, "LUSR")
	if err != nil {
		t.Fatal(err)
	}
	if !result["m1"] {
		t.Fatal("expected m1 in result")
	}
}

func TestLoadExistingRatingIDs_ErrorPropagation(t *testing.T) {
	db := openLUSRDB(t)
	if _, err := db.Exec(`DROP TABLE match_skill_rank`); err != nil {
		t.Fatal(err)
	}
	if _, err := loadExistingRatingIDs(db, "CSR"); err == nil {
		t.Fatal("expected error when table missing, got nil")
	}
}

// seedCSR insère un rating CSR de référence pour les tests d'écrasement.
func seedCSR(t *testing.T, db *sql.DB, matchID string, ratingValue float64) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, rating_deviation,
			 tier, tier_fr, sub_tier, tier_label,
			 rating_delta, playlist_group, start_time)
		VALUES (?, 'CSR', ?, 50.0, 'Diamond', 'Diamant', 2, 'Diamond II',
		        NULL, 'ranked', '2025-01-01 10:00:00'::TIMESTAMPTZ)`,
		matchID, ratingValue)
	if err != nil {
		t.Fatalf("seedCSR(%s): %v", matchID, err)
	}
}

func assertCSRPreserved(t *testing.T, db *sql.DB, matchID string, expectedValue float64) {
	t.Helper()
	var typ string
	var val float64
	err := db.QueryRow(
		"SELECT rating_type, rating_value FROM match_skill_rank WHERE match_id = ?",
		matchID,
	).Scan(&typ, &val)
	if err != nil {
		t.Fatalf("read %s: %v", matchID, err)
	}
	if typ != "CSR" {
		t.Fatalf("CSR overwritten by %s on match %s", typ, matchID)
	}
	if val != expectedValue {
		t.Fatalf("CSR rating_value changed: want %.2f, got %.2f", expectedValue, val)
	}
}

// TestUpsertLUSR_DoesNotOverwriteExistingCSR_NormalMode vérifie que la garde Go
// (filtre existingCSR) bloque l'UPSERT LUSR sur un match déjà CSR.
func TestUpsertLUSR_DoesNotOverwriteExistingCSR_NormalMode(t *testing.T) {
	db := openLUSRDB(t)
	seedCSR(t, db, "m1", 1500.0)

	existingCSR := map[string]bool{"m1": true}
	existingLUSR := map[string]bool{}
	results := []lusrResult{{
		MatchID:         "m1",
		RatingValue:     1234.5,
		RatingDeviation: 50.0,
		PlaylistGroup:   "arena_slayer",
	}}

	updated, err := upsertLUSRRatings(db, results, existingCSR, existingLUSR, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated != 0 {
		t.Fatalf("expected 0 updates, got %d", updated)
	}
	assertCSRPreserved(t, db, "m1", 1500.0)
}

// TestUpsertLUSR_SQLGuardWhenGoFilterBypassed simule le cas où la garde Go a sauté
// (map existingCSR vide) : seul le garde-fou SQL doit empêcher l'écrasement, et
// le compteur `updated` doit refléter qu'aucune ligne n'a été modifiée.
func TestUpsertLUSR_SQLGuardWhenGoFilterBypassed(t *testing.T) {
	db := openLUSRDB(t)
	seedCSR(t, db, "m1", 1500.0)

	emptyCSR := map[string]bool{} // garde Go désactivée
	emptyLUSR := map[string]bool{}
	results := []lusrResult{{
		MatchID:         "m1",
		RatingValue:     999.9,
		RatingDeviation: 80.0,
		PlaylistGroup:   "arena_slayer",
	}}

	updated, err := upsertLUSRRatings(db, results, emptyCSR, emptyLUSR, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated != 0 {
		t.Fatalf("updated=%d : le SQL guard a bloqué l'UPDATE, le compteur doit rester à 0", updated)
	}
	assertCSRPreserved(t, db, "m1", 1500.0)
}

// TestUpsertLUSR_CounterReflectsOnlyRealWrites valide le compteur sur un batch
// hétérogène : CSR (filtré par Go), LUSR existant (filtré par Go), CSR avec
// garde Go bypassée (bloqué par SQL guard), et 2 nouveaux matchs valides.
// Seuls les 2 nouveaux doivent compter.
func TestUpsertLUSR_CounterReflectsOnlyRealWrites(t *testing.T) {
	db := openLUSRDB(t)
	seedCSR(t, db, "m_csr_filtered", 1500.0)
	seedCSR(t, db, "m_csr_sql_only", 1600.0)
	if _, err := db.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, playlist_group)
		VALUES ('m_lusr_existing', 'LUSR', 25.0, 'arena_slayer')`); err != nil {
		t.Fatal(err)
	}

	// Garde Go : connaît m_csr_filtered et m_lusr_existing, MAIS PAS m_csr_sql_only
	// (simule un loadExistingRatingIDs incomplet ou un refactor buggué).
	existingCSR := map[string]bool{"m_csr_filtered": true}
	existingLUSR := map[string]bool{"m_lusr_existing": true}

	results := []lusrResult{
		{MatchID: "m_csr_filtered", RatingValue: 100.0, PlaylistGroup: "arena_slayer"},
		{MatchID: "m_lusr_existing", RatingValue: 200.0, PlaylistGroup: "arena_slayer"},
		{MatchID: "m_csr_sql_only", RatingValue: 300.0, PlaylistGroup: "arena_slayer"},
		{MatchID: "m_new_1", RatingValue: 400.0, PlaylistGroup: "arena_slayer"},
		{MatchID: "m_new_2", RatingValue: 500.0, PlaylistGroup: "arena_slayer"},
	}

	updated, err := upsertLUSRRatings(db, results, existingCSR, existingLUSR, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated != 2 {
		t.Fatalf("updated=%d, attendu 2 (seulement m_new_1 et m_new_2)", updated)
	}
	// Les deux CSR doivent être intacts.
	assertCSRPreserved(t, db, "m_csr_filtered", 1500.0)
	assertCSRPreserved(t, db, "m_csr_sql_only", 1600.0)
	// Le LUSR existant n'a pas été touché (skip par Go).
	var lusrVal float64
	if err := db.QueryRow(
		"SELECT rating_value FROM match_skill_rank WHERE match_id = 'm_lusr_existing'",
	).Scan(&lusrVal); err != nil {
		t.Fatal(err)
	}
	if lusrVal != 25.0 {
		t.Fatalf("m_lusr_existing rating_value=%.2f, attendu 25.0 (skip par filtre Go)", lusrVal)
	}
	// Les deux nouveaux sont bien insérés en LUSR.
	for _, mid := range []string{"m_new_1", "m_new_2"} {
		var typ string
		if err := db.QueryRow(
			"SELECT rating_type FROM match_skill_rank WHERE match_id = ?", mid,
		).Scan(&typ); err != nil {
			t.Fatalf("read %s: %v", mid, err)
		}
		if typ != "LUSR" {
			t.Fatalf("%s rating_type=%s, attendu LUSR", mid, typ)
		}
	}
}

// TestBatchComputeLUSR_ForceMode_PreservesCSR exerce le pipeline complet en
// mode force (chemin le plus exposé) : un CSR pré-existant doit survivre.
func TestBatchComputeLUSR_ForceMode_PreservesCSR(t *testing.T) {
	db := openLUSRDB(t)
	if _, err := db.Exec(`INSERT INTO match_registry VALUES
		('m1', '2025-01-01 10:00:00'::TIMESTAMPTZ, 'Arena Slayer', 'Slayer', FALSE, FALSE, 600)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_participants
		(match_id, xuid, outcome, kills, deaths, assists, kills_expected, deaths_expected,
		 damage_dealt, damage_taken, accuracy, team_id)
		VALUES ('m1', 'xuid1', 2, 10, 5, 2, 10.0, 5.0, 2000.0, 1000.0, 0.5, 0)`); err != nil {
		t.Fatal(err)
	}
	seedCSR(t, db, "m1", 1500.0)

	if _, err := batchComputeLUSR(db, db, "xuid1", nil, true); err != nil {
		t.Fatal(err)
	}
	assertCSRPreserved(t, db, "m1", 1500.0)
}
