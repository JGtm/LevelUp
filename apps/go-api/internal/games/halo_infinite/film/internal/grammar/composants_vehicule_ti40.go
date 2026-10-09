package grammar

import "levelup/go-api/internal/games/halo_infinite/film/types"

// composants_vehicule_ti40.go — LES COMPOSANTS PROPRES AU VEHICULE (`ti=40`, i30 a i47).
//
// Chaque grammaire est celle du DESERIALISEUR du jeu (Ghidra, HaloInfinite.exe, base
// 0x140000000 ; table nom -> descripteur -> deserialiseur relevee dans
// `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/T6_vehicules_ti40.md` §1 et §4). Aucune valeur
// n est interpretee ni publiee : ces lecteurs rendent la LARGEUR juste, pour que la liste de la
// vue B continue.
//
// # LA PORTE `+0x818` (i33, i34) EST UNE LOI DU MASQUE
//
// Les deserialiseurs de `i33` (`FUN_142f02474`) et de `i34` (`FUN_142f02498`) ne lisent leur
// corps que si l octet `+0x818` de l etat du vehicule est pose. Cet octet n est pas dans le flux :
// le constructeur de l etat (`FUN_14058c2ec`, slot `+0x88` de la vtable d archetype) le pose a
// (type de physique du tag `vehi` == 6). Mais les DEUX ecrivains du masque de presence ne posent
// les bits 33 et 34 que sous ce meme octet :
//
//	FUN_142f09c74 (slot +0x78, masque de difference) : FUN_14320c8fc sous `*(etat+0x818) != 0`
//	FUN_142f0cca0 (slot +0x130, capture, qui REMPLACE le masque complet de FUN_142e32138 via
//	              FUN_142ee98ec -> FUN_142f13c1c) : FUN_143208c18, nul si `*(etat+0x818) == 0`
//
// Un record lu AVEC un masque (DELTA, NEW) qui annonce `i33` ou `i34` a donc ete ecrit porte
// posee, et le lecteur, qui construit son etat par le meme `FUN_14058c2ec` depuis le meme tag,
// lit le corps. C est la loi de l ecrivain, pas une supposition : aucun chassis n est consulte.
//
// UN ETAT COMPLET D IMAGE-CLE N A PAS DE MASQUE (`FUN_142e2c690` deserialise toutes les entrees
// nommees) : `i33` et `i34` y sont appeles pour TOUT vehicule, et la porte y depend du chassis,
// que ce paquet ne lit pas. C est la seule loi du jeu qui empeche une lecture ici.
//
// LE RESTE EST UN CHOIX CONSERVATEUR DE CE PORT, PAS UNE REGLE DU JEU : sous [Lecteur.etatComplet],
// tous les composants de ce fichier rendent « non porte », sauf `i37` (minuteur EMP, lu en tout
// contexte : sa largeur ne depend de rien). La boucle d etat complet s arrete donc au premier
// d entre eux dans le registre, `i30`, alors que `i30`..`i32`, a largeur du seul flux, y seraient
// lisibles : les lire ne ferait que deplacer l arret jusqu a `i33`, et la marche d image-cle reste
// ainsi celle d avant ce maillon. Lire l etat complet du vehicule demande la porte du chassis.

// Etiquettes de registre des composants `ti=40` (nom ASCII lu dans le binaire, T6 §1).
const (
	compVehicleAutoTurretTriggers     = "vehicle-auto-turret-triggers-component"                      // i30
	compVehicleAutoTurretAimingVector = "vehicle-auto-turret-aiming-vector-component"                 // i31
	compVehicleTransformedOpenState   = "vehicle-transformed-or-desired-open-state-changed-component" // i32
	compVehicleTypeState              = "vehicle-type-state-component"                                // i33
	compVehicleAutoTurretTarget       = "vehicle-auto-turret-target-component"                        // i35
	compVehicleSentryState            = "vehicle-sentry-state-component"                              // i36
	compVehicleWeaponSet              = "vehicle-weapon-set-component"                                // i38
	compVehicleAutoTurret             = "vehicle-auto-turret-component"                               // i39
	compVehicleEquipmentTurretParent  = "vehicle-equipment-turret-parent-component"                   // i40
	compVehicleSeatsOverridePitch     = "vehicle-seats-override-pitch-component"                      // i41
	compVehicleSeatsOverrideYaw       = "vehicle-seats-override-yaw-component"                        // i42
	compAirDropFlight                 = "air-drop-flight-component"                                   // i45
	compWarp                          = "warp-component"                                              // i46
	compVehicleLowFrequency           = "vehicle-low-frequency-component"                             // i47
)

