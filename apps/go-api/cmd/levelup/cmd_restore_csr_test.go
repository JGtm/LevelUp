package main

// cmd_restore_csr_test.go — restore-csr n'écrit dans match_skill_rank (append-only, ADR 0026)
// que par INSERT : aucun LUSR n'est effacé, le CSR ajouté prime à la lecture par la vue
// match_skill_rank_latest, et une seconde exécution n'insère rien.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/titleseams"
	"levelup/go-api/internal/migration"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

func TestRestoreCSR_InsertSeul_CSRPrimeEtIdempotent(t *testing.T) {
	ctx := context.Background()
	repoRoot := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: repoRoot}
	wireStartupSeams(cfg)
	t.Cleanup(func() { titleseams.RegisterAll("") })

	playerPath := titlePkg.NewPathResolver(repoRoot).PlayerDBPath(titlePkg.DefaultSlug, "Joueur")
	if err := os.MkdirAll(filepath.Dir(playerPath), 0o755); err != nil {
		t.Fatalf("mkdir : %v", err)
	}
	if err := applyMigrationsOnDB(playerPath, migration.TargetPlayer); err != nil {
		t.Fatalf("migrations player : %v", err)
	}
	execOnPlayer(t, playerPath,
		`INSERT INTO match_skill_rank (match_id, rating_type, rating_value, rating_deviation, playlist_group)
		 VALUES ('m1', 'LUSR', 1100, 50, 'ranked'), ('m2', 'CSR', 1400, 40, 'ranked')`)

	backup := filepath.Join(t.TempDir(), "legacy.duckdb")
	writeLegacyBackup(t, backup)

	args := []string{"--gamertag", "Joueur", "--backup", backup}
	if err := runRestoreCSR(cfg, args); err != nil {
		t.Fatalf("restore-csr : %v", err)
	}
	if err := runRestoreCSR(cfg, args); err != nil {
		t.Fatalf("restore-csr (seconde exécution) : %v", err)
	}

	handle, err := duckdbpkg.OpenReadWrite(playerPath)
	if err != nil {
		t.Fatalf("réouverture : %v", err)
	}
	defer func() { _ = handle.Close() }()
	db := handle.SQLDb()

	// m1 : LUSR conservé + CSR ajouté ; m2 : CSR préexistant, rien d'ajouté ; m3 : une ligne
	// malgré le doublon du backup ; m4 (LUSR du backup) : ignoré.
	want := map[string]int{"m1/LUSR": 1, "m1/CSR": 1, "m2/CSR": 1, "m3/CSR": 1, "m5/CSR": 1}
	got := map[string]int{}
	rows, err := db.QueryContext(ctx,
		`SELECT match_id || '/' || rating_type, COUNT(*) FROM match_skill_rank GROUP BY 1`)
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var k string
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			t.Fatalf("scan : %v", err)
		}
		got[k] = n
	}
	if len(got) != len(want) {
		t.Fatalf("lignes brutes = %v, attendu %v", got, want)
	}
	for k, n := range want {
		if got[k] != n {
			t.Errorf("%s : %d ligne(s), attendu %d (tout = %v)", k, got[k], n, got)
		}
	}

	var latestType string
	var value float64
	if err := db.QueryRowContext(ctx,
		`SELECT rating_type, rating_value FROM match_skill_rank_latest WHERE match_id = 'm1'`,
	).Scan(&latestType, &value); err != nil {
		t.Fatalf("lecture _latest : %v", err)
	}
	if latestType != "CSR" || value != 1500 {
		t.Errorf("m1 lu par _latest = %s %.0f, attendu CSR 1500", latestType, value)
	}
	assertRestoredCSR(t, db, "m1", restoredCSR{tier: "Diamond", subTier: 2, group: "ranked_arena",
		start: "2026-01-01 10:00:00"})
	// sub_tier absent → 0, playlist_group absent → « ranked », start_time absent → NULL.
	assertRestoredCSR(t, db, "m5", restoredCSR{tier: "Gold", subTier: 0, group: "ranked", start: ""})
}

// restoredCSR : colonnes d'une ligne CSR restaurée ; start vide = start_time NULL.
type restoredCSR struct {
	tier, group, start string
	subTier            int
}

