//go:build cgo

package migration

// steps_player_drop_psa_secondary_indexes_test.go — verrouille le retrait des TROIS
// derniers index secondaires de personal_score_awards (2026-09-20).
//
// Pourquoi ce retrait : la sonde data-health signalait l'index DÉSYNCHRONISÉ à chaque
// boot et la réparation manuelle du 2026-09-20 a montré que les clés en écart étaient
// des match_id du mois courant — le défaut DuckDB #23645 se reforme sur les insertions
// COURANTES, donc un index ART est structurellement non fiable sur cette table. Aucun
// lecteur n'y perd : tous passent par la vue _latest (plan Sequential Scan).

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// psaSecondaryIndexes — les trois index retirés par drop_psa_secondary_art_indexes_v1.
var psaSecondaryIndexes = []string{"idx_psa_match", "idx_psa_category", "idx_psa_gen"}

func openPSATestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "stats.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := execScript(db, PlayerPersonalScoreAwardsDDL); err != nil {
		t.Fatalf("DDL personal_score_awards: %v", err)
	}
	return db
}

func psaIndexCount(t *testing.T, db *sql.DB, name string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duckdb_indexes() WHERE index_name = ?`, name).Scan(&n); err != nil {
		t.Fatalf("duckdb_indexes(%s): %v", name, err)
	}
	return n
}

func applyDropPSASecondaryIndexes(t *testing.T, db *sql.DB) {
	t.Helper()
	m, ok := ByName("drop_psa_secondary_art_indexes_v1")
	if !ok {
		t.Fatal("step drop_psa_secondary_art_indexes_v1 absent du registre")
	}
	if m.ApplySchema == nil {
		t.Fatal("step drop_psa_secondary_art_indexes_v1 sans ApplySchema")
	}
	if err := m.ApplySchema(db); err != nil {
		t.Fatalf("drop_psa_secondary_art_indexes_v1: %v", err)
	}
}

// TestPSADDLNeCreePlusAucunIndexSecondaire : sur une DB FRAÎCHE, l'autorité de schéma
// (PlayerPersonalScoreAwardsDDL, consommée par les migrations, sync.EnsurePlayerSchema
// et le seed démo) ne pose plus aucun des trois index.
func TestPSADDLNeCreePlusAucunIndexSecondaire(t *testing.T) {
	db := openPSATestDB(t)
	for _, idx := range psaSecondaryIndexes {
		if n := psaIndexCount(t, db, idx); n != 0 {
			t.Errorf("index %s créé par PlayerPersonalScoreAwardsDDL — retiré le 2026-09-20 "+
				"(récidive #23645 sur les insertions courantes, aucun lecteur ne l'emprunte)", idx)
		}
	}
}

// TestDropPSASecondaryARTIndexes_RetireEtEstIdempotent : le step retire les trois index
// d'une DB EXISTANTE qui les porte, et un second passage ne casse pas (DROP IF EXISTS).
func TestDropPSASecondaryARTIndexes_RetireEtEstIdempotent(t *testing.T) {
	db := openPSATestDB(t)
	// État d'une DB de prod d'avant le 2026-09-20 : les trois index sont posés.
	for _, stmt := range []string{
		`CREATE INDEX IF NOT EXISTS idx_psa_match    ON personal_score_awards(match_id)`,
		`CREATE INDEX IF NOT EXISTS idx_psa_category ON personal_score_awards(award_category)`,
		`CREATE INDEX IF NOT EXISTS idx_psa_gen      ON personal_score_awards(match_id, xuid, generation_id)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("pose de l'index legacy: %v", err)
		}
	}
	for _, idx := range psaSecondaryIndexes {
		if n := psaIndexCount(t, db, idx); n != 1 {
			t.Fatalf("préalable : index %s absent (count=%d) — le test ne mordrait pas", idx, n)
		}
	}

	applyDropPSASecondaryIndexes(t, db)
	for _, idx := range psaSecondaryIndexes {
		if n := psaIndexCount(t, db, idx); n != 0 {
			t.Errorf("index %s toujours présent après drop_psa_secondary_art_indexes_v1", idx)
		}
	}

	// Idempotence : rejoué sur une DB qui ne les porte plus.
	applyDropPSASecondaryIndexes(t, db)
}

// TestDropPSASecondaryARTIndexes_PreserveLesDonnees : le step ne touche QUE de la DDL
// d'index — aucune ligne ne disparaît, et la vue _latest reste lisible.
func TestDropPSASecondaryARTIndexes_PreserveLesDonnees(t *testing.T) {
	db := openPSATestDB(t)
	if _, err := db.Exec(psaLatestViewSQL); err != nil {
		t.Fatalf("vue _latest: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO personal_score_awards
		    (match_id, xuid, award_name, award_category, award_count, award_score, generation_id, is_tombstone)
		VALUES ('m1','u1','flag_captured','objective',1,100,1,FALSE),
		       ('m1','u1','zone_secured','objective',2,50,1,FALSE),
		       ('m2','u1','killed_player','combat',1,10,1,FALSE)`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	applyDropPSASecondaryIndexes(t, db)

	var rows, score int
	if err := db.QueryRow(`SELECT COUNT(*) FROM personal_score_awards`).Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 3 {
		t.Errorf("lignes après drop = %d, want 3 (le step ne touche que la DDL d'index)", rows)
	}
	if err := db.QueryRow(`SELECT COALESCE(SUM(award_score),0)::INTEGER
		FROM personal_score_awards_latest WHERE match_id = 'm1' AND xuid = 'u1'`).Scan(&score); err != nil {
		t.Fatalf("lecture via la vue _latest: %v", err)
	}
	if score != 150 {
		t.Errorf("SUM(award_score) m1/u1 via _latest = %d, want 150", score)
	}
}
