package main

import (
	"fmt"
	"sort"
)

// solve.go — PASSE 1.3 voie (b), SECONDE FORME : le handle n'est pas un mot de 32 bits.
//
// CE QUE LES DEUX PASSES PRECEDENTES ONT ELIMINE, et c'est un acquis, pas un echec :
//   - la chaine sequentielle du flux delta atteint i21 mais y lit du BRUIT (slots a 8e8,
//     generations uniformes sur 0..3) : le parcours derive avant d'y arriver ;
//   - AUCUN mot de 32 bits aligne, dans les 832 records ti=5 des images-cles, ne vaut un
//     handle de biped. La forme « R(32) brut » est donc exclue, pas seulement improbable.
//
// LA FORME RESTANTE est celle que le format emploie partout ailleurs pour un handle :
// readRecordID lit `low = R(idLowBits)` puis `tag = R(2)`, et compose id = (tag<<30)|low.
// C'est aussi ce que le port d'i10 (player-primary-respawn-object) decrit deja :
// « R(1)[si1 : R(W)+R(2) runtime] ». Un handle coute donc ~13 bits, pas 32.
//
// POURQUOI UNE RESOLUTION PAR CONTRAINTE ET PAS UN BALAYAGE. A 11 bits de slot, le
// predicat « slot est un biped connu » a une selectivite de ~3 %, soit une quinzaine de
// touches par record : inexploitable seul. La contrainte qui tranche est STRUCTURELLE et
// ne depend d'aucune valeur : a une image-cle donnee, les huit joueurs vivants designent
// huit bipeds DISTINCTS, et ils le font a la MEME position dans leur record, puisque
// c'est le meme archetype et le meme composant. On cherche donc un couple (largeur,
// decalage) qui produit huit slots biped distincts — pas une valeur qui nous plairait.

// solveHit est un couple (largeur de slot, decalage) et son score sur une image-cle.
type solveHit struct {
	width    int // nb de bits du champ slot
	off      int // decalage, depuis le debut (fromEnd=false) ou la fin (fromEnd=true)
	fromEnd  bool
	valid    int // nb de records joueur ou la lecture donne un slot biped connu
	distinct int // nb de slots DISTINCTS parmi ceux-la
	tagOK    int // nb de lectures dont le tag vaut 1..3
	fullKF   int // nb d'images-cles ou TOUS les joueurs designent un biped distinct
}

// playerRecords rend les records ti=5 « larges » (joueur actif) d'une image-cle, tries.
func playerRecords(kf kfView) []recSpan {
	var out []recSpan
	for _, s := range kf.sp {
		if s.ti == playerTI && s.to-s.from > 600 {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].slot < out[j].slot })
	return out
}

// scoreAt lit (slot,tag) a un decalage donne dans chaque record joueur et rend le score.
func scoreAt(kf kfView, recs []recSpan, biped map[int]bool, width, off int, fromEnd bool) solveHit {
	h := solveHit{width: width, off: off, fromEnd: fromEnd}
	seen := map[int]bool{}
	for _, s := range recs {
		p := s.from + off
		if fromEnd {
			p = s.to - off
		}
		if p < s.from || p+width+2 > s.to {
			continue
		}
		slot := int(bits(kf.pay, p, width))
		tag := int(bits(kf.pay, p+width, 2))
		if !biped[slot] {
			continue
		}
		h.valid++
		if tag >= 1 && tag <= 3 {
			h.tagOK++
		}
		if !seen[slot] {
			seen[slot] = true
			h.distinct++
		}
	}
	return h
}

