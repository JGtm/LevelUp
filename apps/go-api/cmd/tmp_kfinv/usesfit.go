package main

import (
	"fmt"
	"sort"
)

// usesfit.go — CALIBRAGE (et non validation) du compteur d'utilisations de capacité.
//
// Ce que fait cette passe : elle cherche une position de bit, relative à une ancre du record,
// où le quartet de poids fort d'un champ de 7 bits reproduit les huit valeurs du relevé
// Theater (5,5,5,4,5,1,5,5). Le relevé est ici l'ORACLE DE RECHERCHE : un succès ne VALIDE
// donc rien, il localise un champ. La validation, si elle vient, viendra d'une autre image-clé
// ou d'un autre film.

var truthUses = map[int]uint32{512: 5, 513: 5, 514: 5, 515: 4, 516: 5, 517: 1, 518: 5, 519: 5}

func runUsesFit(dir string, kfWanted int, grenMax uint32) {
	views, pay, _ := firstKeyframeViews(dir, kfWanted, grenMax)
	if views == nil {
		fmt.Println("image-clé introuvable")
		return
	}
	sort.Slice(views, func(i, j int) bool { return views[i].slot < views[j].slot })
	type anchor struct {
		name string
		at   func(recView) int
	}
	anchors := []anchor{
		{"debut", func(v recView) int { return v.from }},
		{"capacite", func(v recView) int { return v.abil }},
		{"i22", func(v recView) int { return v.i22 }},
		{"arme0", func(v recView) int {
			if len(v.wpos) > 0 {
				return v.wpos[0]
			}
			return -1
		}},
		{"arme1", func(v recView) int {
			if len(v.wpos) > 1 {
				return v.wpos[len(v.wpos)-1]
			}
			return -1
		}},
		{"fin", func(v recView) int { return v.to }},
	}
	for _, a := range anchors {
		hits := 0
		for off := -3000; off <= 3000; off++ {
			okAll := true
			for _, v := range views {
				want, has := truthUses[v.slot]
				base := a.at(v)
				if !has || base < 0 {
					okAll = false
					break
				}
				p := base + off
				if p < v.from || p+7 > v.to {
					okAll = false
					break
				}
				if (bits(pay, p, 7)>>3)&0xF != want {
					okAll = false
					break
				}
			}
			if okAll {
				hits++
				fmt.Printf("ancre %-9s offset %+6d : les 8 slots reproduisent le relevé\n", a.name, off)
				for _, v := range views {
					p := a.at(v) + off
					fmt.Printf("    slot %d v7=%3d (0x%02x) quartet=%d\n", v.slot,
						bits(pay, p, 7), bits(pay, p, 7), (bits(pay, p, 7)>>3)&0xF)
				}
			}
		}
		if hits == 0 {
			fmt.Printf("ancre %-9s : AUCUNE position sur 6001 ne reproduit les 8 valeurs\n", a.name)
		}
	}
}
