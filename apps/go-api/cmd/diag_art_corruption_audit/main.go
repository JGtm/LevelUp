// cmd/diag_art_corruption_audit — Audit read-only de l'impact bug ART DuckDB
// sur shared_matches_v2.duckdb.
//
// Cherche les symptômes typiques de corruption ART :
//  1. Matchs dans match_registry sans aucune row participants
//  2. Matchs avec un nombre anormalement bas de participants (<4)
//  3. Distribution des participants par match (histogramme)
//  4. Focus sur la fenêtre d'incident 2026-05-22 19h33 (cycle FATAL ART)
//
// Read-only. Skip si shared DB absente.
//
// Usage : go run ./cmd/diag_art_corruption_audit
//
// Sortie : table résumé + JSON détaillé optionnel via -json
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

var (
	dataRoot  = flag.String("data", `../../data`, "Racine du dossier data/")
	titleSlug = flag.String("title", "halo_infinite", "Title slug")
)

func main() {
	flag.Parse()

	sharedPath := filepath.Join(*dataRoot, "titles", *titleSlug, "warehouse", "shared_matches_v2.duckdb")
	if _, err := os.Stat(sharedPath); err != nil {
		log.Fatalf("shared DB introuvable : %s", sharedPath)
	}

	db, err := sql.Open("duckdb", sharedPath+"?access_mode=read_only")
	if err != nil {
		log.Fatalf("ouverture %s : %v", sharedPath, err)
	}
	defer db.Close()

	fmt.Println("=== AUDIT ART CORRUPTION ===")
	fmt.Printf("DB : %s\n\n", sharedPath)

	// ─── 1. Totaux globaux ──────────────────────────────────────────────────
	var (
		totalRegistry        int
		totalParticipantRows int
		distinctMatchInPart  int
		distinctXuidInPart   int
	)
	must(db.QueryRow(`SELECT COUNT(*) FROM match_registry`).Scan(&totalRegistry))
	must(db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&totalParticipantRows))
	must(db.QueryRow(`SELECT COUNT(DISTINCT match_id) FROM match_participants`).Scan(&distinctMatchInPart))
	must(db.QueryRow(`SELECT COUNT(DISTINCT xuid) FROM match_participants`).Scan(&distinctXuidInPart))

	fmt.Println("## 1. Totaux")
	fmt.Printf("  match_registry rows                : %d\n", totalRegistry)
	fmt.Printf("  match_participants rows            : %d\n", totalParticipantRows)
	fmt.Printf("  match_participants distinct match  : %d\n", distinctMatchInPart)
	fmt.Printf("  match_participants distinct xuid   : %d\n", distinctXuidInPart)

	missing := totalRegistry - distinctMatchInPart
	fmt.Printf("\n  ➤ Registry sans participants : %d (%.2f%%)\n",
		missing, percent(missing, totalRegistry))
	fmt.Println()

	// ─── 2. Histogramme nb participants par match ──────────────────────────
	fmt.Println("## 2. Distribution nb participants par match (registry-side)")
	rows, err := db.Query(`
		SELECT participant_count, COUNT(*) AS n_matches
		FROM (
			SELECT r.match_id,
			       COALESCE((SELECT COUNT(*) FROM match_participants p WHERE p.match_id = r.match_id), 0) AS participant_count
			FROM match_registry r
		)
		GROUP BY participant_count
		ORDER BY participant_count ASC
	`)
	must(err)
	defer rows.Close()
	fmt.Printf("  %-6s  %-10s  %s\n", "count", "n_matches", "interpretation")
	fmt.Printf("  %s\n", "──────  ──────────  ────────────────────────────────────────")
	suspect := 0
	for rows.Next() {
		var pc, n int
		if err := rows.Scan(&pc, &n); err != nil {
			log.Fatal(err)
		}
		interp := classifyParticipantCount(pc)
		if pc < 4 {
			suspect += n
		}
		fmt.Printf("  %-6d  %-10d  %s\n", pc, n, interp)
	}
	fmt.Printf("\n  ➤ Matchs suspects (<4 participants) : %d (%.2f%%)\n",
		suspect, percent(suspect, totalRegistry))
	fmt.Println()

	// ─── 3. Matchs récents (incident window 2026-05-22) ────────────────────
	fmt.Println("## 3. Fenêtre incident ART (2026-05-22 19:00 → 2026-05-23 23:59)")
	var winRegistry, winMissing, winSuspect int
	must(db.QueryRow(`
		SELECT COUNT(*)
		FROM match_registry
		WHERE first_sync_at >= '2026-05-22 19:00:00'
		  AND first_sync_at <= '2026-05-23 23:59:59'
	`).Scan(&winRegistry))
	must(db.QueryRow(`
		SELECT COUNT(*)
		FROM match_registry r
		WHERE r.first_sync_at >= '2026-05-22 19:00:00'
		  AND r.first_sync_at <= '2026-05-23 23:59:59'
		  AND NOT EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = r.match_id)
	`).Scan(&winMissing))
	must(db.QueryRow(`
		SELECT COUNT(*)
		FROM match_registry r
		WHERE r.first_sync_at >= '2026-05-22 19:00:00'
		  AND r.first_sync_at <= '2026-05-23 23:59:59'
		  AND (SELECT COUNT(*) FROM match_participants p WHERE p.match_id = r.match_id) < 4
	`).Scan(&winSuspect))

	fmt.Printf("  match_registry dans fenêtre : %d\n", winRegistry)
	fmt.Printf("  → sans participants         : %d (%.2f%%)\n", winMissing, percent(winMissing, winRegistry))
	fmt.Printf("  → < 4 participants          : %d (%.2f%%)\n", winSuspect, percent(winSuspect, winRegistry))
	fmt.Println()

	// ─── 4. Top 10 matchs récents corrompus (sans participants) ────────────
	fmt.Println("## 4. Top 10 matchs récents avec 0 participants (à backfill)")
	mrows, err := db.Query(`
		SELECT r.match_id, r.first_sync_at, r.first_sync_by, r.playlist_name
		FROM match_registry r
		WHERE NOT EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = r.match_id)
		ORDER BY r.first_sync_at DESC
		LIMIT 10
	`)
	must(err)
	defer mrows.Close()
	for mrows.Next() {
		var mid, by string
		var ts sql.NullTime
		var pl sql.NullString
		if err := mrows.Scan(&mid, &ts, &by, &pl); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  %s  %s  by=%s  playlist=%s\n",
			mid, formatTime(ts), by, pl.String)
	}
	fmt.Println()

	// ─── 5. CSR backfill timeline (pour aider décision restore backup) ────
	fmt.Println("## 5. Repères temporels (pour décision restore backup)")
	var csrFirstSync, csrLastSync sql.NullTime
	_ = db.QueryRow(`
		SELECT MIN(first_sync_at), MAX(first_sync_at)
		FROM match_registry
		WHERE is_ranked = TRUE
	`).Scan(&csrFirstSync, &csrLastSync)
	fmt.Printf("  Première sync ranked (registry)   : %s\n", formatTime(csrFirstSync))
	fmt.Printf("  Dernière sync ranked (registry)   : %s\n", formatTime(csrLastSync))

	var matchCsrCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_csrs`).Scan(&matchCsrCount)
	fmt.Printf("  shared.match_csrs rows totales    : %d (CSR de tous participants des matchs ranked)\n", matchCsrCount)
	fmt.Println()

	fmt.Println("=== VERDICT ===")
	if missing == 0 && suspect == 0 {
		fmt.Println("  ✅ Aucune trace de corruption ART détectable (registry vs participants cohérent).")
	} else {
		fmt.Printf("  ⚠️  %d matchs sans participants + %d matchs <4 participants détectés.\n", missing, suspect)
		fmt.Println("     → Ces matchs peuvent être backfilled depuis l'API Halo (re-sync forcé).")
		fmt.Println("     → Pas besoin de restore un backup si le volume est gérable (<100 matchs).")
	}
}

func classifyParticipantCount(pc int) string {
	switch {
	case pc == 0:
		return "AUCUN participant — INSERT a échoué (ART corruption probable)"
	case pc < 4:
		return "anormalement bas — partial corruption ou mode firefight ?"
	case pc <= 16:
		return "normal (4v4 à 8v8 selon mode)"
	default:
		return "élevé (BTB ou autre mode large)"
	}
}

func percent(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return 100.0 * float64(n) / float64(total)
}

func formatTime(t sql.NullTime) string {
	if !t.Valid {
		return "(null)"
	}
	return t.Time.Format("2006-01-02 15:04:05")
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
