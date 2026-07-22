//go:build cgo

// purge_corrupt_records — purge one-off des PB corrompus (A5) : valeurs hors
// bornes de vraisemblance (accuracy « 7333 % », best_kda 107, …) ou métriques
// hors catalogue (best_kda legacy). Neutralisation = recette ADR 0026 (rebuild
// CTAS filtré + swap transactionnel dans internal/ops), JAMAIS de DELETE brut.
//
// Tables purgées :
//
//	{title}/warehouse/shared_social.duckdb : player_records_history (+ vue player_records_latest)
//	{title}/players/{gt}/stats.duckdb       : record_history
//
//	Usage : go run -tags cgo ./cmd/purge_corrupt_records \
//		    [--data-root ../../data/titles/halo_infinite] [--apply]
//
// Sans --apply : DRY-RUN (aucune mutation, comptage seulement). Avec --apply :
// reconstruction transactionnelle. Le serveur Go DOIT être stoppé (lock DuckDB
// exclusif — modèle mono-process writer).
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/ops"
)

func main() {
	dataRoot := flag.String("data-root", "../../data/titles/halo_infinite", "racine du titre (data/titles/{slug})")
	apply := flag.Bool("apply", false, "applique la purge (sinon dry-run, aucune mutation)")
	flag.Parse()

	mode := "DRY-RUN (aucune mutation)"
	if *apply {
		mode = "APPLY (reconstruction)"
	}
	fmt.Printf("=== purge_corrupt_records ===\n")
	fmt.Printf("data_root: %s\n", *dataRoot)
	fmt.Printf("mode     : %s\n\n", mode)

	ctx := context.Background()
	totalRemoved := 0

	// 1. shared_social.duckdb → player_records_history
	socialPath := filepath.Join(*dataRoot, "warehouse", "shared_social.duckdb")
	if _, err := os.Stat(socialPath); err != nil {
		log.Fatalf("shared_social introuvable : %s", socialPath)
	}
	fmt.Printf("--- shared_social.duckdb ---\n")
	totalRemoved += runPurge(ctx, socialPath, ops.PurgePlayerRecordsHistory, *apply)

	// 2. Per-player stats.duckdb → record_history
	playersDir := filepath.Join(*dataRoot, "players")
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		log.Fatalf("lecture playersDir: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		statsPath := filepath.Join(playersDir, e.Name(), "stats.duckdb")
		if _, statErr := os.Stat(statsPath); statErr != nil {
			continue
		}
		fmt.Printf("--- %s/stats.duckdb ---\n", e.Name())
		totalRemoved += runPurge(ctx, statsPath, ops.PurgeRecordHistory, *apply)
	}

	fmt.Println()
	verb := "à retirer"
	if *apply {
		verb = "retirées"
	}
	fmt.Printf("=== Total lignes corrompues %s : %d ===\n", verb, totalRemoved)
	if !*apply {
		fmt.Println("\nRelance avec --apply pour confirmer (serveur Go stoppé).")
	}
}

// runPurge ouvre la DB en RW, exécute la fonction de purge fournie et imprime le
// détail des lignes concernées. Retourne le nombre de lignes retirées/à retirer.
func runPurge(ctx context.Context, path string,
	purge func(context.Context, *sql.DB, bool) (ops.RecordsPurgeResult, error), apply bool) int {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		log.Fatalf("open %s: %v", path, err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	res, err := purge(ctx, db, apply)
	if err != nil {
		log.Fatalf("purge %s: %v", path, err)
	}
	if res.Removed == 0 {
		fmt.Printf("  %-24s aucune ligne corrompue\n", res.Table)
		return 0
	}
	fmt.Printf("  %-24s %d/%d ligne(s) corrompue(s)\n", res.Table, res.Removed, res.Before)
	for _, o := range res.Offenders {
		fmt.Printf("      %-20s value=%-14g x%d\n", o.Metric, o.Value, o.Count)
	}
	return res.Removed
}
