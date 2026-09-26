package grammar

// Décodage de la VISÉE MODALE du record de tir `action_weapon_fire` (type 36).
//
// CE QUE CE FICHIER PORTE. La visée ne se lit qu'au bout de la grammaire réelle du record —
// tracée instruction par instruction dans FUN_14080C1F8 par l'agent Ghidra, cf.
// .ai/V7.5/film_re/NOTE_VISEE_TIR_2026-08-31.md — à la position POST-COMPTES + 2. Il couvre TOUT le
// record MODAL (0 cible, 0 composante de dégât), soit 3 à 6× plus de tirs que l'ancien chemin à
// offsets fixes (mesuré : 33→210, 143→491, 48→218 sur trois films).
//
// DEPUIS LE LOT M4b (2026-09-24), LA TÊTE EST CELLE DE [lireEnteteTir36] : il n'y a plus qu'UNE
// lecture de la tête du record dans le paquet, et ce fichier repart de sa fin (après `i`, `j`).
// Le chemin à offsets fixes qui lisait la visée au bit 113 sur le seul « record vide » a disparu
// avec les offsets fixes : sur ce sous-ensemble, post-comptes + 2 vaut 113 (mesuré sur 5 films),
// donc la visée qu'il rendait est celle que ce chemin rend.
//
// DEUX CORRECTIONS vs l'ancien décodeur modèle-M (prouvées sur 5 films, commit 8a8aa3239) :
//  1. POLARITÉ du champ d : sauter R(5) quand son bit de garde vaut 0 (désassemblage), pas 1.
//  2. Les deux lecteurs composites (FUN_1406cd5b8 / FUN_1408eff64) sont PARASITES dans le chemin
//     modal : la vraie visée est à post-comptes + 2 (les deux derniers drapeaux), AUCUNE lecture
//     composite avant elle.
//
// RÉSERVE (agent Ghidra, JUSTE) : les boucles cibles/composantes NON vides ont une largeur venant
// d'une table peuplée au runtime (0x1451f98d0) — non localisable hors ligne. On ne perce donc QUE
// le cas modal (le tir « propre ») ; les records à ≥ 1 cible / composante rendent ok=false.

// modalAimGap est l'écart, en bits, entre la position post-comptes et le début de la visée : les
// deux derniers drapeaux du record. La visée suit IMMÉDIATEMENT, sans lecteur composite.
const modalAimGap = 2

// modalAimBit rend la position de la visée d'un record modal, ou ok=false.
func modalAimBit(pay []byte) (int, bool) {
	h, ok := lireEnteteTir36(pay)
	if !ok {
		return 0, false
	}
	return modalAimBitFrom(pay, h)
}

// modalAimBitFrom rend la position de la visée d'un record dont la tête est DÉJÀ lue.
func modalAimBitFrom(pay []byte, h enteteTir36) (int, bool) {
	pos, ok := modalPostCountsBitFrom(pay, h)
	if !ok {
		return 0, false
	}
	return pos + modalAimGap, true
}

// modalPostCountsBitFrom lit, après la tête, le bloc horodatage et les comptes, et rend la position
// post-comptes d'un record MODAL.
func modalPostCountsBitFrom(pay []byte, h enteteTir36) (int, bool) {
	br := LecteurSur(pay)
	br.Skip(h.apresDrapeaux)
	if h.bloc {
		br.Skip(1)
		if br.ReadBit() {
			return 0, false // horodatage bloc non résolu hors ligne
		}
	}
	if h.court {
		return 0, false // la variante courte ne porte pas ce préambule
	}
	var nCibles, nComp uint64
	if !br.ReadBit() {
		if br.ReadBit() {
			nCibles = 1
		} else {
			nCibles = br.ReadBits(4)
		}
		if !br.ReadBit() {
			if br.ReadBit() {
				nComp = 1
			} else {
				nComp = br.ReadBits(4)
			}
		}
	}
	if nCibles != 0 || nComp != 0 {
		return 0, false // record non-modal : les boucles sont de largeur runtime
	}
	return br.BitPos(), br.BitPos() <= len(pay)*8
}

func readAimAt(pay []byte, e *FireEvent, aimBit int) {
	if aimBit < 0 || len(pay)*8 < aimBit+int(FireAimBits) {
		return
	}
	if v, ok := DecodeAimVectorChecked(readBitsAt(pay, aimBit, int(FireAimBits)), FireAimBits); ok {
		e.HasAim, e.Aim = true, v
	}
}
