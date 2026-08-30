//go:build cgo

// shared_kill_positions_appendonly_test.go — G.2 (2026-08-30) : kill_positions bascule
// append-only (id PK + written_at + vue kill_positions_latest), même recette que ses deux
// sœurs de steps_appendonly_misc.go (match_csrs, pve_match_stats). Le mécanisme GÉNÉRIQUE du
// swap (transaction, garde anti-perte, recoverOrphan, idempotence) est déjà verrouillé par
// internal/migration/append_only_rebuild_test.go — ces tests verrouillent la SPEC de CETTE
// table : la clé fonctionnelle (match_id, killer_xuid, time_ms) — PAS de victim_xuid, cf.
// steps_shared_kill_positions.go — et la préservation des lignes Halo 5 déjà en prod.
package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// setupKillPositionsSharedDB monte une DB shared via la chaîne de migration RÉELLE
// (provider title-owned câblé, canonicalOrder respecté) : kill_positions créée PUIS convertie
// append-only dans le même run — c'est la preuve que shared_append_only_kill_positions_v1 est
// bien positionnée APRÈS shared_create_kill_positions dans internal/migration/order.go (une
// inversion ferait échouer ce test : la conversion no-operait sur une table absente et ne
// serait jamais rejouée).
func setupKillPositionsSharedDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(TargetShared): %v", err)
	}
	return db
}

// TestKillPositions_FreshSharedDB_IsAppendOnly : garde anti-régression — une DB shared NEUVE
// doit produire kill_positions au schéma append-only (id/written_at) ET la vue
// kill_positions_latest, dans le MÊME run de migration (ordre canonicalOrder correct).
func TestKillPositions_FreshSharedDB_IsAppendOnly(t *testing.T) {
	db := setupKillPositionsSharedDB(t)

	for _, col := range []string{"id", "written_at"} {
		has, err := migration.ColumnExists(db, "kill_positions", col)
		if err != nil {
			t.Fatalf("ColumnExists(%s): %v", col, err)
		}
		if !has {
			t.Fatalf("kill_positions.%s absente — DB neuve pas convertie append-only (régression d'ordre ?)", col)
		}
	}
	has, err := migration.TableExists(db, "kill_positions_latest")
	if err != nil {
		t.Fatalf("TableExists(kill_positions_latest): %v", err)
	}
	if !has {
		t.Fatal("vue kill_positions_latest absente")
	}
}