// compVehicleTypePhysics est l etiquette de registre de `ti=40 i34` : nom `143d0a838`, accesseur
// `141177260`, descripteur `143d0b308`, deser `FUN_142f02498` :
//
//	si objet[+0x818] :                        porte de la loi du masque (en-tete du fichier)
//	   c = R(1)                               mode = c ? 2 : 0
//	   FUN_140c5f938(lecteur, +0x7f4, +0x800, mode)   mode 2 : R(192) ; 0 : [decodeObjectForwardAndUp]
//	   FUN_14076e1c8(lecteur, +0x80c, mode)           mode 2 : R(96)  ; 0 : [consumeDynPrecVec3]
//
// C est EXACTEMENT la paire (avant/haut, vitesse angulaire) des composants dynamiques de precision
// de l archetype (`FUN_14076e1c8` est le lecteur de [consumeObjectAngularVelocity]). L ecrivain
// (`FUN_142f04e90`) teste le MEME octet et n ecrit rien sans lui.
const compVehicleTypePhysics = "vehicle-type-physics-component"

// compVehicleEmpTimer est l etiquette de registre de `ti=40 i37` : nom `143c995e8`, accesseur
// `141177300`, descripteur `143d0b498`, deser `FUN_142f049dc` -> `FUN_1432065d8` =
// `FUN_1406d84b4(..., 8, ...)` : R(8), le meme minuteur quantifie que celui du bipede (`i51`).
const compVehicleEmpTimer = "vehicle-emp-timer-component"

// Largeurs et categories lues chez les deserialiseurs (T6 §4 ; `FUN_1406d84b4` consomme la
// largeur que l appelant pousse en `[RSP+0x20]`).
const (
	// largeurMinuteurEMPVehicule : la largeur de `FUN_1432065d8` (i37).
	largeurMinuteurEMPVehicule = 8
	// largeurVecteurViseeTourelle : `FUN_14076dc04(..., 0x13)` @14115f346 (i31).
	largeurVecteurViseeTourelle = 19
	// largeurEtatOuverture : `MOV [RSP+0x20],0x8` @142f04ba1 (i32, apres le R(1)).
	largeurEtatOuverture = 8
	// largeurEtatDeType, largeurComplementEtatDeType : `FUN_142af27f8` R(2), puis R(6) quand
	// l etat lu vaut 1 ou 3 (`FUN_14320c4c8`, i33).
	largeurEtatDeType, largeurComplementEtatDeType = 2, 6
	// largeurEtatSentinelle : `FUN_1424d9a30` R(3) (i36, avant le R(1)).
	largeurEtatSentinelle = 3
	// largeurTourelleAuto : `FUN_142f04884` R(2) (`ADD [RDX+0x2c],0x2` @142f048aa, i39).
	largeurTourelleAuto = 2
	// largeurAngleDeSiege : `FUN_1406d84b4(..., 8, ...)` deux fois (i41 `FUN_142f04a4c`, i42
	// `FUN_142f04ac0`), bornes [-pi, +pi].
	largeurAngleDeSiege = 8
	// largeurEtatLargage, largeurTempsLargage, largeurHauteurLargage : `FUN_142af27f8` R(2) puis
	// `FUN_1406d84b4` a 0xe @142f02540 et a 0x8 @142f02562 (i45 `FUN_142f02508`).
	largeurEtatLargage, largeurTempsLargage, largeurHauteurLargage = 2, 14, 8
	// largeurValeurWarp : `FUN_1406d84b4` a 0x8 @142f04c1a et @142f04c48 (i46 `FUN_142f04bcc`).
	largeurValeurWarp = 8
	// categorieCibleTourelle : `FUN_1408f0ac4(+0x83c, lecteur, 1)` (`FUN_142f0496c`, i35).
	categorieCibleTourelle = 1
	// categorieParentTourelle : `FUN_1408f0ac4(+0x834, lecteur, 0)` (`FUN_142f04a00`, i40).
	categorieParentTourelle = 0
)

// consumeVehicleTypePhysics lit le corps de `ti=40 i34`, porte posee (cf. la constante).
func consumeVehicleTypePhysics(br *Lecteur) {
	if br.ReadBit() { // c : mode 2, les deux vecteurs bruts puis la vitesse brute
		br.ReadBits(fwdUpDynPrecMode2Bits)
		br.ReadBits(rawVec3Bits)
		return
	}
	consumeObjectForwardAndUp(br)                            // FUN_140c5f938 mode 0 -> FUN_140c5fa84
	consumeDynPrecVec3(br, angularMagBits, angularScaleBits) // FUN_14076e1c8 mode 0 -> FUN_14076d528
}

// consumeVehicleTypeState lit le corps de `ti=40 i33`, porte posee : `FUN_142f02474` ->
// `FUN_14320c4c8` : R(2) v (`FUN_142af27f8`) ; si v vaut 1 ou 3, R(6).
func consumeVehicleTypeState(br *Lecteur) {
	if v := br.ReadBits(largeurEtatDeType); v == 1 || v == 3 {
		br.ReadBits(largeurComplementEtatDeType)
	}
}

