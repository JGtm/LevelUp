//go:build cgo

// force_rebuild_art — outil CLI standalone qui force le rebuild swap CTAS
// de match_participants pour défaire une corruption d'index ART. Utilisé
// quand BootARTGuard est aveugle au bug (typiquement le bug DELETE-side
// "Invalid Input Error: Failed to delete all rows from index" qui n'est
// PAS détecté par la probe SELECT count).
//
// Étapes :
//   1. Ouvre shared_matches_v2.duckdb en RW (exclusivement — le serveur
//      doit être arrêté).
//   2. Efface le sentinel `match_participants_rebuilt_v1` dans sync_meta
//      (sinon la migration historique skip).
//   3. Appelle migration.RebuildMatchParticipantsART qui fait le swap CTAS
//      complet : PRAGMA table_info → CREATE TABLE _rebuilt AS SELECT
//      → DROP → RENAME → ADD PK → recreate views + indexes.
//   4. Compte les rows avant/après pour log.
//
// Idempotent : peut être rappelé sans risque. Pas de paramètre — pointe
// par défaut vers data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb
// depuis la racine du repo. Sortie texte console.
//
// **Pré-requis** : le binaire server.exe NE DOIT PAS être en cours
// d'exécution (sinon DuckDB refuse l'ouverture RW exclusive). Vérifie via
// `tasklist /FI "IMAGENAME eq server.exe"` avant.

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func main() {
	// --all : rebuild shared + tous les player DBs (extension 2026-05-23).
	all := flag("all", "", "rebuild shared + toutes les player DBs (data/titles/halo_infinite/players/*)")
	// --db : path explicite vers UNE DB shared.
	dbPath := flag("db",
		filepath.Join("data", "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb"),
		"chemin vers shared_matches_v2.duckdb (mode shared uniquement)")
	// --player-db : path explicite vers UN player stats.duckdb.
	playerDB := flag("player-db", "", "chemin vers un player stats.duckdb (mode player uniquement)")

	if all != "" && all != "false" {
		rebuildAll()
		return
	}

	if playerDB != "" {
		rebuildPlayerDB(playerDB)
		return
	}

	rebuildSharedDB(dbPath)
}

// rebuildSharedDB rebuild la table match_participants dans shared_matches_v2.duckdb.
func rebuildSharedDB(path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("abs path: %v", err)
	}
	if _, err := os.Stat(abs); err != nil {
		log.Fatalf("DB introuvable: %s (err: %v)", abs, err)
	}

	fmt.Printf("══ force_rebuild_art (shared) ══\n")
	fmt.Printf("DB cible : %s\n\n", abs)

	db, err := sql.Open("duckdb", abs)
	if err != nil {
		log.Fatalf("open RW: %v (serveur encore en cours ?)", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	ctx := context.Background()

	var rowsBefore int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_participants`).Scan(&rowsBefore); err != nil {
		log.Fatalf("count before: %v", err)
	}
	fmt.Printf("Rows pre-rebuild : %d\n", rowsBefore)

	if _, err := db.ExecContext(ctx,
		`DELETE FROM sync_meta WHERE key = 'match_participants_rebuilt_v1'`); err != nil {
		fmt.Printf("WARN: delete sentinel: %v (non-bloquant)\n", err)
	} else {
		fmt.Println("Sentinel match_participants_rebuilt_v1 effacé.")
	}

	fmt.Println("\nLancement rebuild swap CTAS...")
	start := time.Now()
	if err := migration.RebuildMatchParticipantsART(ctx, db); err != nil {
		log.Fatalf("RebuildMatchParticipantsART: %v", err)
	}
	dur := time.Since(start)

	var rowsAfter int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_participants`).Scan(&rowsAfter); err != nil {
		log.Fatalf("count after: %v", err)
	}

	fmt.Printf("\n══ Résultat shared ══\n")
	fmt.Printf("Rows avant : %d\n", rowsBefore)
	fmt.Printf("Rows après : %d\n", rowsAfter)
	fmt.Printf("Durée      : %v\n", dur)

	if rowsAfter != rowsBefore {
		fmt.Printf("\n⚠ ALERTE : row count différent ! Investigation requise.\n")
		os.Exit(1)
	}
	fmt.Println("\n✓ Rebuild shared terminé sans perte.")
}

