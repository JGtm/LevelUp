// tmp_mapinv — outil jetable : inventaire des cartes distinctes de match_registry
// (map_id, map_version_id, nb matchs, nb matchs CTF).
package main

import (
	"database/sql"
	"fmt"

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
	rows, err := db.Query(`
		SELECT map_id,
		       COUNT(*) AS n,
		       COUNT(map_version_id) AS n_ver,
		       ANY_VALUE(map_version_id) AS a_ver,
		       ANY_VALUE(map_name) AS a_name,
		       COUNT(DISTINCT game_variant_id) AS n_var
		FROM s.match_registry
		WHERE map_id IS NOT NULL
		GROUP BY map_id ORDER BY n DESC`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, ver, name sql.NullString
		var n, nver, nvar int
		if err := rows.Scan(&id, &n, &nver, &ver, &name, &nvar); err != nil {
			panic(err)
		}
		fmt.Printf("%-38s n=%4d ver_non_null=%4d ver=%-38s variantes=%2d %s\n",
			id.String, n, nver, ver.String, nvar, name.String)
	}
}
