//go:build cgo

// diag_medals — inspecte les medals_earned d'un match pour valider le calcul
// des citations medal/multi-medal (e.g. spartan_carnage qui agrège 8 medal_ids).
//
// Usage : go run -tags cgo ./cmd/diag_medals <player_gt> <match_id>
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"path/filepath"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

const tz = "Europe/Paris"

// medal_ids référencés par les citations multi-medal du Python source.
var spartanCarnageIDs = []uint64{
	2780740615, 4261842076, 418532952, 1486797009,
	710323196, 1720896992, 2567026752, 2875941471,
}
var opportunistIDs = []uint64{
	622331684, 2063152177, 4261842076, 2137071619,
	1486797009, 1430343434, 2242633421,
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("usage: diag_medals <player_gt> <match_id>")
	}
	playerGT := os.Args[1]
	matchID := os.Args[2]

	playerDB := filepath.Join("..", "..", "data", "titles", "halo_infinite", "players", playerGT, "stats.duckdb")
	sharedPath := filepath.Join("..", "..", "data", "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb")

	db := openDuckDB(playerDB)
	defer db.Close()
	mustExec(db, fmt.Sprintf("ATTACH '%s' AS shared (READ_ONLY)", sharedPath))

	// xuid via shared.xuid_aliases
	var xuid string
	_ = db.QueryRowContext(context.Background(),
		"SELECT xuid FROM shared.xuid_aliases WHERE gamertag = ? LIMIT 1", playerGT).Scan(&xuid)
	if xuid == "" {
		log.Fatalf("xuid not found for %s", playerGT)
	}

	fmt.Printf("=== medals_earned pour match=%s xuid=%s ===\n", matchID, xuid)
	rows, err := db.QueryContext(context.Background(),
		`SELECT medal_name_id, count FROM shared.medals_earned WHERE match_id = ? AND xuid = ? ORDER BY medal_name_id`,
		matchID, xuid)
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	all := map[int64]int{}
	for rows.Next() {
		var id int64
		var c int
		if err := rows.Scan(&id, &c); err != nil {
			log.Fatalf("scan: %v", err)
		}
		all[id] = c
		fmt.Printf("  medal_id=%-15d count=%d\n", id, c)
	}
	rows.Close()
	fmt.Printf("  Total: %d médailles distinctes\n\n", len(all))

	check := func(name string, ids []uint64) {
		fmt.Printf("=== Citation %q (medal_ids %v) ===\n", name, ids)
		total := 0
		hits := 0
		for _, id := range ids {
			c, ok := all[int64(id)]
			if ok {
				fmt.Printf("  HIT  medal_id=%d count=%d\n", id, c)
				total += c
				hits++
			}
		}
		fmt.Printf("  Résultat: %d hits / %d ids → total = %d\n\n", hits, len(ids), total)
	}
	check("spartan_carnage", spartanCarnageIDs)
	check("opportunist", opportunistIDs)
}

func openDuckDB(path string) *sql.DB {
	connector, err := duckdb.NewConnector(path+"?access_mode=READ_ONLY", func(execer driver.ExecerContext) error {
		_, e := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return e
	})
	if err != nil {
		log.Fatalf("connector(%s): %v", path, err)
	}
	return sql.OpenDB(connector)
}

func mustExec(db *sql.DB, q string) {
	if _, err := db.ExecContext(context.Background(), q); err != nil {
		log.Fatalf("exec(%s): %v", q, err)
	}
}
