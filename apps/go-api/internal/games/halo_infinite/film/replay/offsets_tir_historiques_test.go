package replay

// offsets_tir_historiques_test.go — LA LECTURE A OFFSETS FIXES DU RECORD DE TIR D AVANT LE LOT M4b,
// pour les SEULS instruments de recherche historiques qui la MESURAIENT (`ctf_shortauthor`,
// `ctf_shortvariant`, `visee_canal_zoom`) : l octet de tete « type 105 + variante » et l index
// d attaquant a quatre bits (bits 36..40, decale d un cran).
//
// La production ne les porte plus : depuis le lot M4b (2026-09-24), la tete du record est lue par
// sa grammaire (`grammar/fire_events.go`), et ces offsets ne valaient que pour la disposition
// canonique. Ils vivent ici, sous `_test.go` ; le ratchet `archlint/record36_grammaire_test.go`
// interdit qu ils reviennent en production.

// ancienTypeTeteTir est le « type 105 » de l ancienne lecture : les SEPT premiers bits du payload
// (configuration, continuation, et les cinq bits hauts du type 36 ou 37).
const ancienTypeTeteTir = 105

// ancienneTeteTirBits est l ancienne garde de longueur (le cinquieme « drapeau », bit 112).
const ancienneTeteTirBits = 113

// ancienIndexAttaquant relit l index d attaquant a QUATRE bits de l ancienne lecture (bits 36..40,
// decale d un cran) ; -1 si le payload est trop court.
func ancienIndexAttaquant(pay []byte) int {
	const bit, largeur = 36, 5
	if len(pay)*8 < bit+largeur {
		return -1
	}
	v := 0
	for i := bit; i < bit+largeur; i++ {
		v = v<<1 | int(pay[i/8]>>uint(7-i%8)&1)
	}
	return v >> 1
}
