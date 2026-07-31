package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// widths.go — PASSE 1.2 du plan : mesurer la largeur consommee par chaque composant
// i0..i21 de l'archetype 5, par difference de curseur entre composants consecutifs.
//
// LE CRITERE EST UNE LARGEUR, PAS UN CONTENU. Un record de keyframe est borne par le
// debut du record suivant (WalkKeyframeWorld les emet en ordre). La chaine de composants
// doit donc ATTERRIR AU BIT PRES sur cette borne. Aucune valeur decodee n'entre dans le
// critere — c'est la meme discipline que solveAmmoBlock du chantier inventaire.
//
// STRUCTURE SUPPOSEE, et pourquoi elle est testable. L'en-tete keyframe fait 64 bits :
// [id:32][field26:26][ti:6]. Les 6 derniers bits sont exactement ce que TraverseEntity
// lit en premier. On place donc le curseur a from+58 et on laisse TraverseEntity derouler
// sa sequence normale — default-state, porte, masque, composants. Si la supposition est
// fausse, aucun default-state ne fera atterrir la chaine sur la borne, et on le verra.

func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

// traceAt deroule la chaine de composants d'un record de keyframe, en supposant un
// default-state de largeur dsb. Le curseur part de from+58 (les 6 bits de ti).
func traceAt(pay []byte, from, dsb int, reg *filmdec.Registry) filmdec.EntityTrace {
	br := filmdec.NewBitReader(pay)
	br.SetBitPos(from + 58)
	return filmdec.TraverseEntity(br, reg, dsb)
}

// solveDefaultState cherche les largeurs de default-state pour lesquelles la chaine
// atterrit EXACTEMENT sur `to` sans desync. Rend toutes les solutions : une seule = la
// grammaire est contrainte, plusieurs = le critere ne suffit pas et il faut le dire.
func solveDefaultState(pay []byte, from, to int, reg *filmdec.Registry, maxDSB int) []int {
	var sols []int
	for dsb := 0; dsb <= maxDSB; dsb++ {
		t := traceAt(pay, from, dsb, reg)
		if t.DesyncAt == -1 && t.EndBit == to {
			sols = append(sols, dsb)
		}
	}
	return sols
}

// runKFChain — TEMOIN NEGATIF : resoudre la largeur du default-state de ti=5, puis publier la
// largeur de chaque composant du record.
func runKFChain(kfs []kfView, dir string) {
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "registre illisible: %v\n", err)
		os.Exit(1)
	}
	arch, ok := reg.Archetype(playerTI)
	if !ok {
		fmt.Fprintf(os.Stderr, "archetype %d absent du registre\n", playerTI)
		os.Exit(1)
	}
	fmt.Printf("=== 1.2 LARGEURS — archetype %d, %d composants\n\n", playerTI, len(arch.Components))

	// 1) resolution du default-state sur TOUS les records ti=5 : l'intersection des
	//    solutions est la largeur commune, s'il en existe une.
	inter := map[int]int{}
	nRec, nSolved := 0, 0
	solCount := map[int]int{}
	for _, kf := range kfs {
		for _, s := range kf.sp {
			if s.ti != playerTI {
				continue
			}
			nRec++
			sols := solveDefaultState(kf.pay, s.from, s.to, reg, 260)
			solCount[len(sols)]++
			if len(sols) > 0 {
				nSolved++
			}
			for _, d := range sols {
				inter[d]++
			}
		}
	}
	fmt.Printf("records ti=5 examines            : %d\n", nRec)
	fmt.Printf("records dont la chaine atterrit  : %d\n", nSolved)
	fmt.Printf("nombre de solutions par record   : %s\n", fmtCount(solCount))
	fmt.Printf("\ndefault-state candidats (largeur : nb de records ou elle marche) :\n")
	var ds []int
	for d := range inter {
		ds = append(ds, d)
	}
	sort.Ints(ds)
	best, bestN := -1, 0
	for _, d := range ds {
		if inter[d] > bestN {
			bestN, best = inter[d], d
		}
		if inter[d] >= nRec/4 {
			fmt.Printf("   %4d bits : %d / %d records\n", d, inter[d], nRec)
		}
	}
	if best < 0 {
		fmt.Println("\nAUCUNE largeur de default-state ne fait atterrir la chaine.")
		fmt.Println("=> la structure supposee (ti a from+58, puis default-state, porte, masque) est FAUSSE.")
		return
	}
	fmt.Printf("\nretenu : default-state = %d bits (%d / %d records)\n\n", best, bestN, nRec)

	// 2) largeur de chaque composant, sur le premier record qui atterrit avec `best`.
	dumpComponentWidths(kfs, reg, arch, best)
}

// dumpComponentWidths affiche la largeur de chaque composant, mesuree par difference de
// curseur, sur le premier record LARGE (joueur actif) qui atterrit proprement.
func dumpComponentWidths(kfs []kfView, reg *filmdec.Registry, arch filmdec.Archetype, dsb int) {
	// distribution du rang atteint, sur tous les records
	reach := map[int]int{}
	var sample *filmdec.EntityTrace
	var sampleSlot, sampleKF int
	for ki, kf := range kfs {
		for _, s := range kf.sp {
			if s.ti != playerTI {
				continue
			}
			t := traceAt(kf.pay, s.from, dsb, reg)
			if t.DesyncAt == -1 && t.EndBit == s.to {
				reach[len(t.Comps)]++
				if sample == nil && s.to-s.from > 600 { // record large = joueur actif
					cp := t
					sample, sampleSlot, sampleKF = &cp, s.slot, ki+1
				}
			} else {
				reach[-1]++
			}
		}
	}
	fmt.Printf("composants presents par record (nb de composants : nb de records) : %s\n\n", fmtCount(reach))
	if sample == nil {
		fmt.Println("aucun record LARGE (> 600 bits) n'atterrit proprement — pas d'echantillon a detailler.")
		return
	}
	fmt.Printf("--- detail : image-cle %d, slot %d, masque %#x, %d composants presents ---\n",
		sampleKF, sampleSlot, sample.Mask, len(sample.Comps))
	fmt.Printf("%-5s %-8s %-8s %s\n", "rang", "debut", "largeur", "composant")
	for i, c := range sample.Comps {
		w := sample.EndBit - c.StartBit
		if i+1 < len(sample.Comps) {
			w = sample.Comps[i+1].StartBit - c.StartBit
		}
		fmt.Printf("i%-4d %-8d %-8d %s\n", c.Index, c.StartBit, w, c.Name)
	}
	_ = arch
}

func fmtCount(m map[int]int) string {
	var ks []int
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	s := ""
	for _, k := range ks {
		s += fmt.Sprintf("%d:%d ", k, m[k])
	}
	return s
}
