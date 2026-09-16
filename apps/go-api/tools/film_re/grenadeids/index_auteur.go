//go:build research

package grenadeids

// index_auteur.go — OU EST L INDEX DE L AUTEUR SUR LES BUILDS ANCIENS.
//
// LA MESURE DE L IDENTIFIANT NE SUFFIT PAS A REGLER CELLE DE L AUTEUR. Le decalage d un bit
// etabli sur l identifiant ne se propage pas mecaniquement au champ d index : les 47 bits qui
// separent les deux dans la grammaire de production sont un TOTAL MESURE, pas une suite de
// champs lue (cf. l en-tete de `grammar/grenade_events.go`, qui le dit lui-meme : « la source de
// reference decrit sauter 47 bits sans dire depuis quoi »). Sur les builds anciens, ni +103 ni
// +102 ne rendent un index plausible : il faut donc le CHERCHER, pas le deduire.
//
// LE TEST EST CELUI QUI A ETABLI +103 : le champ fait cinq bits et le film compte au plus huit
// joueurs par equipe indexes 0..7, donc la bonne position est celle ou TOUTES les valeurs
// tombent dans 0..7 — un decalage faux rend des valeurs au-dela une fois sur deux. L instrument
// balaye les decalages autour de +103 et publie, pour chacun, la part des lancers CONFIRMES
// (identifiant reconnu, a +24 ou a +23) dont l index tient dans 0..7.

import (
	"fmt"
	"io"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/grammar"
)

// indexAuteurBit : la position de production du champ d index, en bits depuis le marqueur.
const indexAuteurBit = 24 + 32 + 47

// indexAuteurBits : la largeur du champ.
const indexAuteurBits = 5

// indexAuteurMax : la plus grande valeur qu un index de joueur peut prendre dans un film a huit.
const indexAuteurMax = 7

// CompteIndex porte, pour un decalage, combien de lancers confirmes y rendent un index plausible.
type CompteIndex struct {
	Dans0a7, Total int
}

// balayerIndexAuteur accumule, pour une occurrence CONFIRMEE, la plausibilite de chaque decalage.
func balayerIndexAuteur(pay []byte, o Occurrence, fenetre int, r *Releve) {
	if !occurrenceConfirmee(o) {
		return
	}
	if r.IndexParDecalage == nil {
		r.IndexParDecalage = map[int]*CompteIndex{}
	}
	r.LancersConfirmes++
	for d := -fenetre; d <= fenetre; d++ {
		c := r.IndexParDecalage[d]
		if c == nil {
			c = &CompteIndex{}
			r.IndexParDecalage[d] = c
		}
		c.Total++
		if grammar.PeekBits(pay, o.BitPos+indexAuteurBit+d, indexAuteurBits) <= indexAuteurMax {
			c.Dans0a7++
		}
	}
}

// occurrenceConfirmee dit si ce marqueur porte un identifiant de grenade reconnu, a la position
// de production (+24) ou a celle des builds anciens (+23).
func occurrenceConfirmee(o Occurrence) bool {
	if _, ok := grammar.GrenadeRankOf(o.ID); ok {
		return true
	}
	_, ok := grammar.GrenadeRankOf(o.IDAlt)
	return ok
}

// EcrireIndexAuteur publie le balayage, du decalage le plus plausible au moins plausible.
func EcrireIndexAuteur(w io.Writer, r *Releve, top int) {
	if r.LancersConfirmes == 0 {
		return
	}
	type ligne struct {
		d int
		c CompteIndex
	}
	var lignes []ligne
	for d, c := range r.IndexParDecalage {
		lignes = append(lignes, ligne{d: d, c: *c})
	}
	sort.Slice(lignes, func(i, j int) bool {
		if lignes[i].c.Dans0a7 != lignes[j].c.Dans0a7 {
			return lignes[i].c.Dans0a7 > lignes[j].c.Dans0a7
		}
		return abs(lignes[i].d) < abs(lignes[j].d)
	})
	fmt.Fprintf(w, "   INDEX AUTEUR  %d lancer(s) confirme(s) ; part des index dans 0..7 par decalage depuis +%d\n",
		r.LancersConfirmes, indexAuteurBit)
	for i, l := range lignes {
		if top > 0 && i >= top {
			break
		}
		fmt.Fprintf(w, "     decalage=%+d  %d/%d\n", l.d, l.c.Dans0a7, l.c.Total)
	}
}
