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

// TestRestore_SauvegardeAnterieureAUneConversionAppendOnly : une sauvegarde prise AVANT une
// conversion append-only (le favori est dans la table legacy, le registre ne porte pas l'étape)
// est restaurée dans l'état de son époque ; le cycle de migrations suivant la convertit, et la
// vue _latest voit le favori. Poser le schéma ACTUEL avant de verser les lignes créerait
// `match_favorites_history` vide : la conversion se sauterait (`if hasHistory`) et le favori
// disparaîtrait de la vue.
func TestRestore_SauvegardeAnterieureAUneConversionAppendOnly(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.duckdb")
	src, err := sql.Open("duckdb", srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := migration.RunForTitleDB(src, "halo_infinite", migration.TargetSharedSocial); err != nil {
		t.Fatalf("migrations source: %v", err)
	}
	// Vieillissement : la base revient à l'état d'avant shared_social_favorites_append_only_v1.
	for _, s := range []string{
		`DROP VIEW match_favorites_latest`,
		`DROP TABLE match_favorites_history`,
		`DELETE FROM schema_migrations WHERE name = 'shared_social_favorites_append_only_v1'`,
		`INSERT INTO match_favorites (player_slug, match_id) VALUES ('p', 'm1')`,
	} {
		if _, err := src.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	_ = src.Close()

	key := warehouseBackupKey("halo_infinite", backupDBSharedSoc)
	target := restoreOne(t, srcPath, key, filepath.Join("halo_infinite", backupDBSharedSoc))

	db, err := sql.Open("duckdb", target)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := migration.RunForTitleDB(db, "halo_infinite", migration.TargetSharedSocial); err != nil {
		t.Fatalf("cycle de migrations après restauration: %v", err)
	}
	var latest int
	if err := db.QueryRow(`SELECT count(*) FROM match_favorites_latest`).Scan(&latest); err != nil {
		t.Fatal(err)
	}
	if latest != 1 {
		t.Fatalf("match_favorites_latest = %d après restauration + migrations, want 1 (favori perdu)", latest)
	}
}

// restoreOne exporte la base srcPath sous la clé key (dossier de staging rel) et la restaure
// dans un fichier neuf, dont il rend le chemin.
func restoreOne(t *testing.T, srcPath, key, rel string) string {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	staging := filepath.Join(dir, "staging")
	if _, err := duckdbbackup.ExportTarget(ctx, duckdbbackup.Target{Key: key, Path: srcPath},
		filepath.Join(staging, rel), 1); err != nil {
		t.Fatalf("export: %v", err)
	}
	target := filepath.Join(dir, "restored.duckdb")
	if _, err := duckdbbackup.ImportFromStaging(ctx, staging, func(k string) string {
		if k == key {
			return target
		}
		return ""
	}, FinalizeRestoredDB, false); err != nil {
		t.Fatalf("import: %v", err)
	}
	return target
}

// TestRestore_BaseJoueurClesVuesEtSequencesAlignees : une base joueur migrée, exportée puis
// restaurée porte de nouveau ses clés primaires (registre des migrations compris) et ses vues
// _latest ; FinalizeRestoredDB aligne la séquence d'une table append-only dont les identifiants
// ont été écrits sans la consommer (sans alignement, la prochaine insertion reprendrait un id
// déjà pris) ; le cycle de migrations suivant passe.
func TestRestore_BaseJoueurClesVuesEtSequencesAlignees(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "src.duckdb")
	src, err := sql.Open("duckdb", srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := migration.RunForTitleDB(src, "halo_infinite", migration.TargetPlayer); err != nil {
		t.Fatalf("migrations source: %v", err)
	}
	for _, s := range []string{
		`INSERT INTO sync_meta (key, value) VALUES ('k', 'v')`,
		`INSERT INTO player_csr_snapshots (id, playlist_id, season_id) VALUES (1, 'p', 's'), (2, 'p', 's'), (3, 'p', 's')`,
	} {
		if _, err := src.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	_ = src.Close()

	key := playerBackupKey("halo_infinite", "GT")
	target := restoreOne(t, srcPath, key, filepath.Join("halo_infinite", backupPlayerSegment, "GT"))
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
	if _, err := db.Exec(`INSERT INTO player_csr_snapshots (playlist_id, season_id) VALUES ('p', 's')`); err != nil {
		t.Fatalf("séquence non alignée par FinalizeRestoredDB : %v", err)
	}
	if _, err := db.Exec(`INSERT INTO sync_meta (key, value) VALUES ('k', 'w')
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatalf("ON CONFLICT sur la base restaurée : %v", err)
	}
	if err := migration.RunForTitleDB(db, "halo_infinite", migration.TargetPlayer); err != nil {
		t.Fatalf("cycle de migrations sur la base restaurée : %v", err)
	}
}