func assertRestoredCSR(t *testing.T, db *sql.DB, matchID string, want restoredCSR) {
	t.Helper()
	var got restoredCSR
	var tier, start sql.NullString
	var sub sql.NullInt64
	if err := db.QueryRow(`SELECT tier, sub_tier, playlist_group, strftime(start_time, '%Y-%m-%d %H:%M:%S')
		FROM match_skill_rank WHERE match_id = ? AND rating_type = 'CSR'`, matchID).
		Scan(&tier, &sub, &got.group, &start); err != nil {
		t.Fatalf("lecture CSR %s : %v", matchID, err)
	}
	got.tier, got.start = tier.String, start.String
	if !sub.Valid {
		got.subTier = -1
	} else {
		got.subTier = int(sub.Int64)
	}
	if got != want {
		t.Errorf("CSR %s restauré = %+v, attendu %+v", matchID, got, want)
	}
}

// TestRestoreCSR_DryRunNEcritRien : --dry-run inspecte le backup sans aucune écriture.
func TestRestoreCSR_DryRunNEcritRien(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: repoRoot}
	wireStartupSeams(cfg)
	t.Cleanup(func() { titleseams.RegisterAll("") })

	playerPath := titlePkg.NewPathResolver(repoRoot).PlayerDBPath(titlePkg.DefaultSlug, "Joueur")
	if err := os.MkdirAll(filepath.Dir(playerPath), 0o755); err != nil {
		t.Fatalf("mkdir : %v", err)
	}
	if err := applyMigrationsOnDB(playerPath, migration.TargetPlayer); err != nil {
		t.Fatalf("migrations player : %v", err)
	}
	backup := filepath.Join(t.TempDir(), "legacy.duckdb")
	writeLegacyBackup(t, backup)

	if err := runRestoreCSR(cfg, []string{"--gamertag", "Joueur", "--backup", backup, "--dry-run"}); err != nil {
		t.Fatalf("restore-csr --dry-run : %v", err)
	}
	handle, err := duckdbpkg.OpenReadWrite(playerPath)
	if err != nil {
		t.Fatalf("réouverture : %v", err)
	}
	defer func() { _ = handle.Close() }()
	var n int
	if err := handle.SQLDb().QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&n); err != nil {
		t.Fatalf("lecture : %v", err)
	}
	if n != 0 {
		t.Errorf("--dry-run a écrit %d ligne(s) dans match_skill_rank, attendu 0", n)
	}
}

func execOnPlayer(t *testing.T, path, q string) {
	t.Helper()
	handle, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("ouverture : %v", err)
	}
	defer func() { _ = handle.Close() }()
	if _, err := handle.SQLDb().Exec(q); err != nil {
		t.Fatalf("exec : %v", err)
	}
}

func writeLegacyBackup(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("backup : %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`
		CREATE TABLE match_skill_rank (
			match_id VARCHAR, rating_type VARCHAR, rating_value DOUBLE, rating_deviation DOUBLE,
			tier VARCHAR, sub_tier INTEGER, tier_label VARCHAR, rating_delta DOUBLE,
			playlist_group VARCHAR, start_time TIMESTAMP);
		INSERT INTO match_skill_rank VALUES
			('m1', 'CSR', 1500, 30, 'Diamond', 2, 'Diamond 3', 12, 'ranked_arena', TIMESTAMP '2026-01-01 10:00:00'),
			('m5', 'CSR', 1100, 30, 'Gold', NULL, 'Gold 1', NULL, NULL, NULL),
			('m2', 'CSR', 1450, 30, 'Diamond', 1, 'Diamond 2', 8, NULL, TIMESTAMP '2026-01-02 10:00:00'),
			('m3', 'CSR', 1300, 30, 'Platinum', NULL, 'Platinum 6', NULL, 'ranked', TIMESTAMP '2026-01-03 10:00:00'),
			('m3', 'CSR', 1310, 30, 'Platinum', NULL, 'Platinum 6', NULL, 'ranked', TIMESTAMP '2026-01-03 11:00:00'),
			('m4', 'LUSR', 900, 30, NULL, NULL, NULL, NULL, 'ranked', NULL);`); err != nil {
		t.Fatalf("backup DDL : %v", err)
	}
}
