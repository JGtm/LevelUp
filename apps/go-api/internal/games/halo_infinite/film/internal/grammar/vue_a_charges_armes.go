package grammar

// vue_a_charges_armes.go — LES CHARGES DES MESSAGES D ARME, DE PROJECTILE ET D EQUIPEMENT DE LA VUE A
// (lot LN de la campagne de grammaire). Memes conventions que `vue_a_charges.go`.

// Les immediats des sites d appel de `FUN_14076dc04` et de `FUN_1406d84b4` lus dans ces lecteurs.
const (
	largeurMagnitudePoussee   = 10 // FUN_14076d528, sixieme argument de FUN_14116c344
	largeurDeuxiemeDirection  = 8  // FUN_1408096f8 : R9D = 8 quand le R(1) qui precede vaut 0
	largeurScalaireProjectile = 7  // FUN_1406d84b4, cinquieme argument 7 (FUN_1410f03b4)
	largeurDetonationIndex    = 9  // R(9) de FUN_1408096f8 et FUN_1410f03b4
)

// lireArmeEtVariante lit le couple commun aux lecteurs d arme : `FUN_14080d69c` puis
// `FUN_14080dec4` « variant_name ».
func lireArmeEtVariante(br *Lecteur) {
	consumeGateR(br, 32)
	br.Skip(32)
}

// lireVarianteSiAbsente lit `R(1) g ; si g == 0 : FUN_14080d69c, FUN_14080dec4` (FUN_1408096f8,
// FUN_1410f03b4, FUN_142f1c6cc).
func lireVarianteSiAbsente(br *Lecteur) {
	if !br.ReadBit() {
		lireArmeEtVariante(br)
	}
}

// chargeDetonation porte `FUN_1408096f8` (`projectile_detonate`).
func chargeDetonation(br *Lecteur) bool {
	br.Skip(6) // FUN_140809454
	lireVarianteSiAbsente(br)
	consumeGateR(br, 32)                                                    // FUN_14080d69c
	if _, ok := lireE494Sur(br, niveauDetonation, br.vueA.positions); !ok { // FUN_14076f91c ? R(96) : FUN_14076e524(..., 0xf)
		return false
	}
	br.Skip(largeurDirection)       // FUN_14076dc04, R9D = R14D = 0x13
	br.Skip(5 + 1)                  // FUN_1406d84b4 (5), FUN_1406cf008
	br.Skip(largeurDetonationIndex) // R(9)
	consumeGateR(br, 10+8)          // R(1) ; si 1 : FUN_140809530 = R(10) + R(8)
	largeur := largeurDeuxiemeDirection
	if br.ReadBit() {
		largeur = largeurDirection
	}
	br.Skip(largeur) // FUN_14076dc04
	br.Skip(2)       // FUN_1424cd2fc
	return true
}

// niveauDetonation et niveauImpact sont les immediats `R9D` des sites de `FUN_14076e524` dans
// `FUN_1408096f8` (0xf) et `FUN_1410f03b4` (0xc). A ces niveaux, la ligne de la table PAR INDEX se
// calcule par la loi (`FUN_140be9b88`) sur les bornes de la structure d index du scenario
// (`FUN_140be9a14`) : celles de la region jouee sont celles de l entree de catalogue de la carte du
// match ([tablesDeLaRegionJouee]) ; un autre index, ou un film lu sans carte, arrete la lecture.
const (
	niveauDetonation = 0xf
	niveauImpact     = 0xc
)

// chargeImpact porte `FUN_1410f03b4` (`projectile_impact_effect`).
func chargeImpact(br *Lecteur) bool {
	lireVarianteSiAbsente(br)
	br.Skip(2 * largeurScalaireProjectile)                              // FUN_1406d84b4 x 2
	br.Skip(largeurDirection)                                           // FUN_14076dc04, R9D = 0x13
	if _, ok := lireE494Sur(br, niveauImpact, br.vueA.positions); !ok { // FUN_14076f91c ? R(96) : FUN_14076e524(..., 0xc)
		return false
	}
	br.Skip(largeurDirection)           // FUN_14076dc04, R9D = 0x13
	br.Skip(largeurDetonationIndex + 1) // R(9), R(1)
	return true
}

// chargeCorpsACorps porte `FUN_140ff8d70` (`biped_melee_initiate`) : `FUN_141102ed0(0x28)` vaut la
// version native 2, donc la tete est R(3) ; si elle vaut 1, R(2).
func chargeCorpsACorps(br *Lecteur) bool {
	if br.ReadBits(3) == 1 {
		br.Skip(2)
	}
	br.Skip(2)
	lireArmeEtVariante(br)
	consumeGate0R(br, 5) // FUN_1407f2058
	return true
}

// chargeClicAVide porte `FUN_142f17ffc` (`weapon_empty_click`).
func chargeClicAVide(br *Lecteur) bool {
	consumeID2(br) // FUN_1406d00ec
	lireArmeEtVariante(br)
	br.Skip(1)
	consumeGate0R(br, 5) // FUN_1407f2034
	return true
}

// chargeLancerDArme porte `FUN_142f18490` (`weapon_throw`).
func chargeLancerDArme(br *Lecteur) bool {
	br.Skip(1)
	consumeID2(br)
	lireArmeEtVariante(br)
	consumeGate0R(br, 5) // FUN_1407f2034
	return true
}

// chargeArmeRangee porte `FUN_142f18284` (`weapon_put_away`).
func chargeArmeRangee(br *Lecteur) bool {
	br.Skip(1)
	consumeID2(br)
	lireArmeEtVariante(br)
	return true
}

