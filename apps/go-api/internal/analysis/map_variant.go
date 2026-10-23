package analysis

import "strings"

// BaseMapName rend le nom de la carte de base dont `name` est la variante, ou ("", false) quand
// `name` n'en est pas une.
//
// `name` est une variante quand il se termine par l'un des `suffixes` du titre
// (config/titles/{slug}/mappings/map_variants.toml), précédé d'une espace, et qu'il reste un nom
// non vide une fois le suffixe et les espaces de fin retirés : « Thunderhead Heavies » avec le
// suffixe « Heavies » donne « Thunderhead ». Un nom qui est le suffixe seul (« Heavies ») n'a pas de
// base. Le premier suffixe qui s'applique l'emporte. La comparaison est sensible à la casse : les
// suffixes sont recopiés des noms du registre.
//
// Aucune heuristique de longueur : sans suffixe déclaré, aucune carte n'est une variante.
func BaseMapName(name string, suffixes []string) (string, bool) {
	name = strings.TrimSpace(name)
	for _, suffix := range suffixes {
		if suffix == "" || !strings.HasSuffix(name, " "+suffix) {
			continue
		}
		base := strings.TrimSpace(strings.TrimSuffix(name, suffix))
		if base != "" {
			return base, true
		}
	}
	return "", false
}
