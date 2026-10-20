package modelabel

import (
	"regexp"
	"strings"
)

// mapSuffixRe — le suffixe « sur <carte> » / « on <map> » d'un pair_name, FR et EN.
//
// UNE SEULE REGEX DANS LE DÉPÔT (règle CLAUDE.md n°6) : `analysis.NormalizeModeLabel`
// l'appelle pour son étape 3, et l'appariement du bloc « Score dans le temps » l'appelle
// seul, sans le reste de la normalisation. Deux expressions du même retrait, ce serait deux
// façons de couper « Slayer on Forest » — et l'une des deux finirait par laisser passer un
// nom de carte dans l'appariement.
var mapSuffixRe = regexp.MustCompile(`(?i)\s+(?:on|sur)\s+.+$`)

// StripMapSuffix retire le suffixe de CARTE d'un libellé de mode et rend le reste, trimé.
//
// POURQUOI CE RETRAIT EST INDISPENSABLE À L'APPARIEMENT, et pourquoi on ne peut pas le
// sauter : les jetons se cherchent comme mots entiers, et un nom de carte peut porter un mot
// de mode. Sans ce retrait, « Slayer on Bazaar » et « Arena:CTF on Streets » resteraient
// justes, mais rien ne protégerait d'une carte nommée d'après un mode.
//
// CE QU'IL NE FAIT PAS, ET C'EST LA DIFFÉRENCE AVEC `NormalizeModeLabel` : il ne touche NI
// au préfixe de playlist, NI au sous-mode. « Super Fiesta:Slayer on Forbidden - Forge » rend
// « Super Fiesta:Slayer », qui porte encore son jeton « Slayer » — là où la normalisation
// complète rendrait « Super Fiesta » et le perdrait.
func StripMapSuffix(label string) string {
	return strings.TrimSpace(mapSuffixRe.ReplaceAllString(strings.TrimSpace(label), ""))
}
