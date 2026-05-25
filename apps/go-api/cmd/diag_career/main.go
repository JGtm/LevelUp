// Outil de diagnostic : affiche les dernières lignes career_progression pour un joueur.
package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

func main() {
	dbPath := `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\titles\halo_infinite\players\JGtm\stats.duckdb`
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	handle, err := duckdbpkg.OpenReadOnly(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer handle.Close()

	db := handle.SQLDb()

	// Dernières rows career_progression
	fmt.Println("=== career_progression (10 dernières) ===")
	rows, err := db.Query(`SELECT rank, rank_name, current_xp, recorded_at FROM career_progression ORDER BY recorded_at DESC LIMIT 10`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query career_progression: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%-6s %-30s %-8s %s\n", "rank", "rank_name", "xp", "recorded_at")
	fmt.Println(strings.Repeat("-", 72))
	for rows.Next() {
		var rank, xp int
		var name sql.NullString
		var ts string
		rows.Scan(&rank, &name, &xp, &ts)
		fmt.Printf("%-6d %-30s %-8d %s\n", rank, name.String, xp, ts)
	}
	rows.Close()

	// Vérifie ce que retourne la metadata pour le rank courant
	metaPath := `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\titles\halo_infinite\warehouse\metadata.duckdb`
	metaHandle, err := duckdbpkg.OpenReadOnly(metaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open metadata: %v\n", err)
		return
	}
	defer metaHandle.Close()
	metaDB := metaHandle.SQLDb()

	fmt.Println("\n=== metadata career_ranks autour du rang courant ===")
	metaRows, err := metaDB.Query(`SELECT rank_id, title_en, tier_type, grade, xp_required FROM career_ranks WHERE rank_id BETWEEN 177 AND 185 ORDER BY rank_id`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query metadata: %v\n", err)
		return
	}
	fmt.Printf("%-8s %-20s %-12s %-6s %s\n", "rank_id", "title_en", "tier_type", "grade", "xp_required")
	fmt.Println(strings.Repeat("-", 60))
	for metaRows.Next() {
		var id, xpReq int
		var title string
		var tier sql.NullString
		var grade sql.NullInt64
		metaRows.Scan(&id, &title, &tier, &grade, &xpReq)
		g := ""
		if grade.Valid {
			g = fmt.Sprintf("%d", grade.Int64)
		}
		fmt.Printf("%-8d %-20s %-12s %-6s %d\n", id, title, tier.String, g, xpReq)
	}
	metaRows.Close()
}
