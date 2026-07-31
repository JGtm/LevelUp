// tmp_dbweapon — THROWAWAY. La DB/API contient-elle l'arme par kill ?
// 1) raw_json des events 'kill' : champ arme ? 2) killer_victim_pairs (matrice DB pour
// valider le film) 3) roster match_participants (xuid->gamertag->kills).
package main

import (
	"database/sql"
	"fmt"
	"sort"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`

func main() {
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	var mid string
	db.QueryRow(`SELECT match_id FROM match_registry WHERE match_id LIKE '000d5950%' LIMIT 1`).Scan(&mid)

	// 1) raw_json de quelques kills
	fmt.Println("=== raw_json d'events 'kill' (cherche un champ arme) ===")
	rs, _ := db.Query(`SELECT time_ms, raw_json FROM highlight_events WHERE match_id=? AND event_type='kill' ORDER BY time_ms LIMIT 3`, mid)
	for rs.Next() {
		var t int
		var j string
		rs.Scan(&t, &j)
		fmt.Printf("  t=%d : %s\n", t, j)
	}
	rs.Close()

	// 2) roster
	fmt.Println("\n=== roster (match_participants) ===")
	type p struct {
		xuid, gt string
		kills    int
	}
	var roster []p
	rs, _ = db.Query(`SELECT xuid, gamertag, kills FROM match_participants WHERE match_id=? ORDER BY kills DESC`, mid)
	for rs.Next() {
		var x p
		rs.Scan(&x.xuid, &x.gt, &x.kills)
		roster = append(roster, x)
		fmt.Printf("  %-20s xuid=%-18s kills=%d\n", x.gt, x.xuid, x.kills)
	}
	rs.Close()

	// 3) killer_victim_pairs (matrice DB)
	fmt.Println("\n=== killer_victim_pairs DB (killer -> victim : kill_count) ===")
	type kv struct {
		k, v string
		c    int
	}
	var pairs []kv
	gtKills := map[string]int{}
	rs, _ = db.Query(`SELECT killer_gamertag, victim_gamertag, kill_count FROM killer_victim_pairs WHERE match_id=?`, mid)
	for rs.Next() {
		var x kv
		rs.Scan(&x.k, &x.v, &x.c)
		pairs = append(pairs, x)
		gtKills[x.k] += x.c
	}
	rs.Close()
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].c > pairs[j].c })
	for _, x := range pairs {
		fmt.Printf("  %-18s -> %-18s : %d\n", x.k, x.v, x.c)
	}
	fmt.Println("\n=== total kills par tueur (DB killer_victim_pairs) ===")
	type t struct {
		g string
		c int
	}
	var tot []t
	for g, c := range gtKills {
		tot = append(tot, t{g, c})
	}
	sort.Slice(tot, func(i, j int) bool { return tot[i].c > tot[j].c })
	for _, x := range tot {
		fmt.Printf("  %-18s : %d\n", x.g, x.c)
	}
}
