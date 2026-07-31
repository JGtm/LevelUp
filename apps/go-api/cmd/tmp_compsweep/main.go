// tmp_compsweep — LA RECETTE : pour CHAQUE composant du bipede, sa position exacte dans le film,
// sa largeur reelle, et la distribution de ses valeurs. En un seul parcours.
//
// CE QUE CET OUTIL PRODUIT, et pourquoi c'est le livrable attendu. Jusqu'ici chaque composant
// demandait sa propre enquete. Ici on publie, en une passe, le tableau complet :
//
//	composant · nombre de lectures · largeur mesuree · valeurs distinctes · bits porteurs
//
// De quoi decider, composant par composant, ce qui est exploitable pour le rejeu 2D et ce qui
// ne l'est pas — sans supposer, sans extrapoler d'un composant a l'autre.
//
// LES DEUX IDENTITES SUR LESQUELLES IL REPOSE, toutes deux etablies par mesure le 2026-07-27 :
//
//  1. POSITION. Le curseur de bits capture EST la position absolue dans le payload du paquet,
//     decalage NUL : `position = paquet.Start*8 + curseur`. Etabli par balayage de l'amorce sur
//     0..8 — seul +0 rend un parse valide, et il en rend 249 sur 249 sur i22.
//  2. LARGEUR. La difference de curseur entre un composant et le suivant DU MEME record EST la
//     largeur consommee, exactement.
//
// La signature de 16 octets ne sert qu'a rattacher une lecture capturee a SON PAQUET dans le
// film. C'est le curseur qui donne le bit.
//
// POURQUOI UNE TABLE DE HACHAGE. Chercher chaque signature par balayage coute
// (nb signatures) x (taille du film). Avec 64 composants et quelques dizaines de lectures
// chacun, cela ferait des dizaines de Go de parcours. On indexe donc les signatures par leurs 8
// premiers octets et on parcourt le film UNE fois, en verifiant les 16 octets complets a chaque
// collision — le hachage selectionne, il ne decide pas.
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const recSize = 40

// minDistinct ecarte les fenetres de bourrage : elles n'identifient rien et polluent l'index.
const minDistinct = 12

type hit struct {
	EID, TypeIndex, CompIndex, Param4, BitCursor uint32
	Sig                                          [16]byte
}

// stat agrege ce qu'on sait d'un composant.
type stat struct {
	Reads     int            // lectures dans la capture
	Located   int            // lectures retrouvees dans le film
	Widths    map[int]int    // largeur consommee -> occurrences
	Values    map[uint64]int // valeur (sur la largeur dominante) -> occurrences
	BitOnes   []int          // nombre de 1 par position de bit
	BitTotal  int            // lectures ayant servi au comptage de bits
	SampleEID uint32
}