// runSolve — 1.3(b) : chercher le couple (largeur, decalage) qui fait designer, aux huit
// joueurs d'une image-cle, huit bipeds distincts.
func runSolve(kfs []kfView) {
	fmt.Printf("=== 1.3(b) RESOLUTION PAR CONTRAINTE — handle compresse R(W)+R(2)\n\n")

	// LE CATALOGUE EST LOCAL A L'IMAGE-CLE, et c'est ce qui fait la selectivite. Un
	// catalogue global (90 slots sur tout le film) a ete essaye d'abord : son temoin l'a
	// refuse — 91,6 % de « liens » sur les records ti=5 VIDES, contre 87,5 % sur les
	// joueurs. Un predicat que le vide satisfait mieux que le plein ne mesure rien.
	// Les bipeds d'UNE image-cle sont sept en moyenne : la selectivite passe de ~9 % a ~1 %,
	// et l'exigence de huit lectures simultanement distinctes la porte a ~1e-16.
	globalCount := 0
	for _, kf := range kfs {
		globalCount += len(bipedSlotsOf(kf))
	}
	fmt.Printf("bipeds par image-cle (moyenne) : %.1f\n", float64(globalCount)/float64(len(kfs)))

	// Score cumule par (largeur, decalage, sens) sur toutes les images-cles : une position
	// reelle est bonne PARTOUT, une coincidence ne l'est qu'ici ou la.
	type key struct {
		w, off  int
		fromEnd bool
	}
	agg := map[key]*solveHit{}
	nKF := 0
	for _, kf := range kfs {
		recs := playerRecords(kf)
		if len(recs) < 8 {
			continue
		}
		nKF++
		local := bipedSlotsOf(kf)
		for _, w := range []int{10, 11, 12, 13, 14} {
			for _, fromEnd := range []bool{false, true} {
				maxOff := 700
				for off := 0; off <= maxOff; off++ {
					h := scoreAt(kf, recs, local, w, off, fromEnd)
					if h.distinct == 0 {
						continue
					}
					k := key{w, off, fromEnd}
					if agg[k] == nil {
						agg[k] = &solveHit{width: w, off: off, fromEnd: fromEnd}
					}
					agg[k].valid += h.valid
					agg[k].distinct += h.distinct
					agg[k].tagOK += h.tagOK
					if h.distinct == len(recs) {
						agg[k].fullKF++
					}
				}
			}
		}
	}
	fmt.Printf("images-cles a 8 joueurs actifs : %d (maximum theorique %d liens)\n\n", nKF, nKF*8)

	var all []*solveHit
	for _, h := range agg {
		all = append(all, h)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].fullKF != all[j].fullKF {
			return all[i].fullKF > all[j].fullKF
		}
		if all[i].distinct != all[j].distinct {
			return all[i].distinct > all[j].distinct
		}
		return all[i].off < all[j].off
	})
	fmt.Printf("--- meilleurs couples (largeur, decalage) par nombre de slots distincts ---\n")
	fmt.Printf("%-8s %-8s %-8s %-10s %-10s %s\n", "largeur", "sens", "decalage", "valides", "distincts", "tag 1..3")
	for i, h := range all {
		if i >= 15 {
			break
		}
		sens := "debut"
		if h.fromEnd {
			sens = "fin"
		}
		fmt.Printf("%-8d %-8s %-8d %-10d %-10d %d\n", h.width, sens, h.off, h.valid, h.distinct, h.tagOK)
	}
	if len(all) == 0 {
		fmt.Println("aucun couple ne produit le moindre slot biped : la forme compressee est exclue aussi.")
		return
	}
	best := all[0]
	fmt.Printf("\nmeilleur : largeur %d, %s +%d -> %d liens distincts sur %d possibles (%.1f %%)\n",
		best.width, map[bool]string{false: "debut", true: "fin"}[best.fromEnd], best.off,
		best.distinct, nKF*8, pct(best.distinct, nKF*8))

	// TEMOIN : le meme couple applique aux records ti=5 VIDES (slots 60..83). Ils ne
	// portent pas de joueur, donc ils ne doivent pas produire de lien. Un score comparable
	// signifierait que la contrainte est satisfaite par le hasard, et non par le format.
	emptyDistinct, emptyTotal := 0, 0
	for _, kf := range kfs {
		var empty []recSpan
		for _, s := range kf.sp {
			if s.ti == playerTI && s.to-s.from <= 600 {
				empty = append(empty, s)
			}
		}
		if len(empty) == 0 {
			continue
		}
		emptyTotal += len(empty)
		h := scoreAt(kf, empty, bipedSlotsOf(kf), best.width, best.off, best.fromEnd)
		emptyDistinct += h.distinct
	}
	fmt.Printf("TEMOIN (memes largeur/decalage sur les records ti=5 VIDES) : %d liens sur %d records (%.1f %%)\n",
		emptyDistinct, emptyTotal, pct(emptyDistinct, emptyTotal))

	// REPARTITION : 26 liens sur 200 peuvent etre huit joueurs lus proprement sur trois
	// images-cles (un signal, avec une condition de presence a trouver) ou un ou deux liens
	// eparpilles sur vingt-cinq (du bruit qui passe le temoin par chance). La difference
	// n'est pas dans le total, elle est dans la FORME de la repartition.
	fmt.Printf("\n--- repartition du meilleur candidat, image-cle par image-cle ---\n")
	full := 0
	for ki, kf := range kfs {
		recs := playerRecords(kf)
		if len(recs) < 8 {
			continue
		}
		local := bipedSlotsOf(kf)
		var got []string
		for _, s := range recs {
			p := s.from + best.off
			if best.fromEnd {
				p = s.to - best.off
			}
			if p < s.from || p+best.width+2 > s.to {
				continue
			}
			slot := int(bits(kf.pay, p, best.width))
			if !local[slot] {
				continue
			}
			got = append(got, fmt.Sprintf("%d->%d", s.slot, slot))
		}
		if len(got) == len(recs) {
			full++
		}
		fmt.Printf("kf%-3d %d/%d joueurs lus  %v\n", ki+1, len(got), len(recs), got)
	}
	fmt.Printf("\nimages-cles ou LES HUIT joueurs sont lus : %d sur %d\n", full, nKF)
}
