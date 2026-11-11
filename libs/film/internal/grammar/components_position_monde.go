package grammar

// components_position_monde.go — LES POSITIONS QUE LE FILM NOMME PAR INDEX DE REGION, LUES PAR LE
// PORTAGE UNIQUE DE `FUN_14076e524` ([lireE524], [lireE494]) : world-object i0 et les deux composants
// de flock.

// consumeObjectPositionMonde lit world-object i0 (`FUN_14076e29c` -> `FUN_14076e420(0x10)`, CALL
// 14076e2c0) : precHigh R(1) ; a 0, `FUN_14076e524(0x10)` ([lireE524] : la porte, l index sur
// `DAT_144632be0` bits, les trois axes aux largeurs de la ligne de CET index, la table DEFAUT quand la
// porte est posee), puis R(2) (`FUN_14076e304`, la queue de handle valant 0 bit quand precHigh vaut 0).
// A 1, [consumePrecHautDeLObjetMonde].
//
// SOUS LA GARDE DE PLEINE PRECISION (`FUN_14076f91c`, la portee de l etat complet) : precHigh et
// R(96), la queue `FUN_14076e3e4(precHigh)`, puis R(2) si les trois flottants sont finis — la forme de
// la branche absolue d i0 du bipede ([consumeAbsoluSousLaPortee]).
func consumeObjectPositionMonde(br *Lecteur) {
	if fullPrecisionGate(br) {
		consumeAbsoluSousLaPortee(br)
		return
	}
	if br.ReadBit() { // precHigh (FUN_14076e420 R(1))
		consumePrecHautDeLObjetMonde(br)
		return
	}
	lireE524(br, niveauPosition) // FUN_14076e524(0x10)
	br.ReadBits(2)               // FUN_14076e304 R(2) fini
}

// consumeFlockPosition lit ti=21 i16 flock-position (`FUN_140ee7270`, CALL 140ee7293, niveau 0x10 en
// 140ee7288) : la garde de pleine precision (R(96) brut), sinon `FUN_14076e524(0x10)`.
func consumeFlockPosition(br *Lecteur) { lireE494(br, niveauPosition) }

// consumeFlockDestination lit ti=21 i2-i11 flock-destination (`FUN_140fb8af0`, CALL 140fb8b3e, niveau
// 0x10 en 140fb8b33) : R(1), la position (`FUN_14076e494(0x10)`), puis R(2) (`FUN_1424e268c`) quand le
// niveau du registre depasse 1 (`param_4 > 1`).
func consumeFlockDestination(br *Lecteur, level uint32) {
	br.ReadBit()
	lireE494(br, niveauPosition)
	if level > 1 {
		br.ReadBits(2)
	}
}
