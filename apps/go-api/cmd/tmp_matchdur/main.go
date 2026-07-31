//go:build cgo

// tmp_matchdur — THROWAWAY : lit duration_seconds du match temoin en LECTURE SEULE, pour
// recouper l'horloge decodee du film. Aucune ecriture.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	prefix := "000d5950"
	if len(os.Args) > 1 {
		prefix = os.Args[1]
	}
	db, err := sql.Open("duckdb", "C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb?access_mode=read_only")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT match_id, duration_seconds,
	    COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC') AS st,
	    COALESCE(map_name,''), COALESCE(playlist_name,'')
	  FROM match_registry WHERE match_id LIKE ? || '%' LIMIT 10`, prefix)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, mp, pl string
		var dur sql.NullFloat64
		var st sql.NullTime
		if err := rows.Scan(&id, &dur, &st, &mp, &pl); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("match=%s duration_seconds=%v start=%v map=%q playlist=%q\n", id, dur.Float64, st.Time, mp, pl)
	}
}