// TestKillPositions_AppendOnlyPreservesExistingH5Rows : le cas de production réel — la table
// porte DÉJÀ des lignes Halo 5 (schéma legacy, sans id/written_at) au moment où la conversion
// s'applique. Le swap CTAS doit les préserver À L'IDENTIQUE (garde rebuilt==before du helper),
// jamais les perdre ni les dupliquer.
func TestKillPositions_AppendOnlyPreservesExistingH5Rows(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Schéma LEGACY exact de steps_shared_kill_positions.go (avant conversion) — H5 y écrit
	// depuis longtemps, cf. ingest.MapKillPositions.
	if _, err := db.Exec(`
		CREATE TABLE kill_positions (
			match_id    VARCHAR NOT NULL,
			killer_xuid VARCHAR,
			time_ms     INTEGER,
			killer_x    DOUBLE, killer_y DOUBLE, killer_z DOUBLE,
			victim_x    DOUBLE, victim_y DOUBLE, victim_z DOUBLE
		)`); err != nil {
		t.Fatalf("create legacy: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO kill_positions (match_id, killer_xuid, time_ms, killer_x, killer_y, killer_z, victim_x, victim_y, victim_z)
		VALUES
			('h5m1', 'K1', 1000, 10.0, 20.0, 0.0, 15.0, 25.0, 0.0),
			('h5m1', 'K2', 2000, 30.0, 40.0, 0.0, NULL, NULL, NULL)`); err != nil {
		t.Fatalf("seed legacy H5 rows: %v", err)
	}

	if err := applyAppendOnlyKillPositions(db); err != nil {
		t.Fatalf("applyAppendOnlyKillPositions: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Fatalf("kill_positions après swap = %d lignes, attendu 2 (les 2 lignes H5 préexistantes)", n)
	}
	var latestN int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions_latest`).Scan(&latestN); err != nil {
		t.Fatalf("count latest: %v", err)
	}
	if latestN != 2 {
		t.Fatalf("kill_positions_latest = %d lignes, attendu 2", latestN)
	}
	// Les coordonnées H5 elles-mêmes doivent être intactes après le swap (pas seulement le
	// compte de lignes).
	var kx float64
	if err := db.QueryRow(`SELECT killer_x FROM kill_positions_latest WHERE match_id='h5m1' AND killer_xuid='K1' AND time_ms=1000`).Scan(&kx); err != nil {
		t.Fatalf("select preserved row: %v", err)
	}
	if kx != 10.0 {
		t.Errorf("killer_x = %v, attendu 10.0 (donnée H5 altérée par le swap)", kx)
	}

	// Idempotence : 2e passe = no-op (marqueur id déjà présent), aucune perte ni erreur.
	if err := applyAppendOnlyKillPositions(db); err != nil {
		t.Fatalf("2e passe (idempotence): %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions`).Scan(&n); err != nil {
		t.Fatalf("count après 2e passe: %v", err)
	}
	if n != 2 {
		t.Fatalf("kill_positions après 2e passe = %d, attendu 2 (idempotence)", n)
	}
}

// TestKillPositions_LatestViewDedupesReDecode : LE défaut que G.2 corrige. Un re-décodage du
// même match (nouvelle passe, decoder_rev bumpé) écrit une SECONDE ligne pour la même clé
// (match_id, killer_xuid, time_ms) — append-only, jamais un UPDATE. La vue _latest doit
// EXPOSER LA PLUS RÉCENTE, jamais les deux (ce qui doublerait silencieusement les positions
// lues, le défaut mesuré à 46,8 % sur killer_victim_pairs avant sa propre conversion). La
// table brute, elle, garde les deux (append-only = zéro perte).
func TestKillPositions_LatestViewDedupesReDecode(t *testing.T) {
	db := setupKillPositionsSharedDB(t)

	// Première "passe" (position mesurée initiale).
	if _, err := db.Exec(`
		INSERT INTO kill_positions (match_id, killer_xuid, time_ms, killer_x, killer_y, killer_z, written_at)
		VALUES ('m1', 'K1', 1000, 1.0, 1.0, 0.0, TIMESTAMP '2026-08-01 00:00:00')`); err != nil {
		t.Fatalf("insert pass 1: %v", err)
	}
	// Re-décodage (decoder_rev bumpé) : MÊME clé fonctionnelle, coordonnées affinées,
	// written_at postérieur — jamais un UPDATE/DELETE de la première ligne.
	if _, err := db.Exec(`
		INSERT INTO kill_positions (match_id, killer_xuid, time_ms, killer_x, killer_y, killer_z, written_at)
		VALUES ('m1', 'K1', 1000, 1.5, 1.5, 0.0, TIMESTAMP '2026-08-15 00:00:00')`); err != nil {
		t.Fatalf("insert pass 2 (re-decode): %v", err)
	}

	var rawN int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions WHERE match_id='m1' AND killer_xuid='K1' AND time_ms=1000`).Scan(&rawN); err != nil {
		t.Fatalf("count raw: %v", err)
	}
	if rawN != 2 {
		t.Fatalf("table brute = %d lignes pour la clé, attendu 2 (append-only : les deux passes coexistent)", rawN)
	}

	var latestN int
	var kx float64
	if err := db.QueryRow(
		`SELECT COUNT(*), MAX(killer_x) FROM kill_positions_latest WHERE match_id='m1' AND killer_xuid='K1' AND time_ms=1000`,
	).Scan(&latestN, &kx); err != nil {
		t.Fatalf("count/select latest: %v", err)
	}
	if latestN != 1 {
		t.Fatalf("kill_positions_latest = %d lignes pour la clé, attendu 1 (dédoublonnage par written_at DESC)", latestN)
	}
	if kx != 1.5 {
		t.Errorf("kill_positions_latest.killer_x = %v, attendu 1.5 (la passe la PLUS RÉCENTE, pas la première)", kx)
	}
}
