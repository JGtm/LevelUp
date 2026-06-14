//go:build cgo

// diag_citation_counters — vérifie la persistance + le cumul des citations.
//
// Pour les 10 derniers matchs d'un joueur :
//  1. Vue persistance : par match, combien de citations (value>0) écrites,
//     somme des deltas, présence du sentinel "_processed", nb total de rows.
//     total_rows=0 => le match n'a AUCUNE ligne match_citations (non persisté).
//  2. Évolution compteur : pour la citation la plus active sur la période,
//     son delta par match + le compteur cumulé reconstruit (SUM(value) à vie).
//
// Usage : go run -tags cgo ./cmd/diag_citation_counters [gamertag]
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

const base = `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\titles\halo_infinite`

func main() {
	gt := "Madina97294"
	if len(os.Args) > 1 {
		gt = os.Args[1]
	}
	playerPath := fmt.Sprintf(`%s\players\%s\stats.duckdb?access_mode=READ_ONLY`, base, gt)
	sharedPath := fmt.Sprintf(`%s\warehouse\shared_matches_v2.duckdb`, base)

	db, err := sql.Open("duckdb", playerPath)
	if err != nil {
		fmt.Println("open player err:", err)
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, fmt.Sprintf("ATTACH '%s' AS shared (READ_ONLY)", sharedPath)); err != nil {
		fmt.Println("attach shared err:", err)
		os.Exit(1)
	}

	var totalRows int
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_citations").Scan(&totalRows)
	fmt.Printf("Joueur: %s | total rows match_citations (toute la base): %d\n", gt, totalRows)

	fmt.Println("\n=== 1. Persistance citations sur les 10 derniers matchs ===")
	fmt.Printf("%-3s  %-19s  %-9s  %-9s  %-9s  %-9s\n", "#", "start_time (UTC)", "cit>0", "Σdelta", "_proc", "rows")
	rows, err := db.QueryContext(ctx, `
WITH recent AS (
  SELECT pme.match_id AS mid, r.start_time AS st
  FROM player_match_enrichment pme
  JOIN shared.match_registry r ON r.match_id = pme.match_id
  ORDER BY r.start_time DESC
  LIMIT 10
)
SELECT
  recent.mid, recent.st,
  COALESCE(SUM(CASE WHEN mc.value > 0 THEN 1 ELSE 0 END), 0)                          AS cit_gained,
  COALESCE(SUM(CASE WHEN mc.value > 0 THEN mc.value ELSE 0 END), 0)                   AS sum_delta,
  COALESCE(SUM(CASE WHEN mc.citation_name_norm = '_processed' THEN 1 ELSE 0 END), 0)  AS sentinel,
  COUNT(mc.citation_name_norm)                                                        AS total_rows
FROM recent
LEFT JOIN match_citations mc ON mc.match_id = recent.mid
GROUP BY recent.mid, recent.st
ORDER BY recent.st DESC`)
	if err != nil {
		fmt.Println("query 1 err:", err)
		os.Exit(1)
	}
	i := 0
	for rows.Next() {
		i++
		var mid string
		var st time.Time
		var cit, sumDelta, sentinel, tot int
		_ = rows.Scan(&mid, &st, &cit, &sumDelta, &sentinel, &tot)
		flag := ""
		if tot == 0 {
			flag = "  <-- AUCUNE ligne (non persisté)"
		} else if cit == 0 && sentinel > 0 {
			flag = "  (traité, 0 citation)"
		}
		fmt.Printf("%-3d  %-19s  %-9d  %-9d  %-9d  %-9d%s\n",
			i, st.Format("2006-01-02 15:04:05"), cit, sumDelta, sentinel, tot, flag)
	}
	rows.Close()

	// 2. Citation la plus active sur les 10 derniers matchs.
	var topNorm string
	_ = db.QueryRowContext(ctx, `
WITH recent AS (
  SELECT pme.match_id AS mid
  FROM player_match_enrichment pme
  JOIN shared.match_registry r ON r.match_id = pme.match_id
  ORDER BY r.start_time DESC
  LIMIT 10
)
SELECT mc.citation_name_norm
FROM match_citations mc
JOIN recent ON recent.mid = mc.match_id
WHERE mc.value > 0
GROUP BY mc.citation_name_norm
ORDER BY SUM(mc.value) DESC
LIMIT 1`).Scan(&topNorm)

	if topNorm == "" {
		fmt.Println("\n=== 2. Aucune citation value>0 sur les 10 derniers matchs ===")
		return
	}

	fmt.Printf("\n=== 2. Évolution du compteur pour la citation la plus active : %q ===\n", topNorm)
	fmt.Println("(running = SUM(value) à vie jusqu'à ce match inclus = le compteur affiché)")
	fmt.Printf("%-3s  %-19s  %-8s  %-10s\n", "#", "start_time (UTC)", "delta", "compteur")
	r2, err := db.QueryContext(ctx, `
WITH per_match AS (
  SELECT r.start_time AS st, mc.value AS v,
         SUM(mc.value) OVER (ORDER BY r.start_time
             ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS running
  FROM match_citations mc
  JOIN shared.match_registry r ON r.match_id = mc.match_id
  WHERE mc.citation_name_norm = ? AND mc.value > 0
)
SELECT st, v, running FROM per_match ORDER BY st DESC LIMIT 10`, topNorm)
	if err != nil {
		fmt.Println("query 2 err:", err)
		os.Exit(1)
	}
	j := 0
	for r2.Next() {
		j++
		var st time.Time
		var v, running int
		_ = r2.Scan(&st, &v, &running)
		fmt.Printf("%-3d  %-19s  +%-7d  %-10d\n", j, st.Format("2006-01-02 15:04:05"), v, running)
	}
	r2.Close()

	var grand int
	_ = db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(value),0) FROM match_citations WHERE citation_name_norm = ?", topNorm).Scan(&grand)
	fmt.Printf("\nCompteur total à vie pour %q : %d\n", topNorm, grand)
}
