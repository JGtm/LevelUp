package grammar

// lecteur_position_exceptions.go — LES TREIZE SITES DE `FUN_14076e524` QUI GARDENT LEUR ANCIEN
// LECTEUR (lot J6.3 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision du superviseur du
// 2026-09-27 pour les trois premiers ; lot J6-bis du 2026-09-28 pour flock-destination,
// tacmap-poiicon et player-desired-respawn-location ; lot R3 du 2026-09-29 pour
// tacmap-displayasset, tacmap-areaofinterest, tacmap-cooptetherarea, crew-order et le precHigh de
// la branche absolue d i0 du bipede ; lot R3-bis du 2026-09-30 pour unit-actor-state et
// tacmap-waypointstate — meme situation, meme format).
//
// Le portage unique (`lecteur_position.go`) lit ces treize sites comme le jeu les ecrit — releve
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
//
// SOUS LA GARDE DE PLEINE PRECISION (`FUN_14076f91c`, [fullPrecisionGate] : la portee de l etat
// complet, plan LK, LK.5), un site dont la lecture sous la garde est relue chez le jeu la porte EN
// TETE de sa fonction, dans un bloc autonome : il y lit comme le jeu, sans noter d exception, et
// l exception ne vaut que hors de la garde. Ses cas : `lecteur_position_sites_portee_test.go`.

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
	br.noterExceptionDatee()
	if br.ReadBit() { // precHigh (FUN_14076e420 R(1))
		br.ReadBits(59) // precHigh=1 : FUN_141f85880 AABB + handle-tail + R(2) (total 60 mesuré)
		return
	}
	if !br.ReadBit() { // FUN_14076e524 index-sel ; si 0 -> lit l'index de région
		br.ReadBits(br.worldObjectPrecision().IndexW)
	}
	for a := range 3 {
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
	br.noterExceptionDatee()
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
	br.noterExceptionDatee()
	mask := br.ReadBits(8)
	for i := range uint(8) {
		if mask&(1<<i) != 0 {
			consumeCompressedDir140c1e79c(br)
			if !br.ReadBit() { // FUN_14076e524 index-present select
				br.ReadBits(br.traversal().IndexW)
			}
			for a := range 3 {
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
	br.noterExceptionDatee()
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
	br.noterExceptionDatee()
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
	br.noterExceptionDatee()
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

// LOT R3 (2026-09-29) — CINQ SITES DE PLUS, LOCALISES AU PAQUET.
//
// Methode : marche des trames (carte de fermeture) du parent du lot J6.3 (`56299bdc3~1`, la
// reference J4.0.5) contre la tete du plan, paquet par paquet, sur douze films ; puis chaque site
// rendu SEUL a son lecteur d avant sur la tete. Les chiffres « ancien » sont ceux de ce site seul
// contre la tete : ce qu il ferme de plus, et ce que la lecture du jeu fermait et qu il rend.

// consumeTacmapDisplayAsset lit ti=33 i0 tacmap-displayasset (`FUN_142ed433c`) : R(32), R(32), R(2),
// la position au lecteur d AVANT le lot J6.3 ([lireCorpsDeTraverseeAncien]), R(96), R(96), R(1).
//
// EXCEPTION (lot R3, 2026-09-29) : chez le jeu, `FUN_142ed7d38` -> thunk `FUN_1424e0e38` (CALL
// 142ed7edf, `LEA R8D,[R9+0x10]` en 142ed7edb ; le thunk pose p5 = p6 = 0) -> `FUN_14076e494(0x10)`.
// Portee ainsi (porte posee : 22 bits par axe au lieu de 6), dix paquets de `51ebbc0f` que l ancien
// lecteur fermait au bit pres ne ferment plus (chunk 7 paquets 2336, 2338, 2340, 2356, 2358, 2362,
// 2382 ; chunk 8 paquets 70, 74, 118 : 61 entrees de controle, 41 records ti=35 dont le saut du
// slot 526 a 7:2358). Elle en ferme cinq autres : `51ebbc0f` 14:42 (5 entrees), `084a804d` 46:10
// (20), `11de8353` 29:208 (16), `fb1a1a72` 7:2380 (0), `60ae07c4` 32:2062 (7). Scindee par la porte,
// aucune forme ne ferme les deux familles : l ancien lecteur sur la seule porte posee ferme les dix
// et perd 14:42, 7:2380 et 32:2062 ; sur la seule porte a 0, il perd 46:10 et 29:208 sans rien fermer.
func consumeTacmapDisplayAsset(br *Lecteur) {
	br.noterExceptionDatee()
	br.ReadBits(32)
	br.ReadBits(32)
	br.ReadBits(2)
	lireCorpsDeTraverseeAncien(br)
	br.ReadBits(64)
	br.ReadBits(32)
	br.ReadBits(64)
	br.ReadBits(32)
	br.ReadBits(1)
}

// consumeTacmapAreaOfInterest lit ti=32 i0 tacmap-areaofinterest (`FUN_142ed3c50`) : R(32), R(3), la
// position au lecteur d AVANT le lot J6.3 ([lireCorpsDeTraverseeAncien]), R(12).
//
// EXCEPTION (lot R3, 2026-09-29) : chez le jeu, `FUN_142ed7764` -> thunk `FUN_1424e0e38(0x10)` (CALL
// 142ed7853). Portee ainsi, le paquet 12:608 de `51ebbc0f` (6 entrees), que l ancien lecteur fermait
// au bit pres, ne ferme plus (le composant passe de 67 a 96 bits). Elle en ferme trois autres :
// `11de8353` 19:394 (liste, 13 entrees), `fb1a1a72` 38:8 (6), `60ae07c4` 3:1790 (0) ; scindee par la
// porte, aucune forme ne ferme les deux familles.
func consumeTacmapAreaOfInterest(br *Lecteur) {
	br.noterExceptionDatee()
	br.ReadBits(32)
	br.ReadBits(3)
	lireCorpsDeTraverseeAncien(br)
	br.ReadBits(12)
}

// consumeTacmapCoopTetherArea lit ti=34 i11 tacmap-cooptetherarea (`FUN_142ed4198`) : la position au
// lecteur d AVANT le lot J6.3 ([lireCorpsDeTraverseeAncien]), R(12), R(12).
//
// EXCEPTION (lot R3, 2026-09-29) : chez le jeu, thunk `FUN_1424e0e38(0x10)` (CALL 142ed41ba). Portee
// ainsi, la liste d evenements du chunk 21 paquet 1012 de `c75f33b8` (HI_1_13_0), que l ancien lecteur
// fermait au bit pres, ne se localise plus (+48 bits, porte posee) ; aucune fermeture ne monte sur
// les douze films.
func consumeTacmapCoopTetherArea(br *Lecteur) {
	br.noterExceptionDatee()
	lireCorpsDeTraverseeAncien(br)
	br.ReadBits(12)
	br.ReadBits(12)
}

// consumeCrewOrder lit ti=14 i0 crew-order (`FUN_142ed4274` -> `FUN_142ed9120`) : `FUN_142b1cf3c`
// R(3), la porte de presence, puis le vecteur au lecteur d AVANT le lot J6.3
// ([lireVecteurAncienAuNiveauDuRegistre] : precHigh, porte, index fige a 1 bit, 6 + niveau du
// registre par axe).
//
// EXCEPTION (lot R3, 2026-09-29) : chez le jeu, `FUN_14076e494(..., 0x10, 0, param_3, 0)` (CALL
// 142ed918e), sans bit precHigh. Portee ainsi, la liste d evenements du chunk 22 paquet 538 de
// `084a804d` (HI_1_10_0, 14 entrees), que l ancien lecteur fermait au bit pres, ne se localise plus
// (+26 bits) ; elle en ferme une autre, `e5adf7b2` chunk 4 paquet 900 (liste, 12 entrees).
func consumeCrewOrder(br *Lecteur, level uint32) {
	br.noterExceptionDatee()
	br.ReadBits(3)    // FUN_142b1cf3c
	if br.ReadBit() { // presence du vecteur
		lireVecteurAncienAuNiveauDuRegistre(br, level)
	}
}

// consumePrecHautDuBipede lit la branche precHigh = 1 de l i0 ABSOLU du bipede (`FUN_1406cfe44`,
// bUsePred = 0, bDelta = 0) comme avant le lot J6.3 : RIEN apres le bit — ni axes, ni le R(2) de
// `LAB_1406cffd7`.
//
// EXCEPTION (lot R3, 2026-09-29) : chez le jeu, precHigh a 1 mene (1406d0093 -> 1422f4cb7) a
// `FUN_141f85880(&DAT_143b8c6d0, 0x10)`, trois axes de 14 bits sur +/-100, puis le R(2) : 44 bits.
// Portee ainsi, trois listes d evenements que l ancien lecteur fermait au bit pres ne se localisent
// plus — `0797ce72` chunk 9 paquet 138 (6 entrees ; record NEW ti=35 slot 530), `084a804d` chunk 25
// paquet 356 (21 entrees ; delta du slot 685, dont la lecture d accroupi) et chunk 37 paquet 22 (10
// entrees ; NEW ti=35 slot 527) ; aucune fermeture ne monte sur les douze films. Les deux autres
// sites a precHigh (grammaire d ecrivain d i0, delta predit a cVar1 = 1) gardent la lecture du jeu.
func consumePrecHautDuBipede(br *Lecteur) {
	br.noterExceptionDatee()
}

// LOT R3-BIS (2026-09-30) — DEUX SITES DE PLUS, LOCALISES AU PAQUET.
//
// Methode : carte de fermeture paquet par paquet de vingt films (les dix-huit de la reference
// J4.0.5, `f75e7053` et `000d5950`), la reference (`56299bdc3~1`) contre la tete du plan ; chaque
// paquet de la reference perdu remonte au premier composant dont la longueur change ; puis chaque
// site rendu SEUL a son lecteur d avant sur la tete, sur les vingt films.

// lireViseeDActeurAncienne lit le vecteur d un emplacement de visee de unit-actor-state
// (`FUN_14058c058`, branches a = 0) comme avant le lot J6.3 : seize bits plats.
//
// SOUS LA GARDE DE PLEINE PRECISION ([fullPrecisionGate] : la portee de l etat complet), LE SITE LIT
// COMME LE JEU (plan LK, LK.5.1, 2026-10-08) : `FUN_14076e494` y lit le vecteur BRUT
// (`FUN_1411b259c`, R(96)), relu sur les deux CALLs. L exception ne vaut que hors de la garde.
//
// EXCEPTION (lot R3-bis, 2026-09-30) : chez le jeu, `FUN_14076e494(..., 0x10, 0, param_3, 0)` (CALLs
// 1422cddc1 et 1422cde0e, une par branche a = 0) — garde, porte, index, trois axes a la ligne
// 0x10, 49 a 67 bits. Portee ainsi, des paquets de `4f77afc1` (HI_1_13_0) que l ancien lecteur
// fermait au bit pres ne ferment plus : les listes 38:410 (17 entrees de controle), 54:316 (5) et
// 54:1140 (11), et 12:1118, ou le slot 570 se lie alors a ti=4 au lieu de ti=35, d ou les paquets
// 12:1120 a 12:1128 (100 entrees). Il faut les deux branches : rendue seule, la branche b = 0 laisse
// 54:316 et la chaine de 12:1118 perdues. Elle en ferme onze autres, jamais fermes a la reference :
// `084a804d` 8:276, 10:384, 10:844, 20:722, 26:656, 33:634 (68 entrees), `e5adf7b2` 6:838 et 7:440
// (14), `111fa685` 10:906, `4f77afc1` 48:776 (6), `d9781168` 33:1176 (3). La porte et l index que
// lit la lecture du jeu ne departagent pas les deux familles (porte a 0, index 0 et 1 de part et
// d autre ; la porte posee n apparait que chez les secondes).
func lireViseeDActeurAncienne(br *Lecteur) {
	if fullPrecisionGate(br) { // FUN_14076f91c, 0 bit : vraie sous la portee de l etat complet
		br.ReadBits(rawVec3Bits) // FUN_1411b259c -> FUN_1406d676c(..., 0x60)
		return
	}
	br.noterExceptionDatee()
	br.ReadBits(16)
}

// consumeTacmapWaypointState lit ti=34 i7 tacmap-waypointstate (`FUN_140f04d74` -> `FUN_140f04d88`)
// comme avant le lot J6.3 : R(1), `FUN_14080dec4` "waypoint-lockedto" = R(32), puis la position au
// lecteur d AVANT ([lireCorpsDeTraverseeAncien]) — et pas le R(1) de `param_4 > 1`.
//
// EXCEPTION (lot R3-bis, 2026-09-30) : chez le jeu, la garde, `FUN_14076e524(0x10)` (CALL 140f04de0,
// 140f04dd5) et un R(1) quand le niveau du registre depasse 1 (`if (1 < param_4)`). Portee ainsi, la
// liste d evenements du chunk 34 paquet 336 de `d9781168` (HI_1_13_0, 4 entrees), que l ancien
// lecteur fermait au bit pres, ne se localise plus (+30 bits) ; il faut les deux ecarts (position et
// R(1)) pour la rendre. Aucune fermeture ne monte sur les vingt films.
func consumeTacmapWaypointState(br *Lecteur) {
	br.noterExceptionDatee()
	br.ReadBit()
	br.ReadBits(32) // FUN_14080dec4 "waypoint-lockedto"
	lireCorpsDeTraverseeAncien(br)
}

// lireCorpsDeTraverseeAncien est la position que les sites tacmap lisaient avant le lot J6.3
// (`consumeE524PositionBody`) : la porte, l index sur la largeur du descripteur de TRAVERSEE, puis
// trois axes a ses largeurs (index 1 bit, 6/6/6 : la valeur du profil pour tous les builds), sans
// garde.
func lireCorpsDeTraverseeAncien(br *Lecteur) {
	if !br.ReadBit() { // porte ; 0 -> l index est present
		br.ReadBits(br.traversal().IndexW)
	}
	for axe := range 3 {
		br.ReadBits(br.traversal().AxisW[axe])
	}
}
