package grammar

import "levelup/go-api/internal/games/halo_infinite/film/types"

// composants_vue_b_m4b.go — LES COMPOSANTS QUI FERMAIENT LA VUE B DANS LES FENETRES DU TIR CONTINU
// (lot M4b de la campagne « retours rejeu », 2026-09-25).
//
// # POURQUOI CES PORTS SONT DANS LE LOT DU TIR CONTINU
//
// Le tir continu se lit dans la vue C, qui ne se lit que si la vue B clot sa liste (sonde P1-S3).
// Un composant non porte OUVRE la vue B du paquet (`desync`) : la vue C n est pas lue, et les
// records qui suivent — dont les CREATIONS d objets — sont perdus, ce qui clot ensuite chaque
// paquet ou l objet non lie reapparait (rejet d en-tete). Les ports ci-dessous sont ceux que les
// gates G1 et G2 du lot ont trouves en travers d une fenetre de tir, un par un, sur pieces :
//
//	ti=47 i2  personal-ai-data              81c02726 : 3 489 paquets ouverts sur 19 933
//	ti=5  i22 player-aim-assist             81c02726 : 247 paquets ouverts (fenetre du G1 comprise)
//	ti=5  i24 player-desired-frame-config   8a485699 : la chaine de tete du paquet de mort p692
//	ti=40 i34 vehicle-type-physics          1cd3848a : 741 paquets sur 785 dans la fenetre LAAG
//	ti=40 i37 vehicle-emp-timer             8a485699 : 608 paquets ouverts ; 1cd3848a : 31
//	ti=10 i24 managed-object-looping-sound  81c02726 : 206 paquets ouverts (125 DELTA, 81 NEW)
//
// Les deux ports `ti=40` vivent avec les autres composants du vehicule
// (`composants_vehicule_ti40.go`), maillon suivant de la chaine.
//
// Chaque grammaire est lue chez l ECRIVAIN (Ghidra, HaloInfinite.exe, base 0x140000000) : nom ->
// accesseur de nom -> descripteur de replication -> deserialiseur a +0x28 (meme gabarit que
// `ti=47 i1`, `FUN_140daebd0` a `143d085d8 + 0x28`). Aucune valeur n est interpretee ni publiee :
// ces ports rendent la LARGEUR juste, pour que la liste continue.

// --- ti=47 i2 -------------------------------------------------------------------------------

// compPersonalAIData est l etiquette de registre de `ti=47 i2`.
const compPersonalAIData = "personal-ai-data-component"

// poigneeAucune est la valeur de `R(32)` qui dit « aucune poignee » (`CMP R9D,-0x1`
// @142c5d6de) : le deser s arrete la.
const poigneeAucune = 0xffffffff

// largeurValeurPersonalAI est la largeur de la valeur quantifiee (`FUN_1406d84b4`, 0xc,
// `MOV [RSP+0x20],0xc` @142c5d710).
const largeurValeurPersonalAI = 12

// consumePersonalAIData lit `ti=47 i2` : nom `143c95b10`, accesseur `141177c90`, descripteur
// `143e0c1e8`, deser `FUN_142ed6c04` -> `FUN_142c5d614(objet + 0xf0, lecteur)` :
//
//	R(32)                         une poignee ; 0xffffffff = aucune
//	si != 0xffffffff : R(1)       `FUN_1406cf008`
//	   si 1 : R(12)               `FUN_1406d84b4`
//
// Soit 32, 33 ou 45 bits. Le lot F2 (2026-08-24, `registre_film/LOTF2_TI47.md`) avait MESURE la
// largeur modale sans le binaire (`R(45)`, bit 1 a 1 et bits 2-19 nuls : une poignee de datum) et
// deux largeurs minoritaires qu il n expliquait pas : ce sont les deux branches courtes.
func consumePersonalAIData(br *Lecteur) {
	if br.ReadBits(32) == poigneeAucune {
		return
	}
	if br.ReadBit() {
		br.ReadBits(largeurValeurPersonalAI)
	}
}

