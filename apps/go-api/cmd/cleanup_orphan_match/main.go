//go:build cgo

// cleanup_orphan_match — supprime un match « dur » (cascade manuelle) à
// travers les DBs partagées et joueurs. Utilisé pour nettoyer les matchs
// orphelins qui polluent le bucket "(no session)" du chart Squad/Synergies
// et qui peuvent bloquer le sync delta s'ils sortent de leur fenêtre
// temporelle attendue.
//
// Tables touchées :
//
//	shared_matches_v2.duckdb : match_registry, match_participants,
//	    medals_earned, highlight_events, killer_victim_pairs, weapon_kills
//	{player}/stats.duckdb     : player_match_enrichment, match_skill_rank,
//	    personal_score_awards, match_citations, media_match_associations
//
// Pas touché : shared.xuid_aliases (alias global, partagé). Pas de cascade
// vers shared_pve.duckdb (pas Firefight ici).
//
//	Usage : go run -tags cgo ./cmd/cleanup_orphan_match \
//		    --match-id <uuid> [--apply] [--data-root ../../data/titles/halo_infinite]
//
// Sans --apply : dry-run, transactions rollback. Avec --apply : commit.
// Le serveur Go DOIT être stoppé (write lock DuckDB exclusif).
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

const tz = "Europe/Paris"

var (
	sharedTables = []string{
		"match_registry",
		"match_participants",
		"medals_earned",
		"highlight_events",
		"killer_victim_pairs",
		"weapon_kills",
	}
	playerTables = []string{
		"player_match_enrichment",
		"match_skill_rank",
		"personal_score_awards",
		"match_citations",
		"media_match_associations",
	}
)

func main() {
	matchID := flag.String("match-id", "", "match_id à supprimer (UUID, requis)")
	apply := flag.Bool("apply", false, "applique les suppressions (sinon dry-run)")
	dataRoot := flag.String("data-root", "../../data/titles/halo_infinite", "racine titre")
	flag.Parse()

	if *matchID == "" {
		log.Fatal("--match-id requis")
	}

	mode := "DRY-RUN (rollback)"
	if *apply {
		mode = "APPLY (commit)"
	}
	fmt.Printf("=== cleanup_orphan_match ===\n")
	fmt.Printf("match_id : %s\n", *matchID)
	fmt.Printf("data_root: %s\n", *dataRoot)
	fmt.Printf("mode     : %s\n", mode)
	fmt.Println()

	totalDeleted := 0

	// 1. shared_matches_v2.duckdb
	sharedPath := filepath.Join(*dataRoot, "warehouse", "shared_matches_v2.duckdb")
	if _, err := os.Stat(sharedPath); err != nil {
		log.Fatalf("shared DB introuvable : %s", sharedPath)
	}
	fmt.Printf("--- shared_matches_v2.duckdb ---\n")
	totalDeleted += cleanupDB(sharedPath, sharedTables, *matchID, *apply)

	// 2. Per-player stats.duckdb
	playersDir := filepath.Join(*dataRoot, "players")
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		log.Fatalf("lecture playersDir: %v", err)
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
		fmt.Printf("--- %s/stats.duckdb ---\n", gamertag)
		totalDeleted += cleanupDB(statsPath, playerTables, *matchID, *apply)
	}

	fmt.Println()
	fmt.Printf("=== Total lignes %s : %d ===\n",
		map[bool]string{true: "supprimées", false: "à supprimer"}[*apply], totalDeleted)
	if !*apply {
		fmt.Println("\nRelance avec --apply pour confirmer.")
	}
}

// cleanupDB ouvre la DB en RW, démarre une transaction, DELETE par table,
// puis COMMIT si apply=true, ROLLBACK sinon. Retourne le total de lignes
// affectées (informatif — toujours réel, même en dry-run, car le DELETE est
// exécuté avant le rollback).
func cleanupDB(path string, tables []string, matchID string, apply bool) int {
	connector, err := duckdb.NewConnector(path, func(execer driver.ExecerContext) error {
		_, e := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return e
	})
	if err != nil {
		log.Fatalf("connector(%s): %v", path, err)
	}
	db := sql.OpenDB(connector)
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("begin(%s): %v", path, err)
	}

	total := 0
	for _, tbl := range tables {
		if !tableExists(tx, tbl) {
			fmt.Printf("  %-30s SKIP (table absente)\n", tbl)
			continue
		}
		res, err := tx.Exec("DELETE FROM "+tbl+" WHERE match_id = ?", matchID)
		if err != nil {
			_ = tx.Rollback()
			log.Fatalf("DELETE FROM %s: %v", tbl, err)
		}
		n, _ := res.RowsAffected()
		fmt.Printf("  %-30s %d ligne(s)\n", tbl, n)
		total += int(n)
	}

	if apply {
		if err := tx.Commit(); err != nil {
			log.Fatalf("commit(%s): %v", path, err)
		}
	} else {
		if err := tx.Rollback(); err != nil {
			log.Fatalf("rollback(%s): %v", path, err)
		}
	}
	return total
}

func tableExists(tx *sql.Tx, name string) bool {
	var n int
	err := tx.QueryRow(
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`,
		name,
	).Scan(&n)
	if err != nil {
		return false
	}
	return n > 0
}
