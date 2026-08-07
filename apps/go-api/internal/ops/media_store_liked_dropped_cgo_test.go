//go:build cgo

// Package ops — media_store_liked_dropped_cgo_test.go : garde-rail du retrait
// définitif de media_files.liked / liked_at (2026-08-04).
//
// POURQUOI CE TEST EXISTE. Le DROP de ces deux colonnes avait été REPORTÉ pour
// une raison précise : une migration DROP COLUMN seule ne tient pas, parce que
// la boucle d'indexation la défait. ensureMediaTables (media_store.go) est
// appelée par IndexMedia à CHAQUE scan de médias et exécute, en plus de son
// CREATE TABLE IF NOT EXISTS, un `ALTER TABLE media_files ADD COLUMN IF NOT
// EXISTS` pour chaque colonne de sa liste. Toute colonne encore mentionnée là
// RENAÎT au scan suivant, silencieusement, sur toutes les DB de prod.
//
// Le test reproduit donc la séquence réelle et complète :
//
//	migrations (dont le DROP) → ensureMediaTables → la colonne est-elle revenue ?
//
// Il échouera si quelqu'un réintroduit `liked` dans le CREATE TABLE d'
// ensureMediaTables, dans sa liste ADD COLUMN, ou dans le CREATE de
// create_base_shared_social_schema.

package ops

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// mediaFilesColumns retourne l'ensemble des colonnes de media_files.
func mediaFilesColumns(t *testing.T, db *sql.DB) map[string]bool {
	t.Helper()
	rows, err := db.Query(
		`SELECT column_name FROM information_schema.columns
		 WHERE table_schema = 'main' AND table_name = 'media_files'`)
	if err != nil {
		t.Fatalf("information_schema.columns: %v", err)
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan column_name: %v", err)
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}
	return cols
}

// mediaFilesIndexNames retourne les index portés par media_files.
func mediaFilesIndexNames(t *testing.T, db *sql.DB) map[string]bool {
	t.Helper()
	rows, err := db.Query(`SELECT index_name FROM duckdb_indexes() WHERE table_name = 'media_files'`)
	if err != nil {
		t.Fatalf("duckdb_indexes: %v", err)
	}
	defer rows.Close()
	names := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan index_name: %v", err)
		}
		names[n] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}
	return names
}

// migratedSharedSocialDB provisionne une shared_social.duckdb neuve via la
// CHAÎNE DE MIGRATIONS RÉELLE (registre global + steps title-owned Halo), donc
// en incluant drop_media_files_liked_columns_v1.
func migratedSharedSocialDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shared_social.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Les steps shared_social sont title-owned (b19/b24) : sans provider, la
	// chaîne ne contiendrait ni create_base_shared_social_schema ni le DROP.
	// SetTitleStepsProvider est idempotent et purement additif — on ne le
	// restaure pas (aucun autre test d'ops ne dépend d'un provider nil).
	migration.SetTitleStepsProvider(halomigrations.StepsFor)

	if err := migration.RunForDB(db, migration.TargetSharedSocial); err != nil {
		t.Fatalf("RunForDB(shared_social): %v", err)
	}
	return db
}

// TestEnsureMediaTables_DoesNotResurrectLikedColumns — LE garde-rail du lot.
//
// (a) DB créée par les migrations → (b) le DROP y est passé → (c) on exécute
// ensureMediaTables (1re instruction d'IndexMedia) → (d) la colonne ne doit PAS
// être revenue. Un ensureMediaTables idempotent est rejoué à chaque scan : on le
// lance deux fois pour couvrir aussi le chemin « table déjà existante ».
func TestEnsureMediaTables_DoesNotResurrectLikedColumns(t *testing.T) {
	ctx := context.Background()
	db := migratedSharedSocialDB(t)

	// (b) Post-migrations : les colonnes sont déjà absentes.
	cols := mediaFilesColumns(t, db)
	if len(cols) == 0 {
		t.Fatal("media_files absente après migrations — la chaîne shared_social n'a pas tourné")
	}
	for _, col := range []string{"liked", "liked_at"} {
		if cols[col] {
			t.Fatalf("media_files.%s présente APRÈS migrations : "+
				"drop_media_files_liked_columns_v1 n'a pas été appliqué, ou une DDL la recrée", col)
		}
	}

	// (c) Le vecteur de résurrection : la boucle d'indexation.
	for i := 1; i <= 2; i++ {
		if err := ensureMediaTables(ctx, db); err != nil {
			t.Fatalf("ensureMediaTables #%d: %v", i, err)
		}
	}

	// (d) Verdict.
	after := mediaFilesColumns(t, db)
	for _, col := range []string{"liked", "liked_at"} {
		if after[col] {
			t.Errorf("RÉGRESSION : media_files.%s est RÉAPPARUE après ensureMediaTables. "+
				"C'est exactement ce qui avait fait reporter le DROP : retirer la colonne du "+
				"CREATE TABLE ET de la liste ADD COLUMN d'ensureMediaTables (internal/ops/media_store.go), "+
				"sans quoi chaque IndexMedia la recrée sur toutes les DB de prod.", col)
		}
	}

	// Contrôle négatif : le test mordrait-il ? Une colonne bien présente doit
	// l'être — sinon l'assertion ci-dessus passerait pour une mauvaise raison
	// (table vide, mauvais schéma, requête information_schema muette).
	if !after["discord_notified"] {
		t.Error("discord_notified absente : la sonde information_schema ne voit pas le vrai schéma, " +
			"l'assertion sur liked/liked_at ne prouve donc rien")
	}
}

