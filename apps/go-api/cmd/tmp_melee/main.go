// tmp_melee — MESURER la recette de detection des coups de melee avant de l'implementer.
//
// DEUX RECETTES EXISTENT ET ELLES DIVERGENT :
//   - .ai/GRENADE_MELEE_DETECTION.md : marqueur 0b10100110010 (11 bits), ancre a +3, type uint8
//     a l'ancre+76 dans {0x42, 0x47, 0x60}, index joueur a l'ancre+20.
//   - .ai/ETAT_DE_L_ART_KILLWEAPON.md (autre branche) : index joueur a l'ancre+23.
//
// Et le portage de reference de l'autre branche porte lui-meme un avertissement : « l'offset
// arme d'acurtis (88/86/101/103) ne valide rien sur notre film. Le type tombe (0x42/47/60 @+76)
// donc l'ancre est bonne. » Autrement dit l'ancre est etablie, l'offset de l'index ne l'est pas.
//
// CE QUE CETTE SONDE FAIT : elle ne choisit pas, elle MESURE. Elle balaye l'offset de l'index
// joueur sur une plage large et publie, pour chacun, la part de valeurs dans 0..7 et le nombre
// de joueurs distincts couverts.
//
// LE CRITERE, et il est le meme que celui qui a valide les grenades : le champ fait 5 bits, il
// peut donc porter 32 valeurs, et sur un match a 8 joueurs il DOIT tomber dans 0..7. Le hasard
// donne 25 %. Un offset juste doit ecraser ce niveau ET couvrir les 8 joueurs.
//
// CONTROLE NEGATIF : la meme mesure sur des positions de marqueur DECALEES d'un bit, qui doit
// s'effondrer si l'ancre porte bien l'information.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// meleeMarker est le motif de 11 bits qui precede un coup de melee. Il DIFFERE du marqueur des
// events de tir (0b10100100110) par deux bits.
const meleeMarker = 0b10100110010

// meleeMarkerBits, meleeAnchorSkip : l'ancre est a 3 bits dans le marqueur.
const (
	meleeMarkerBits = 11
	meleeAnchorSkip = 3
	meleeTypeOffset = 76
)

// meleeTypes est la liste blanche du type d'attaque. Toute autre valeur est un faux positif.
var meleeTypes = map[uint64]string{
	0x42: "non propulse (rate ou arme ordinaire)",
	0x47: "marteau gravitationnel",
	0x60: "epee energetique / coup non propulse touche",
}

func main() {
	dir := flag.String("dir", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache\film_chunks\000d5950`, "chunks du film")
	flag.Parse()

	type hit struct {
		ts   uint64
		pay  []byte
		anch int
		typ  uint64
	}
	var hits []hit
	markers := 0
	n := filmdec.CountFilmChunks(*dir)
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(*dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			pay := p.Payload(chunk)
			lim := len(pay)*8 - 260
			for bp := 0; bp <= lim; bp++ {
				if filmdec.PeekBits(pay, bp, meleeMarkerBits) != meleeMarker {
					continue
				}
				markers++
				a := bp + meleeAnchorSkip
				t := filmdec.PeekBits(pay, a+meleeTypeOffset, 8)
				if _, ok := meleeTypes[t]; !ok {
					continue
				}
				hits = append(hits, hit{p.TimestampUS, pay, a, t})
			}
		}
	}
	fmt.Printf("MELEE — %s\n\n", *dir)
	fmt.Printf("  %d marqueurs de 11 bits, %d retenus par la liste blanche des types (%.2f %%)\n",
		markers, len(hits), 100*float64(len(hits))/float64(max(1, markers)))
	if len(hits) == 0 {
		fmt.Fprintln(os.Stderr, "aucun coup retenu — la recette ne tient pas sur ce film")
		os.Exit(1)
	}

	byType := map[uint64]int{}
	for _, h := range hits {
		byType[h.typ]++
	}
	fmt.Println("\n  par type d'attaque :")
	for _, t := range []uint64{0x42, 0x47, 0x60} {
		if byType[t] > 0 {
			fmt.Printf("    0x%02X  %-46s %4d\n", t, meleeTypes[t], byType[t])
		}
	}

	// --- BALAYAGE de l'offset de l'index joueur -------------------------------------------
	fmt.Println("\n  BALAYAGE de l'offset de l'index joueur (5 bits depuis l'ancre)")
	fmt.Printf("  %6s  %8s  %10s  %s\n", "offset", "dans 0..7", "joueurs", "distribution")
	type row struct {
		off, in07, players int
	}
	var rows []row
	for off := 0; off <= 140; off++ {
		in07, seen := 0, map[uint64]bool{}
		for _, h := range hits {
			v := filmdec.PeekBits(h.pay, h.anch+off, 5)
			if v <= 7 {
				in07++
				seen[v] = true
			}
		}
		rows = append(rows, row{off, in07, len(seen)})
	}
	best := rows[0]
	for _, r := range rows {
		if r.in07 > best.in07 || (r.in07 == best.in07 && r.players > best.players) {
			best = r
		}
	}
	// on montre les offsets revendiques, le meilleur, et le voisinage du meilleur
	show := map[int]bool{20: true, 23: true, best.off: true,
		best.off - 1: true, best.off + 1: true}
	keys := make([]int, 0, len(show))
	for k := range show {
		if k >= 0 && k <= 140 {
			keys = append(keys, k)
		}
	}
	sort.Ints(keys)
	for _, off := range keys {
		r := rows[off]
		hist := map[uint64]int{}
		for _, h := range hits {
			hist[filmdec.PeekBits(h.pay, h.anch+off, 5)]++
		}
		var d []string
		for v := uint64(0); v <= 7; v++ {
			if hist[v] > 0 {
				d = append(d, fmt.Sprintf("%d:%d", v, hist[v]))
			}
		}
		lbl := ""
		switch off {
		case 20:
			lbl = "  <- revendique par GRENADE_MELEE_DETECTION"
		case 23:
			lbl = "  <- revendique par ETAT_DE_L_ART_KILLWEAPON"
		}
		if off == best.off {
			lbl += "  <== MEILLEUR"
		}
		fmt.Printf("  %6d  %5.1f %%  %6d/8   %s%s\n",
			off, 100*float64(r.in07)/float64(len(hits)), r.players,
			joinN(d, 8), lbl)
	}
	fmt.Printf("\n  NIVEAU DU HASARD : un champ de 5 bits tombe dans 0..7 une fois sur quatre = 25,0 %%\n")

	// --- CONTROLE NEGATIF : ancre decalee d'un bit -----------------------------------------
	for _, shift := range []int{-1, +1} {
		in07 := 0
		for _, h := range hits {
			if filmdec.PeekBits(h.pay, h.anch+best.off+shift, 5) <= 7 {
				in07++
			}
		}
		fmt.Printf("  CONTROLE ancre decalee de %+d bit : %.1f %%\n",
			shift, 100*float64(in07)/float64(len(hits)))
	}
}

func joinN(s []string, n int) string {
	out := ""
	for i, x := range s {
		if i >= n {
			out += "…"
			break
		}
		if i > 0 {
			out += " "
		}
		out += x
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
