package migration

// steps_player_repair_append_only_ids_test.go — repair_player_append_only_ids_v1 : ids NULL
// ou en double réparés par swap, aucune ligne perdue, clé et défauts reposés, table saine
// laissée intacte.

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openRepairIDsDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "stats.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func execRepairIDs(t *testing.T, db *sql.DB, stmts ...string) {
	t.Helper()
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%.70s: %v", s, err)
		}
	}
}

func scanInt(t *testing.T, db *sql.DB, q string) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRow(q).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

// TestRepairAppendOnlyIDs_NullsEtDoublons : l'état relevé sur les bases réelles (table sans
// clé, ids en double et NULL, séquence en retard) est réparé sans perte ; la première
// occurrence d'un id le garde ; DEFAULT, NOT NULL, clé et vue tiennent ; second passage sans
// effet.
func TestRepairAppendOnlyIDs_NullsEtDoublons(t *testing.T) {
	db := openRepairIDsDB(t)
	execRepairIDs(t, db,
		`CREATE SEQUENCE pme_seq START 1`,
		`CREATE TABLE player_match_enrichment (
			id BIGINT, match_id VARCHAR NOT NULL, stage VARCHAR DEFAULT 'legacy',
			performance_score DOUBLE, written_at TIMESTAMP)`,
		`INSERT INTO player_match_enrichment (id, match_id, stage, performance_score, written_at) VALUES
			(1, 'm1', 'live', 10, TIMESTAMP '2026-09-01 10:00:00'),
			(2, 'm2', 'live', 20, TIMESTAMP '2026-09-01 11:00:00'),
			(2, 'm3', 'live', 30, TIMESTAMP '2026-09-21 11:00:00'),
			(NULL, 'm4', 'live', 40, TIMESTAMP '2026-09-22 11:00:00'),
			(NULL, 'm5', 'live', 50, TIMESTAMP '2026-09-22 12:00:00')`,
		`CREATE VIEW player_match_enrichment_latest AS SELECT * FROM player_match_enrichment
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id ORDER BY written_at DESC, id DESC) = 1`,
	)

	if err := applyRepairAppendOnlyIDs(db); err != nil {
		t.Fatalf("réparation: %v", err)
	}

	if n := scanInt(t, db, `SELECT COUNT(*) FROM player_match_enrichment`); n != 5 {
		t.Errorf("lignes = %d, attendu 5", n)
	}
	if n := scanInt(t, db, `SELECT COUNT(*) - COUNT(DISTINCT id) + (COUNT(*) - COUNT(id)) FROM player_match_enrichment`); n != 0 {
		t.Errorf("%d id NULL ou en double après réparation", n)
	}
	if id := scanInt(t, db, `SELECT id FROM player_match_enrichment WHERE match_id = 'm2'`); id != 2 {
		t.Errorf("m2 (première occurrence de l'id 2) a l'id %d, attendu 2", id)
	}
	if id := scanInt(t, db, `SELECT id FROM player_match_enrichment WHERE match_id = 'm3'`); id <= 2 {
		t.Errorf("m3 (doublon de l'id 2) a l'id %d, attendu un id neuf > 2", id)
	}
	if n := scanInt(t, db, `SELECT COUNT(*) FROM duckdb_constraints()
		WHERE table_name = 'player_match_enrichment' AND constraint_type = 'PRIMARY KEY'`); n != 1 {
		t.Errorf("clé primaire : %d, attendu 1", n)
	}
	if n := scanInt(t, db, `SELECT COUNT(*) FROM player_match_enrichment_latest`); n != 5 {
		t.Errorf("vue _latest après swap : %d lignes, attendu 5", n)
	}
	// DEFAULT de l'id (séquence réalignée), DEFAULT de stage relu, DEFAULT UTC posé sur
	// l'horloge qui n'en avait pas, NOT NULL de match_id relu.
	execRepairIDs(t, db, `INSERT INTO player_match_enrichment (match_id) VALUES ('m6')`)
	var stage string
	var wa sql.NullTime
	var id int64
	if err := db.QueryRow(`SELECT id, stage, written_at FROM player_match_enrichment WHERE match_id = 'm6'`).
		Scan(&id, &stage, &wa); err != nil {
		t.Fatalf("lecture m6: %v", err)
	}
	if stage != "legacy" || !wa.Valid {
		t.Errorf("m6 : stage=%q written_at valide=%v, attendu legacy et horloge posée", stage, wa.Valid)
	}
	if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id) VALUES (NULL)`); err == nil {
		t.Error("NOT NULL de match_id non reposé")
	}
	before := scanInt(t, db, `SELECT SUM(id) FROM player_match_enrichment`)
	if err := applyRepairAppendOnlyIDs(db); err != nil {
		t.Fatalf("second passage: %v", err)
	}
	if after := scanInt(t, db, `SELECT SUM(id) FROM player_match_enrichment`); after != before {
		t.Errorf("second passage a changé des ids (%d -> %d) : la table saine doit rester intacte", before, after)
	}
}

// TestRepairAppendOnlyIDs_TableSaineIntacte : une table sans défaut n'est pas reconstruite
// (ses ids et son type restent ceux d'origine), une table absente est ignorée.
func TestRepairAppendOnlyIDs_TableSaineIntacte(t *testing.T) {
	db := openRepairIDsDB(t)
	execRepairIDs(t, db,
		`CREATE SEQUENCE personal_score_awards_id_seq START 1`,
		`CREATE TABLE personal_score_awards (
			id INTEGER PRIMARY KEY DEFAULT nextval('personal_score_awards_id_seq'),
			match_id VARCHAR NOT NULL, written_at TIMESTAMP)`,
		`INSERT INTO personal_score_awards (match_id) VALUES ('a'), ('b')`,
	)
	if err := applyRepairAppendOnlyIDs(db); err != nil {
		t.Fatalf("réparation: %v", err)
	}
	if n := scanInt(t, db, `SELECT SUM(id) FROM personal_score_awards`); n != 3 {
		t.Errorf("ids modifiés sur une table saine (somme %d, attendu 3)", n)
	}
	var typ string
	if err := db.QueryRow(`SELECT data_type FROM duckdb_columns()
		WHERE table_name = 'personal_score_awards' AND column_name = 'id'`).Scan(&typ); err != nil {
		t.Fatal(err)
	}
	if typ != "INTEGER" {
		t.Errorf("type de l'id = %s, attendu INTEGER (table saine non reconstruite)", typ)
	}
}
