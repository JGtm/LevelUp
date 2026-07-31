// tmp_mapmatch — trouve des matchs joués sur une map donnée (def catalyst) qui ont
// un film en cache, pour tester l'overlay positions-joueurs sur la carte 2D.
package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
const cacheDir = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`

func main() {
	want := "catalyst"
	if len(os.Args) > 1 {
		want = os.Args[1]
	}
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	// distribution des maps (pour repérer le nom exact).
	rows, _ := db.Query(`SELECT map_name, COUNT(*) c FROM match_registry GROUP BY map_name ORDER BY c DESC`)
	fmt.Println("=== maps (match_registry) ===")
	for rows.Next() {
		var mn sql.NullString
		var c int
		rows.Scan(&mn, &c)
		fmt.Printf("  %-30s %d\n", mn.String, c)
	}
	rows.Close()

	// matchs de la map voulue avec film en cache.
	q := `SELECT match_id, map_name FROM match_registry WHERE LOWER(map_name) LIKE '%' || ? || '%' ORDER BY match_id`
	rows, err = db.Query(q, strings.ToLower(want))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rows.Close()
	fmt.Printf("\n=== matchs '%s' avec film en cache ===\n", want)
	n := 0
	for rows.Next() {
		var id, mn sql.NullString
		rows.Scan(&id, &mn)
		short := id.String
		if len(short) > 8 {
			short = short[:8]
		}
		if _, err := os.Stat(filepath.Join(cacheDir, short, "chunk_00.bin")); err == nil {
			fmt.Printf("  %s  (%s)  [film cache OK]\n", id.String, mn.String)
			n++
		}
	}
	fmt.Printf("→ %d matchs %s avec film\n", n, want)
}
