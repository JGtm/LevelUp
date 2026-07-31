// tmp_dbcheck — THROWAWAY, OEIL NEUF : que contient DÉJÀ la DB/API pour ce match, sur l'arme
// par kill et l'assistant ? Avant de décoder le film, vérifier killer_victim_pairs.weapon_id,
// highlight_events, medals_earned. Ne présume rien.
package main

import (
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`

func main() {
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var full string
	if err := db.QueryRow(`SELECT match_id FROM match_registry WHERE match_id LIKE '000d5950%' LIMIT 1`).Scan(&full); err != nil {
		fmt.Println("match introuvable:", err)
		return
	}
	fmt.Printf("match_id = %s\n\n", full)

	// 1) colonnes de killer_victim_pairs
	fmt.Println("=== schéma killer_victim_pairs ===")
	dumpCols(db, "killer_victim_pairs")

	// 2) killer_victim_pairs pour ce match
	fmt.Println("\n=== killer_victim_pairs (ce match) ===")
	rows, err := db.Query(`SELECT killer_xuid, victim_xuid, count, weapon_id FROM killer_victim_pairs WHERE match_id=? ORDER BY count DESC`, full)
	if err != nil {
		fmt.Println("  (pas de weapon_id ? err:", err, ")")
		// retry sans weapon_id
		rows, err = db.Query(`SELECT killer_xuid, victim_xuid, count FROM killer_victim_pairs WHERE match_id=? ORDER BY count DESC`, full)
		if err != nil {
			fmt.Println("  err:", err)
		} else {
			defer rows.Close()
			n := 0
			for rows.Next() {
				var k, v sql.NullString
				var c sql.NullInt64
				rows.Scan(&k, &v, &c)
				if n < 20 {
					fmt.Printf("  %s -> %s : count=%d\n", k.String, v.String, c.Int64)
				}
				n++
			}
			fmt.Printf("  total %d paires\n", n)
		}
	} else {
		defer rows.Close()
		n, withW := 0, 0
		for rows.Next() {
			var k, v, w sql.NullString
			var c sql.NullInt64
			rows.Scan(&k, &v, &c, &w)
			if w.Valid && w.String != "" && w.String != "0" {
				withW++
			}
			if n < 25 {
				fmt.Printf("  %s -> %s : count=%d  weapon_id=%q\n", k.String, v.String, c.Int64, w.String)
			}
			n++
		}
		fmt.Printf("  total %d paires ; %d avec weapon_id non vide\n", n, withW)
	}

	// 3) highlight_events schéma + sample
	fmt.Println("\n=== schéma highlight_events ===")
	dumpCols(db, "highlight_events")
	fmt.Println("\n=== highlight_events : distinct event_type (ce match) ===")
	rows2, err := db.Query(`SELECT event_type, count(*) FROM highlight_events WHERE match_id=? GROUP BY event_type ORDER BY 2 DESC`, full)
	if err != nil {
		fmt.Println("  err:", err)
	} else {
		defer rows2.Close()
		for rows2.Next() {
			var t sql.NullString
			var c sql.NullInt64
			rows2.Scan(&t, &c)
			fmt.Printf("  %-20s : %d\n", t.String, c.Int64)
		}
	}

	// 4) medals_earned schéma + assist-like ?
	fmt.Println("\n=== schéma medals_earned ===")
	dumpCols(db, "medals_earned")
}

func dumpCols(db *sql.DB, table string) {
	rows, err := db.Query(`SELECT column_name, data_type FROM information_schema.columns WHERE table_name=? ORDER BY ordinal_position`, table)
	if err != nil {
		fmt.Println("  err:", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var c, t sql.NullString
		rows.Scan(&c, &t)
		fmt.Printf("  %-28s %s\n", c.String, t.String)
	}
}
