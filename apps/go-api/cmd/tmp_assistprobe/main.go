// tmp_assistprobe — THROWAWAY, OEIL NEUF : l'assistant par kill est-il deja dans chunk_27 ?
// Hypothese fraiche : un event highlight = 1 joueur (xuid) + type_hint + time. Un ASSIST en Halo
// est une medaille. Donc l'assistant pourrait etre un event MEDAL time-correle au kill, sans avoir
// a decoder le composant kill-event enfoui dans le flux ECS.
//
// Ne presume RIEN des resultats anterieurs : roster (xuid/gamertag/kills/deaths/ASSISTS) chargé
// FRAIS depuis la DB (verite-terrain API). On dumpe l'inventaire des events, la distribution des
// medalType, et on cherche quel medalType a un compte ~= total assists DB.
//
// Usage : tmp_assistprobe
package main

import (
	"database/sql"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"

	_ "github.com/duckdb/duckdb-go/v2"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`

type player struct {
	gt                   string
	kills, deaths, assis int
}

func main() {
	// 1) roster FRAIS depuis la DB
	roster, total := loadRoster()
	fmt.Printf("=== ROSTER DB (match 000d5950) — %d joueurs ===\n", len(roster))
	var xs []uint64
	for x := range roster {
		xs = append(xs, x)
	}
	sort.Slice(xs, func(i, j int) bool { return roster[xs[i]].kills > roster[xs[j]].kills })
	for _, x := range xs {
		p := roster[x]
		fmt.Printf("  %-18s xuid=%d  K=%-3d D=%-3d A=%-3d\n", p.gt, x, p.kills, p.deaths, p.assis)
	}
	fmt.Printf("  TOTAL : kills=%d deaths=%d ASSISTS=%d\n\n", total.kills, total.deaths, total.assis)

	// 2) events chunk_27
	raw, err := os.ReadFile(cache + "/chunk_27.bin")
	if err != nil {
		panic(err)
	}
	events, err := analysis.ParseHighlightEvents(raw, 0)
	if err != nil {
		fmt.Println("parse err:", err)
	}
	fmt.Printf("=== chunk_27 : %d events parsés ===\n", len(events))

	// 2a) distribution par EventType
	byType := map[string]int{}
	for _, e := range events {
		byType[e.EventType]++
	}
	fmt.Println("  par type :")
	for t, n := range byType {
		fmt.Printf("    %-8s : %d\n", t, n)
	}

	// 2b) distribution par (EventType, TypeHint, MedalType, IsMedal)
	type key struct {
		et      string
		th, mt  int
		isMedal bool
	}
	dist := map[key]int{}
	for _, e := range events {
		dist[key{e.EventType, e.TypeHint, e.MedalType, e.IsMedal}]++
	}
	var keys []key
	for k := range dist {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return dist[keys[i]] > dist[keys[j]] })
	fmt.Println("\n  distribution fine (type / typeHint / medalType / isMedal -> count) :")
	for _, k := range keys {
		flag := ""
		if dist[k] == total.assis || dist[k] == total.assis+1 || dist[k] == total.assis-1 {
			flag = "   <<< ~= total ASSISTS DB !"
		}
		fmt.Printf("    %-7s th=%-3d mt=%-3d medal=%-5v -> %-3d%s\n", k.et, k.th, k.mt, k.isMedal, dist[k], flag)
	}

	// 3) per-joueur : nb d'events medal, a comparer aux assists DB
	fmt.Println("\n=== par joueur : kills/deaths/medals d'events vs DB ===")
	evByX := map[uint64]map[string]int{}
	medalByX := map[uint64]int{}
	for _, e := range events {
		if evByX[e.XUID] == nil {
			evByX[e.XUID] = map[string]int{}
		}
		evByX[e.XUID][e.EventType]++
		if e.EventType == analysis.EventTypeMedal {
			medalByX[e.XUID]++
		}
	}
	for _, x := range xs {
		p := roster[x]
		ev := evByX[x]
		fmt.Printf("  %-18s | events: kill=%-3d death=%-3d medal=%-3d mode=%-3d | DB: K=%-3d D=%-3d A=%-3d\n",
			p.gt, ev["kill"], ev["death"], ev["medal"], ev["mode"], p.kills, p.deaths, p.assis)
	}

	fmt.Println("\n>>> Si un medalType a un compte ~= total ASSISTS, c'est le candidat assist -> time-correler aux kills.")
	fmt.Println(">>> Si le medal-count par joueur ~= ses assists DB, l'assistant est dans chunk_27 (pas besoin du composant ECS).")
}

func loadRoster() (map[uint64]player, player) {
	out := map[uint64]player{}
	var tot player
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		fmt.Println("db open err:", err)
		return out, tot
	}
	defer db.Close()
	var full string
	if db.QueryRow(`SELECT match_id FROM match_registry WHERE match_id LIKE '000d5950%' LIMIT 1`).Scan(&full) != nil {
		fmt.Println("match introuvable")
		return out, tot
	}
	rows, err := db.Query(`SELECT xuid, gamertag, kills, deaths, assists FROM match_participants WHERE match_id=?`, full)
	if err != nil {
		fmt.Println("query err:", err)
		return out, tot
	}
	defer rows.Close()
	for rows.Next() {
		var xs, gt sql.NullString
		var k, d, a sql.NullInt64
		rows.Scan(&xs, &gt, &k, &d, &a)
		var xu uint64
		fmt.Sscanf(xs.String, "%d", &xu)
		out[xu] = player{gt.String, int(k.Int64), int(d.Int64), int(a.Int64)}
		tot.kills += int(k.Int64)
		tot.deaths += int(d.Int64)
		tot.assis += int(a.Int64)
	}
	return out, tot
}
