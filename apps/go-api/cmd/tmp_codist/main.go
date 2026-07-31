// tmp_codist — MESURER la distance en bits entre deux composants d'un MEME record.
//
// LA QUESTION QUI DECIDE. Le scan par catalogue est refute pour les champs etroits : 9 bits et
// 25 valeurs valides donnent une acceptation sur vingt, donc deux millions de candidats sur un
// film — la selectivite vient de la LARGEUR du champ, pas du catalogue (mesure du 2026-07-27,
// cmd/tmp_catscan). La seule voie qui reste est de COMBINER plusieurs champs etroits a distances
// imposees : quatre champs de 9 bits a des ecarts connus forment un champ de 36 bits.
//
// Mais cela ne vaut que si la distance est PREDICTIBLE. Or elle depend du masque : entre i22 et
// i47 se trouvent tous les composants d'indice intermediaire PRESENTS dans ce record, et le
// masque varie. Si la distance prend trente valeurs, le test ne selectionne rien.
//
// CE QUE CET OUTIL FAIT. Il lit la capture CE, regroupe les composants par record (meme entite,
// curseur croissant), et publie la distribution des ecarts de curseur entre deux composants
// donnes. C'est une mesure directe : le curseur EST la position, l'identite a ete etablie.
//
// CE QU'IL FAUT REGARDER : le nombre de valeurs distinctes de l'ecart. Peu de valeurs = test
// utilisable. Beaucoup = la voie de la co-occurrence a distance fixe est morte, et il faudra
// enumerer les masques plutot que les distances.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"sort"
)

const recSize = 40

type hit struct {
	EID, TypeIndex, CompIndex, Param4, BitCursor uint32
}

func main() {
	in := flag.String("in", "", "dump binaire de la capture CE")
	a := flag.Int("a", 22, "premier composant")
	b := flag.Int("b", 47, "second composant")
	ti := flag.Int("ti", 35, "archetype")
	flag.Parse()
	if *in == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_codist -in <capture.bin> [-a 22] [-b 47]")
		os.Exit(2)
	}
	hits, err := readHits(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Regroupement en records : meme entite, curseur qui avance. Regle conservatrice — elle peut
	// fusionner deux records consecutifs d'une meme entite, jamais couper un record en deux.
	type rec struct {
		eid  uint32
		comp map[uint32]uint32 // composant -> curseur
		mask []uint32
	}
	var recs []rec
	var cur *rec
	var lastCur uint32
	for _, h := range hits {
		if int(h.TypeIndex) != *ti {
			cur = nil
			continue
		}
		if cur == nil || cur.eid != h.EID || h.BitCursor < lastCur {
			if cur != nil {
				recs = append(recs, *cur)
			}
			cur = &rec{eid: h.EID, comp: map[uint32]uint32{}}
		}
		if _, seen := cur.comp[h.CompIndex]; !seen {
			cur.comp[h.CompIndex] = h.BitCursor
			cur.mask = append(cur.mask, h.CompIndex)
		}
		lastCur = h.BitCursor
	}
	if cur != nil {
		recs = append(recs, *cur)
	}

	fmt.Printf("DISTANCE i%d -> i%d dans un meme record (archetype %d)\n\n", *a, *b, *ti)
	fmt.Printf("  %d records reconstruits\n", len(recs))

	dist := map[int]int{}
	both, onlyA, onlyB := 0, 0, 0
	maskOf := map[string]int{}
	for _, r := range recs {
		ca, okA := r.comp[uint32(*a)]
		cb, okB := r.comp[uint32(*b)]
		switch {
		case okA && okB:
			both++
			dist[int(cb)-int(ca)]++
			m := append([]uint32(nil), r.mask...)
			sort.Slice(m, func(i, j int) bool { return m[i] < m[j] })
			maskOf[fmt.Sprint(m)]++
		case okA:
			onlyA++
		case okB:
			onlyB++
		}
	}
	fmt.Printf("  les deux presents : %d · i%d seul : %d · i%d seul : %d\n\n", both, *a, onlyA, *b, onlyB)
	if both == 0 {
		fmt.Println("  AUCUN record ne porte les deux : la co-occurrence n'est pas exploitable ainsi.")
		return
	}

	type kv struct{ d, n int }
	var rows []kv
	for d, n := range dist {
		rows = append(rows, kv{d, n})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	fmt.Printf("  %d ECARTS DISTINCTS sur %d couples —\n", len(rows), both)
	cum := 0
	for i, r := range rows {
		if i >= 14 {
			fmt.Printf("    … %d autres ecarts\n", len(rows)-14)
			break
		}
		cum += r.n
		fmt.Printf("    %+6d bits : %4d fois  %5.1f %%   (cumul %.1f %%)\n",
			r.d, r.n, 100*float64(r.n)/float64(both), 100*float64(cum)/float64(both))
	}

	fmt.Printf("\n  %d MASQUES DISTINCTS parmi les records portant les deux\n", len(maskOf))
	fmt.Println("\n  VERDICT —")
	switch {
	case len(rows) <= 4:
		fmt.Printf("    %d ecarts seulement : la distance EST predictible, le test combine est utilisable.\n", len(rows))
	case len(rows) <= 20:
		fmt.Printf("    %d ecarts : exploitable en enumerant les ecarts, au prix d'autant de tests.\n", len(rows))
	default:
		fmt.Printf("    %d ecarts : trop disperse pour un test a distance fixe. Il faudra enumerer\n", len(rows))
		fmt.Println("    les MASQUES (peu nombreux) plutot que les distances, et derouler le record.")
	}
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
