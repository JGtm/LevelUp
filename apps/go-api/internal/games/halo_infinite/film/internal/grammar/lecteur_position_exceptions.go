package grammar

// lecteur_position_exceptions.go — LES TROIS SITES DE `FUN_14076e524` QUI GARDENT LEUR ANCIEN
// LECTEUR (lot J6.3 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision du superviseur du
// 2026-09-27).
//
// Le portage unique (`lecteur_position.go`) lit ces trois sites comme le jeu les ecrit — releve
// Ghidra du 2026-09-27 — et la FERMETURE DES BOBINES baisse sur chacun : la lecture du jeu y est
// donc contredite par une mesure que ce lot ne sait pas expliquer. Chaque site garde ici son
// lecteur d AVANT le lot, et figure comme EXCEPTION DATEE dans la table des sites
// (`lecteur_position_ratchet_test.go`, `exceptionsDuPortage`), avec l immediat du jeu, les chiffres
// de la baisse et le critere de retrait :
//
//	la lecture du jeu fait monter la fermeture sans aucune baisse sur les bobines, ou la
//	grammaire dependante du build est etablie.
//
// Le test de chaque site (`lecteur_position_sites_test.go`) est marque « ecart attendu » par
// l exception : il rougit si le site est migre sans que l exception soit retiree, et inversement.
// Ce fichier est le SEUL exempte des formes de lecteur local interdites par le ratchet.

// consumeObjectPositionMonde lit world-object i0 (`FUN_14076e29c` -> `FUN_14076e420(0x10)`, CALL
// 14076e2c0) avec le lecteur d AVANT le lot J6.3 : precHigh R(1) ; a 1, R(59) mesure ; a 0, la
// porte, l index sur `DAT_144632be0` bits, TROIS AXES AUX LARGEURS DE LA CARTE QUELLE QUE SOIT LA
// PORTE, puis R(2).
//
// EXCEPTION (GA2-5) : chez le jeu la porte posee lit la table DEFAUT, 22/22/22. Portee ainsi, la
// fermeture d image-cle monte sur les builds recents (ti=38 : bcb6d393 134 -> 713, fb1a1a72
// 317 -> 349, 60ae07c4 30 -> 46, e5adf7b2 49 -> 56 ; ti=42 : bcb6d393 5 -> 70, fb1a1a72 9 -> 25)
// et BAISSE sur les anciens (ti=38 : 11de8353 99 -> 83, a521164d 122 -> 119 ; ti=42 : 60ae07c4
// 5 -> 4, 11de8353 3 -> 2, 111fa685 2 -> 1). La garde de pleine precision n est pas lue non plus.
func consumeObjectPositionMonde(br *Lecteur) {
	if br.ReadBit() { // precHigh (FUN_14076e420 R(1))
		br.ReadBits(59) // precHigh=1 : FUN_141f85880 AABB + handle-tail + R(2) (total 60 mesuré)
		return
	}
	if !br.ReadBit() { // FUN_14076e524 index-sel ; si 0 -> lit l'index de région
		br.ReadBits(br.worldObjectPrecision().IndexW)
	}
	for a := 0; a < 3; a++ {
		br.ReadBits(br.worldObjectPrecision().AxisW[a]) // FUN_140cc5128 axe a
	}
	br.ReadBits(2) // FUN_14076e304 R(2) finite (handle-tail = 0 bit quand precHigh=0)
}

// consumeFlockPosition lit ti21 i16 (`FUN_140ee7270`, CALL 140ee7293, niveau 0x10 en 140ee7288)
// avec le lecteur d AVANT le lot J6.3 : la garde, la porte, un index fige a 1 bit, puis trois axes
// a `min(26, 6 + niveau du registre)`.
//
// EXCEPTION (GA2-2) : le jeu lit l index sur `DAT_144632be0` bits et les axes a la ligne 0x10
// (22/22/22 porte posee, les largeurs de la carte sinon). Portee ainsi, la carte de fermeture
// monte sur ks_000d5950 (paquets 1823 -> 1847, ti=21 44/490 -> 84/536, ti=35 3549 -> 3638) et
// BAISSE sur ks_e5adf7b2 (paquets 371 -> 370, ti=21 1/60 -> 0/60, ti=4 240 -> 239).
func consumeFlockPosition(br *Lecteur, level uint) {
	if fullPrecisionGate(br) {
		br.ReadBits(rawVec3Bits) // FUN_1411b259c -> FUN_1406d676c(..., 0x60)
		return
	}
	w := largeurAncienneDuFlock(level)
	if !br.ReadBit() { // FUN_1406cf008 ; bit==0 -> l'index est present
		br.ReadBits(1)
	}
	br.ReadBits(w)
	br.ReadBits(w)
	br.ReadBits(w)
}

// largeurAncienneDuFlock est la largeur `min(26, 6 + niveau du registre)` que le flock lisait
// avant le lot J6.3 — infirmee par le releve (aucun site ne transmet le niveau du registre au
// lecteur), gardee pour ce seul site tant que l exception tient.
func largeurAncienneDuFlock(level uint) uint {
	if w := 6 + level; w < 26 {
		return w
	}
	return 26
}

// consumeGenericRigidBodyTransforms lit ti=38 i18 (`FUN_142f036f0`, CALL 142f03837, niveau 0x10)
// avec le lecteur d AVANT le lot J6.3 : masque R(8), puis par bit `FUN_140c1e79c` et la position
// aux largeurs du descripteur de TRAVERSEE (index 1 bit, 6/6/6), sans garde.
//
// EXCEPTION : la decompilation de `FUN_142f036f0` (relue le 2026-09-27) appelle
// `FUN_14076e494(..., 0x10, 0, *(param_3+0x38), 0)` par bit — le portage unique. Portee ainsi, la
// fermeture d image-cle ti=38 BAISSE sans aucune hausse (fb1a1a72 317 -> 245, 111fa685 72 -> 30,
// 11de8353 99 -> 19) ; lue R(96) brut, elle baisse aussi (281, 30, 19).
func consumeGenericRigidBodyTransforms(br *Lecteur) {
	mask := br.ReadBits(8)
	for i := uint(0); i < 8; i++ {
		if mask&(1<<i) != 0 {
			consumeCompressedDir140c1e79c(br)
			if !br.ReadBit() { // FUN_14076e524 index-present select
				br.ReadBits(br.traversal().IndexW)
			}
			for a := 0; a < 3; a++ {
				br.ReadBits(br.traversal().AxisW[a]) // FUN_140cc5128 axis a
			}
		}
	}
}