func main() {
	in := flag.String("in", "", "dump binaire de la capture CE")
	dir := flag.String("film", "", "dossier des chunks du film")
	ti := flag.Int("ti", 35, "archetype a analyser")
	perComp := flag.Int("per", 60, "lectures echantillonnees par composant")
	maxW := flag.Int("maxbits", 48, "largeur maximale relue pour la distribution des valeurs")
	flag.Parse()
	if *in == "" || *dir == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_compsweep -in <capture.bin> -film <dir> [-ti 35]")
		os.Exit(2)
	}

	hits, err := readHits(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// LARGEURS : mesurees sur TOUTES les lectures, pas sur l'echantillon — c'est gratuit et plus
	// solide. Un couple compte si les deux composants appartiennent a la meme entite et que le
	// curseur avance.
	widths := map[uint32]map[int]int{}
	reads := map[uint32]int{}
	for i, h := range hits {
		if int(h.TypeIndex) != *ti {
			continue
		}
		reads[h.CompIndex]++
		if i+1 >= len(hits) {
			continue
		}
		n := hits[i+1]
		if n.EID != h.EID || n.BitCursor < h.BitCursor {
			continue
		}
		w := int(n.BitCursor - h.BitCursor)
		if w <= 0 || w > 4096 {
			continue
		}
		if widths[h.CompIndex] == nil {
			widths[h.CompIndex] = map[int]int{}
		}
		widths[h.CompIndex][w]++
	}

	// ECHANTILLON a localiser : quelques lectures par composant suffisent pour la distribution
	// des valeurs, et cela borne le cout de l'indexation.
	type want struct {
		comp uint32
		h    hit
	}
	index := map[uint64][]want{}
	taken := map[uint32]int{}
	nSig := 0
	for _, h := range hits {
		if int(h.TypeIndex) != *ti || !usable(h.Sig) {
			continue
		}
		if taken[h.CompIndex] >= *perComp {
			continue
		}
		taken[h.CompIndex]++
		k := binary.LittleEndian.Uint64(h.Sig[:8])
		index[k] = append(index[k], want{h.CompIndex, h})
		nSig++
	}

	fmt.Printf("BALAYAGE DES COMPOSANTS — archetype %d\n\n", *ti)
	fmt.Printf("  %d lectures capturees · %d signatures indexees (max %d par composant)\n",
		len(hits), nSig, *perComp)

	// UN SEUL PARCOURS du film. A chaque position on lit 8 octets, on interroge l'index, et on
	// ne confirme que si les 16 octets complets coincident.
	nc := filmdec.CountFilmChunks(*dir)
	st := map[uint32]*stat{}
	for c := 1; c <= nc; c++ {
		chunk, err := filmdec.ReadFilmChunk(*dir, c)
		if err != nil {
			continue
		}
		type pk struct{ start, size int }
		var packets []pk
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type == filmdec.PacketTypeDelta {
				packets = append(packets, pk{p.Start, p.Size})
			}
		}
		for off := 0; off+16 <= len(chunk); off++ {
			cands, ok := index[binary.LittleEndian.Uint64(chunk[off:off+8])]
			if !ok {
				continue
			}
			for _, w := range cands {
				if !bytes.Equal(chunk[off:off+16], w.h.Sig[:]) {
					continue
				}
				var f *pk
				for i := range packets {
					if off >= packets[i].start && off < packets[i].start+packets[i].size {
						f = &packets[i]
						break
					}
				}
				if f == nil {
					continue
				}
				pay := chunk[f.start : f.start+f.size]
				cur := int(w.h.BitCursor)
				if st[w.comp] == nil {
					st[w.comp] = &stat{Widths: map[int]int{}, Values: map[uint64]int{},
						BitOnes: make([]int, *maxW), SampleEID: w.h.EID}
				}
				s := st[w.comp]
				s.Located++
				rw := dominantWidth(widths[w.comp])
				if rw <= 0 || rw > *maxW {
					rw = *maxW
				}
				if cur >= 0 && cur+rw <= len(pay)*8 {
					s.Values[filmdec.PeekBits(pay, cur, rw)]++
					s.BitTotal++
					for i := 0; i < rw && i < *maxW; i++ {
						if filmdec.PeekBits(pay, cur+i, 1) == 1 {
							s.BitOnes[i]++
						}
					}
				}
			}
		}
	}

	comps := make([]uint32, 0, len(reads))
	for c := range reads {
		comps = append(comps, c)
	}
	sort.Slice(comps, func(i, j int) bool { return comps[i] < comps[j] })

	fmt.Printf("\n  %-5s %9s %9s %11s %10s  %s\n",
		"comp", "lectures", "localis.", "largeur", "valeurs", "note")
	for _, c := range comps {
		s := st[c]
		loc, nv := 0, 0
		if s != nil {
			loc, nv = s.Located, len(s.Values)
		}
		w := dominantWidth(widths[c])
		wtxt := "?"
		if w > 0 {
			wtxt = fmt.Sprintf("%d bits", w)
			if len(widths[c]) > 1 {
				wtxt += fmt.Sprintf(" (%d)", len(widths[c]))
			}
		}
		note := ""
		if nv > 0 && nv <= 16 {
			note = "PEU DE VALEURS -> enumerable"
		}
		if s != nil && s.BitTotal > 0 {
			if d := deadBits(s, w); d > 0 {
				note += fmt.Sprintf("  %d bits toujours nuls", d)
			}
		}
		fmt.Printf("  i%-4d %9d %9d %11s %10d  %s\n", c, reads[c], loc, wtxt, nv, note)
	}

	fmt.Println("\n  LECTURE DU TABLEAU —")
	fmt.Println("    « lectures »  ce que le moteur a deserialise sur tout le film")
	fmt.Println("    « localis. »  echantillon retrouve dans le film (borne par -per)")
	fmt.Println("    « largeur »   mesuree par difference de curseurs ; (n) = n largeurs distinctes")
	fmt.Println("    « valeurs »   valeurs distinctes sur l'echantillon, a la largeur dominante")
	fmt.Println("    Un composant a PEU DE VALEURS est enumerable donc exploitable tout de suite ;")
	fmt.Println("    des bits toujours nuls signalent un champ plus etroit que sa largeur.")
}

// dominantWidth rend la largeur majoritaire, ou 0 si aucune mesure.
func dominantWidth(m map[int]int) int {
	best, bestN := 0, 0
	for w, n := range m {
		if n > bestN {
			bestN, best = n, w
		}
	}
	return best
}

// deadBits compte les positions de bit jamais a 1 : elles bornent la taille reelle du champ.
func deadBits(s *stat, w int) int {
	if w <= 0 || w > len(s.BitOnes) {
		return 0
	}
	n := 0
	for i := 0; i < w; i++ {
		if s.BitOnes[i] == 0 {
			n++
		}
	}
	return n
}

func usable(s [16]byte) bool {
	seen := map[byte]bool{}
	for _, b := range s {
		seen[b] = true
	}
	return len(seen) >= minDistinct
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
		copy(h.Sig[:], b[24:40])
		if h.EID == 0 && h.TypeIndex == 0 && h.BitCursor == 0 && h.CompIndex == 0 {
			break
		}
		out = append(out, h)
	}
	return out, nil
}