// rebuildPlayerDB rebuild la table player_match_enrichment dans le stats.duckdb
// d'un joueur.
func rebuildPlayerDB(path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("abs path: %v", err)
	}
	if _, err := os.Stat(abs); err != nil {
		log.Fatalf("player DB introuvable: %s (err: %v)", abs, err)
	}

	fmt.Printf("══ force_rebuild_art (player) ══\n")
	fmt.Printf("Player DB : %s\n\n", abs)

	db, err := sql.Open("duckdb", abs)
	if err != nil {
		log.Fatalf("open RW: %v (serveur encore en cours ?)", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	ctx := context.Background()

	var rowsBefore int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&rowsBefore); err != nil {
		log.Fatalf("count before: %v", err)
	}
	fmt.Printf("Rows pre-rebuild : %d\n", rowsBefore)

	start := time.Now()
	if err := migration.RebuildPlayerMatchEnrichmentART(ctx, db); err != nil {
		log.Fatalf("RebuildPlayerMatchEnrichmentART: %v", err)
	}
	dur := time.Since(start)

	var rowsAfter int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&rowsAfter); err != nil {
		log.Fatalf("count after: %v", err)
	}

	fmt.Printf("\n══ Résultat player ══\n")
	fmt.Printf("Rows avant : %d\n", rowsBefore)
	fmt.Printf("Rows après : %d\n", rowsAfter)
	fmt.Printf("Durée      : %v\n", dur)

	if rowsAfter != rowsBefore {
		fmt.Printf("\n⚠ ALERTE : row count différent ! Investigation requise.\n")
		os.Exit(1)
	}
	fmt.Println("\n✓ Rebuild player (player_match_enrichment) terminé sans perte.")

	// Phase 4.5 follow-up 2026-05-24 : rebuild également match_skill_rank,
	// dont l'ART corruption bloque les DELETE batch LUSR (PostSyncLUSRPersister.Upsert).
	var msrBefore, msrAfter int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank`).Scan(&msrBefore); err != nil {
		// Table absente (player DB legacy sans LUSR) : silence.
		return
	}
	fmt.Printf("\nmatch_skill_rank rows pre-rebuild : %d\n", msrBefore)
	startMSR := time.Now()
	if err := migration.RebuildMatchSkillRankART(ctx, db); err != nil {
		log.Fatalf("RebuildMatchSkillRankART: %v", err)
	}
	durMSR := time.Since(startMSR)
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank`).Scan(&msrAfter); err != nil {
		log.Fatalf("count msr after: %v", err)
	}
	fmt.Printf("match_skill_rank rows : avant=%d après=%d durée=%v\n", msrBefore, msrAfter, durMSR)
	if msrAfter != msrBefore {
		fmt.Printf("\n⚠ ALERTE match_skill_rank : row count différent ! Investigation requise.\n")
		os.Exit(1)
	}
	fmt.Println("✓ Rebuild player (match_skill_rank) terminé sans perte.")
}

// rebuildAll itère shared puis tous les player DBs présents sous
// data/titles/halo_infinite/players/{gamertag}/stats.duckdb.
func rebuildAll() {
	fmt.Println("══ force_rebuild_art --all ══")

	// 1. Shared.
	rebuildSharedDB(filepath.Join("data", "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb"))

	// 2. Player DBs : itère data/titles/halo_infinite/players/*/stats.duckdb.
	playersDir := filepath.Join("data", "titles", "halo_infinite", "players")
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		log.Fatalf("read players dir %s: %v", playersDir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		playerDB := filepath.Join(playersDir, entry.Name(), "stats.duckdb")
		if _, err := os.Stat(playerDB); err != nil {
			fmt.Printf("\nSkip %s (pas de stats.duckdb)\n", entry.Name())
			continue
		}
		fmt.Printf("\n--- Player : %s ---\n", entry.Name())
		rebuildPlayerDB(playerDB)
	}

	fmt.Println("\n✓ Rebuild --all terminé.")
}

// flag : helper minimal pour args CLI sans dépendance externe.
func flag(name, defaultValue, _ string) string {
	for i, arg := range os.Args[1:] {
		if arg == "--"+name && i+1 < len(os.Args)-1 {
			return os.Args[i+2]
		}
	}
	return defaultValue
}
