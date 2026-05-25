//go:build cgo

// migrate-roman-ranks — convertit les tier_label CSR/LUSR stockés avec chiffres
// arabes ("Or 3", "Diamant 6") en chiffres romains ("Or III", "Diamant VI").
//
// Tables mises à jour :
//   - shared_matches_v2.duckdb  : match_csrs.tier_label
//   - {player}/stats.duckdb     : match_skill_rank.tier_label
//
// Seules les lignes dont tier_label se termine par ' [1-6]' sont touchées
// (Onyx "1850", "En placement", valeurs nulles sont inchangées).
//
// Usage :
//
//	go run -tags cgo ./cmd/migrate-roman-ranks [--apply] [--data-root ../../data/titles/halo_infinite]
//
// Sans --apply : dry-run (SELECT COUNT + affichage, pas de modification).
// Avec --apply : UPDATE committé.
// Le serveur Go DOIT être arrêté (write lock DuckDB exclusif).
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
)

const updateSQL = `
UPDATE %s
SET tier_label = (
    CASE RIGHT(tier_label, 2)
        WHEN ' 1' THEN LEFT(tier_label, LENGTH(tier_label)-1) || 'I'
        WHEN ' 2' THEN LEFT(tier_label, LENGTH(tier_label)-1) || 'II'
        WHEN ' 3' THEN LEFT(tier_label, LENGTH(tier_label)-1) || 'III'
        WHEN ' 4' THEN LEFT(tier_label, LENGTH(tier_label)-1) || 'IV'
        WHEN ' 5' THEN LEFT(tier_label, LENGTH(tier_label)-1) || 'V'
        WHEN ' 6' THEN LEFT(tier_label, LENGTH(tier_label)-1) || 'VI'
        ELSE tier_label
    END
)
WHERE tier_label IS NOT NULL
  AND RIGHT(tier_label, 2) IN (' 1', ' 2', ' 3', ' 4', ' 5', ' 6')
`

const countSQL = `
SELECT COUNT(*) FROM %s
WHERE tier_label IS NOT NULL
  AND RIGHT(tier_label, 2) IN (' 1', ' 2', ' 3', ' 4', ' 5', ' 6')
`

func migrateDB(dbPath, table string, apply bool) (int64, error) {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", dbPath, err)
	}
	defer db.Close()

	if !apply {
		var n int64
		if err := db.QueryRow(fmt.Sprintf(countSQL, table)).Scan(&n); err != nil {
			return 0, fmt.Errorf("count %s.%s: %w", dbPath, table, err)
		}
		return n, nil
	}

	res, err := db.Exec(fmt.Sprintf(updateSQL, table))
	if err != nil {
		return 0, fmt.Errorf("update %s.%s: %w", dbPath, table, err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func main() {
	apply := flag.Bool("apply", false, "applique les mises à jour (sinon dry-run)")
	dataRoot := flag.String("data-root", "../../data/titles/halo_infinite", "racine titre halo_infinite")
	flag.Parse()

	mode := "DRY-RUN"
	if *apply {
		mode = "APPLY"
	}
	fmt.Printf("=== migrate-roman-ranks [%s] ===\n", mode)
	fmt.Printf("data-root : %s\n\n", *dataRoot)

	var totalAffected int64

	// --- shared_matches_v2.duckdb : match_csrs ---
	sharedPath := filepath.Join(*dataRoot, "warehouse", "shared_matches_v2.duckdb")
	if _, err := os.Stat(sharedPath); err == nil {
		n, err := migrateDB(sharedPath, "match_csrs", *apply)
		if err != nil {
			log.Printf("WARN shared match_csrs: %v", err)
		} else {
			verb := "à migrer"
			if *apply {
				verb = "migrées"
			}
			fmt.Printf("shared/match_csrs          : %d lignes %s\n", n, verb)
			totalAffected += n
		}
	} else {
		fmt.Printf("shared_matches_v2.duckdb   : introuvable — skip\n")
	}

	// --- {player}/stats.duckdb : match_skill_rank ---
	playersDir := filepath.Join(*dataRoot, "players")
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		log.Fatalf("lecture players/: %v", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gamertag := e.Name()
		statsPath := filepath.Join(playersDir, gamertag, "stats.duckdb")
		if _, err := os.Stat(statsPath); err != nil {
			continue
		}

		n, err := migrateDB(statsPath, "match_skill_rank", *apply)
		if err != nil {
			log.Printf("WARN %s/match_skill_rank: %v", gamertag, err)
			continue
		}
		verb := "à migrer"
		if *apply {
			verb = "migrées"
		}
		fmt.Printf("%-26s : %d lignes %s\n", gamertag+"/match_skill_rank", n, verb)
		totalAffected += n
	}

	fmt.Printf("\nTotal : %d lignes %s\n", totalAffected, func() string {
		if *apply {
			return "mises à jour"
		}
		return "à mettre à jour"
	}())
	if !*apply && totalAffected > 0 {
		fmt.Println("\nRelancez avec --apply pour appliquer.")
	}
}
