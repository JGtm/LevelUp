// tmp_recgap — MESURER l'en-tete d'un record, et en isoler la largeur du DEFAULT-STATE.
//
// LE PROBLEME. Le gabarit rigide atteint 96,7 % de precision mais plafonne a 24 % de rappel,
// parce qu'il ne teste que les records DELTA. Les 75 % de lectures d'i22 qu'il rate vivent dans
// des records NEW (spawn : nouvelle entite, etat complet transmis) — etabli par param_4, qui
// vaut 3 vingt-quatre fois chez les gros records et ZERO sur 36 606 deltas.
//
// Le gabarit NEW est plus contraint que le DELTA (3 bits de prefixe au lieu d'1, plus 6 bits de
// typeIndex valant 35 pour un bipede), sauf sur un point : la largeur du DEFAULT-STATE, qui est
// une valeur de RUNTIME non sourcee statiquement.
//
// LA MESURE, ET POURQUOI ELLE EST POSSIBLE. Le crochet CE ne voit jamais le debut d'un record —
// il ne journalise que les composants. Mais deux records consecutifs d'un meme paquet se
// touchent : la fin de l'un est le debut de l'autre. Donc
//
//	ecart = curseur(premier composant du record N+1)
//	      - curseur(dernier composant du record N) - largeur(ce dernier composant)
//
// et cet ecart EST l'en-tete complet du record N+1 : prefixe + identifiant + masque, plus le
// default-state s'il s'agit d'un NEW.
//
// CE QUI REND LA DEDUCTION VALIDE : on connait la formule de l'en-tete DELTA
// (1 + idLow + 2 + 1 + 3 + 6*count, tous termes mesures ou calibres). L'ecart observe sur les
// records COURTS doit donc la reproduire — c'est le controle. S'il la reproduit, alors l'exces
// observe sur les records LONGS est le default-state, et rien d'autre.
//
// CE QUE CET OUTIL NE FAIT PAS : il ne suppose pas que les records longs sont des NEW. Il publie
// les deux distributions et laisse l'ecart parler. Si les deux se superposent, l'hypothese NEW
// tombe et il faudra chercher ailleurs.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"sort"
)

const recSize = 40

// compWidth : largeur dominante mesuree (capture CE, film 9e8fb31b).
var compWidth = map[uint32]int{
	0: 47, 1: 31, 2: 9, 4: 11, 5: 29, 6: 358, 9: 334, 13: 8, 14: 4, 15: 72,
	17: 52, 18: 2, 21: 25, 22: 35, 23: 19, 24: 12, 25: 10, 26: 22, 28: 10,
	30: 14, 31: 11, 32: 9, 33: 14, 34: 11, 35: 9, 42: 7, 43: 15, 44: 15,
	45: 17, 46: 17, 47: 9, 48: 10, 49: 3, 52: 68, 53: 31, 54: 2, 56: 10, 57: 2,
}

type hit struct {
	EID, TypeIndex, CompIndex, Param4, BitCursor uint32
}

type rec struct {
	eid                 uint32
	firstCur, lastCur   uint32
	lastComp            uint32
	nComp               int
	hasUnknownLastWidth bool
	param4Set           map[uint32]bool
}