// chargeArmeLachee porte `FUN_142f17d74` (`weapon_drop`).
func chargeArmeLachee(br *Lecteur) bool {
	br.Skip(1)
	consumeID2(br)
	lireArmeEtVariante(br)
	br.Skip(1)
	return true
}

// chargeArmeRamassee porte `FUN_142f18158` (`weapon_pickup`).
func chargeArmeRamassee(br *Lecteur) bool {
	br.Skip(largeurQueue1406d0f20) // FUN_1406d0f20
	consumeID2(br)
	consumeID2(br)
	br.Skip(3)
	return true
}

// chargeSurchauffe porte `FUN_142ef94f4` (`weapon_overheat`).
func chargeSurchauffe(br *Lecteur) bool {
	lireArmeEtVariante(br)
	consumeID2(br)
	return true
}

// chargeAttache porte `FUN_142f183f0` (`weapon_tether_request`) : `FUN_141102ed0(0x30)` vaut la
// version native 2, donc le R(1) final est lu.
func chargeAttache(br *Lecteur) bool {
	consumeID2(br)
	lireArmeEtVariante(br)
	br.Skip(1)
	return true
}

// chargeBonusApplique porte `FUN_142ef8a64` (`PowerUpApplied`).
func chargeBonusApplique(br *Lecteur) bool {
	lireArmeEtVariante(br)
	consumeGateR(br, 32)
	return true
}

// chargePorte32 porte les lecteurs qui ne lisent que `FUN_14080d69c` : `FUN_142ef9074`
// (`repair_complete`), `FUN_142f17480` (`projectile_supercombine_request`, suivi d appels sans
// lecture), et `FUN_141118a00` (`EquipmentObjectKnockedBack` : R(1) puis `FUN_14080d6f0`).
func chargePorte32(br *Lecteur) bool {
	consumeGateR(br, 32)
	return true
}

// chargeBalise porte `FUN_142ef8a40` (`NavpointRequest`) -> `FUN_142ef4a98` : R(32), R(32).
func chargeBalise(br *Lecteur) bool {
	br.Skip(64)
	return true
}

// chargeEquipement porte `FUN_142eebd68` (`Equipment`) : R(8), R(1).
func chargeEquipement(br *Lecteur) bool {
	br.Skip(8 + 1)
	return true
}

// chargeTeleportationDEquipement porte `FUN_142ef8ec8` (`equipment_teleport_request`).
func chargeTeleportationDEquipement(br *Lecteur) bool {
	br.Skip(4)           // FUN_1424e1d48
	consumeGate0R(br, 5) // FUN_1407f2034
	return true
}

// chargeFigureDeVehicule porte `FUN_142f17c84` (`vehicle_trick`).
func chargeFigureDeVehicule(br *Lecteur) bool {
	br.Skip(3)
	consumeGate0R(br, 5) // FUN_1407f2034
	return true
}

// chargePousseeDuJoueur porte `FUN_14116c344` (`EquipmentKnockbackPlayer`) -> `FUN_14076d528` :
// R(1) ; si 0 : R(0x13), R(10).
func chargePousseeDuJoueur(br *Lecteur) bool {
	consumeGate0R(br, largeurDirection+largeurMagnitudePoussee)
	return true
}

// chargeDemandeDePoussee porte `FUN_142eebcec` (`EquipmentKnockbackRequest`).
func chargeDemandeDePoussee(br *Lecteur) bool {
	br.Skip(largeurDirection) // FUN_14076dc04, R9D = 0x13
	n := br.ReadBits(4) + 1   // FUN_142ed0abc
	for range n {
		readVarWidthInt(br, 0)    // FUN_1406d3140(..., 0, ...)
		br.Skip(largeurDirection) // FUN_14076dc04, R9D = 0x13
	}
	return true
}

// chargeEsquive porte `FUN_142f169d0` (`biped_dodge`).
func chargeEsquive(br *Lecteur) bool {
	br.Skip(32)               // FUN_141015740
	br.Skip(largeurDirection) // FUN_14076dc04, R9D = 0x13
	br.Skip(8)
	consumeGate0R(br, 5) // FUN_1407f2034
	return true
}

// chargeUnBit porte `FUN_142f16c90` (`biped_laser_designation`) : R(1).
func chargeUnBit(br *Lecteur) bool {
	br.Skip(1)
	return true
}

// chargeMot32 porte `FUN_142f1615c` (`EngineClientEvent`) -> `FUN_142b19a90` : R(32).
func chargeMot32(br *Lecteur) bool {
	br.Skip(32)
	return true
}

// chargeDArme rend le lecteur de charge d un genre d arme, de projectile ou d equipement porte ici ;
// nil sinon.
func chargeDArme(genre int) func(*Lecteur) bool {
	switch genre {
	case 5:
		return chargeDetonation
	case 6:
		return chargeImpact
	case 11:
		return chargeClicAVide
	case 31:
		return chargeTeleportationDEquipement
	case 37:
		return chargeSurchauffe
	case 40:
		return chargeCorpsACorps
	case 41:
		return chargeFigureDeVehicule
	case 42:
		return chargeEsquive
	case 44:
		return chargeArmeRamassee
	case 45:
		return chargeArmeRangee
	case 46:
		return chargeArmeLachee
	case 47:
		return chargeLancerDArme
	case 48:
		return chargeAttache
	case 58, 105, 118:
		return chargePorte32
	case 63:
		return chargeUnBit
	case 86:
		return chargeMot32
	case 98:
		return chargeEquipement
	case 100:
		return chargeBonusApplique
	case 104:
		return chargePousseeDuJoueur
	case 108:
		return chargeBalise
	case 119:
		return chargeDemandeDePoussee
	}
	return nil
}
