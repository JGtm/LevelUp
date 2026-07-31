// tmp_hdrtruth — MESURE de la longueur de l'EN-TETE de record DELTA sur la VERITE TERRAIN
// live (.ai/re_dump/ce_capture_delta.csv : eid,typeIndex,compIndex,param4,bitCursor).
// THROWAWAY.
//
// PRINCIPE (aucune hypothese de grammaire) : dans un record, la largeur d'un composant se
// lit comme la difference des curseurs de deux composants consecutifs. Entre deux records
// N et N+1 du meme paquet :
//
//	curseur(1er comp de N+1) - curseur(dernier comp de N)
//	   = largeur(dernier comp de N) + ENTETE(N+1) + 6 x nbComposants(N+1)   [masque epars]
//
// donc  ENTETE = gap - largeur(dernier) - 6 x count.  On mesure les largeurs par le mode
// des deltas intra-record, puis on publie l'histogramme de l'entete ainsi obtenue.
// L'entete attendue vaut 1 (prefixe type) + idLow + 2 (tag) + 1 (gate) + 3 (count).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_hdrtruth [csv]
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type hit struct {
	eid, ti, comp, cursor int
}

type rec struct {
	eid, ti int
	comps   []hit
}

func main() {
	path := `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_capture_delta.csv`
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var hits []hit
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		ln := strings.TrimSpace(sc.Text())
		if ln == "" || strings.HasPrefix(ln, "#") || strings.HasPrefix(ln, "eid") {
			continue
		}
		p := strings.Split(ln, ",")
		if len(p) < 5 {
			continue
		}
		eid, _ := strconv.Atoi(p[0])
		ti, _ := strconv.Atoi(p[1])
		ci, _ := strconv.Atoi(p[2])
		cur, _ := strconv.Atoi(p[4])
		hits = append(hits, hit{eid, ti, ci, cur})
	}
	fmt.Printf("hits lus : %d\n", len(hits))

	// Segmentation en records : un record = suite de hits de MEME eid a curseurs croissants.
	// Un changement d'eid OU un curseur qui recule ouvre un nouveau record.
	var recs []rec
	var cur rec
	last := -1
	for _, h := range hits {
		if cur.eid != h.eid || h.cursor <= last || len(cur.comps) == 0 {
			if len(cur.comps) > 0 {
				recs = append(recs, cur)
			}
			cur = rec{eid: h.eid, ti: h.ti}
		}
		cur.comps = append(cur.comps, h)
		last = h.cursor
	}
	if len(cur.comps) > 0 {
		recs = append(recs, cur)
	}
	fmt.Printf("records segmentes : %d\n", len(recs))

	// Largeurs : mode des deltas intra-record, par (ti, compIndex) ET seulement quand les
	// deux composants sont CONSECUTIFS dans la liste capturee (sinon on additionne des
	// composants absents de la capture).
	widthHist := map[[2]int]map[int]int{}
	for _, r := range recs {
		for i := 0; i+1 < len(r.comps); i++ {
			k := [2]int{r.ti, r.comps[i].comp}
			if widthHist[k] == nil {
				widthHist[k] = map[int]int{}
			}
			widthHist[k][r.comps[i+1].cursor-r.comps[i].cursor]++
		}
	}
	width := map[[2]int]int{}
	widthConf := map[[2]int]float64{}
	for k, h := range widthHist {
		bestV, bestN, tot := 0, 0, 0
		for v, n := range h {
			tot += n
			if n > bestN {
				bestV, bestN = v, n
			}
		}
		width[k] = bestV
		widthConf[k] = float64(bestN) / float64(tot)
	}
	fmt.Printf("largeurs de composants mesurees : %d couples (ti,comp)\n", len(width))
	fmt.Printf("  ex ti=35 i00=%d bits (confiance %.2f) ; i01=%d (%.2f) ; i25=%d (%.2f)\n",
		width[[2]int{35, 0}], widthConf[[2]int{35, 0}],
		width[[2]int{35, 1}], widthConf[[2]int{35, 1}],
		width[[2]int{35, 25}], widthConf[[2]int{35, 25}])

	// ENTETE : sur les paires de records consecutifs du meme paquet (curseur croissant),
	// avec largeur du dernier composant de N fiable (confiance >= 0.8) et count(N+1) <= 7
	// (branche eparse plausible).
	hdr := map[int]int{}
	hdrByCount := map[int]map[int]int{}
	pairs, skipped := 0, 0
	for i := 0; i+1 < len(recs); i++ {
		a, b := recs[i], recs[i+1]
		la := a.comps[len(a.comps)-1]
		fb := b.comps[0]
		if fb.cursor <= la.cursor { // frontiere de paquet
			skipped++
			continue
		}
		k := [2]int{a.ti, la.comp}
		w, ok := width[k]
		if !ok || widthConf[k] < 0.8 || w <= 0 {
			skipped++
			continue
		}
		c := len(b.comps)
		if c > 7 {
			skipped++
			continue
		}
		h := fb.cursor - la.cursor - w - 6*c
		hdr[h]++
		if hdrByCount[c] == nil {
			hdrByCount[c] = map[int]int{}
		}
		hdrByCount[c][h]++
		pairs++
	}
	fmt.Printf("\npaires exploitees : %d (ecartees : %d)\n", pairs, skipped)
	fmt.Println("\n=== ENTETE MESUREE = gap - largeur(dernier comp) - 6*count ===")
	fmt.Println("(attendu : 18 si idLow=11 sans maskSel ; 19 avec maskSel ; 20 si idLow=13 sans ; 21 avec)")
	top(hdr, 14)

	// TEST SANS AUCUNE LARGEUR ESTIMEE : le PREMIER record d'un paquet demarre au bit 0 du
	// payload (le decodeur offline le verifie : le walk part de 0). Pour lui,
	//   curseur(1er composant) = ENTETE + 6 x count   (masque epars)
	//   curseur(1er composant) = ENTETE_DENSE         (masque dense)
	// Un segment = suite de hits a curseur croissant ; un recul de curseur ouvre un paquet.
	firstHdr := map[int]int{}
	firstRaw := map[int]int{}
	segs := 0
	prevCursor := 1 << 30
	for i := 0; i < len(recs); i++ {
		c0 := recs[i].comps[0].cursor
		if c0 > prevCursor {
			prevCursor = recs[i].comps[len(recs[i].comps)-1].cursor
			continue
		}
		segs++
		n := len(recs[i].comps)
		firstRaw[c0]++
		if n <= 7 {
			firstHdr[c0-6*n]++
		}
		prevCursor = recs[i].comps[len(recs[i].comps)-1].cursor
	}
	fmt.Printf("\n=== PREMIER RECORD DE PAQUET (%d segments) : curseur(1er comp) - 6*count ===\n", segs)
	fmt.Println("(c'est l'ENTETE nue, aucune largeur estimee)")
	top(firstHdr, 10)
	fmt.Println("-- curseur brut du 1er composant du 1er record (dense = entete complete) --")
	top(firstRaw, 10)

	// TEST INDEPENDANT : les records a masque DENSE (gate=1 -> R(64), aucun index de 6
	// bits) ne peuvent porter plus de 7 composants en epars ; au-dela de 7 composants
	// captures le masque est FORCEMENT dense. Pour eux :
	//   gap - largeur(dernier) = 1 (prefixe) + idLow + 2 (tag) + 1 (gate) + 64
	// soit 68 + idLow. Cela mesure idLow SANS passer par la branche eparse.
	dense := map[int]int{}
	for i := 0; i+1 < len(recs); i++ {
		a, b := recs[i], recs[i+1]
		la := a.comps[len(a.comps)-1]
		fb := b.comps[0]
		if fb.cursor <= la.cursor || len(b.comps) <= 7 {
			continue
		}
		k := [2]int{a.ti, la.comp}
		w, ok := width[k]
		if !ok || widthConf[k] < 0.8 || w <= 0 {
			continue
		}
		dense[fb.cursor-la.cursor-w]++
	}
	fmt.Println("\n=== MASQUE DENSE (>7 composants) : gap - largeur(dernier) = 68 + idLow ===")
	fmt.Println("(82 -> idLow=14 ; 81 -> idLow=13 ; 79 -> idLow=11)")
	top(dense, 8)

	fmt.Println("\n-- entete par nombre de composants du record suivant (verifie le facteur 6) --")
	cs := []int{}
	for c := range hdrByCount {
		cs = append(cs, c)
	}
	sort.Ints(cs)
	for _, c := range cs {
		bv, bn, tot := 0, 0, 0
		for v, n := range hdrByCount[c] {
			tot += n
			if n > bn {
				bv, bn = v, n
			}
		}
		fmt.Printf("   count=%d : mode=%d  (%d/%d = %.1f %%)\n", c, bv, bn, tot, 100*float64(bn)/float64(tot))
	}
}

func top(h map[int]int, n int) {
	type kv struct{ k, v int }
	var a []kv
	tot := 0
	for k, v := range h {
		a = append(a, kv{k, v})
		tot += v
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	for i := 0; i < len(a) && i < n; i++ {
		fmt.Printf("   entete=%4d : %6d  (%.2f %%)\n", a[i].k, a[i].v, 100*float64(a[i].v)/float64(tot))
	}
}
