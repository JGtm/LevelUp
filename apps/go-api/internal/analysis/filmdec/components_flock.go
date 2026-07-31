package filmdec

// components_flock.go — deserialiseurs de l'archetype ti21 (flock).
//
// Resolus 100% STATIQUEMENT par la regle du descripteur : chaine .rdata -> unique xref
// (getName, slot 0 de la vtable) -> unique motif d'octets de cette adresse -> vtable ->
// **vtable+0x28 = le deserialiseur**, qui est le slot appele par la boucle de composants
// `FUN_14076cb60` (`(**(code **)(*desc + 0x28))(desc, reader, ctx, baseline, count)`).
//
// Ils bloquaient la marche des records (mesure `evtl` sur 000d5950) :
// flock-position 222 paquets, flock-remembered-danger 10, flock-current-destination 9,
// flock-fleeing 6 — dont 2 des 14 morts encore manquantes.

// consumeFlockCurrentDestination : ti21 i17, chaine 143c95778 -> getName 141177be0 ->
// vtable 143d07dc0 -> +0x28 = FUN_142ed4668 -> FUN_142ed0674 = R(4) puis `valeur - 1`.
func consumeFlockCurrentDestination(br *BitReader) {
	br.ReadBits(4)
}

// consumeFlockRememberedDanger : ti21 i15, chaine 143c957a0 -> getName 141177c00 ->
// vtable 143d07e60 -> +0x28 = FUN_142ed477c :
//
//	FUN_1424d9a30(reader, ..., etat+0xd4) = R(3)              [desassemblage : ADD [rcx+0x2c],3]
//	FUN_1406d84b4(reader, ..., 0x8, 0, 0) = R(8)              [dword ptr [RSP+0x20] = 8]
//	si etat[0xd4] != 0 : FUN_14076dc04(reader, ..., 0x13)     [R9D = 0x13 = R(19)]
//
// L'octet teste est CELUI QUI VIENT D'ETRE LU (les 3 bits) : la largeur est donc entierement
// determinee par le flux — 11 bits si la valeur est nulle, 30 sinon.
func consumeFlockRememberedDanger(br *BitReader) {
	v := br.ReadBits(3)
	br.ReadBits(8)
	if v != 0 {
		br.ReadBits(19)
	}
}

// consumeFlockFleeing : ti21 i14, chaine 143c95690 -> getName 141177c30 -> vtable 143d07c30
// -> +0x28 = FUN_142ed4704 = un unique FUN_1406cf008 = R(1) (drapeau, bit 2 de etat+0xe8).
func consumeFlockFleeing(br *BitReader) {
	br.ReadBits(1)
}

// consumeFlockPosition : ti21 i16, chaine 143c95758 -> getName 141177bf0 -> vtable 143d07e10
// -> +0x28 = FUN_140ee7270 :
//
//	si FUN_14076f91c() != 0 : FUN_1411b259c (remplissage NaN/keep) = 0 bit
//	sinon                   : FUN_14076e524(&dst, reader, buf, 0x10) = l'epine vec3 quantifiee
//	                          (R(1) gate ; si 0 -> R(1) index ; puis 3 x R(6+L))
//
// C'est exactement le meme lecteur que le chemin ABSOLU de object-position : on reutilise
// donc `consumeQuantVec3WithGate` et le meme drapeau global `PositionFullPrecision`.
func consumeFlockPosition(br *BitReader, level uint) {
	if PositionFullPrecision {
		return
	}
	consumeQuantVec3WithGate(br, quantAxisWidth(level))
}
