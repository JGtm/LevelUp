//go:build cgo

// playlists_catalog_no_index_test.go — garde-fou RC-E déplacé depuis internal/migration
// (Phase 1.5 b12). playlists_catalog NE DOIT PAS avoir d'index secondaire : un UPDATE
// sur une colonne ART-indexée (experience/is_active, mutées chaque cycle par
// seedPlaylistsCatalog) corrompt l'index DuckDB et FATAL-invalide metadata.duckdb. Ce
// test câble le provider title-owned (StepsFor) car add_catalog_playlists +
// drop_playlists_catalog_secondary_indexes y vivent désormais.
package migrations

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func TestPlaylistsCatalog_NoSecondaryIndex(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
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
		// compte pas. Seuls les index nommés idx_* sont la surface de corruption.
		if strings.HasPrefix(name, "idx_") {
			offending = append(offending, name)
		}
	}
	if len(offending) > 0 {
		t.Errorf("playlists_catalog a des index secondaires interdits %v — un UPDATE sur "+
			"colonne indexée corrompt metadata.duckdb (RC-E). Voir le step "+
			"drop_playlists_catalog_secondary_indexes (steps.go)", offending)
	}
}
