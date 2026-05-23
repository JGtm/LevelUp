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
	// Path par défaut (relatif au repo root). Override possible via --db.
	dbPath := flag("db",
		filepath.Join("data", "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb"),
		"chemin vers shared_matches_v2.duckdb")

	abs, err := filepath.Abs(dbPath)
	if err != nil {
		log.Fatalf("abs path: %v", err)
	}
	if _, err := os.Stat(abs); err != nil {
		log.Fatalf("DB introuvable: %s (err: %v)", abs, err)
	}

	fmt.Printf("══ force_rebuild_art ══\n")
	fmt.Printf("DB cible : %s\n\n", abs)

	// Ouvre en RW. DuckDB refusera si un autre process tient déjà le fichier.
	db, err := sql.Open("duckdb", abs)
	if err != nil {
		log.Fatalf("open RW: %v (serveur encore en cours ?)", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	ctx := context.Background()

	// 1. Diag pré-rebuild : count rows.
	var rowsBefore int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_participants`).Scan(&rowsBefore); err != nil {
		log.Fatalf("count before: %v", err)
	}
	fmt.Printf("Rows pre-rebuild : %d\n", rowsBefore)

	// 2. Efface le sentinel pour que la prochaine boot relance applyRebuild...
	// Mais on appelle RebuildMatchParticipantsART direct ici, donc le sentinel
	// n'est pas consulté. On l'efface quand même au cas où le serveur
	// référencerait ailleurs.
	if _, err := db.ExecContext(ctx,
		`DELETE FROM sync_meta WHERE key = 'match_participants_rebuilt_v1'`); err != nil {
		// Pas fatal — la table peut ne pas avoir cette key.
		fmt.Printf("WARN: delete sentinel: %v (non-bloquant)\n", err)
	} else {
		fmt.Println("Sentinel match_participants_rebuilt_v1 effacé.")
	}

	// 3. Rebuild swap CTAS.
	fmt.Println("\nLancement rebuild swap CTAS...")
	start := time.Now()
	if err := migration.RebuildMatchParticipantsART(ctx, db); err != nil {
		log.Fatalf("RebuildMatchParticipantsART: %v", err)
	}
	dur := time.Since(start)

	// 4. Validation post-rebuild.
	var rowsAfter int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_participants`).Scan(&rowsAfter); err != nil {
		log.Fatalf("count after: %v", err)
	}

	fmt.Printf("\n══ Résultat ══\n")
	fmt.Printf("Rows avant : %d\n", rowsBefore)
	fmt.Printf("Rows après : %d\n", rowsAfter)
	fmt.Printf("Durée      : %v\n", dur)

	if rowsAfter != rowsBefore {
		fmt.Printf("\n⚠ ALERTE : row count différent ! Investigation requise.\n")
		os.Exit(1)
	}
	fmt.Println("\n✓ Rebuild terminé sans perte.")
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
