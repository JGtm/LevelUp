// Outil de diagnostic : dump la table spartan_identity de chaque player DB
// pour valider que le refactor §11 PLAN_SPARTAN_IDENTITY est opérationnel.
package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

func main() {
	playersDir := `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\titles\halo_infinite\players`
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open players dir: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-20s %-22s %-10s %-8s %-8s %-8s %-12s %s\n",
		"player", "xuid", "spartan_id", "banner", "emblem", "backdrop", "status", "last_refreshed_at")
	fmt.Println(strings.Repeat("-", 130))

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dbPath := filepath.Join(playersDir, e.Name(), "stats.duckdb")
		if _, err := os.Stat(dbPath); err != nil {
			continue
		}
		handle, err := duckdbpkg.OpenReadOnly(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open %s: %v\n", e.Name(), err)
			continue
		}
		db := handle.SQLDb()

		rows, err := db.Query(`SELECT xuid, spartan_id, banner_image_url, emblem_image_url, backdrop_image_url, last_attempt_status, last_refreshed_at FROM spartan_identity`)
		if err != nil {
			fmt.Printf("%-20s [no spartan_identity table or error: %v]\n", e.Name(), err)
			handle.Close()
			continue
		}
		count := 0
		for rows.Next() {
			var (
				xuid                                        string
				spartanID, banner, emblem, backdrop, status sql.NullString
				lastRefreshed                               sql.NullTime
			)
			if err := rows.Scan(&xuid, &spartanID, &banner, &emblem, &backdrop, &status, &lastRefreshed); err != nil {
				fmt.Printf("  scan: %v\n", err)
				continue
			}
			short := func(s sql.NullString) string {
				if !s.Valid || s.String == "" {
					return "-"
				}
				if len(s.String) > 7 {
					return "✓"
				}
				return s.String
			}
			ts := "(never)"
			if lastRefreshed.Valid {
				ts = lastRefreshed.Time.Format("2006-01-02 15:04:05")
			}
			fmt.Printf("%-20s %-22s %-10s %-8s %-8s %-8s %-12s %s\n",
				e.Name(), xuid,
				short(spartanID), short(banner), short(emblem), short(backdrop),
				strings.TrimSpace(status.String), ts)
			count++
		}
		rows.Close()
		if count == 0 {
			fmt.Printf("%-20s [table exists, 0 rows]\n", e.Name())
		}
		handle.Close()
	}
}
