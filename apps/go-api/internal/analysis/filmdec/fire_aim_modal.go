package filmdec

// Décodage de la VISÉE MODALE du record de tir (type 105 / 0xD2), au-delà des ~19 % que le
// chemin à offsets fixes de fire_events.go couvrait.
//
// CE QUE CE FICHIER AJOUTE. fire_events.go lit la visée à l'offset FIXE bit 113, mais seulement
// sur le « chemin record vide » (drapeaux 110==1, 111==0, 112==0). Ce décodeur AVANCE dans la
// grammaire réelle du record — tracée instruction par instruction dans FUN_14080C1F8 par l'agent
// Ghidra, cf. .ai/V7.5/film_re/NOTE_VISEE_TIR_2026-08-31.md — jusqu'à la position POST-COMPTES,
// et rend la visée à post-comptes + 2. Il couvre ainsi TOUT le record MODAL (0 cible, 0 composante
// de dégât), soit 3 à 6× plus de tirs (mesuré : 33→210, 143→491, 48→218 sur trois films).
//
// POURQUOI « type 36 » ET NON 105. Le même paquet 0xD2 se lit de deux façons. fire_events.go lit
// bits 0..6 = 105 puis bit 7 = variante (atterrissage empirique du cas canonique). La grammaire
// Ghidra, elle, lit 2 bits de préfixe config/continuation PUIS 7 bits de type = 36, PUIS trois
// références d'entité, PUIS les champs a..k. Les deux modèles décrivent les mêmes octets ; celui-ci
// est le modèle réel du désérialiseur, seul capable d'avancer jusqu'aux comptes.
//
// DEUX CORRECTIONS vs l'ancien décodeur modèle-M (prouvées sur 5 films, commit 8a8aa3239) :
//  1. POLARITÉ du champ d : sauter R(5) quand son bit de garde vaut 0 (désassemblage), pas 1.
//     Ici CÂBLÉE EN DUR — la branche buggée (saut si garde==1) n'existe pas dans ce chemin.
//  2. Les deux lecteurs composites (FUN_1406cd5b8 / FUN_1408eff64) sont PARASITES dans le chemin
//     modal : la vraie visée est à post-comptes + 2 (les deux derniers drapeaux), AUCUNE lecture
//     composite avant elle. Pour un record flags-vide, post-comptes vaut 111 sur 100 % des témoins,
//     donc post-comptes + 2 = 113 = exactement ce que fire_events lit déjà : ZÉRO régression.
//
// RÉSERVE (agent Ghidra, JUSTE) : les boucles cibles/composantes NON vides ont une largeur venant
// d'une table peuplée au runtime (0x1451f98d0) — non localisable hors ligne. On ne perce donc QUE
// le cas modal (le tir « propre ») ; les records à ≥ 1 cible / composante rendent ok=false.

const (
	// modalRecordType est le type du record de tir/dégât dans la NUMÉROTATION Ghidra (2 bits de
	// préfixe config/continuation puis 7 bits de type). C'est le même paquet 0xD2 que fire_events
	// numérote 105.
	modalRecordType = 36
	// modalAimGap est l'écart, en bits, entre la position post-comptes et le début de la visée :
	// les deux derniers drapeaux du record. La visée suit IMMÉDIATEMENT, sans lecteur composite.
	modalAimGap = 2
)

// modalAimBit rend la position de bit de la visée d'un record de tir MODAL (0 cible, 0 composante
// de dégât), ou ok=false pour un record non-modal, court, ou à bloc-horodatage non résolu. La
// lecture finale (aimBit bits de large) reste à la charge de l'appelant, qui la borne.
func modalAimBit(pay []byte) (int, bool) {
	pos, ok := modalPostCountsBit(pay)
	if !ok {
		return 0, false
	}
	return pos + modalAimGap, true
}

// modalPostCountsBit avance dans la grammaire Ghidra du record type 36 jusqu'à la position APRÈS
// les comptes cibles/composantes, pour le seul cas MODAL (0 cible, 0 composante). Rend ok=false
// si le paquet n'est pas un type 36 modal borné. Le BitReader est borné : les bits hors tampon se
// lisent en 0 (pas de panique) ; l'appelant borne la lecture de visée elle-même.
func modalPostCountsBit(pay []byte) (int, bool) {
	br := NewBitReader(pay)
	br.Skip(2) // préfixe config / continuation
	if br.ReadBits(7) != modalRecordType {
		return 0, false
	}
	if br.ReadBit() { // ref0 dom1 (FUN_141fcf670)
		w := 13
		if br.ReadBit() {
			w = 9
		}
		br.Skip(w + 2)
	}
	for range 2 { // ref1 dom8, ref2 dom7
		if br.ReadBit() {
			br.Skip(15)
		}
	}
	estCourt := br.ReadBit() // a : variante courte
	estBloc := br.ReadBit()  // b : bloc supplémentaire
	br.Skip(8)               // c : attaquant R(7)+R(1)
	if !br.ReadBit() {       // d : R(1) + [si 0] R(5) — POLARITÉ Ghidra câblée
		br.Skip(5)
	}
	if !br.ReadBit() { // e : R(1) + [si 0] R(2)
		br.Skip(2)
	}
	if br.ReadBit() { // f : arme famille R(1) + [si 1] R(32)
		br.Skip(32)
	}
	br.Skip(32) // g : arme variante R(32)
	br.Skip(2)  // i, j : R(1), R(1)
	if estBloc {
		br.Skip(1)
		if br.ReadBit() {
			return 0, false // horodatage bloc non résolu hors ligne
		}
	}
	if estCourt {
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
	return br.BitPos(), true
}

// readAimAt déquantifie 30 bits de visée à la position aimBit et les pose sur e, si la lecture
// tient dans le payload et si le vecteur est valide (face < 6). Centralisé pour que le chemin fixe
// (bit 113) et le chemin modal (post-comptes+2) partagent une seule lecture.
func readAimAt(pay []byte, e *FireEvent, aimBit int) {
	if aimBit < 0 || len(pay)*8 < aimBit+int(FireAimBits) {
		return
	}
	if v, ok := DecodeAimVectorChecked(readBitsAt(pay, aimBit, int(FireAimBits)), FireAimBits); ok {
		e.HasAim, e.Aim = true, v
	}
}
