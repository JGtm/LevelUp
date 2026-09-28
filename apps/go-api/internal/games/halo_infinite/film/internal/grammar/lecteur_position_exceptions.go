package grammar

// lecteur_position_exceptions.go — LES SIX SITES DE `FUN_14076e524` QUI GARDENT LEUR ANCIEN
// LECTEUR (lot J6.3 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision du superviseur du
// 2026-09-27 pour les trois premiers ; lot J6-bis du 2026-09-28 pour flock-destination,
// tacmap-poiicon et player-desired-respawn-location, meme situation, meme format).
//
// Le portage unique (`lecteur_position.go`) lit ces six sites comme le jeu les ecrit — releve
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

// consumeFlockDestination lit ti=21 i2-i11 flock-destination (`FUN_140fb8af0`, CALL 140fb8b3e,
// niveau 0x10 en 140fb8b33) avec le lecteur d AVANT le lot J6.3 : R(1), le vecteur au niveau du
// registre ([lireVecteurAncienAuNiveauDuRegistre]), puis R(2) quand ce niveau depasse 1.
//
// EXCEPTION (lot J6-bis, 2026-09-28) : chez le jeu, R(1), la garde, `FUN_14076e524(0x10)` et R(2)
// (`FUN_1424e268c`, `param_4 > 1`). Portee ainsi, deux listes d evenements de `11de8353`
// (HI_1_9_0) que l ancien lecteur fermait au bit pres ne se localisent plus : chunk 19 paquet
// 494 (16 entrees de controle) et chunk 9 paquet 1146 (22 entrees) — l ancien lit ce composant sur
// 29 et 4 bits, la lecture du jeu sur 52 et 70. Elle en ferme une autre, `000d5950` (HI_1_13_0) chunk 20
// paquet 1322 (8 entrees) : le format depend du build ou du contenu, ce que ce lot n etablit pas.
func consumeFlockDestination(br *Lecteur, level uint32) {
	br.ReadBit()
	lireVecteurAncienAuNiveauDuRegistre(br, level)
	if level > 1 {
		br.ReadBits(2) // FUN_1424e268c
	}
}

// consumeTacmapPoiIcon lit ti=30 i0 tacmap-poiicon (`FUN_142ed8418`, CALL 142ed86d7, le thunk
// `FUN_1424e0e38` au niveau 0x10) avec le lecteur d AVANT le lot J6.3 : le bloc de l icone, le
// vecteur au niveau du registre ([lireVecteurAncienAuNiveauDuRegistre]), puis la queue.
//
// EXCEPTION (lot J6-bis, 2026-09-28) : chez le jeu, le vecteur est `FUN_14076e494(0x10)`. Porte
// ainsi, la liste d evenements du chunk 21 paquet 1032 de `11de8353` (HI_1_9_0), que l ancien
// lecteur fermait au bit pres (0 entree de controle), ne se localise plus : le composant passe de
// 239 a 264 bits ; aucune fermeture ne monte sur les huit builds.
func consumeTacmapPoiIcon(br *Lecteur, level uint32) {
	br.ReadBits(32) // icon-id
	br.ReadBits(32) // icon-missionid
	br.ReadBit()
	br.ReadBits(3)
	br.ReadBits(32) // icon-text
	br.ReadBits(32) // icon-bitmapbg
	br.ReadBits(9)
	br.ReadBits(9)
	lireVecteurAncienAuNiveauDuRegistre(br, level)
	br.ReadBits(32) // string-id
	br.ReadBit()
	br.ReadBits(8)
	br.ReadBits(8)
	br.ReadBits(8)
	br.ReadBits(8)
}

// consumePlayerDesiredRespawnLocation lit ti=5 i12 player-desired-respawn-location (`FUN_142f03ec8`,
// descripteur 143d0f2f8 + 0x28, `FUN_14076e494(0x10)` chez le jeu) avec le lecteur d AVANT le lot
// J6.3 : R(1) porte ; si 1, le vecteur au niveau du registre, puis R(19) (`FUN_14076dc04`,
// l identifiant de reapparition). Publie `[qx, qy, qz, identifiant, niveau]` — le niveau parce que
// la largeur des quanta en depend ; precHigh leve : l identifiant seul, `present` faux ; porte
// fermee : rien.
//
// EXCEPTION (lot J6-bis, 2026-09-28) : portee comme le jeu, la liste d evenements du chunk 25
// paquet 344 de `e5adf7b2` (HI_1_11_0, 14 entrees de controle), que l ancien lecteur fermait au bit
// pres, ne se localise plus (le composant passe de 44 a 71 bits) ; elle en ferme deux autres
// (`e5adf7b2` chunk 6 paquet 50, 4 entrees ; `111fa685` chunk 14 paquet 552, 2 entrees).
func consumePlayerDesiredRespawnLocation(br *Lecteur, level uint32) {
	if !br.ReadBit() {
		br.obs.publishPlayerState(PlayerDesiredRespawnLocation, false)
		return
	}
	q, ok := lireVecteurAncienAuNiveauDuRegistre(br, level)
	id := br.ReadBits(19)
	if !ok {
		br.obs.publishPlayerState(PlayerDesiredRespawnLocation, false, id)
		return
	}
	br.obs.publishPlayerState(PlayerDesiredRespawnLocation, true, q[0], q[1], q[2], id, uint64(level))
}

// lireVecteurAncienAuNiveauDuRegistre est le vecteur que ces sites lisaient avant le lot J6.3
// (`consumeQuantVec3Values`) : un bit precHigh — a 1, le vecteur par defaut, 0 bit, et ok faux —,
// la porte, un index fige a 1 bit, puis trois axes a `min(26, 6 + niveau du registre)`.
func lireVecteurAncienAuNiveauDuRegistre(br *Lecteur, level uint32) (q [3]uint64, ok bool) {
	if br.ReadBit() { // precHigh
		return q, false
	}
	if !br.ReadBit() { // porte ; 0 -> l index est present
		br.ReadBits(1)
	}
	w := largeurAncienneDuFlock(uint(level))
	for axe := range q {
		q[axe] = br.ReadBits(w)
	}
	return q, true
}
