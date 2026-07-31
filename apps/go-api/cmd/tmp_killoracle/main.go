// tmp_killoracle — THROWAWAY : extrait les kills (killer/victim/time_ms) du match
// 000d5950 depuis killer_victim_pairs + le détail des highlight_events autour des
// kills connus, pour ancrer la sonde empirique d'arme-par-kill.
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`

func main() {
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Match id complet
	var fullID string
	if err := db.QueryRow(`SELECT match_id FROM match_registry WHERE match_id LIKE '000d5950%' LIMIT 1`).Scan(&fullID); err != nil {
		fmt.Println("match lookup:", err)
		os.Exit(1)
	}
	fmt.Println("full match_id =", fullID)

	// Kills ordonnés
	fmt.Println("\n=== killer_victim_pairs (time_ms ASC) ===")
	rows, err := db.Query(`SELECT killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, time_ms
		FROM killer_victim_pairs WHERE match_id = ? ORDER BY time_ms`, fullID)
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		var kx, kg, vx, vg sql.NullString
		var t sql.NullInt64
		rows.Scan(&kx, &kg, &vx, &vg, &t)
		fmt.Printf("  t=%-8d killer=%s(%s) -> victim=%s(%s)\n", t.Int64, kg.String, kx.String, vg.String, vx.String)
	}
	rows.Close()

	// highlight_events autour de quelques kills (raw_json complet)
	fmt.Println("\n=== highlight_events (type_hint, time_ms) — premiers 40 ===")
	rows2, err := db.Query(`SELECT event_type, time_ms, xuid, type_hint, raw_json
		FROM highlight_events WHERE match_id = ? ORDER BY time_ms LIMIT 40`, fullID)
	if err != nil {
		fmt.Println("highlight err:", err)
		return
	}
	for rows2.Next() {
		var et, xuid, th, rj sql.NullString
		var t sql.NullInt64
		rows2.Scan(&et, &t, &xuid, &th, &rj)
		rjs := rj.String
		if len(rjs) > 160 {
			rjs = rjs[:160]
		}
		fmt.Printf("  t=%-8d et=%-14s th=%-4s xuid=%s\n     raw=%s\n", t.Int64, et.String, th.String, xuid.String, rjs)
	}
	rows2.Close()

	// weapon_kills v2 (pi attribution) si présent — colonnes : player_index,
	// weapon_id, time_ms, match_id, reconciled_as
	fmt.Println("\n=== weapon_kills (v2) time_ms ASC ===")
	rows3, err := db.Query(`SELECT player_index, weapon_id, time_ms, reconciled_as FROM weapon_kills WHERE match_id = ? ORDER BY time_ms LIMIT 60`, fullID)
	if err != nil {
		fmt.Println("weapon_kills err:", err)
		return
	}
	for rows3.Next() {
		var pi sql.NullInt64
		var wid, rec sql.NullString
		var t sql.NullInt64
		rows3.Scan(&pi, &wid, &t, &rec)
		fmt.Printf("  t=%-8d pi=%-3d weapon_id=%-20s reconciled=%s\n", t.Int64, pi.Int64, wid.String, rec.String)
	}
	rows3.Close()

	// player_index -> xuid mapping via match_participants (pour relier pi au killer)
	fmt.Println("\n=== match_participants player_index -> xuid/gamertag ===")
	rows4, err := db.Query(`SELECT player_index, xuid, gamertag FROM match_participants WHERE match_id = ? ORDER BY player_index`, fullID)
	if err != nil {
		fmt.Println("participants err:", err)
		return
	}
	for rows4.Next() {
		var pi sql.NullInt64
		var x, g sql.NullString
		rows4.Scan(&pi, &x, &g)
		fmt.Printf("  pi=%-3d xuid=%-18s gamertag=%s\n", pi.Int64, x.String, g.String)
	}
	rows4.Close()
}