// --- ti=5 i22 et i24 ------------------------------------------------------------------------

// compPlayerAimAssist est l etiquette de registre de `ti=5 i22`.
const compPlayerAimAssist = "player-aim-assist-component"

// categorieAimAssist est la categorie que le deser pousse dans R8D avant `FUN_1408f0ac4`
// (`MOV R8D,0x1` @141139f41).
const categorieAimAssist = 1

// consumePlayerAimAssist lit `ti=5 i22` : nom `143c98370`, accesseur `1411776a0`, descripteur
// `143d0dd18`, deser `FUN_141139f30` :
//
//	FUN_1408f0ac4(objet + 0xb0, lecteur, 1)   porte R(1), puis l entier a largeur variable
//	FUN_1406cf008                             R(1) -> objet + 0xb8
func consumePlayerAimAssist(br *Lecteur) {
	consume1408f0ac4(br, categorieAimAssist)
	br.ReadBit()
}

// compPlayerDesiredFrameConfiguration est l etiquette de registre de `ti=5 i24` : nom
// `143c982a8`, accesseur `141177680`, descripteur `143d0dcc8`, deser `FUN_1410dcb34`, qui appelle
// `FUN_1407f0550(objet + 200, lecteur)` — le bloc de trois elements deja porte pour l arme tenue
// ([consume1407f0550]).
const compPlayerDesiredFrameConfiguration = "player-desired-frame-configuration-component"

// --- ti=10 i24 ------------------------------------------------------------------------------

// compLoopingSound est l etiquette de registre de `ti=10 i24` : nom `143c94ac0`, accesseur
// `1411756d0`, descripteur `143c97280`, deser `FUN_140fb89f4` : R(32), range dans le tableau des
// sons en boucle de l objet a l index du descripteur (`objet + 0x174 + desc[8] * 4`).
const compLoopingSound = "managed-object-looping-sound-component"

// largeurSonEnBoucle : la largeur du mot de `FUN_140fb89f4` (`+0x2c += 0x20`).
const largeurSonEnBoucle = 32

// consumeComposantsVueBM4b est un maillon de la chaine de dispatch (cf. l en-tete de
// `dispatch_object.go`) : les ports du lot M4b, et `ti=10 i2` a `i21` et `i23` (lots des arrets de la vue B).
// Un maillon a lui plutot qu un case de plus dans un maillon existant : ceux-ci sont au plafond du
// ratchet de longueur de fonction.
func consumeComposantsVueBM4b(br *Lecteur, name string) (variant uint32, dead *types.DeadState, ported bool) {
	variant = noVariant
	switch name {
	case compPersonalAIData: // ti=47 i2 (FUN_142ed6c04 -> FUN_142c5d614)
		consumePersonalAIData(br)
	case compPlayerAimAssist: // ti=5 i22 (FUN_141139f30)
		consumePlayerAimAssist(br)
	case compPlayerDesiredFrameConfiguration: // ti=5 i24 (FUN_1410dcb34 -> FUN_1407f0550)
		consume1407f0550(br)
	case compLoopingSound: // ti=10 i24 et i25 (FUN_140fb89f4) — R(32)
		br.ReadBits(largeurSonEnBoucle)
	case compManagedObjectNavpoint: // ti=10 i2 a i17 (FUN_14107cea4) — R(32), lot des arrets de la vue B
		consumeManagedObjectNavpoint(br)
	case compManagedObjectFlags: // ti=10 i23 (FUN_1410d9b5c) — R(2)
		consumeManagedObjectFlags(br)
	case compManagedObjectNetworkedProperty: // ti=10 i18 a i21 (FUN_142ed5358) — R(32)
		consumeManagedObjectNetworkedProperty(br)
	default:
		return consumeComposantsDispositif(br, name)
	}
	return variant, nil, true
}
