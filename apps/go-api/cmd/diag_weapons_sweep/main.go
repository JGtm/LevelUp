// diag_weapons_sweep — état brut weapon_kills pour les N derniers matchs.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"levelup/go-api/internal/analysis"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	path := "../../../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"
	if env := os.Getenv("DIAG_SHARED_DB"); env != "" {
		path = env
	}
	fmt.Printf("[shared] %s\n\n", path)
	db, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== Inventaire weapon_kills — 50 matchs les plus récents ===")
	fmt.Println()
	fmt.Printf("%-10s %-17s %-15s %-7s %-7s %-7s %-7s\n",
		"match_id8", "start_time", "category", "wk", "kills", "bit21", "bit22")

	// Détecte si la colonne start_time_utc existe (schéma local) ou non (VPS legacy).
	var hasUTC int
	_ = db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name='match_registry' AND column_name='start_time_utc'`).Scan(&hasUTC)
	stExpr := "mr.start_time"
	if hasUTC > 0 {
		stExpr = analysis.SQLStartTimeCanonical("mr")
	}

	q := fmt.Sprintf(`
		SELECT mr.match_id,
		       %s AS st,
		       COALESCE(mr.pair_name, '') AS pair,
		       COALESCE(mr.backfill_completed, 0) AS bits,
		       (SELECT COUNT(*) FROM weapon_kills wk WHERE wk.match_id = mr.match_id) AS wk,
		       (SELECT COUNT(*) FROM highlight_events he
		         WHERE he.match_id = mr.match_id
		           AND LOWER(COALESCE(he.event_type,'')) LIKE '%%kill%%') AS kills
		FROM match_registry mr
		WHERE COALESCE(mr.is_firefight, FALSE) = FALSE
		ORDER BY st DESC LIMIT 50`, stExpr)
	rows, err := db.Query(q)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	totalRows := 0
	withWK := 0
	bit21Set := 0
	bit21SetButEmpty := 0
	for rows.Next() {
		var mid, pair string
		var st sql.NullTime
		var bits int64
		var wk, kills int
		if err := rows.Scan(&mid, &st, &pair, &bits, &wk, &kills); err != nil {
			continue
		}
		totalRows++
		stStr := ""
		if st.Valid {
			stStr = st.Time.Format("2006-01-02 15:04")
		}
		hasBit21 := bits&(1<<21) != 0
		hasBit22 := bits&(1<<22) != 0
		bit21Mark := "-"
		if hasBit21 {
			bit21Mark = "✓"
			bit21Set++
			if wk == 0 {
				bit21SetButEmpty++
			}
		}
		bit22Mark := "-"
		if hasBit22 {
			bit22Mark = "✓"
		}
		if wk > 0 {
			withWK++
		}
		short := pair
		if len(short) > 14 {
			short = short[:14]
		}
		fmt.Printf("%-10s %-17s %-15s %-7d %-7d %-7s %-7s\n",
			mid[:8], stStr, short, wk, kills, bit21Mark, bit22Mark)
	}
	fmt.Println()

	// Totaux globaux
	var totalMatches int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE COALESCE(is_firefight, FALSE) = FALSE`).Scan(&totalMatches)
	var globalWithWK int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_registry mr
		WHERE COALESCE(is_firefight, FALSE) = FALSE
		  AND EXISTS (SELECT 1 FROM weapon_kills wk WHERE wk.match_id = mr.match_id)`).Scan(&globalWithWK)
	var globalBit21 int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_registry mr
		WHERE COALESCE(is_firefight, FALSE) = FALSE
		  AND (COALESCE(backfill_completed,0) & ?) != 0`, 1<<21).Scan(&globalBit21)
	var globalBit21Empty int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_registry mr
		WHERE COALESCE(is_firefight, FALSE) = FALSE
		  AND (COALESCE(backfill_completed,0) & ?) != 0
		  AND NOT EXISTS (SELECT 1 FROM weapon_kills wk WHERE wk.match_id = mr.match_id)`, 1<<21).Scan(&globalBit21Empty)

	fmt.Printf("\n=== GLOBAL (matchs non-firefight) ===\n")
	fmt.Printf("  Total matchs        : %d\n", totalMatches)
	fmt.Printf("  Avec weapon_kills   : %d (%.1f%%)\n", globalWithWK, 100.0*float64(globalWithWK)/float64(totalMatches))
	fmt.Printf("  bit21 set           : %d\n", globalBit21)
	fmt.Printf("  bit21 set MAIS vide : %d (faux positifs)\n", globalBit21Empty)

	fmt.Printf("\n=== Échantillon affiché (50 derniers) ===\n")
	fmt.Printf("  Matchs avec wk      : %d / %d\n", withWK, totalRows)
	fmt.Printf("  bit21 set           : %d\n", bit21Set)
	fmt.Printf("  bit21 set MAIS vide : %d\n", bit21SetButEmpty)
}
