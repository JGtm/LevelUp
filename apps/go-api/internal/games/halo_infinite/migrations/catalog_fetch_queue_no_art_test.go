//go:build cgo

// catalog_fetch_queue_no_art_test.go — garde-fou RC-E déplacé depuis
// internal/migration (campagne append-only ART #23046). catalog_fetch_queue est
// créée par le créateur de schéma metadata TITLE-OWNED (steps.go) ; ce test câble
// donc le provider title-owned (StepsFor) pour faire tourner RunForDB(TargetMetadata)
// bout-en-bout (sans le provider, la table n'est pas créée dans le binaire de test
// du package global internal/migration — cycle d'import).
//
// catalog_fetch_queue NE DOIT avoir AUCUNE surface ART (ni PRIMARY KEY ni index
// secondaire). Le drain catalogue (CatalogFetcherService) DELETE chaque ligne
// traitée (deleteFromQueue) et UPDATE attempts sur erreur (markError) ; un
// DELETE/UPDATE per-row sur une table ART-indexée déclenche le bug DuckDB 1.5.x
// (#23046) qui FATAL-invalide metadata.duckdb (→ noms d'assets cassés tout le
// reste de la vie du process). Ce test échoue si quelqu'un réintroduit une PK ou
// un index sur cette table. Voir le rebuild global
// rebuild_catalog_fetch_queue_drop_art_indexes (internal/migration) + le créateur
// title-owned (steps.go).
package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func TestCatalogFetchQueue_NoArtIndexSurface(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("migrate metadata: %v", err)
	}

	// 1. AUCUN index (ni secondaire idx_*, ni PK matérialisée en ART interne).
	var idxCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM duckdb_indexes() WHERE table_name = 'catalog_fetch_queue'`,
	).Scan(&idxCount); err != nil {
		t.Fatalf("query indexes: %v", err)
	}
	if idxCount != 0 {
		t.Errorf("catalog_fetch_queue a %d index — un DELETE/UPDATE per-row sur index ART "+
			"corrompt metadata.duckdb (RC-E catalog drain). Aucune surface ART autorisée.", idxCount)
	}

	// 2. AUCUNE PRIMARY KEY (surface ART du DELETE deleteFromQueue).
	var pkCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM duckdb_constraints()
		 WHERE table_name = 'catalog_fetch_queue' AND constraint_type = 'PRIMARY KEY'`,
	).Scan(&pkCount); err != nil {
		t.Fatalf("query constraints: %v", err)
	}
	if pkCount != 0 {
		t.Errorf("catalog_fetch_queue a une PRIMARY KEY — le DELETE de drain touche son index ART. " +
			"Dédup d'enqueue : SELECT-then-INSERT (NOT EXISTS), pas de PK.")
	}

	// 3. Fonctionnel : INSERT + UPDATE attempts + DELETE doivent réussir sans
	//    invalider la DB (reproduit la séquence exacte du drain).
	if _, err := db.Exec(
		`INSERT INTO catalog_fetch_queue (title_slug, asset_type, asset_id, version_id)
		 VALUES ('halo_infinite', 'playlist', 'asset-x', 'v1')`,
	); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := db.Exec(
		`UPDATE catalog_fetch_queue SET attempts = attempts + 1
		 WHERE title_slug = 'halo_infinite' AND asset_type = 'playlist' AND asset_id = 'asset-x'`,
	); err != nil {
		t.Fatalf("update attempts (markError simulé): %v", err)
	}
	if _, err := db.Exec(
		`DELETE FROM catalog_fetch_queue
		 WHERE title_slug = 'halo_infinite' AND asset_type = 'playlist' AND asset_id = 'asset-x'`,
	); err != nil {
		t.Fatalf("delete (deleteFromQueue simulé) a échoué — surface ART résiduelle ?: %v", err)
	}

	// DB toujours utilisable après la séquence.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM catalog_fetch_queue`).Scan(&n); err != nil {
		t.Fatalf("metadata.duckdb invalidée après DELETE: %v", err)
	}
	if n != 0 {
		t.Errorf("catalog_fetch_queue: attendu 0 après DELETE, obtenu %d", n)
	}
}
