// tmp_killevent_scalars — THROWAWAY. Les 2 scalaires bruts du kill-event (FUN_14104bd08)
// sont-ils l'ARME, ou des % de dégâts ? Test : distribution par tueur + somme s1+s2 sur kills assistés.
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

const base = 3780116480
const step = 0x10002

var roster = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm",
	3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}

func idx(v int64) int {
	if v < base || (v-base)%step != 0 {
		return -1
	}
	return int((v - base) / step)
}

func main() {
	f, _ := os.Open(`c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/apps/go-api/cmd/tmp_killevent_scalars/data.csv`)
	defer f.Close()
	type row struct{ victim, killer, s1, b, assist, s2 int64 }
	var rows []row
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		c := strings.Split(strings.TrimSpace(sc.Text()), ",")
		if len(c) < 6 {
			continue
		}
		p := func(s string) int64 { n, _ := strconv.ParseInt(s, 10, 64); return n }
		rows = append(rows, row{p(c[0]), p(c[1]), p(c[2]), p(c[3]), p(c[4]), p(c[5])})
	}

	// 1. s1 par tueur : si ARME, un mono-arme aurait s1 ~constant. Si %dmg, varie 0..~100.
	fmt.Println("=== scalaire1 (s1) par tueur — distinct values ===")
	s1byK := map[int]map[int64]int{}
	for _, r := range rows {
		k := idx(r.killer)
		if s1byK[k] == nil {
			s1byK[k] = map[int64]int{}
		}
		s1byK[k][r.s1]++
	}
	var ks []int
	for k := range s1byK {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	for _, k := range ks {
		var vs []int64
		for v := range s1byK[k] {
			vs = append(vs, v)
		}
		sort.Slice(vs, func(i, j int) bool { return vs[i] < vs[j] })
		fmt.Printf("  k%d %-18s : %d valeurs distinctes %v\n", k, roster[k], len(vs), vs)
	}

	// 2. Kills assistés (assist != -1) : s1+s2 = ?
	fmt.Println("\n=== kills assistés : s1 + s2 (test %dmg : doit ~99/100) ===")
	nAssist, sumOK := 0, 0
	for _, r := range rows {
		if r.assist == 4294967295 {
			continue
		}
		nAssist++
		sum := r.s1 + r.s2
		flag := ""
		if sum >= 95 && sum <= 100 {
			sumOK++
			flag = "OK~100"
		}
		fmt.Printf("  s1=%-3d s2=%-3d sum=%-3d %s\n", r.s1, r.s2, sum, flag)
	}
	fmt.Printf("  -> %d/%d kills assistés ont s1+s2 dans [95..100]\n", sumOK, nAssist)

	// 3. Kills solo (assist=-1) : s2 doit etre le sentinel 149
	fmt.Println("\n=== kills solo (assist=-1) : valeurs de s2 ===")
	s2solo := map[int64]int{}
	s1solo := map[int64]int{}
	for _, r := range rows {
		if r.assist != 4294967295 {
			continue
		}
		s2solo[r.s2]++
		s1solo[r.s1]++
	}
	fmt.Printf("  s2 solo distinct : %v\n", mapKeys(s2solo))
	fmt.Printf("  s1 solo distinct : %v (range = part de dégâts du tueur ?)\n", mapKeys(s1solo))

	// 4. Verdict ARME : s1/s2 ont-ils ~15-30 valeurs clusterisées (armes) ou un continuum (%) ?
	allS1, allS2 := map[int64]int{}, map[int64]int{}
	for _, r := range rows {
		allS1[r.s1]++
		allS2[r.s2]++
	}
	fmt.Printf("\n=== global : s1 = %d valeurs distinctes, s2 = %d valeurs distinctes ===\n", len(allS1), len(allS2))
	fmt.Println("  (une ARME aurait des valeurs HANDLE/grandes ou un petit set clusterisé par type ; un % est un continuum 0..~100)")
}

func mapKeys(m map[int64]int) []int64 {
	var out []int64
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
