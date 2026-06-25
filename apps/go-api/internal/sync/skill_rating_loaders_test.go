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
			start_time_utc TIMESTAMPTZ,
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
		CREATE SEQUENCE msr_seq START 1;
		CREATE TABLE match_skill_rank (
			id               BIGINT DEFAULT nextval('msr_seq') PRIMARY KEY,
			match_id         VARCHAR NOT NULL,
			rating_type      VARCHAR NOT NULL,
			rating_value     DOUBLE,
			rating_deviation DOUBLE,
			tier             VARCHAR,
			tier_fr          VARCHAR,
			sub_tier         SMALLINT DEFAULT 0,
			tier_label       VARCHAR,
			rating_delta     DOUBLE,
			playlist_group   VARCHAR,
			expected_win_prob FLOAT,
			start_time       TIMESTAMPTZ,
			written_at       TIMESTAMP NOT NULL DEFAULT now(),
			created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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

func TestLoadLUSRMatchData_Empty(t *testing.T) {
	db := openLUSRDB(t)
	data, err := loadLUSRMatchData(t.Context(), db, "xuid_none")
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
		('m1', '2025-01-01 10:00:00'::TIMESTAMPTZ, NULL, 'Ranked Arena', 'Slayer', FALSE, FALSE, 600)`)
	db.Exec(`INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, assists, kills_expected, deaths_expected, damage_dealt, damage_taken, accuracy, team_id) VALUES
		('m1', 'xuid1', 2, 15, 5, 3, 12.0, 6.0, 3000.0, 1500.0, 0.55, 0)`)

	data, err := loadLUSRMatchData(t.Context(), db, "xuid1")
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
		('m1', '2025-01-01 10:00:00'::TIMESTAMPTZ, NULL, 'Ranked', 'Slayer', TRUE, FALSE, 600)`)
	db.Exec(`INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, assists, kills_expected, deaths_expected, damage_dealt, damage_taken, accuracy, team_id) VALUES
		('m1', 'xuid1', 2, 10, 5, 2, 10.0, 5.0, 2000.0, 1000.0, 0.5, 0)`)

	data, err := loadLUSRMatchData(t.Context(), db, "xuid1")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("expected 0 (ranked filtered), got %d", len(data))
	}
}

func TestLoadLUSRParticipants_Empty(t *testing.T) {
	db := openLUSRDB(t)
	result, err := loadLUSRParticipants(t.Context(), db, nil)
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

	result, err := loadLUSRParticipants(t.Context(), db, []string{"m1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result["m1"]) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(result["m1"]))
	}
}

func TestLoadExistingRatingIDs_Empty(t *testing.T) {
	db := openLUSRDB(t)
	result, err := loadExistingRatingIDs(t.Context(), db, "LUSR")
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

	result, err := loadExistingRatingIDs(t.Context(), db, "LUSR")
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
	if _, err := loadExistingRatingIDs(t.Context(), db, "CSR"); err == nil {
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
	// Vue latest avec priorité CSR : si LUSR a été inséré physiquement par
	// un test (cas "Go filter bypassed"), la vue renvoie quand même le CSR
	// pour ce match_id, donc la sémantique "CSR jamais écrasé" est
	// préservée fonctionnellement.
	err := db.QueryRow(
		"SELECT rating_type, rating_value FROM match_skill_rank_latest WHERE match_id = ?",
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

	updated, err := upsertLUSRRatings(t.Context(), db, results, existingCSR, existingLUSR, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated != 0 {
		t.Fatalf("expected 0 updates, got %d", updated)
	}
	assertCSRPreserved(t, db, "m1", 1500.0)
}

// TestUpsertLUSR_SQLGuardWhenGoFilterBypassed simule le cas où la garde Go
// a sauté (map existingCSR vide).
//
// Sémantique post-Phase-2.E : avec append-only INSERT pur, le LUSR est
// physiquement écrit (updated=1) mais la vue match_skill_rank_latest
// priorise CSR > LUSR — le CSR reste fonctionnellement visible et intact.
// Le garde-fou s'est déplacé du SQL vers la vue, sans perte fonctionnelle.
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

	updated, err := upsertLUSRRatings(t.Context(), db, results, emptyCSR, emptyLUSR, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated != 1 {
		t.Fatalf("updated=%d : append-only INSERT physique réussi, attendu 1", updated)
	}
	// Le CSR reste fonctionnellement visible via la vue latest (priorité CSR).
	assertCSRPreserved(t, db, "m1", 1500.0)
}

// TestUpsertLUSR_CounterReflectsOnlyRealWrites valide le compteur sur un batch
// hétérogène : CSR filtré par Go, LUSR existant filtré par Go, CSR avec garde
// Go bypassée, 2 nouveaux matchs.
//
// Sémantique post-Phase-2.E (append-only) : `updated` = nombre d'INSERTs
// physiques réussis. Le cas "Go filter bypassed" produit un INSERT LUSR
// physique (compté), mais la vue match_skill_rank_latest priorise CSR donc
// le CSR reste fonctionnellement intact (cf. assertCSRPreserved).
//
// Cela contraste avec l'ancien comportement où le SQL guard rejetait
// l'UPDATE (updated=2). Désormais updated=3 car m_csr_sql_only reçoit aussi
// un INSERT LUSR physique. C'est le compromis : INSERT pur élimine le bug
// ART, la vue garantit la sémantique métier.
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

	updated, err := upsertLUSRRatings(t.Context(), db, results, existingCSR, existingLUSR, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated != 3 {
		t.Fatalf("updated=%d, attendu 3 (m_csr_sql_only INSERT physique + m_new_1 + m_new_2)", updated)
	}
	// Les deux CSR sont intacts fonctionnellement (via vue latest priorité CSR).
	assertCSRPreserved(t, db, "m_csr_filtered", 1500.0)
	assertCSRPreserved(t, db, "m_csr_sql_only", 1600.0)
	// Le LUSR existant n'a pas été touché (skip par Go). Lecture via vue
	// latest pour cohérence.
	var lusrVal float64
	if err := db.QueryRow(
		"SELECT rating_value FROM match_skill_rank_latest WHERE match_id = 'm_lusr_existing'",
	).Scan(&lusrVal); err != nil {
		t.Fatal(err)
	}
	if lusrVal != 25.0 {
		t.Fatalf("m_lusr_existing rating_value=%.2f, attendu 25.0 (skip par filtre Go)", lusrVal)
	}
	// Les deux nouveaux sont bien insérés en LUSR (visibles via vue latest).
	for _, mid := range []string{"m_new_1", "m_new_2"} {
		var typ string
		if err := db.QueryRow(
			"SELECT rating_type FROM match_skill_rank_latest WHERE match_id = ?", mid,
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
		('m1', '2025-01-01 10:00:00'::TIMESTAMPTZ, NULL, 'Arena Slayer', 'Slayer', FALSE, FALSE, 600)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_participants
		(match_id, xuid, outcome, kills, deaths, assists, kills_expected, deaths_expected,
		 damage_dealt, damage_taken, accuracy, team_id)
		VALUES ('m1', 'xuid1', 2, 10, 5, 2, 10.0, 5.0, 2000.0, 1000.0, 0.5, 0)`); err != nil {
		t.Fatal(err)
	}
	seedCSR(t, db, "m1", 1500.0)

	if _, err := batchComputeLUSR(t.Context(), db, db, "xuid1", nil, true); err != nil {
		t.Fatal(err)
	}
	assertCSRPreserved(t, db, "m1", 1500.0)
}
