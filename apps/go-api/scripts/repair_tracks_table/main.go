// Script one-shot : supprime et recrée les tables Battle Pass corrompues
// (tracks + items), pour corriger les FatalException DuckDB par clé primaire dupliquée.
// Usage : cd apps/go-api && go run ./scripts/repair_tracks_table/
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	dbPath := "../../data/warehouse/metadata.duckdb"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	fmt.Printf("Ouverture de %s...\n", dbPath)
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		log.Fatalf("Impossible d'ouvrir la DB : %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Ping échoué : %v", err)
	}

	// Vérifier le nombre d'items avant
	var itemCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM battlepass_item_definitions").Scan(&itemCount)
	fmt.Printf("Items (avant) : %d\n", itemCount)

	// Vérifier le nombre de tracks avant
	var trackCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM battlepass_track_definitions").Scan(&trackCount)
	fmt.Printf("Tracks (avant) : %d\n", trackCount)

	// DROP de toutes les tables Battle Pass (tracks + items)
	steps := []string{
		"DROP INDEX IF EXISTS idx_battlepass_track_translations_lookup",
		"DROP INDEX IF EXISTS idx_battlepass_track_definitions_lookup",
		"DROP TABLE IF EXISTS battlepass_track_translations",
		"DROP TABLE IF EXISTS battlepass_track_definitions",
		"DROP INDEX IF EXISTS idx_battlepass_item_translations_lookup",
		"DROP INDEX IF EXISTS idx_battlepass_item_definitions_lookup",
		"DROP TABLE IF EXISTS battlepass_item_translations",
		"DROP TABLE IF EXISTS battlepass_item_definitions",
	}
	for _, q := range steps {
		fmt.Printf("Exécution : %s\n", q)
		if _, err := db.Exec(q); err != nil {
			log.Fatalf("Erreur : %v", err)
		}
	}

	// CHECKPOINT pour écrire proprement sur disque
	fmt.Println("CHECKPOINT...")
	if _, err := db.Exec("CHECKPOINT"); err != nil {
		log.Printf("CHECKPOINT warning: %v", err)
	}

	// Recréer les tables
	create := []string{
		`CREATE TABLE IF NOT EXISTS battlepass_track_definitions (
			reward_track_path VARCHAR NOT NULL, content_hash VARCHAR NOT NULL,
			xp_per_rank INTEGER, battlepass_image_path VARCHAR, background_image_path VARCHAR,
			raw_payload_json VARCHAR NOT NULL, first_seen_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			last_seen_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), is_current BOOLEAN NOT NULL DEFAULT TRUE,
			PRIMARY KEY (reward_track_path, content_hash)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_battlepass_track_definitions_lookup ON battlepass_track_definitions(reward_track_path, is_current)`,
		`CREATE TABLE IF NOT EXISTS battlepass_track_translations (
			reward_track_path VARCHAR NOT NULL, content_hash VARCHAR NOT NULL,
			lang VARCHAR NOT NULL, track_name VARCHAR,
			first_seen_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), last_seen_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (reward_track_path, content_hash, lang)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_battlepass_track_translations_lookup ON battlepass_track_translations(reward_track_path, lang)`,
		`CREATE TABLE IF NOT EXISTS battlepass_item_definitions (
			inventory_item_path VARCHAR NOT NULL, content_hash VARCHAR NOT NULL,
			quality VARCHAR, item_type VARCHAR, display_path VARCHAR,
			raw_payload_json VARCHAR NOT NULL, first_seen_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			last_seen_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), is_current BOOLEAN NOT NULL DEFAULT TRUE,
			PRIMARY KEY (inventory_item_path, content_hash)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_battlepass_item_definitions_lookup ON battlepass_item_definitions(inventory_item_path, is_current)`,
		`CREATE TABLE IF NOT EXISTS battlepass_item_translations (
			inventory_item_path VARCHAR NOT NULL, content_hash VARCHAR NOT NULL,
			lang VARCHAR NOT NULL, title VARCHAR, description VARCHAR,
			first_seen_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), last_seen_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (inventory_item_path, content_hash, lang)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_battlepass_item_translations_lookup ON battlepass_item_translations(inventory_item_path, lang)`,
	}
	for _, q := range create {
		fmt.Printf("Création : %s...\n", q[:50])
		if _, err := db.Exec(q); err != nil {
			log.Fatalf("Erreur création : %v", err)
		}
	}

	// CHECKPOINT final
	if _, err := db.Exec("CHECKPOINT"); err != nil {
		log.Printf("CHECKPOINT final warning: %v", err)
	}

	// Vérifier le résultat
	var newItemCount, newTrackCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM battlepass_item_definitions").Scan(&newItemCount)
	_ = db.QueryRow("SELECT COUNT(*) FROM battlepass_track_definitions").Scan(&newTrackCount)
	fmt.Printf("Items (après recréation) : %d (doit être 0)\n", newItemCount)
	fmt.Printf("Tracks (après recréation) : %d (doit être 0)\n", newTrackCount)

	fmt.Println("OK — tables tracks + items recréées proprement.")
	fmt.Println("Relancer ensuite :")
	fmt.Println("  go run ./scripts/import_bp_items/")
	fmt.Println("  go run ./scripts/import_bp_tracks/")
}
