//go:build research

package grenadeids

// typeindex.go — QUEL ARCHETYPE NAIT VRAIMENT DERRIERE LE MARQUEUR.
//
// LE MARQUEUR NE PORTE QUE CINQ BITS DE L INDEX, et c est la decouverte D2 (3.3r) du lot :
// `marqueur(ti) = ((ti & 31) << 19) | 0x40C00` est la meme valeur pour `ti=41` (le projectile)
// et pour `ti=9` (`managed-player`, cf. `grammar/player_teams.go:61`). Un balayage qui compte
// des marqueurs compte donc DEUX populations, et l histogramme de ce qui les suit melange deux
// grammaires de default-state. Sur les builds ou la liste blanche ne reconnait rien, ce
// melange est exactement ce qui pourrait faire prendre une valeur de `managed-player` pour un
// identifiant de grenade.
//
// LE SIXIEME BIT EST JUSTE AVANT. Le typeIndex d un record fait SIX bits — `traverse.go:94`
// (`t.TypeIndex = uint32(br.ReadBits(6))`) et `keyframe_fullstate_loop.go:88`
// (`kfReadBits(pay, recBit+keyframeRecordTIBit, 6)`). Le marqueur commence au deuxieme de ces
// six bits : son bit de poids fort (valeur 32) est donc a `marqueur - 1`.
//
//	41 = 0b101001  -> bit a -1 : 1
//	 9 = 0b001001  -> bit a -1 : 0
//
// UNE LECTURE, PAS UNE HEURISTIQUE : le bit est ecrit par le jeu au meme titre que les cinq
// autres. La seule position ou il manque est `marqueur = 0` (debut de payload), comptee a part
// — jamais devinee a zero en silence.

import "levelup/go-api/internal/games/halo_infinite/film/internal/grammar"

// bitsTypeIndexBasDuMarqueur : les cinq bits bas du typeIndex, en tete du marqueur.
const bitsTypeIndexBasDuMarqueur = 5

// typeIndexDuRecord rend le typeIndex complet du record dont le marqueur est a `bp`, et si son
// bit de poids fort etait hors du payload (auquel cas seul le reste module 32 est connu).
func typeIndexDuRecord(pay []byte, bp int) (int, bool) {
	bas := int(grammar.PeekBits(pay, bp, bitsTypeIndexBasDuMarqueur))
	if bp < 1 {
		return bas, true
	}
	return int(grammar.PeekBits(pay, bp-1, 1))<<bitsTypeIndexBasDuMarqueur | bas, false
}

// FiltrerParTi rend les occurrences dont le typeIndex resolu vaut `ti`. `ti < 0` les rend
// toutes : c est la valeur qui dit « ne filtre pas », pas un archetype.
func FiltrerParTi(occ []Occurrence, ti int) []Occurrence {
	if ti < 0 {
		return occ
	}
	out := make([]Occurrence, 0, len(occ))
	for _, o := range occ {
		if o.TypeIndex == ti && !o.TiIndetermine {
			out = append(out, o)
		}
	}
	return out
}