func main() {
	in := flag.String("in", "", "dump binaire de la capture CE")
	ti := flag.Int("ti", 35, "archetype")
	idLow := flag.Int("idlow", 10, "largeur du champ identifiant bas (calibree par le gabarit)")
	flag.Parse()
	if *in == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_recgap -in <capture.bin> [-ti 35] [-idlow 10]")
		os.Exit(2)
	}
	hits, err := readHits(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Reconstruction des records. Un nouveau record commence quand l'entite change OU que le
	// curseur recule (autre flux). Regle conservatrice, deja eprouvee.
	var recs []rec
	var cur *rec
	for _, h := range hits {
		if int(h.TypeIndex) != *ti {
			if cur != nil {
				recs = append(recs, *cur)
				cur = nil
			}
			continue
		}
		if cur == nil || cur.eid != h.EID || h.BitCursor < cur.lastCur {
			if cur != nil {
				recs = append(recs, *cur)
			}
			cur = &rec{eid: h.EID, firstCur: h.BitCursor, param4Set: map[uint32]bool{}}
		}
		cur.lastCur = h.BitCursor
		cur.lastComp = h.CompIndex
		cur.nComp++
		cur.param4Set[h.Param4] = true
	}
	if cur != nil {
		recs = append(recs, *cur)
	}
	fmt.Printf("EN-TETE DE RECORD — %d records reconstruits (archetype %d)\n\n", len(recs), *ti)

	// ECART entre la fin d'un record et le premier composant du suivant.
	shortGap := map[int]int{}
	longGap := map[int]int{}
	nS, nL, skipped := 0, 0, 0
	for i := 0; i+1 < len(recs); i++ {
		a, b := recs[i], recs[i+1]
		if a.eid == b.eid || b.firstCur < a.lastCur {
			continue // pas deux records consecutifs du meme flux
		}
		w, ok := compWidth[a.lastComp]
		if !ok {
			skipped++
			continue // largeur du dernier composant inconnue : ecart indeterminable
		}
		gap := int(b.firstCur) - int(a.lastCur) - w
		if gap < 0 || gap > 4096 {
			continue
		}
		if b.nComp > 7 {
			longGap[gap]++
			nL++
		} else {
			shortGap[gap]++
			nS++
		}
	}
	fmt.Printf("  %d ecarts sur records COURTS (<= 7 composants) · %d sur records LONGS (> 7)\n", nS, nL)
	fmt.Printf("  %d couples ecartes (largeur du dernier composant inconnue)\n\n", skipped)

	// CONTROLE : l'ecart des records COURTS doit reproduire la formule de l'en-tete DELTA.
	fmt.Println("  ECARTS des records COURTS — doivent valoir 1 + idLow + 2 + 1 + 3 + 6*count")
	fmt.Printf("  soit %d + 6*count : count=1 -> %d · 2 -> %d · 3 -> %d · 4 -> %d · 5 -> %d\n\n",
		1+*idLow+2+1+3, 7+*idLow+6, 7+*idLow+12, 7+*idLow+18, 7+*idLow+24, 7+*idLow+30)
	printTop(shortGap, nS, 12)

	fmt.Println("\n  ECARTS des records LONGS — l'exces sur les courts EST le default-state")
	printTop(longGap, nL, 12)

	if nS > 0 && nL > 0 {
		ms, ml := median(shortGap, nS), median(longGap, nL)
		fmt.Printf("\n  mediane COURTS %d bits · mediane LONGS %d bits · ECART %+d bits\n", ms, ml, ml-ms)
		fmt.Println("\n  LECTURE — si les deux distributions se superposent, l'hypothese NEW tombe et")
		fmt.Println("  les records longs ne sont pas d'une autre nature. Si l'ecart est net et stable,")
		fmt.Println("  il donne la largeur du default-state, a corriger de la difference d'en-tete")
		fmt.Println("  (NEW = 3 bits de prefixe + 6 de typeIndex contre 1 bit pour DELTA).")
	}
}

func printTop(m map[int]int, tot, n int) {
	type kv struct{ g, n int }
	var rows []kv
	for g, c := range m {
		rows = append(rows, kv{g, c})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	for i, r := range rows {
		if i >= n {
			fmt.Printf("    … %d autres ecarts\n", len(rows)-n)
			break
		}
		fmt.Printf("    %5d bits : %5d fois  %5.1f %%\n", r.g, r.n, 100*float64(r.n)/float64(max(1, tot)))
	}
}

func median(m map[int]int, tot int) int {
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	seen := 0
	for _, k := range ks {
		seen += m[k]
		if seen*2 >= tot {
			return k
		}
	}
	return 0
}

func readHits(path string) ([]hit, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture %s : %w", path, err)
	}
	out := make([]hit, 0, len(raw)/recSize)
	for i := 0; i+recSize <= len(raw); i += recSize {
		b := raw[i : i+recSize]
		var h hit
		h.EID = binary.LittleEndian.Uint32(b[0:])
		h.TypeIndex = binary.LittleEndian.Uint32(b[4:])
		h.CompIndex = binary.LittleEndian.Uint32(b[8:])
		h.Param4 = binary.LittleEndian.Uint32(b[12:])
		h.BitCursor = binary.LittleEndian.Uint32(b[16:])
		if h.EID == 0 && h.TypeIndex == 0 && h.BitCursor == 0 && h.CompIndex == 0 {
			break
		}
		out = append(out, h)
	}
	return out, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
