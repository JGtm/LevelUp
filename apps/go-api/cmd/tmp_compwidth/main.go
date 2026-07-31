// tmp_compwidth — THROWAWAY : mesure la LARGEUR EXACTE (en bits) de chaque composant
// par archétype, depuis la capture CE filmdec_delta_capture.csv (eid,typeIndex,
// compIndex,param4,bitCursor,skipCount). Dans un record, les dispatches de composants
// sont contigus -> width(compIndex) = bitCursor(suivant) - bitCursor(courant). C'est la
// vérité-terrain pour caler les desers world-object (ex object-position ti=41).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_compwidth <csv> [typeIndex]
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type row struct{ eid, ti, ci, cursor int }

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: tmp_compwidth <csv> [typeIndex]")
		return
	}
	filterTi := -1
	if len(os.Args) >= 3 {
		filterTi, _ = strconv.Atoi(os.Args[2])
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("ouverture impossible: %v\n", err)
		return
	}
	defer f.Close()

	// width[ti][ci] -> (width -> count)
	width := map[int]map[int]map[int]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	var prev *row
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "eid") {
			continue
		}
		p := strings.Split(line, ",")
		if len(p) < 6 {
			continue
		}
		eid, _ := strconv.Atoi(p[0])
		ti, _ := strconv.Atoi(p[1])
		ci, _ := strconv.Atoi(p[2])
		cur, _ := strconv.Atoi(p[4])
		cur4 := cur
		r := row{eid, ti, ci, cur4}
		// diff vs précédent SI même record (même eid) et curseur croissant et compIndex croissant
		if prev != nil && prev.eid == r.eid && prev.ti == r.ti && r.cursor > prev.cursor && r.ci > prev.ci {
			w := r.cursor - prev.cursor
			if w >= 0 && w < 2000 {
				if width[prev.ti] == nil {
					width[prev.ti] = map[int]map[int]int{}
				}
				if width[prev.ti][prev.ci] == nil {
					width[prev.ti][prev.ci] = map[int]int{}
				}
				width[prev.ti][prev.ci][w]++
			}
		}
		rr := r
		prev = &rr
	}

	tis := []int{}
	for ti := range width {
		if filterTi < 0 || ti == filterTi {
			tis = append(tis, ti)
		}
	}
	sort.Ints(tis)
	for _, ti := range tis {
		fmt.Printf("=== ti=%d : largeur par composant (compIndex -> width modal [n], autres) ===\n", ti)
		cis := []int{}
		for ci := range width[ti] {
			cis = append(cis, ci)
		}
		sort.Ints(cis)
		for _, ci := range cis {
			m := width[ti][ci]
			type wn struct{ w, n int }
			var ws []wn
			for w, n := range m {
				ws = append(ws, wn{w, n})
			}
			sort.Slice(ws, func(i, j int) bool { return ws[i].n > ws[j].n })
			s := ""
			for i, x := range ws {
				if i >= 4 {
					break
				}
				s += fmt.Sprintf("%db:%d ", x.w, x.n)
			}
			fmt.Printf("  i%-2d : %s\n", ci, s)
		}
		fmt.Println()
	}
}
