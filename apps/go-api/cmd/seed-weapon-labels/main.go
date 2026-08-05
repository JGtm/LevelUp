//go:build cgo

// cmd/seed-weapon-labels — repare metadata.weapon_labels quand la table est
// vide alors que la migration `add_weapon_labels` est marquee done dans
// schema_migrations (cas observe sur les .duckdb prebuilts ou apres reset
// partiel de la metadata).
//
// Usage : go run -tags cgo ./cmd/seed-weapon-labels
//
// Idempotent : INSERT OR IGNORE — n'ecrase pas les rows deja en place.
//
// IMPORTANT : metadata.duckdb doit etre librable, donc stopper le serveur API
// avant de lancer (le pool DuckDB ouvre metadata.duckdb en RW au boot).
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/weapons"
)

func main() {
	metaPath := filepath.Join("..", "..", "data", "titles", "halo_infinite", "warehouse", "metadata.duckdb")
	if _, err := os.Stat(metaPath); err != nil {
		log.Fatalf("metadata.duckdb introuvable %s: %v", metaPath, err)
	}

	db, err := sql.Open("duckdb", metaPath)
	if err != nil {
		log.Fatalf("open %s: %v", metaPath, err)
	}
	defer db.Close()

	var before int
	if err := db.QueryRow("SELECT COUNT(*) FROM weapon_labels").Scan(&before); err != nil {
		// Table absente — on continue, ApplyLabels la cree.
		fmt.Printf("Avant : table absente ou erreur (%v)\n", err)
		before = -1
	} else {
		fmt.Printf("Avant : %d rows dans weapon_labels\n", before)
	}

	if err := weapons.ApplyLabels(db); err != nil {
		log.Fatalf("ApplyLabels: %v", err)
	}

	var after int
	if err := db.QueryRow("SELECT COUNT(*) FROM weapon_labels").Scan(&after); err != nil {
		log.Fatalf("count after: %v", err)
	}
	fmt.Printf("Apres : %d rows\n", after)
	if before >= 0 && after > before {
		fmt.Printf("Inserees : %d nouvelles entrees\n", after-before)
	}
}
