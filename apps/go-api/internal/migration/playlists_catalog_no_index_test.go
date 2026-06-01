//go:build cgo

// Package migration — playlists_catalog_no_index_test.go : garde-fou RC-E.
//
// playlists_catalog NE DOIT PAS avoir d'index secondaire. seedPlaylistsCatalog
// (sync/career.go) UPDATE les colonnes experience/is_active chaque cycle ; un
// UPDATE sur une colonne ART-indexée corrompt l'index DuckDB et FATAL-invalide
// metadata.duckdb (→ cascade shared read-only → complétion combat bloquée). La PK
// (title_slug, playlist_asset_id) n'est jamais touchée par ces UPDATE, donc son
// index reste sain ; seuls des index SECONDAIRES sur les colonnes mutées posent
// problème. Ce test échoue si quelqu'un réintroduit un tel index.
package migration

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestPlaylistsCatalog_NoSecondaryIndex(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("migrate metadata: %v", err)
	}

	rows, err := db.Query(`SELECT index_name FROM duckdb_indexes() WHERE table_name = 'playlists_catalog'`)
	if err != nil {
		t.Fatalf("query indexes: %v", err)
	}
	defer rows.Close()

	var offending []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		// DuckDB matérialise la PRIMARY KEY comme un index ART interne ; on ne le
		// compte pas (la PK n'est pas mutée par les UPDATE de seed). Seuls les index
		// nommés idx_* (créés via CREATE INDEX) sont la surface de corruption.
		if strings.HasPrefix(name, "idx_") {
			offending = append(offending, name)
		}
	}
	if len(offending) > 0 {
		t.Errorf("playlists_catalog a des index secondaires interdits %v — un UPDATE sur "+
			"colonne indexée corrompt metadata.duckdb (RC-E). Voir "+
			"steps_metadata_drop_playlists_catalog_indexes.go", offending)
	}
}