// TestDropMediaFilesLikedColumns_Idempotent — rejouer la chaîne de migrations
// sur la même DB (cas d'un boot suivant, ou d'un step re-tenté) reste un no-op :
// le DROP COLUMN IF EXISTS ne doit pas échouer sur des colonnes déjà absentes.
func TestDropMediaFilesLikedColumns_Idempotent(t *testing.T) {
	db := migratedSharedSocialDB(t)

	if err := migration.RunForDB(db, migration.TargetSharedSocial); err != nil {
		t.Fatalf("2e RunForDB (idempotence): %v", err)
	}
	cols := mediaFilesColumns(t, db)
	for _, col := range []string{"liked", "liked_at"} {
		if cols[col] {
			t.Errorf("media_files.%s présente après 2e passage des migrations", col)
		}
	}
}

// TestDropMediaFilesLikedColumns_LegacyDB — le cas qui compte en prod : une DB
// ANTÉRIEURE, dont media_files porte encore liked/liked_at avec des données.
// Le step doit les retirer sans toucher aux autres colonnes ni aux lignes.
func TestDropMediaFilesLikedColumns_LegacyDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared_social.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Schéma legacy : media_files AVEC les deux colonnes et deux lignes likées.
	if _, err := db.Exec(`
		CREATE TABLE media_files (
			id VARCHAR PRIMARY KEY,
			player_slug VARCHAR,
			file_path VARCHAR,
			file_name VARCHAR,
			kind VARCHAR,
			liked BOOLEAN DEFAULT FALSE,
			liked_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		INSERT INTO media_files (id, player_slug, file_path, file_name, kind, liked, liked_at)
		VALUES ('m1', 'alice', 'alice/a.mp4', 'a.mp4', 'video', TRUE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)),
		       ('m2', 'bob',   'bob/b.mp4',   'b.mp4', 'video', FALSE, NULL);
	`); err != nil {
		t.Fatalf("seed legacy: %v", err)
	}

	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetSharedSocial); err != nil {
		t.Fatalf("RunForDB sur DB legacy: %v", err)
	}

	cols := mediaFilesColumns(t, db)
	for _, col := range []string{"liked", "liked_at"} {
		if cols[col] {
			t.Errorf("media_files.%s survit sur DB legacy — le DROP n'a pas été appliqué", col)
		}
	}
	for _, col := range []string{"id", "player_slug", "file_path", "file_name", "kind"} {
		if !cols[col] {
			t.Errorf("colonne %s perdue : le retrait a débordé de son périmètre", col)
		}
	}
	var rows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM media_files`).Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 2 {
		t.Errorf("rows = %d, want 2 (le DROP COLUMN ne doit perdre aucune ligne)", rows)
	}

	// LA propriété que le premier jet du step avait ratée : DuckDB refuse
	// « DROP COLUMN » tant qu'un index porte sur une colonne d'ordinal SUPÉRIEUR
	// (« an index depends on a column after it! »). Le step démonte donc les index
	// avant de droper — et DOIT les remonter. Les perdre en silence serait une
	// régression de perf invisible sur la galerie.
	idx := mediaFilesIndexNames(t, db)
	for _, name := range []string{"idx_mf_player_slug", "idx_mf_created", "idx_mf_player_stem"} {
		if !idx[name] {
			t.Errorf("index %s absent après le DROP : le step l'a démonté sans le remonter", name)
		}
	}
}
