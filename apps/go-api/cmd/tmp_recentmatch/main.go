// tmp_recentmatch — THROWAWAY. Identifie le match le plus récent en BDD + son contexte
// (nb kills, joueurs, et si les chunks de film sont en cache local pour le décode offline).
package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
const cacheRoot = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`

func main() {
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 5 matchs les plus récents
	fmt.Println("=== 5 matchs les plus récents (match_registry) ===")
	rows, err := db.Query(`SELECT match_id, start_time, pair_name FROM match_registry ORDER BY start_time DESC LIMIT 5`)
	if err != nil {
		panic(err)
	}
	var recent []string
	for rows.Next() {
		var mid, pair string
		var st sql.NullString
		rows.Scan(&mid, &st, &pair)
		recent = append(recent, mid)
		fmt.Printf("  %s  %s  %s\n", mid, st.String, pair)
	}
	rows.Close()
	if len(recent) == 0 {
		fmt.Println("  (aucun match)")
		return
	}
	top := recent[0]
	fmt.Printf("\n=== match le plus récent = %s ===\n", top)

	// nb kills (killer_victim_pairs) + nb participants
	var kc, pc int
	db.QueryRow(`SELECT COUNT(*) FROM killer_victim_pairs WHERE match_id=?`, top).Scan(&kc)
	db.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE match_id=?`, top).Scan(&pc)
	fmt.Printf("  kills (killer_victim_pairs) = %d ; participants = %d\n", kc, pc)

	// gamertags des participants
	prows, _ := db.Query(`SELECT gamertag, xuid, kills FROM match_participants WHERE match_id=? ORDER BY kills DESC`, top)
	fmt.Println("  roster :")
	for prows.Next() {
		var gt, xuid string
		var k sql.NullInt64
		prows.Scan(&gt, &xuid, &k)
		fmt.Printf("    %-20s xuid=%-18s kills=%d\n", gt, xuid, k.Int64)
	}
	prows.Close()

	// chunks de film en cache ?
	fmt.Println("\n=== cache film_chunks (offline decode possible ?) ===")
	dirs, _ := os.ReadDir(cacheRoot)
	var cached []string
	for _, d := range dirs {
		if d.IsDir() {
			cached = append(cached, d.Name())
		}
	}
	sort.Strings(cached)
	fmt.Printf("  matchs en cache : %v\n", cached)
	topShort := top[:8]
	found := ""
	for _, c := range cached {
		if len(c) >= 8 && c[:8] == topShort {
			found = c
		}
	}
	if found != "" {
		cnt := 0
		filepath.Walk(filepath.Join(cacheRoot, found), func(p string, info os.FileInfo, e error) error {
			if e == nil && !info.IsDir() {
				cnt++
			}
			return nil
		})
		fmt.Printf("  >>> le match récent %s EST en cache (%d fichiers) -> décode offline possible\n", topShort, cnt)
	} else {
		fmt.Printf("  >>> le match récent %s n'est PAS en cache -> film à (re)télécharger pour décode offline\n", topShort)
	}
}
