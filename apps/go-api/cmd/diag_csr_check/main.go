//go:build cgo

// diag_csr_check — vérifie la présence de matchs classés CSR dans match_skill_rank.
// Read-only. Sortie texte console.
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"path/filepath"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

var players = []string{"Madina97294", "Chocoboflor", "JGtm", "XxDaemonGamerxX"}

func openDB(path string) *sql.DB {
	connector, err := duckdb.NewConnector(path+"?access_mode=READ_ONLY", func(execer driver.ExecerContext) error {
		return nil
	})
	if err != nil {
		log.Fatalf("open %s: %v", path, err)
	}
	return sql.OpenDB(connector)
}

func main() {
	dataRoot := "../../data"
	ctx := context.Background()

	fmt.Println("══ match_skill_rank : répartition LUSR vs CSR par joueur ══\n")

	for _, player := range players {
		playerPath := filepath.Join(dataRoot, "titles", "halo_infinite", "players", player, "stats.duckdb")
		db := openDB(playerPath)

		fmt.Printf("--- %s ---\n", player)

		rows, err := db.QueryContext(ctx, `
			SELECT
				COALESCE(rating_type, 'NULL') AS rating_type,
				COUNT(*) AS nb,
				MIN(rating_value) AS min_val,
				MAX(rating_value) AS max_val,
				ROUND(AVG(rating_value), 1) AS avg_val
			FROM match_skill_rank
			GROUP BY rating_type
			ORDER BY rating_type`)
		if err != nil {
			fmt.Printf("  erreur query: %v\n\n", err)
			db.Close()
			continue
		}

		anyRow := false
		for rows.Next() {
			anyRow = true
			var rtype string
			var nb int
			var minVal, maxVal, avgVal float64
			if err := rows.Scan(&rtype, &nb, &minVal, &maxVal, &avgVal); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("  %-8s  nb=%-5d  min=%-8.1f  max=%-8.1f  avg=%.1f\n",
				rtype, nb, minVal, maxVal, avgVal)
		}
		rows.Close()

		if !anyRow {
			fmt.Println("  (table vide)")
		}

		// Total
		var total int
		db.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_skill_rank").Scan(&total)
		fmt.Printf("  TOTAL: %d rows\n\n", total)

		// Quelques exemples de matchs CSR (si présents)
		sampleRows, err := db.QueryContext(ctx, `
			SELECT match_id, rating_type, rating_value
			FROM match_skill_rank
			WHERE rating_type = 'CSR'
			ORDER BY match_id
			LIMIT 5`)
		if err == nil {
			hasSample := false
			for sampleRows.Next() {
				if !hasSample {
					fmt.Printf("  Exemples CSR :\n")
					hasSample = true
				}
				var mid, rt string
				var rv float64
				sampleRows.Scan(&mid, &rt, &rv)
				fmt.Printf("    match_id=%s  rating_value=%.1f\n", mid, rv)
			}
			sampleRows.Close()
			if hasSample {
				fmt.Println()
			}
		}

		db.Close()
	}
}
