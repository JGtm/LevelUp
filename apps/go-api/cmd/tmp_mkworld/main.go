// tmp_mkworld — THROWAWAY : construit un world_dump COMPLET (slot:typeIndex) depuis
// la capture CE filmdec_delta_capture.csv (eid,typeIndex,...). Pour chaque slot, prend
// le typeIndex le PLUS FRÉQUENT (le binding live est stable hors slots réutilisés).
// Sert à injecter le binding complet offline et isoler binding-cassé vs composants-cassés.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_mkworld <in.csv> <out_world_dump.txt>
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: tmp_mkworld <in.csv> <out.txt>")
		return
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("ouverture CSV impossible: %v\n", err)
		return
	}
	defer f.Close()

	// slot -> (typeIndex -> count)
	votes := map[int]map[int]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	nrows := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "eid") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		eid, e1 := strconv.Atoi(parts[0])
		ti, e2 := strconv.Atoi(parts[1])
		if e1 != nil || e2 != nil || eid < 0 || ti < 0 {
			continue
		}
		if votes[eid] == nil {
			votes[eid] = map[int]int{}
		}
		votes[eid][ti]++
		nrows++
	}

	// pour chaque slot, typeIndex modal
	type bind struct{ slot, ti, n, distinct int }
	var binds []bind
	for slot, m := range votes {
		bestTi, bestN := -1, 0
		for ti, n := range m {
			if n > bestN {
				bestN, bestTi = n, ti
			}
		}
		binds = append(binds, bind{slot, bestTi, bestN, len(m)})
	}
	sort.Slice(binds, func(i, j int) bool { return binds[i].slot < binds[j].slot })

	out, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Printf("création sortie impossible: %v\n", err)
		return
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	fmt.Fprintf(w, "# world_dump COMPLET via filmdec_delta_capture (slot:typeIndex modal). %d slots, %d rows.\n", len(binds), nrows)
	reused := 0
	for _, b := range binds {
		fmt.Fprintf(w, "%d:%d\n", b.slot, b.ti)
		if b.distinct > 1 {
			reused++
		}
	}
	w.Flush()
	fmt.Printf("%d slots écrits (%d rows lus) -> %s\n", len(binds), nrows, os.Args[2])
	fmt.Printf("slots à typeIndex multiple (réutilisés, ambigus sans horodatage) : %d\n", reused)
}
