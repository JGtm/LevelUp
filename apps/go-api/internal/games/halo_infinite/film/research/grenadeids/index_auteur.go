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
// LE TEST QUI A ETABLI +103 — ET POURQUOI IL NE SUFFIT PAS (mesure du 2026-09-17, lot 3.3.1).
// Le champ fait cinq bits et un match d arene compte huit joueurs indexes 0..7 : la position
// juste est donc celle ou TOUTES les valeurs tombent dans 0..7. Ce critere ne discrimine RIEN
// sur les films anciens — sur `60ae07c4` (Ranked:Oddball, huit joueurs), TRENTE decalages sur
// 81 rendent 310/310. La cause est mesuree : ce sont des SUITES DE ZEROS de l etat par defaut,
// ou le champ lu vaut 0 a chaque lancer.
//
// LE CRITERE FORT EST LA DISPERSION, ET IL EST GRATUIT. Un champ d index de joueur prend
// PLUSIEURS valeurs sur un match — autant que de lanceurs — tandis qu une suite de zeros n en
// prend qu UNE. L instrument publie donc, par decalage, le nombre de valeurs DISTINCTES et le
// MAXIMUM, a cote de la part dans 0..7 :
//
//	film d arene (8 joueurs)  la position juste rend max <= 7 ET plusieurs distincts
//	film BTB (jusqu a 24)     la MEME position doit rendre un max > 7 : un decalage qui rend
//	                          0..7 des deux cotes est une suite de zeros, pas un index
//
// C est le recoupement que la decouverte D5 (3.3r) reclamait, et il se lit sans le pont
// d identite du rejeu.

import (
	"fmt"
	"io"
	"math/bits"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// indexAuteurBit : la position de production du champ d index, en bits depuis le marqueur.
const indexAuteurBit = 24 + 32 + 47

// indexAuteurBits : la largeur du champ.
const indexAuteurBits = 5

// indexAuteurMax : la plus grande valeur qu un index de joueur peut prendre dans un film a huit.
const indexAuteurMax = 7

// CompteIndex porte, pour un decalage, combien de lancers confirmes y rendent un index plausible,
// et QUELLES valeurs y ont ete vues — c est la dispersion qui separe un index d une suite de
// zeros.
type CompteIndex struct {
	Dans0a7, Total int
	// Vues est le masque des valeurs rencontrees (le champ fait cinq bits : 0..31).
	Vues uint32
}

// Distincts rend le nombre de valeurs differentes vues a ce decalage.
func (c CompteIndex) Distincts() int { return bits.OnesCount32(c.Vues) }

// Max rend la plus grande valeur vue, ou -1 quand aucune ne l a ete.
func (c CompteIndex) Max() int {
	if c.Vues == 0 {
		return -1
	}
	return 31 - bits.LeadingZeros32(c.Vues)
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
		v := grammar.PeekBits(pay, o.BitPos+indexAuteurBit+d, indexAuteurBits)
		c.Vues |= 1 << uint(v&0x1F)
		if v <= indexAuteurMax {
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
//
// L ORDRE EST CELUI DU CRITERE FORT : la part dans 0..7 d abord, puis la DISPERSION — un
// decalage qui rend 100 % avec UNE seule valeur distincte est une suite de zeros, et il tombe
// donc derriere un decalage qui rend 100 % avec huit valeurs.
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
		a, b := lignes[i], lignes[j]
		switch {
		case a.c.Dans0a7 != b.c.Dans0a7:
			return a.c.Dans0a7 > b.c.Dans0a7
		case a.c.Distincts() != b.c.Distincts():
			return a.c.Distincts() > b.c.Distincts()
		default:
			return abs(a.d) < abs(b.d)
		}
	})
	fmt.Fprintf(w, "   INDEX AUTEUR  %d lancer(s) confirme(s) ; part des index dans 0..7 par decalage depuis +%d\n",
		r.LancersConfirmes, indexAuteurBit)
	for i, l := range lignes {
		if top > 0 && i >= top {
			break
		}
		fmt.Fprintf(w, "     decalage=%+d  %d/%d  distincts=%d max=%d\n",
			l.d, l.c.Dans0a7, l.c.Total, l.c.Distincts(), l.c.Max())
	}
}
