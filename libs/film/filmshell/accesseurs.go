package filmshell

// accesseurs.go — LA LECTURE DU CATALOGUE. Les tables sont des variables NON exportees, remplies
// une fois a l'initialisation du paquet ; leurs lecteurs passent par ces fonctions, qui consultent
// ou parcourent la table sans jamais la livrer : aucun lecteur ne peut la modifier.

import "iter"

// NomDArme rend le nom canonique de l'arme `id`, et si elle est au catalogue.
func NomDArme(id uint64) (string, bool) {
	n, ok := weaponIDToName[id]
	return n, ok
}

// NombreDArmes rend le nombre d'armes du catalogue.
func NombreDArmes() int { return len(weaponIDToName) }

// Armes parcourt le catalogue, (identifiant, nom), dans l'ordre d'une map.
func Armes() iter.Seq2[uint64, string] {
	return func(yield func(uint64, string) bool) {
		for id, nom := range weaponIDToName {
			if !yield(id, nom) {
				return
			}
		}
	}
}

// EstUneArme dit si l'identifiant `id` est au catalogue.
func EstUneArme(id uint64) bool { return weaponIDs[id] }

// EstOctetsDArme dit si les huit octets filmshell `b` sont ceux d'une arme du catalogue.
func EstOctetsDArme(b [8]byte) bool { return weaponBytesMap[b] }

// OctetsDesArmes parcourt les huit octets filmshell de chaque arme, dans l'ordre d'une map.
func OctetsDesArmes() iter.Seq[[8]byte] {
	return func(yield func([8]byte) bool) {
		for b := range weaponBytesMap {
			if !yield(b) {
				return
			}
		}
	}
}

// SuffixeCommun rend les quatre derniers octets de la majorite des identifiants d'arme.
func SuffixeCommun() [4]byte { return commonWeaponSuffix }

// SuffixesFormuleA parcourt les suffixes de quatre octets distincts des armes du catalogue, dans
// l'ordre d'une map.
func SuffixesFormuleA() iter.Seq[[4]byte] {
	return func(yield func([4]byte) bool) {
		for s := range allFormulaASuffixes {
			if !yield(s) {
				return
			}
		}
	}
}

// NomCanonique rend la tete de famille d'une variante d'arme nommee (`Duelist Energy Sword` ->
// `Energy Sword`), et si `nom` est une variante.
func NomCanonique(nom string) (string, bool) {
	c, ok := weaponFusionMap[nom]
	return c, ok
}

// IDCanonique rend l'identifiant de la tete de famille d'une variante, et si `id` est une variante.
func IDCanonique(id uint64) (uint64, bool) {
	c, ok := weaponFusionMapID[id]
	return c, ok
}

// EstSentinelle dit si `id` est un identifiant reserve non-arme (melee, grenade, vehicule).
func EstSentinelle(id uint64) bool { return sentinelIDs[id] }

// Sentinelles parcourt les identifiants reserves non-arme, dans l'ordre d'une map.
func Sentinelles() iter.Seq[uint64] {
	return func(yield func(uint64) bool) {
		for id := range sentinelIDs {
			if !yield(id) {
				return
			}
		}
	}
}
