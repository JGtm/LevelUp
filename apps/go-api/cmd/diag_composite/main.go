// cmd/diag_composite/main.go — diagnostic temporaire citations composites
package main

import (
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	meta, err := sql.Open("duckdb", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\titles\halo_infinite\warehouse\metadata.duckdb?access_mode=READ_ONLY`)
	if err != nil {
		fmt.Println("open meta err:", err)
		return
	}
	defer meta.Close()
	meta.SetMaxOpenConns(1)

	rows, _ := meta.Query(`SELECT citation_name_norm, composite_children, tier_targets FROM citation_mappings WHERE mapping_type='composite' ORDER BY citation_name_norm`)
	fmt.Println("=== COMPOSITES ===")
	for rows.Next() {
		var name string
		var children, tiers *string
		rows.Scan(&name, &children, &tiers)
		c := "<nil>"
		if children != nil {
			c = *children
		}
		t := "<nil>"
		if tiers != nil {
			t = *tiers
		}
		fmt.Printf("  %s | children=%s | tiers=%s\n", name, c, t)
	}
	rows.Close()

	fmt.Println("\n=== TIER_TARGETS des enfants non-composites ===")
	rows2, _ := meta.Query(`SELECT citation_name_norm, mapping_type, tier_targets FROM citation_mappings WHERE mapping_type NOT IN ('composite') AND tier_targets IS NOT NULL ORDER BY citation_name_norm`)
	for rows2.Next() {
		var name, mtype string
		var tiers *string
		rows2.Scan(&name, &mtype, &tiers)
		t := "<nil>"
		if tiers != nil {
			t = *tiers
		}
		fmt.Printf("  %s (%s) tiers=%s\n", name, mtype, t)
	}
	rows2.Close()

	for _, gt := range []string{"Madina97294", "Chocoboflor"} {
		path := fmt.Sprintf(`C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\titles\halo_infinite\players\%s\stats.duckdb?access_mode=READ_ONLY`, gt)
		pdb, _ := sql.Open("duckdb", path)
		pdb.SetMaxOpenConns(1)
		fmt.Printf("\n=== match_citations %s ===\n", gt)
		r, err := pdb.Query(`SELECT citation_name_norm, COUNT(*) as n, SUM(value) as total FROM match_citations GROUP BY citation_name_norm ORDER BY citation_name_norm LIMIT 30`)
		if err != nil {
			fmt.Printf("  err: %v\n", err)
			pdb.Close()
			continue
		}
		for r.Next() {
			var name string
			var n, total int
			r.Scan(&name, &n, &total)
			fmt.Printf("  %s: %d matchs, total=%d\n", name, n, total)
		}
		r.Close()
		pdb.Close()
	}
}
