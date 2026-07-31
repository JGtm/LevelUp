package main

import "fmt"

// temoin_i42.go — ORACLE INTERNE pour le sélecteur d'arme, sans aucune vérité terrain.
//
// Principe (RECETTE §3) : l'emplacement dégainé est celui dont les munitions bougent. On ne
// dispose pas ici d'un suivi continu, mais d'un état par image-clé ; l'équivalent statique est :
// l'emplacement dégainé est PLUS SOUVENT ENTAMÉ (chargeur strictement inférieur à la dotation
// pleine de son arme) que l'emplacement rangé.
//
// Le test qui peut le réfuter : si sel ne désignait pas l'emplacement dégainé, les deux
// fréquences seraient égales.

func temoinI42(states []InvState) {
	// [sel][emplacement] -> entamés / total testables
	var entame [2][2]int
	var total [2][2]int
	for _, s := range states {
		if s.DrawnRaw != 0 && s.DrawnRaw != 1 {
			continue
		}
		if len(s.Ammo) < 2 || len(s.Weapons) < 2 {
			continue
		}
		for k := 0; k < 2; k++ {
			ref, has := ammoTable[s.Weapons[k]]
			if !has || s.Ammo[k].Mag == nil {
				continue
			}
			total[s.DrawnRaw][k]++
			if *s.Ammo[k].Mag < ref[0] {
				entame[s.DrawnRaw][k]++
			}
		}
	}
	fmt.Printf("\n=== TÉMOIN INTERNE DU SÉLECTEUR i42 (aucune vérité terrain)\n")
	fmt.Printf("part des chargeurs ENTAMÉS, par régime du sélecteur :\n")
	for sel := 0; sel < 2; sel++ {
		for k := 0; k < 2; k++ {
			p := 0.0
			if total[sel][k] > 0 {
				p = 100 * float64(entame[sel][k]) / float64(total[sel][k])
			}
			mark := "rangé   "
			if sel == k {
				mark = "DÉGAINÉ"
			}
			fmt.Printf("  sel=%d emplacement %d (%s) : %d / %d = %.1f %%\n",
				sel, k, mark, entame[sel][k], total[sel][k], p)
		}
	}
}
