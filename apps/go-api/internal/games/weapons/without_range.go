package weapons

// without_range.go — L'ATTRIBUT « SANS PORTÉE » du registre d'armes canonique.
//
// Une source de dégât est SANS PORTÉE quand la distance tueur -> victime au coup fatal ne
// mesure pas une portée d'engagement :
//   - les ARMES DE CONTACT, qui ne tuent qu'au corps à corps (épée à énergie, marteau
//     antigravité, mains nues ; club de golf et balle sur Halo 5) — leur « portée » est
//     toujours celle du contact, la publier n'apprend rien et tasse l'axe du graphe ;
//   - la CHUTE ET L'ENVIRONNEMENT, où la distance relie la victime à un point de décor.
//
// ATTRIBUT DISTINCT DE LA CLASSE ET DU RÔLE, à dessein : la classe et le rôle alimentent la
// répartition des frags, le profil d'armes du Face-à-face et l'Explorer (l'épée et le marteau
// d'Infinite y restent des armes lourdes de rôle `power`). Cet attribut ne sert QU'À écarter
// une ligne d'un graphe de portée par arme ; il ne reclasse rien.
//
// Déclaré ici, dans le registre, et nulle part ailleurs : aucun filtre de noms côté web ni
// côté service. Garde-rail : `TestWithoutRange_ClesConnuesDuRegistre` (toute clé citée existe
// dans `weaponRegistryWeapons`).
var weaponKeysWithoutRange = map[string]bool{
	keyHinfEnergySword:   true,
	keyHinfGravityHammer: true,
	keyHinfUnarmed:       true,
	keyHinfEnvironment:   true,
	"h5_energy_sword":    true,
	"h5_gravity_hammer":  true,
	"h5_golf_club":       true,
	"h5_oddball":         true,
}

// IsWithoutRange dit si la source `weaponKey` est SANS PORTÉE (cf. en-tête du fichier).
// Une clé inconnue du registre n'est pas sans portée : elle reste publiée.
//
// STATIQUE ET IN-PROCESS, comme [RolesByKey] : aucune ouverture de base.
func IsWithoutRange(weaponKey string) bool {
	return weaponKeysWithoutRange[weaponKey]
}

// keyHinfEnvironment : la chute et l'environnement d'Infinite (une seule entrée de registre pour
// les neuf sources de dégât globales), partagée avec la ligne du registre.
const keyHinfEnvironment = "hinf_environment"
