//go:build cgo

// cleanup_post_art — supprime les matchs inseres apres 2026-05-05 23:59:59
// suite au bug ART DuckDB. Cascade sur shared_matches_v2.duckdb,
// shared_pve.duckdb et toutes les stats.duckdb joueurs.
//
// Tables touchees :
//
//	shared_matches_v2.duckdb : match_participants, medals_earned,
//	    highlight_events, killer_victim_pairs, weapon_kills, match_csrs,
//	    match_registry (en dernier)
//	shared_pve.duckdb        : pve_match_stats
//	{player}/stats.duckdb    : player_match_enrichment, match_skill_rank,
//	    personal_score_awards, match_citations, media_match_associations
//
//	Usage : go run -tags cgo ./cmd/cleanup_post_art \
//		    [--apply] [--data-root ../../data/titles/halo_infinite]
//
// Sans --apply : dry-run (DELETE + rollback). Avec --apply : commit.
// Le serveur Go DOIT etre stoppe (write lock DuckDB exclusif).
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
	"strings"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

const (
	tz     = "Europe/Paris"
	cutoff = "2026-05-05 23:59:59"
)

var (
	sharedTables = []string{
		"match_participants",
		"medals_earned",
		"highlight_events",
		"killer_victim_pairs",
		"weapon_kills",
		"match_csrs",
		"match_registry",
	}
	playerTables = []string{
		"player_match_enrichment",
		"match_skill_rank",
		"personal_score_awards",
		"match_citations",
		"media_match_associations",
	}
	pveTables = []string{
		"pve_match_stats",
	}
)

func main() {
	apply := flag.Bool("apply", false, "applique les suppressions (sinon dry-run)")
	dataRoot := flag.String("data-root", "../../data/titles/halo_infinite", "racine titre")
	flag.Parse()

	mode := "DRY-RUN (rollback)"
	if *apply {
		mode = "APPLY (commit)"
	}
	fmt.Printf("=== cleanup_post_art ===\n")
	fmt.Printf("cutoff   : > %s\n", cutoff)
	fmt.Printf("data_root: %s\n", *dataRoot)
	fmt.Printf("mode     : %s\n", mode)
	fmt.Println()

	sharedPath := filepath.Join(*dataRoot, "warehouse", "shared_matches_v2.duckdb")
	if _, err := os.Stat(sharedPath); err != nil {
		log.Fatalf("shared DB introuvable : %s", sharedPath)
	}

	matchIDs := collectMatchIDs(sharedPath)
	if len(matchIDs) == 0 {
		fmt.Println("Aucun match trouve apres le cutoff — rien a supprimer.")
		return
	}
	fmt.Printf("%d match(s) trouves apres %s :\n", len(matchIDs), cutoff)
	for _, id := range matchIDs {
		fmt.Printf("  %s\n", id)
	}
	fmt.Println()

	totalDeleted := 0

	fmt.Println("--- shared_matches_v2.duckdb ---")
	totalDeleted += cleanupDB(sharedPath, sharedTables, matchIDs, *apply, true)

	pvePath := filepath.Join(*dataRoot, "warehouse", "shared_pve.duckdb")
	if _, err := os.Stat(pvePath); err == nil {
		fmt.Println("--- shared_pve.duckdb ---")
		totalDeleted += cleanupDB(pvePath, pveTables, matchIDs, *apply, false)
	}

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
		totalDeleted += cleanupDB(statsPath, playerTables, matchIDs, *apply, false)
	}

	fmt.Println()
	label := "a supprimer (dry-run)"
	if *apply {
		label = "supprimees"
	}
	fmt.Printf("=== Total lignes %s : %d ===\n", label, totalDeleted)
	if !*apply {
		fmt.Println("Relance avec --apply pour confirmer.")
	}
}

// collectMatchIDs lit les match_ids a supprimer depuis shared (lecture seule).
func collectMatchIDs(sharedPath string) []string {
	connector, err := duckdb.NewConnector(sharedPath, func(execer driver.ExecerContext) error {
		_, e := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return e
	})
	if err != nil {
		log.Fatalf("connector(collect): %v", err)
	}
	db := sql.OpenDB(connector)
	defer db.Close()

	rows, err := db.QueryContext(context.Background(),
		"SELECT match_id FROM match_registry WHERE start_time > TIMESTAMP '"+cutoff+"' ORDER BY start_time",
	)
	if err != nil {
		log.Fatalf("collectMatchIDs: %v", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Fatalf("scan match_id: %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}

// cleanupDB ouvre la DB en RW, execute les DELETE en transaction, commit ou rollback.
// Si fatal=true, echec connecteur = log.Fatal (shared DBs). Sinon : warning (player DBs).
func cleanupDB(path string, tables, matchIDs []string, apply, fatal bool) int {
	connector, err := duckdb.NewConnector(path, func(execer driver.ExecerContext) error {
		_, e := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return e
	})
	if err != nil {
		if fatal {
			log.Fatalf("connector(%s): %v", path, err)
		}
		fmt.Printf("  ERREUR ouverture (DB ignoree) : %v\n", err)
		return 0
	}
	db := sql.OpenDB(connector)
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		if fatal {
			log.Fatalf("begin(%s): %v", path, err)
		}
		fmt.Printf("  ERREUR begin (DB ignoree) : %v\n", err)
		return 0
	}

	clause := buildInClause(matchIDs)
	total := 0
	for _, tbl := range tables {
		if !tableExists(tx, tbl) {
			fmt.Printf("  %-35s SKIP (table absente)\n", tbl)
			continue
		}
		res, err := tx.Exec("DELETE FROM " + tbl + " WHERE match_id IN " + clause)
		if err != nil {
			_ = tx.Rollback()
			fmt.Printf("  %-35s ERREUR : %v\n", tbl, err)
			return total
		}
		n, _ := res.RowsAffected()
		fmt.Printf("  %-35s %d ligne(s)\n", tbl, n)
		total += int(n)
	}

	if apply {
		if err := tx.Commit(); err != nil {
			log.Fatalf("commit(%s): %v", path, err)
		}
	} else {
		_ = tx.Rollback()
	}
	return total
}

// buildInClause construit le fragment SQL IN ('id1','id2',...).
// Les match_ids sont des UUIDs (hex + tirets), sans risque d'injection SQL.
func buildInClause(ids []string) string {
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = "'" + id + "'"
	}
	return "(" + strings.Join(quoted, ",") + ")"
}

func tableExists(tx *sql.Tx, name string) bool {
	var n int
	err := tx.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?",
		name,
	).Scan(&n)
	return err == nil && n > 0
}
