// tmp_slayerfilm — liste les films locaux dont le match est un SLAYER. LECTURE SEULE.
// THROWAWAY. Croise data/cache/film_chunks/<prefixe> avec shared_matches_v2.match_registry.
//
// Usage : go run ./cmd/tmp_slayerfilm  (CGO requis pour DuckDB)
package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	filmDir = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
	dbPath  = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
)

func main() {
	ents, err := os.ReadDir(filmDir)
	if err != nil {
		panic(err)
	}
	prefixes := map[string]bool{}
	for _, e := range ents {
		if e.IsDir() {
			prefixes[strings.ToLower(e.Name())] = true
		}
	}
	fmt.Printf("films locaux : %d\n", len(prefixes))

	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		panic(err)
	}
	defer func() { _ = db.Close() }()

	rows, err := db.Query(`SELECT match_id, COALESCE(game_variant_name,''), COALESCE(map_name,''),
		COALESCE(playlist_name,'') FROM match_registry`)
	if err != nil {
		panic(err)
	}
	defer func() { _ = rows.Close() }()

	type hit struct{ pre, variant, mapName, playlist string }
	var slayer, autres []hit
	n := 0
	for rows.Next() {
		var id, gv, mp, pl string
		if err := rows.Scan(&id, &gv, &mp, &pl); err != nil {
			panic(err)
		}
		pre := strings.ToLower(strings.ReplaceAll(id, "-", ""))
		if len(pre) < 8 {
			continue
		}
		pre = pre[:8]
		if !prefixes[pre] {
			continue
		}
		n++
		h := hit{pre, gv, mp, pl}
		if strings.Contains(strings.ToLower(gv), "slayer") {
			slayer = append(slayer, h)
		} else {
			autres = append(autres, h)
		}
	}
	fmt.Printf("films appariés au registre : %d — dont SLAYER : %d\n\n", n, len(slayer))
	for i, h := range slayer {
		if i >= 40 {
			fmt.Printf("... (%d de plus)\n", len(slayer)-40)
			break
		}
		fmt.Printf("SLAYER  %s  variante=%-28s carte=%-18s playlist=%s\n", h.pre, h.variant, h.mapName, h.playlist)
	}
	fmt.Println()
	for i, h := range autres {
		if i >= 10 {
			break
		}
		fmt.Printf("(autre) %s  variante=%-28s carte=%s\n", h.pre, h.variant, h.mapName)
	}
}
