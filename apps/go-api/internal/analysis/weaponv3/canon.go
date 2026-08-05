package weaponv3

// canon.go — identité d'arme par high-32 (fold des variantes).
//
// Règle (réf .ai/REFERENCE_WEAPON_IDS.md) : l'identité d'une arme = ses 32 bits
// HAUTS (high-32 = uint32(id>>32)). Le suffixe bas (le plus souvent 0x42c9679f)
// est ignoré : il porte la variante cosmétique/upgrade et NON l'identité du type
// d'arme. Folder par high-32 récupère les variantes partageant la même base
// (Gravity Hammer → Diminisher of Hope/Rushdown Hammer, Energy Sword → Duelist/
// Elite Bloodblade, Sentinel Beam → sa variante a0955e9e…).
//
// Les bases DISTINCTES (high-32 propre) restent des entrées séparées : Infected
// Energy Sword 0x0C55765F, MA5K Avenger 0xF5C335DF, Mythic Sandwich 0xB7262CA1,
// Shock Rifle Ranked 0x1A22FEE6.
//
// Valider le high-32 contre ce set est la défense anti-bruit du parser FormulaA
// (cf. REFERENCE §"VÉRIFIÉ workflows v3" : 137 ids junk = high-32 aléatoires).
//
// SOURCE UNIQUE (DRY) : le set est DÉRIVÉ de l'enum d'armes v2 (analysis.WeaponIDToName,
// liste maître weapon_data.go), PAS d'une hand-list qui dérive. Toute arme ajoutée à
// l'enum v2 (sandbox/Fiesta/grenades) est ainsi automatiquement connue du canon.

import "levelup/go-api/internal/analysis"

// KnownWeaponHigh32 — map high-32 → nom canonique, dérivée de l'enum v2.
var KnownWeaponHigh32 = buildKnownWeaponHigh32()

// buildKnownWeaponHigh32 dérive le set high-32 → nom depuis analysis.WeaponIDToName.
// Le fold par high-32 est DÉTERMINISTE (indépendant de l'ordre d'itération de la map) :
// toutes les variantes partageant un high-32 (Energy Sword/Duelist/Bloodblade ;
// Gravity Hammer/Diminisher/Rushdown) résolvent vers le MÊME nom canonique via
// analysis.WeaponFusionMap, donc l'écriture est identique quel que soit l'ordre.
func buildKnownWeaponHigh32() map[uint32]string {
	m := make(map[uint32]string, len(analysis.WeaponIDToName))
	for id, name := range analysis.WeaponIDToName {
		if canon, ok := analysis.WeaponFusionMap[name]; ok {
			name = canon // fold variante → tête de famille
		}
		m[uint32(id>>32)] = name
	}
	return m
}

// commonWeaponSuffix — low-32 partagé par la majorité des weapon-ids réels
// (cf. analysis.CommonWeaponSuffix {0x42,0xc9,0x67,0x9f}, journal §C/§L). C'est LE
// tag d'identité weapon-id : un id qui le porte est une VRAIE arme, même si son
// high-32 est absent de l'enum v2.
const commonWeaponSuffix uint32 = 0x42c9679f

// CanonWeaponID extrait le high-32 (identité) d'un weapon-id 64-bit et indique s'il
// correspond à une arme RÉELLE. Deux signaux d'identité (le suffixe bas est sinon
// ignoré pour le fold des variantes) :
//  1. high-32 ∈ enum v2 (armes nommées + variantes à suffixe spécial : MA5K,
//     Mythic Sandwich, Infected Sword, grenades) ;
//  2. suffixe low-32 == 0x42c9679f = tag weapon-id réel. Couvre les armes
//     sandbox/Fiesta absentes de l'enum (l'enum a été bâti sur du jeu standard).
//     VÉRIFIÉ 000d5950 (Super Fiesta) : 69 occurrences étaient rejetées à tort par
//     le seul check high-32, TOUTES suffixe 42c9679f + récurrentes (14×, 13×…) =
//     vraies armes, pas du bruit object-id (qui a un suffixe aléatoire). Le bruit
//     FormulaA (object-ids §C) n'a PAS ce suffixe → toujours rejeté.
func CanonWeaponID(id uint64) (high uint32, known bool) {
	high = uint32(id >> 32)
	if _, ok := KnownWeaponHigh32[high]; ok {
		return high, true
	}
	if uint32(id&0xffffffff) == commonWeaponSuffix {
		return high, true
	}
	return high, false
}

// WeaponName retourne le nom canonique d'un high-32, ou "" si inconnu.
func WeaponName(high uint32) string {
	return KnownWeaponHigh32[high]
}
