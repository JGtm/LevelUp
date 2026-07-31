// tmp_killvalidate — THROWAWAY. Valide l'extraction tueur/victime du film (CSV CE) contre
// la DB (killer_victim_pairs), résout index-film -> gamertag par force brute (8!), et isole
// les kills d'outro (film en excès vs DB). Si une permutation couvre 100% de la DB avec
// exactement 5 kills film en excès -> extraction film PROUVÉE correcte.
package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

const csvPath = `C:/Users/Guillaume/Downloads/filmdec_killweapon.csv`
const sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
const step = 0x10002

func main() {
	film, base := filmMatrix() // 8x8 par index
	dbMat, names := dbMatrix() // 8x8 par gamertag-index

	// force brute des 8! permutations : film index i -> db gamertag perm[i]
	best := []int{}
	bestExcess, bestUncovered := 1<<30, 1<<30
	perm := []int{0, 1, 2, 3, 4, 5, 6, 7}
	permute(perm, 0, func(p []int) {
		excess, uncovered := 0, 0
		for i := 0; i < 8; i++ {
			for j := 0; j < 8; j++ {
				f, d := film[i][j], dbMat[p[i]][p[j]]
				if f > d {
					excess += f - d
				}
				if d > f {
					uncovered += d - f
				}
			}
		}
		if uncovered < bestUncovered || (uncovered == bestUncovered && excess < bestExcess) {
			bestUncovered, bestExcess = uncovered, excess
			best = append([]int{}, p...)
		}
	})

	fmt.Printf("=== Meilleure bijection : DB non-couverte=%d kills, film en excès=%d kills ===\n", bestUncovered, bestExcess)
	if bestUncovered == 0 {
		fmt.Println("  >>> 100% de la DB couverte par le film. Extraction tueur/victime PROUVÉE.")
	}
	fmt.Printf("  >>> film en excès = %d kills = OUTRO (attendu ~5)\n\n", bestExcess)

	fmt.Println("=== Mapping index-film -> gamertag (+ kills film / kills DB) ===")
	for i := 0; i < 8; i++ {
		fk, dk := 0, 0
		for j := 0; j < 8; j++ {
			fk += film[i][j]
			dk += dbMat[best[i]][best[j]]
		}
		fmt.Printf("  idx%d -> %-18s  (film %d kills, DB %d kills)\n", i, names[best[i]], fk, dk)
	}

	// localiser les kills d'outro : entrées (i,j) où film > db
	fmt.Println("\n=== Kills d'outro identifiés (film > DB pour la paire) ===")
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			if d := film[i][j] - dbMat[best[i]][best[j]]; d > 0 {
				fmt.Printf("  %-18s -> %-18s : +%d kill(s) hors-stats\n", names[best[i]], names[best[j]], d)
			}
		}
	}
	_ = base
}

func filmMatrix() ([8][8]int, uint32) {
	f, err := os.Open(csvPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	type rec struct{ killer, victim uint32 }
	var recs []rec
	base := ^uint32(0)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "weaponDefId") {
			continue
		}
		c := strings.Split(line, ",")
		if len(c) < 6 {
			continue
		}
		u := func(s string) uint32 { n, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64); return uint32(n) }
		victim, killer := u(c[4]), u(c[5]) // p04=victime, p08=tueur
		recs = append(recs, rec{killer, victim})
		if victim < base {
			base = victim
		}
		if killer < base {
			base = killer
		}
	}
	var m [8][8]int
	idx := func(v uint32) int { return int((v - base) / step) }
	for _, r := range recs {
		ki, vi := idx(r.killer), idx(r.victim)
		if ki >= 0 && ki < 8 && vi >= 0 && vi < 8 {
			m[ki][vi]++
		}
	}
	return m, base
}

func dbMatrix() ([8][8]int, []string) {
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	var mid string
	db.QueryRow(`SELECT match_id FROM match_registry WHERE match_id LIKE '000d5950%' LIMIT 1`).Scan(&mid)
	rs, _ := db.Query(`SELECT killer_gamertag, victim_gamertag, kill_count FROM killer_victim_pairs WHERE match_id=?`, mid)
	defer rs.Close()
	gi := map[string]int{}
	names := []string{}
	id := func(g string) int {
		if v, ok := gi[g]; ok {
			return v
		}
		gi[g] = len(names)
		names = append(names, g)
		return gi[g]
	}
	type row struct {
		k, v string
		c    int
	}
	var rows []row
	for rs.Next() {
		var r row
		rs.Scan(&r.k, &r.v, &r.c)
		rows = append(rows, r)
		id(r.k)
		id(r.v)
	}
	var m [8][8]int
	for _, r := range rows {
		m[id(r.k)][id(r.v)] += r.c
	}
	return m, names
}

func permute(a []int, k int, f func([]int)) {
	if k == len(a) {
		f(a)
		return
	}
	for i := k; i < len(a); i++ {
		a[k], a[i] = a[i], a[k]
		permute(a, k+1, f)
		a[k], a[i] = a[i], a[k]
	}
}
