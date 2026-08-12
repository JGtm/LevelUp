package replay

// inventory_grenade_selection.go — LA RÈGLE R5 : la grenade SÉLECTIONNÉE (composant i47)
// dans le record de biped des images-clés. Extraite d'inventory_decode.go (seuil de taille
// du dépôt) ; les règles R1..R4 et leurs helpers restent là-bas.

// invGrenadeSelLo / invGrenadeSelHi bornent, en bits APRÈS la fin de la dernière occurrence
// de famille d'arme du record, la fenêtre où vit le composant i47 (jeu de grenades :
// [6 bits masque][3 bits sélection en base 1] — grammaire du dispatch, accord i22↔i47
// 194/194, RECETTE_LOADOUT_2026-07-27 §1). Position MESURÉE sur les 120 records à i22 lu de
// 000d5950 : 69 lectures sur 92 tombent à +203..+205, le reste du jitter venant de petits
// champs à porte en amont ; la fenêtre l'absorbe. LA PUBLICATION NE REPOSE PAS SUR LA
// POSITION : elle exige masque == bitmap des compteurs i22 ET une sélection UNANIME dans la
// fenêtre — un débordement rend « non lu », jamais un faux. Validation interne : stabilité
// intra-vie 26/29, et l'oracle des décréments (un compteur qui décroît entre deux keyframes
// dit le type lancé) donne 7/7 d'accord côté AVANT, dont 2/2 non tautologiques (porteur à
// 2+ types). Instrument rejouable : i47_research_test.go (garde I47_FILM).
const (
	invGrenadeSelLo = 200
	invGrenadeSelHi = 210
)

// invLastFamily rend la position bit de la DERNIÈRE famille d'arme connue du record : c'est
// le repère avant du composant i47 (cf. invGrenadeSelLo).
func invLastFamily(pay []byte, from, to int, known map[uint32]bool) (int, bool) {
	var w uint32
	last, found := 0, false
	for b := from; b < to; b++ {
		w = w<<1 | invBitAt(pay, b)
		if b-from < 31 {
			continue
		}
		if known[w] {
			last, found = b-31, true
		}
	}
	return last, found
}

// invGrenadeSelection lit le composant i47 dans la fenêtre qui suit la fin de la dernière
// famille d'arme (famEnd = position de la famille + 32) et rend le RANG sélectionné, ou -1.
//
// LE MOTIF EST CONTRAINT PAR i22, PAS PAR SA POSITION : [6 bits masque][3 bits sélection en
// base 1], où le masque doit être EXACTEMENT le bitmap des compteurs non nuls (bit r = rang r)
// et la sélection désigner un rang porté. Neuf bits sont trop courts pour valoir preuve seuls ;
// la fenêtre les cadre, et l'UNANIMITÉ départage : plusieurs occurrences qui ne disent pas la
// même sélection = non lu. Mesures et oracle : en-tête d'invGrenadeSelLo.
func invGrenadeSelection(pay []byte, famEnd, to int, gren [invGrenadeSlots]uint32) int {
	var mask uint32
	for r, v := range gren {
		if v > 0 {
			mask |= 1 << uint(r)
		}
	}
	if mask == 0 {
		return -1
	}
	sel := -1
	for off := invGrenadeSelLo; off <= invGrenadeSelHi; off++ {
		b := famEnd + off
		if b+9 > to {
			break
		}
		if invBits(pay, b, 6) != mask {
			continue
		}
		s := int(invBits(pay, b+6, 3))
		if s < 1 || s > invGrenadeSlots || mask&(1<<uint(s-1)) == 0 {
			continue
		}
		if sel >= 0 && sel != s-1 {
			return -1 // deux occurrences en désaccord : on ne départage pas au hasard
		}
		sel = s - 1
	}
	return sel
}
