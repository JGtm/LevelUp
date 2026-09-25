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
// Chaque grammaire est lue chez l ECRIVAIN (Ghidra, HaloInfinite.exe, base 0x140000000) : nom ->
// accesseur de nom -> descripteur de replication -> deserialiseur a +0x28 (meme gabarit que
// `ti=47 i1`, `FUN_140daebd0` a `143d085d8 + 0x28`). Aucune valeur n est interpretee ni publiee :
// ces ports rendent la LARGEUR juste, pour que la liste continue. Une seule porte n est pas dans
// le flux (i34) : elle est un repli NOMME et COMPTE ([compVehicleTypePhysics]).

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

// --- ti=40 i34 et i37 -----------------------------------------------------------------------

// compVehicleTypePhysics est l etiquette de registre de `ti=40 i34` : nom `143d0a838`, accesseur
// `141177260`, descripteur `143d0b308`, deser `FUN_142f02498` :
//
//	si objet[+0x818] :                        porte RUNTIME, hors du flux
//	   c = R(1)                               mode = c ? 2 : 0
//	   FUN_140c5f938(lecteur, +0x7f4, +0x800, mode)   mode 2 : R(192) ; 0 : [decodeObjectForwardAndUp]
//	   FUN_14076e1c8(lecteur, +0x80c, mode)           mode 2 : R(96)  ; 0 : [consumeDynPrecVec3]
//
// C est EXACTEMENT la paire (avant/haut, vitesse angulaire) des composants dynamiques de precision
// de l archetype (`FUN_14076e1c8` est le lecteur de [consumeObjectAngularVelocity]). L ecrivain
// (`FUN_142f04e90`, descripteur +0x10) teste le MEME octet et n ecrit rien sans lui.
//
// LA PORTE EST UN REPLI NOMME (`repli_physique_de_type_de_vehicule_supposee`, 2026-09-25). L octet
// `+0x818` n est pas ecrit par le flux lu (ni par ce composant, ni par `i33`, qui le teste aussi) ;
// il est pose au moment ou le jeu construit le vehicule. Le port le suppose POSE des que le masque
// annonce le composant, et la preuve est l ORACLE DE CADRAGE : sur `1cd3848a`, la fenetre de la LAAG
// (vehicule 806) passe de 0 a 765 paquets fermes sur 785 — la vue C finit au bit pres sur ses
// 0 a 7 bits de bourrage nuls. Chaque lecture supposee est COMPTEE
// ([types.MovementStateStats.VehicleTypePhysicsAssumed]) ; le retrait attend la lecture de
// l ecrivain de l octet.
const compVehicleTypePhysics = "vehicle-type-physics-component"

// consumeVehicleTypePhysics lit `ti=40 i34`, porte runtime supposee posee (cf. la constante).
func consumeVehicleTypePhysics(br *Lecteur) {
	if br.ReadBit() { // c : mode 2, les deux vecteurs bruts puis la vitesse brute
		br.ReadBits(fwdUpDynPrecMode2Bits)
		br.ReadBits(rawVec3Bits)
		return
	}
	consumeObjectForwardAndUp(br)                            // FUN_140c5f938 mode 0 -> FUN_140c5fa84
	consumeDynPrecVec3(br, angularMagBits, angularScaleBits) // FUN_14076e1c8 mode 0 -> FUN_14076d528
}

// compVehicleEmpTimer est l etiquette de registre de `ti=40 i37` : nom `143c995e8`, accesseur
// `141177300`, descripteur `143d0b498`, deser `FUN_142f049dc` -> `FUN_1432065d8` =
// `FUN_1406d84b4(..., 8, ...)` : R(8), le meme minuteur quantifie que celui du bipede (`i51`).
const compVehicleEmpTimer = "vehicle-emp-timer-component"

// lecturesDeComposant rend le nombre de lectures du composant `nom` dans un record (0 ou 1).
func lecturesDeComposant(r FrameRecord, nom string) int {
	for _, cr := range r.Trace.Comps {
		if cr.Name == nom {
			return 1
		}
	}
	return 0
}

// largeurMinuteurEMPVehicule : la largeur de `FUN_1432065d8`.
const largeurMinuteurEMPVehicule = 8

// --- ti=10 i24 ------------------------------------------------------------------------------

// compLoopingSound est l etiquette de registre de `ti=10 i24` : nom `143c94ac0`, accesseur
// `1411756d0`, descripteur `143c97280`, deser `FUN_140fb89f4` : R(32), range dans le tableau des
// sons en boucle de l objet a l index du descripteur (`objet + 0x174 + desc[8] * 4`).
const compLoopingSound = "managed-object-looping-sound-component"

// largeurSonEnBoucle : la largeur du mot de `FUN_140fb89f4` (`+0x2c += 0x20`).
const largeurSonEnBoucle = 32

// consumeComposantsVueBM4b est le DERNIER maillon de la chaine de dispatch (cf. l en-tete de
// `dispatch_object.go`) : les ports du lot M4b. Un maillon a lui plutot qu un case de plus dans un
// maillon existant : ceux-ci sont au plafond du ratchet de longueur de fonction.
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
	case compVehicleTypePhysics: // ti=40 i34 (FUN_142f02498) — porte runtime supposee : repli nomme
		consumeVehicleTypePhysics(br)
	case compVehicleEmpTimer: // ti=40 i37 (FUN_142f049dc -> FUN_1432065d8) — R(8)
		br.ReadBits(largeurMinuteurEMPVehicule)
	default:
		// Non porte (object position/velocity/angular/region/damage/constraint/parent/
		// scale/..., unit-actor-control/state/malleable, biped-* tail) : arret propre.
		return variant, nil, false
	}
	return variant, nil, true
}
