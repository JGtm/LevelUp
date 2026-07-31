package main

import (
	"fmt"
	"sort"
)

// anchor.go — PASSE 1.3 voie (b) : ANCRAGE PAR SIGNATURE.
//
// POURQUOI CETTE VOIE. La voie (a) — prolonger la chaine sequentielle — a ete mesuree
// (mode `delta`) : elle atteint i21 avec la bonne largeur (32 bits, 34 fois sur 34) mais
// le flux delta ne rencontre que 125 records ti=5 sur tout le film et desynchronise dans
// 47 % des cas. Le plafond n'est pas dans le deser d'i21, il est dans le parcours.
//
// LA SIGNATURE, et pourquoi elle est forte. i21 porte un HANDLE d'entite : les deux bits
// hauts sont la generation (1..3), les 30 bas le slot. On cherche donc, dans l'emprise
// d'un record ti=5, les mots de 32 bits dont le slot appartient a l'ensemble des slots
// BIPED (ti=35) de la MEME image-cle. Ce catalogue vient du record walker, pas de nous.
//
// PROBABILITE PAR HASARD : 3/4 (generation) x 90/2^30 (slots biped connus) = 6,3e-8 par
// position. Le film porte ~30 millions de bits ; l'esperance de faux positifs sur un
// balayage complet est de l'ordre de 2. C'est le meme ordre de selectivite que l'ancrage
// des familles d'arme (911 occurrences pour 0,52 attendue), et le meme geste.
//
// CE QUE LA MESURE DOIT MONTRER, sans quoi la voie est refusee :
//   - le nombre de touches PAR RECORD est petit et stable (une signature, pas du bruit) ;
//   - les 8 slots joueur actifs touchent, les 24 slots vides ne touchent pas ;
//   - a une image-cle donnee, deux joueurs ne designent JAMAIS le meme biped (injectivite).

// hit est une touche de signature dans l'emprise d'un record.
type hit struct {
	bit  int    // position du premier bit du mot
	off  int    // decalage depuis le debut du record
	word uint32 // le handle complet
	slot int    // word & 0x3fffffff
	gen  int    // word >> 30
}

// bipedSlotsOf rend l'ensemble des slots d'archetype biped d'une image-cle.
func bipedSlotsOf(kf kfView) map[int]bool {
	m := map[int]bool{}
	for _, s := range kf.sp {
		if s.ti == bipedTI {
			m[s.slot] = true
		}
	}
	return m
}

// handlesIn balaye [from,to) et rend les mots de 32 bits qui sont des handles de biped.
func handlesIn(pay []byte, from, to int, biped map[int]bool) []hit {
	var out []hit
	var w uint32
	for b := from; b < to; b++ {
		w = w<<1 | bitAt(pay, b)
		if b-from < 31 {
			continue
		}
		gen := int(w >> 30)
		if gen == 0 {
			continue
		}
		slot := int(w & 0x3fffffff)
		if !biped[slot] {
			continue
		}
		out = append(out, hit{bit: b - 31, off: b - 31 - from, word: w, slot: slot, gen: gen})
	}
	return out
}

// runAnchor — 1.3(b) et 1.4 : la signature designe-t-elle une entite, et est-elle unique ?
func runAnchor(kfs []kfView) {
	fmt.Printf("=== 1.3(b) ANCRAGE PAR SIGNATURE — %d images-cles\n\n", len(kfs))

	// A. selectivite : combien de touches par record, et sur quels slots joueur ?
	hitsPerRec := map[int]int{}
	bySlot := map[int]map[int]int{} // slot joueur -> nb de touches -> compte
	offDist := map[int]int{}        // decalage depuis le debut du record -> compte
	var nRec, nLarge int
	for _, kf := range kfs {
		biped := bipedSlotsOf(kf)
		for _, s := range kf.sp {
			if s.ti != playerTI {
				continue
			}
			nRec++
			large := s.to-s.from > 600
			if large {
				nLarge++
			}
			hs := handlesIn(kf.pay, s.from, s.to, biped)
			hitsPerRec[len(hs)]++
			if bySlot[s.slot] == nil {
				bySlot[s.slot] = map[int]int{}
			}
			bySlot[s.slot][len(hs)]++
			for _, h := range hs {
				offDist[h.off]++
			}
		}
	}
	fmt.Printf("records ti=5 examines : %d (dont %d larges = joueur actif)\n", nRec, nLarge)
	fmt.Printf("touches par record    : %s\n\n", fmtCount(hitsPerRec))

	fmt.Printf("--- par slot joueur (slot : distribution du nb de touches) ---\n")
	var slots []int
	for s := range bySlot {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	for _, s := range slots {
		fmt.Printf("  slot %-3d : %s\n", s, fmtCount(bySlot[s]))
	}

	fmt.Printf("\n--- decalage de la touche depuis le debut du record (bits : compte) ---\n")
	var offs []int
	for o := range offDist {
		offs = append(offs, o)
	}
	sort.Ints(offs)
	shown := 0
	for _, o := range offs {
		if offDist[o] < 3 {
			continue
		}
		fmt.Printf("  +%-5d : %d\n", o, offDist[o])
		if shown++; shown >= 20 {
			fmt.Printf("  ... (tronque)\n")
			break
		}
	}

	// B. le lien lui-meme : image-cle par image-cle, slot joueur -> biped designe.
	fmt.Printf("\n--- 1.4 LE LIEN LU : image-cle -> (slot joueur = slot biped) ---\n")
	dupTotal, kfWithDup := 0, 0
	for ki, kf := range kfs {
		biped := bipedSlotsOf(kf)
		type link struct {
			pslot int
			hs    []hit
		}
		var links []link
		for _, s := range kf.sp {
			if s.ti != playerTI || s.to-s.from <= 600 {
				continue
			}
			hs := handlesIn(kf.pay, s.from, s.to, biped)
			if len(hs) > 0 {
				links = append(links, link{s.slot, hs})
			}
		}
		if len(links) == 0 {
			continue
		}
		sort.Slice(links, func(i, j int) bool { return links[i].pslot < links[j].pslot })
		// injectivite : deux joueurs ne peuvent pas designer le meme biped
		used := map[int][]int{}
		var parts []string
		for _, l := range links {
			var ss []string
			for _, h := range l.hs {
				ss = append(ss, fmt.Sprintf("%d(g%d)", h.slot, h.gen))
				used[h.slot] = append(used[h.slot], l.pslot)
			}
			parts = append(parts, fmt.Sprintf("%d->%v", l.pslot, ss))
		}
		dup := 0
		for _, owners := range used {
			if len(owners) > 1 {
				dup++
			}
		}
		if dup > 0 {
			kfWithDup++
			dupTotal += dup
		}
		fmt.Printf("kf%-3d t=%-12d liens=%d collisions=%d\n", ki+1, kf.tsUS, len(links), dup)
		for _, p := range parts {
			fmt.Printf("        %s\n", p)
		}
	}
	fmt.Printf("\nINJECTIVITE : %d images-cles portent au moins une collision (%d au total)\n",
		kfWithDup, dupTotal)
}
