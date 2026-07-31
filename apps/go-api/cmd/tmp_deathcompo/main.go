// tmp_deathcompo — THROWAWAY : composition RÉELLE d'un death-delta depuis le CSV CE (vérité-terrain,
// PAS le décodeur cassé). Parse filmdec_delta_death.csv, segmente les records biped (typeIndex=35),
// isole ceux contenant le composant 11 (dead-state), et dumpe la séquence (compIndex -> largeur).
// => dit EXACTEMENT quels composants sont présents avant i11 dans un vrai delta-de-mort et leur largeur.
//
// Usage : tmp_deathcompo <csv>
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type row struct {
	eid, typeIndex, compIndex, bitCursor int
}

func main() {
	path := `c:/Users/Guillaume/Downloads/filmdec_delta_death.csv`
	if len(os.Args) >= 2 {
		path = os.Args[1]
	}
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var rows []row
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if len(line) == 0 || line[0] == '#' || line[0] == 'e' {
			continue
		}
		p := strings.Split(line, ",")
		if len(p) < 5 {
			continue
		}
		eid, _ := strconv.Atoi(p[0])
		ti, _ := strconv.Atoi(p[1])
		ci, _ := strconv.Atoi(p[2])
		bc, _ := strconv.Atoi(p[4])
		rows = append(rows, row{eid, ti, ci, bc})
	}

	// segmenter en records : un nouveau record commence quand compIndex DIMINUE ou eid change.
	type rec struct {
		comps  []int // compIndex en ordre
		widths []int // largeur (cursor diff) par comp
	}
	var recs []rec
	var cur rec
	var lastEid, lastCi, lastCursor int = -1, -1, -1
	flush := func() {
		if len(cur.comps) > 0 {
			recs = append(recs, cur)
		}
		cur = rec{}
	}
	for _, r := range rows {
		if r.typeIndex != 35 {
			flush()
			lastEid, lastCi = -1, -1
			continue
		}
		if r.eid != lastEid || r.compIndex <= lastCi {
			flush()
		} else if lastCursor >= 0 && len(cur.comps) > 0 {
			cur.widths[len(cur.widths)-1] = r.bitCursor - lastCursor
		}
		cur.comps = append(cur.comps, r.compIndex)
		cur.widths = append(cur.widths, 0)
		lastEid, lastCi, lastCursor = r.eid, r.compIndex, r.bitCursor
	}
	flush()

	// records contenant le composant 11
	var deaths []rec
	for _, rc := range recs {
		for _, ci := range rc.comps {
			if ci == 11 {
				deaths = append(deaths, rc)
				break
			}
		}
	}
	fmt.Printf("=== %d records biped au total ; %d contiennent i11 (dead-state) ===\n\n", len(recs), len(deaths))

	// signature : ensemble des composants <=11. Fréquence.
	sigCount := map[string]int{}
	sigSample := map[string]rec{}
	for _, rc := range deaths {
		var pre []string
		for j, ci := range rc.comps {
			if ci <= 11 {
				pre = append(pre, fmt.Sprintf("i%d:%d", ci, rc.widths[j]))
			}
		}
		sig := strings.Join(pre, " ")
		sigCount[sig]++
		if _, ok := sigSample[sig]; !ok {
			sigSample[sig] = rc
		}
	}
	type kv struct {
		sig string
		c   int
	}
	var sigs []kv
	for s, c := range sigCount {
		sigs = append(sigs, kv{s, c})
	}
	sort.Slice(sigs, func(i, j int) bool { return sigs[i].c > sigs[j].c })
	fmt.Println("=== composition i0..i11 des death-deltas (composant:largeur), par fréquence ===")
	for i, s := range sigs {
		if i >= 12 {
			fmt.Printf("  ... (%d signatures distinctes)\n", len(sigs))
			break
		}
		fmt.Printf("  ×%-4d : %s\n", s.c, s.sig)
	}

	// par composant <=11 : fréquence de présence dans les death-deltas + largeurs distinctes
	fmt.Println("\n=== par composant i0..i11 dans les death-deltas : présence + largeurs distinctes ===")
	presence := map[int]int{}
	widthsOf := map[int]map[int]int{}
	for _, rc := range deaths {
		for j, ci := range rc.comps {
			if ci > 11 {
				continue
			}
			presence[ci]++
			if widthsOf[ci] == nil {
				widthsOf[ci] = map[int]int{}
			}
			widthsOf[ci][rc.widths[j]]++
		}
	}
	for ci := 0; ci <= 11; ci++ {
		if presence[ci] == 0 {
			continue
		}
		var ws []kv2
		for w, c := range widthsOf[ci] {
			ws = append(ws, kv2{w, c})
		}
		sort.Slice(ws, func(i, j int) bool { return ws[i].c > ws[j].c })
		s := ""
		variable := len(ws) > 1
		for k, w := range ws {
			if k >= 4 {
				break
			}
			s += fmt.Sprintf(" %dbit×%d", w.w, w.c)
		}
		flag := ""
		if variable {
			flag = "  <<< VARIABLE (deser à porter, pas calibrable)"
		}
		fmt.Printf("  i%-2d présent %-4d :%s%s\n", ci, presence[ci], s, flag)
	}
}

type kv2 struct{ w, c int }
