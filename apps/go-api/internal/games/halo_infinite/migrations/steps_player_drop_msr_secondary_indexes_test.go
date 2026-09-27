//go:build cgo

package migrations

// steps_player_drop_msr_secondary_indexes_test.go — verrouille le retrait des TROIS
// index secondaires de match_skill_rank (idx_msr_match_lookup, idx_msr_rating_type,
// idx_msr_playlist), plan backlog 2026-09-26, lot B3.
//
// Pourquoi ce retrait : le 2026-09-13, idx_msr_playlist désynchronisé servait 22 lignes
// pour 1 826 réelles (player DB de JGtm) — un index ART désynchronisé et EMPRUNTÉ rend
// des lectures FAUSSES (duckdb#23645, ouvert en 1.5.5). Mesure D-4 amendée
// (migration/psa_index_repro_msr_planprobe_test.go) : aucune lecture ne ralentit sans
// eux, et la seule qui les empruntait (rating_type = 'CSR') est ~10x plus rapide sans.

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

const dropMSRStepName = "drop_msr_secondary_art_indexes_v1"

// msrLegacyIndexDDL : les trois index tels que les posaient les autorités avant le
// 2026-09-27 — l'état d'une player DB de prod, ou d'une DB rouverte par un binaire ancien.
var msrLegacyIndexDDL = []string{
	`CREATE INDEX IF NOT EXISTS idx_msr_match_lookup ON match_skill_rank(match_id, rating_type, written_at)`,
	`CREATE INDEX IF NOT EXISTS idx_msr_rating_type ON match_skill_rank(rating_type)`,
	`CREATE INDEX IF NOT EXISTS idx_msr_playlist    ON match_skill_rank(playlist_group)`,
}

// migratedPlayerDBFile : player DB FICHIER construite par la seule chaîne de migrations.
func migratedPlayerDBFile(t *testing.T) *sql.DB {
	t.Helper()
	migration.SetTitleStepsProvider(StepsFor)
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "stats.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(player): %v", err)
	}
	return db
}

func msrSecondaryIndexCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duckdb_indexes()
		WHERE table_name = 'match_skill_rank' AND index_name LIKE 'idx#_msr#_%' ESCAPE '#'`).Scan(&n); err != nil {
		t.Fatalf("duckdb_indexes(): %v", err)
	}
	return n
}

func applyDropMSRSecondaryIndexes(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, m := range playerMatchSkillRankSteps() {
		if m.Name != dropMSRStepName {
			continue
		}
		if m.ApplySchema == nil {
			t.Fatalf("step %s sans ApplySchema", dropMSRStepName)
		}
		if err := m.ApplySchema(db); err != nil {
			t.Fatalf("%s: %v", dropMSRStepName, err)
		}
		return
	}
	t.Fatalf("step %s absent de la chaîne match_skill_rank", dropMSRStepName)
}

// TestMSRChainNeCreePlusAucunIndexSecondaire : une player DB NEUVE, construite par la
// seule chaîne de migrations, ne porte aucun idx_msr_* (la baseline scellée en pose
// deux, le step de retrait les ôte).
func TestMSRChainNeCreePlusAucunIndexSecondaire(t *testing.T) {
	db := migratedPlayerDBFile(t)
	if n := msrSecondaryIndexCount(t, db); n != 0 {
		t.Errorf("%d idx_msr_* sur une player DB fraîchement migrée, attendu 0 "+
			"(step %s + retrait des autorités non scellées)", n, dropMSRStepName)
	}
}

// TestDropMSRSecondaryARTIndexes_RetireEtEstIdempotent : le step retire les trois index
// d'une DB EXISTANTE qui les porte, et un second passage ne casse pas (DROP IF EXISTS).
func TestDropMSRSecondaryARTIndexes_RetireEtEstIdempotent(t *testing.T) {
	db := migratedPlayerDBFile(t)
	for _, stmt := range msrLegacyIndexDDL {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("pose de l'index legacy: %v", err)
		}
	}
	if n := msrSecondaryIndexCount(t, db); n != 3 {
		t.Fatalf("préalable : %d idx_msr_* posés, attendu 3 — le test ne mordrait pas", n)
	}

	applyDropMSRSecondaryIndexes(t, db)
	if n := msrSecondaryIndexCount(t, db); n != 0 {
		t.Errorf("%d idx_msr_* toujours présents après %s", n, dropMSRStepName)
	}
	applyDropMSRSecondaryIndexes(t, db) // idempotence
}

// TestDropMSRSecondaryARTIndexes_PreserveLignesEtVues : le step ne touche QUE de la DDL
// d'index — aucune ligne ne disparaît, les deux vues restent lisibles et rendent la même
// chose, et un INSERT sans id marche toujours (PK technique et séquence intactes).
func TestDropMSRSecondaryARTIndexes_PreserveLignesEtVues(t *testing.T) {
	db := migratedPlayerDBFile(t)
	for _, stmt := range msrLegacyIndexDDL {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("pose de l'index legacy: %v", err)
		}
	}
	if _, err := db.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, playlist_group, written_at) VALUES
		('m1', 'LUSR',    1200, 'arena_slayer', TIMESTAMP '2026-09-01 10:00:00'),
		('m1', 'LUSR_V2', 1203, 'arena_slayer', TIMESTAMP '2026-09-01 10:00:00'),
		('m1', 'LUSR',    1210, 'arena_slayer', TIMESTAMP '2026-09-02 10:00:00'),
		('m2', 'CSR',     1500, 'ranked_arena', TIMESTAMP '2026-09-03 10:00:00'),
		('m2', 'LUSR',    1250, 'btb',          TIMESTAMP '2026-09-03 10:00:00')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	type counts struct{ rows, latest, byType int }
	read := func() counts {
		var c counts
		for q, dst := range map[string]*int{
			`SELECT COUNT(*) FROM match_skill_rank`:                &c.rows,
			`SELECT COUNT(*) FROM match_skill_rank_latest`:         &c.latest,
			`SELECT COUNT(*) FROM match_skill_rank_latest_by_type`: &c.byType,
		} {
			if err := db.QueryRow(q).Scan(dst); err != nil {
				t.Fatalf("%s: %v", q, err)
			}
		}
		return c
	}
	before := read()
	if before != (counts{rows: 5, latest: 2, byType: 4}) {
		t.Fatalf("préalable : %+v, attendu {rows:5 latest:2 byType:4}", before)
	}

	applyDropMSRSecondaryIndexes(t, db)

	if after := read(); after != before {
		t.Errorf("après %s : %+v, attendu %+v (le step ne touche que la DDL d'index)",
			dropMSRStepName, after, before)
	}
	var winner float64
	if err := db.QueryRow(`SELECT rating_value FROM match_skill_rank_latest WHERE match_id = 'm2'`).
		Scan(&winner); err != nil {
		t.Fatalf("lecture _latest: %v", err)
	}
	if winner != 1500 {
		t.Errorf("_latest m2 = %v, attendu 1500 (priorité CSR)", winner)
	}
	if _, err := db.Exec(`INSERT INTO match_skill_rank (match_id, rating_type, rating_value)
		VALUES ('m3', 'LUSR', 1300)`); err != nil {
		t.Errorf("INSERT sans id après le step (PK/séquence intactes ?): %v", err)
	}
}
