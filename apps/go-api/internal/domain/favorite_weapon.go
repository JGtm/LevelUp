package domain

// favorite_weapon.go — QUI PEUT ETRE « L ARME FAVORITE » D UN JOUEUR (scoreboard du match, top des
// armes de l Explorateur, encart de l Accueil).
//
// AVANT LE 2026-09-24, LA REGLE ETAIT IMPLICITE : `WeaponID != 0`. Elle tenait par une propriete du
// registre d armes de Halo Infinite — aucun objet hors arsenal (vehicule, tourelle, equipement,
// environnement) n y avait d identifiant numerique (garde-rail
// `weapons.TestHorsArsenalHINFSansIdNumerique`). Le lot M6 des retours du rejeu a donne a la bobine
// a fusion UNSC (classe `environmental`) les identifiants de l objet que le joueur TIENT, pour la
// nommer sur les fiches du rejeu : la propriete est tombee, et la bobine serait devenue candidate
// (revue adverse du lot, constat R1). LA REGLE EST DESORMAIS ECRITE, ET PAR LA CLASSE.
//
// POURQUOI LA PROVENANCE. Une ligne MESUREE dans la source de degat du film (Halo Infinite) porte
// une classe hors arsenal pour chaque objet ; une ligne de `weapon_kills` (Halo 5) garde la regle
// d avant, a l identique : son bucket « Spartan » (`h5_unattributed`, id 3168248199) peut etre
// l arme favorite aujourd hui — le changer ne releve pas de ce lot (consigne en decouverte).

// IsFavoriteWeaponCandidate dit si une ligne d arme peut etre l arme favorite d un joueur : un
// identifiant numerique (la vignette en depend), et — pour une ligne mesuree dans le film — une
// classe de l ARSENAL, jamais un objet (vehicule, tourelle, equipement, environnement) ni un
// bucket non-combat.
func IsFavoriteWeaponCandidate(weaponID int64, class string, fromDamageSource bool) bool {
	if weaponID == 0 {
		return false
	}
	if !fromDamageSource {
		return true
	}
	return !IsPerWeaponFragClass(class) && !IsNonCombatFragClass(class)
}