// lireComposantStatiqueTi40 lit un composant `ti=40` dont la largeur ne depend que du flux.
// Chaque cas cite le deserialiseur du jeu ; la liste est celle du cas « largeur du flux » de
// [consumeComposantsVehiculeTi40].
func lireComposantStatiqueTi40(br *Lecteur, name string) {
	switch name {
	case compVehicleAutoTurretTriggers: // i30 FUN_142f04994 : trois FUN_1406cf008
		br.ReadBit()
		br.ReadBit()
		br.ReadBit()
	case compVehicleAutoTurretAimingVector: // i31 FUN_14115f33c : R(19), FUN_1404fedf8 0 bit
		br.ReadBits(largeurVecteurViseeTourelle)
	case compVehicleTransformedOpenState: // i32 FUN_142f04b70 : R(1) puis R(8)
		br.ReadBit()
		br.ReadBits(largeurEtatOuverture)
	case compVehicleAutoTurretTarget: // i35 FUN_142f0496c -> FUN_1408f0ac4 categorie 1
		consume1408f0ac4(br, categorieCibleTourelle)
	case compVehicleSentryState: // i36 FUN_142f04b34 : R(3) puis R(1)
		br.ReadBits(largeurEtatSentinelle)
		br.ReadBit()
	case compVehicleWeaponSet: // i38 14116d3cc -> FUN_1406d01fc
		lireJeuDArmes(br)
	case compVehicleAutoTurret: // i39 FUN_142f04884 : R(2)
		br.ReadBits(largeurTourelleAuto)
	case compVehicleEquipmentTurretParent: // i40 FUN_142f04a00 -> FUN_1408f0ac4 categorie 0
		consume1408f0ac4(br, categorieParentTourelle)
	case compVehicleSeatsOverridePitch, compVehicleSeatsOverrideYaw: // i41, i42 : deux R(8)
		br.ReadBits(largeurAngleDeSiege)
		br.ReadBits(largeurAngleDeSiege)
	case compAirDropFlight: // i45 FUN_142f02508 : R(2), R(14), R(8)
		br.ReadBits(largeurEtatLargage)
		br.ReadBits(largeurTempsLargage)
		br.ReadBits(largeurHauteurLargage)
	case compWarp: // i46 FUN_142f04bcc : deux R(1), puis R(8) sous chacun
		lireWarp(br)
	case compVehicleLowFrequency: // i47 FUN_142f04a20 -> FUN_1407f1ff4 -> FUN_1407f2058
		consumeOpt5(br)
	}
}

// lireWarp lit `ti=40 i46` (`FUN_142f04bcc`) : deux drapeaux R(1) (`+0x8d4`, `+0x8d5`), puis
// R(8) sous chacun, dans cet ordre.
func lireWarp(br *Lecteur) {
	a, b := br.ReadBit(), br.ReadBit()
	if a {
		br.ReadBits(largeurValeurWarp)
	}
	if b {
		br.ReadBits(largeurValeurWarp)
	}
}

// consumeComposantsVehiculeTi40 est le DERNIER maillon de la chaine de dispatch (cf. l en-tete
// de `dispatch_object.go`) : les composants propres au vehicule. Son `default` rend le `default`
// d origine de la chaine (composant non porte : arret propre). Dans un etat complet d image-cle,
// seul `i37` se lit (cf. l en-tete du fichier).
func consumeComposantsVehiculeTi40(br *Lecteur, name string) (variant uint32, dead *types.DeadState, ported bool) {
	variant = noVariant
	switch name {
	case compVehicleEmpTimer: // i37 FUN_142f049dc -> FUN_1432065d8 : R(8), en tout contexte
		br.ReadBits(largeurMinuteurEMPVehicule)
		return variant, nil, true
	case compVehicleTypeState, compVehicleTypePhysics: // i33, i34 : porte de la loi du masque
		if br.etatComplet {
			return variant, nil, false
		}
		if name == compVehicleTypeState {
			consumeVehicleTypeState(br)
		} else {
			consumeVehicleTypePhysics(br)
		}
		return variant, nil, true
	case compVehicleAutoTurretTriggers, compVehicleAutoTurretAimingVector, compVehicleTransformedOpenState,
		compVehicleAutoTurretTarget, compVehicleSentryState, compVehicleWeaponSet, compVehicleAutoTurret,
		compVehicleEquipmentTurretParent, compVehicleSeatsOverridePitch, compVehicleSeatsOverrideYaw,
		compAirDropFlight, compWarp, compVehicleLowFrequency: // largeur du flux
		if br.etatComplet {
			return variant, nil, false
		}
		lireComposantStatiqueTi40(br, name)
		return variant, nil, true
	default:
		// Non porte (object position/velocity/angular/region/damage/constraint/parent/
		// scale/..., unit-actor-control/state/malleable, biped-* tail) : arret propre.
		return variant, nil, false
	}
}
