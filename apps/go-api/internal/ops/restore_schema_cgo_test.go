//go:build cgo

package ops

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/migration"
	"levelup/go-api/pkg/duckdbbackup"
)

// TestRestoreSchema_BaseJoueurRetrouveSesCles : une base joueur migrée (clés posées), exportée en
// Parquet puis restaurée avec RestoreSchema, porte de nouveau ses clés primaires — dont celle
// du registre des migrations — et accepte les écritures de prod qui en dépendent.
func TestRestoreSchema_BaseJoueurRetrouveSesCles(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.duckdb")
	src, err := sql.Open("duckdb", srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := migration.RunForTitleDB(src, "halo_infinite", migration.TargetPlayer); err != nil {
		t.Fatalf("migrations source: %v", err)
	}
	if _, err := src.Exec(`INSERT INTO sync_meta (key, value) VALUES ('k', 'v')`); err != nil {
		t.Fatal(err)
	}
	_ = src.Close()

	key := playerBackupKey("halo_infinite", "GT")
	staging := filepath.Join(dir, "staging")
	if _, err := duckdbbackup.ExportTarget(ctx, duckdbbackup.Target{Key: key, Path: srcPath},
		filepath.Join(staging, "halo_infinite", "player", "GT"), 1); err != nil {
		t.Fatalf("export: %v", err)
	}
	target := filepath.Join(dir, "restored.duckdb")
	res, err := duckdbbackup.ImportFromStaging(ctx, staging, func(k string) string {
		if k == key {
			return target
		}
		return ""
	}, RestoreSchema{}, false)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if len(res.Restored) != 1 || len(res.FallbackTables) != 0 || res.DroppedRows != 0 {
		t.Fatalf("résultat : %+v", res)
	}

	db, err := sql.Open("duckdb", target)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	for _, table := range []string{"schema_migrations", "sync_meta", "arc", "challenge", "streak", "match_skill_rank"} {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM duckdb_constraints()
			WHERE table_name = ? AND constraint_type = 'PRIMARY KEY'`, table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("%s : clé primaire absente après restauration", table)
		}
	}
	var views int
	if err := db.QueryRow(`SELECT count(*) FROM duckdb_views() WHERE view_name = 'match_skill_rank_latest'`).Scan(&views); err != nil || views != 1 {
		t.Errorf("vue _latest absente après restauration (n=%d, err=%v)", views, err)
	}
	// Écriture de prod (migrations sync_meta) : ON CONFLICT sur la clé restaurée.
	if _, err := db.Exec(`INSERT INTO sync_meta (key, value) VALUES ('k', 'w')
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatalf("ON CONFLICT sur la base restaurée : %v", err)
	}
	// Le registre restauré est celui de la sauvegarde : un nouveau cycle n'applique rien.
	var pending int
	if err := db.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&pending); err != nil || pending == 0 {
		t.Fatalf("registre des migrations vide après restauration (n=%d, err=%v)", pending, err)
	}
	if err := migration.RunForTitleDB(db, "halo_infinite", migration.TargetPlayer); err != nil {
		t.Fatalf("cycle de migrations sur la base restaurée : %v", err)
	}
}
