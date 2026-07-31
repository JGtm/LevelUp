// tmp_mapname : resout le nom de carte des films en cache (prefixe match_id).
package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if _, err := db.Exec(`ATTACH 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb' AS s (READ_ONLY)`); err != nil {
		panic(err)
	}
	rows, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name='match_registry'`)
	if err != nil {
		panic(err)
	}
	var cols []string
	for rows.Next() {
		var c string
		_ = rows.Scan(&c)
		cols = append(cols, c)
	}
	rows.Close()
	fmt.Println("colonnes match_registry:", strings.Join(cols, ","))
	mapCol := ""
	for _, c := range cols {
		if strings.EqualFold(c, "map_name") {
			mapCol = c
			break
		}
	}
	if mapCol == "" {
		return
	}
	q := fmt.Sprintf(`SELECT substr(match_id,1,8), any_value(%s) FROM s.match_registry GROUP BY 1`, mapCol)
	rows2, err := db.Query(q)
	if err != nil {
		panic(err)
	}
	defer rows2.Close()
	f, _ := os.Create(`C:/Users/GUILLA~1/AppData/Local/Temp/claude/C--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration--claude-worktrees-filmdec-continuation/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad/mapnames.csv`)
	defer f.Close()
	for rows2.Next() {
		var id string
		var mp sql.NullString
		_ = rows2.Scan(&id, &mp)
		fmt.Fprintf(f, "%s,%s\n", id, mp.String)
	}
	fmt.Println("ecrit mapnames.csv")
}
