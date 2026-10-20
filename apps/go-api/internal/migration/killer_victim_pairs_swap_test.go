package migration

// killer_victim_pairs_swap_test.go — ReplaceKillerVictimPairs remplace les couples des matchs
// donnes par ECHANGE de table (schema et index gardes), et la migration
// shared_killer_victim_pairs_drop_legacy_indexes_v1 retire les trois index herites.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// seedKillerVictimPairs : la table telle qu une base nee avant le schema Go la porte, AVEC ses trois
// index ART herites.
func seedKillerVictimPairs(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, s := range []string{
		`CREATE TABLE killer_victim_pairs (match_id VARCHAR NOT NULL, killer_xuid VARCHAR NOT NULL,
			killer_gamertag VARCHAR, victim_xuid VARCHAR NOT NULL, victim_gamertag VARCHAR,
			kill_count INTEGER DEFAULT 1, time_ms INTEGER, is_validated BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`,
		`CREATE INDEX idx_kv_match ON killer_victim_pairs(match_id)`,
		`CREATE INDEX idx_kv_killer ON killer_victim_pairs(killer_xuid)`,
		`CREATE INDEX idx_kv_victim ON killer_victim_pairs(victim_xuid)`,
		`INSERT INTO killer_victim_pairs (match_id,killer_xuid,killer_gamertag,victim_xuid,victim_gamertag,time_ms)
			VALUES ('m1','jgtm','JGtm','sprifi','Sprifi',593093),('m1','a','A','b','B',100),
			       ('m2','c','C','d','D',200),('m3','e','E','f','F',300)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return db
}

func compterCouples(t *testing.T, db *sql.DB, where string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM killer_victim_pairs ` + where).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Les couples de m1 sont remplaces, ceux de m3 retires (liste vide), ceux de m2 gardes ; le DDL et
// les trois index sont identiques apres l echange, et les tables de travail ont disparu.
func TestReplaceKillerVictimPairsParEchange(t *testing.T) {
	db := seedKillerVictimPairs(t)
	ctx := context.Background()
	avant, err := ddlDesIndex(ctx, db, kvpTable)
	if err != nil || len(avant) != 3 {
		t.Fatalf("index avant = %v (err=%v), attendu 3", avant, err)
	}
	n, err := ReplaceKillerVictimPairs(ctx, db, map[string][]KillerVictimPair{
		"m1": {{KillerXUID: "jgtm", KillerGamertag: "JGtm", VictimXUID: "sprifi", TimeMS: 593095}},
		"m3": {},
	})
	if err != nil || n != 2 {
		t.Fatalf("ReplaceKillerVictimPairs = %d, %v ; attendu 2 lignes", n, err)
	}
	if c := compterCouples(t, db, `WHERE match_id = 'm1' AND time_ms = 593095 AND victim_gamertag IS NULL`); c != 1 {
		t.Errorf("m1 : %d couple(s) recalcule(s), attendu 1 (gamertag inconnu = NULL)", c)
	}
	if compterCouples(t, db, `WHERE match_id = 'm2'`) != 1 || compterCouples(t, db, `WHERE match_id = 'm3'`) != 0 {
		t.Error("m2 doit garder son couple, m3 n en a plus")
	}
	apres, err := ddlDesIndex(ctx, db, kvpTable)
	if err != nil || len(apres) != 3 {
		t.Errorf("index apres = %v (err=%v), attendu les trois memes", apres, err)
	}
	var travail int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duckdb_tables() WHERE table_name LIKE 'killer_victim_pairs__%'`).
		Scan(&travail); err != nil || travail != 0 {
		t.Errorf("tables de travail restantes : %d (err=%v)", travail, err)
	}
}

// La migration retire les trois index herites, et se rejoue sans erreur.
func TestMigrationRetireLesIndexHeritesDeKillerVictimPairs(t *testing.T) {
	db := seedKillerVictimPairs(t)
	var m Migration
	for _, r := range registry {
		if r.Name == "shared_killer_victim_pairs_drop_legacy_indexes_v1" {
			m = r
		}
	}
	if m.ApplySchema == nil {
		t.Fatal("migration non enregistree")
	}
	for pass := 1; pass <= 2; pass++ {
		if err := m.ApplySchema(db); err != nil {
			t.Fatalf("passe %d : %v", pass, err)
		}
	}
	idx, err := ddlDesIndex(context.Background(), db, kvpTable)
	if err != nil || len(idx) != 0 {
		t.Errorf("index restants = %v (err=%v), attendu aucun", idx, err)
	}
}
