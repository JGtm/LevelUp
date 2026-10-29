package grammar

// vue_a_poussees.go — LES DEUX MESSAGES DE POUSSEE D EQUIPEMENT DE LA VUE A, LUS AVEC LEURS VALEURS
// (lot F du plan d ajustements du 2026-10-14).
//
// Quand un equipement repousse (le repulseur), l ecrivain enregistre dans la vue A, pour chaque objet
// pousse, un `EquipmentObjectKnockedBack` (genre 105), et, quand l objet est un joueur, a sa suite
// dans la meme liste, un `EquipmentKnockbackPlayer` (genre 104) de meme reference 0. Les lecteurs et
// les gestionnaires (`vtable + 0x78`) sont lus dans l executable :
//
//	105  FUN_141118a00 : R(1) ; si 1 : R(32)          gestionnaire FUN_1410c20bc : ref 0 = l objet
//	                                                    pousse, ref 1 = l unite qui pousse (0x1003)
//	104  FUN_14116c344 -> FUN_14076d528(..., 10, 0x13) gestionnaire FUN_140bc8170 : applique le
//	     R(1) ; si 0 : direction R(19) (FUN_1406d8288)  vecteur a l unite de la ref 0
//	     puis norme R(10) (FUN_14076d6dc, bornes 0,05 et 20 : DAT_143cd8648, DAT_143cd8f60)
//
// Ce fichier porte les deux charges et range leurs valeurs sur le lecteur ; la lecture de la vue A
// les recueille ([lireLaVueA]). L appariement et le dequantifie de l impulsion vivent dans le
// balayage (`poussees.go`).

import "github.com/JGtm/LevelUp/libs/film/internal/grammar/lecture"

// Les genres des deux messages de poussee (numerotation de `FUN_140e453b4`).
const (
	GenrePousseeDuJoueur = 104
	GenreObjetRepousse   = 105
)

// chargePousseeDuJoueur porte `FUN_14116c344` (`EquipmentKnockbackPlayer`) -> `FUN_14076d528` :
// R(1) ; si 0 : R(0x13) direction, R(10) norme.
func chargePousseeDuJoueur(br *Lecteur) bool {
	br.pousseeLue = lecture.MessageDePoussee{}
	if !br.ReadBit() {
		br.pousseeLue.Direction = uint32(br.ReadBits(largeurDirection))    //nolint:gosec // R(19)
		br.pousseeLue.Norme = uint16(br.ReadBits(largeurMagnitudePoussee)) //nolint:gosec // R(10)
		br.pousseeLue.Drapeaux |= lecture.PousseeImpulsionLue
	}
	return true
}

// chargeObjetRepousse porte `FUN_141118a00` (`EquipmentObjectKnockedBack`) : R(1) ; si 1 :
// `FUN_14080d6f0` = R(32).
func chargeObjetRepousse(br *Lecteur) bool {
	br.pousseeLue = lecture.MessageDePoussee{}
	if br.ReadBit() {
		br.pousseeLue.Mot = uint32(br.ReadBits(32)) //nolint:gosec // R(32)
		br.pousseeLue.Drapeaux |= lecture.PousseeMotLu
	}
	return true
}

// messageDePoussee assemble le message de poussee que la lecture de la vue A vient de lire : sa
// position, son genre, la charge rangee sur le lecteur et ses deux premieres references.
func messageDePoussee(br *Lecteur, genre, debut int) lecture.MessageDePoussee {
	m := br.pousseeLue
	m.Debut = uint32(debut) //nolint:gosec // position dans un payload
	m.Genre = uint8(genre)  //nolint:gosec // genre R(7)
	for i, d := range [2]uint8{lecture.PousseeRef0, lecture.PousseeRef1} {
		if r := br.refsDuMessage[i]; r.presente {
			m.Ref[i], m.Gen[i] = uint16(r.index), uint8(r.gen) //nolint:gosec // index du domaine 0, 13 bits ; generation 2 bits
			m.Drapeaux |= d
		}
	}
	return m
}

// aLeDrapeau dit que le drapeau `d` est pose sur le message `m`.
func aLeDrapeau(m lecture.MessageDePoussee, d uint8) bool { return m.Drapeaux&d != 0 }
