package canonical

import "strings"

// Jeu canonique de catégories de citations / commendations, partagé par TOUS les
// titres — même rôle que MedalCategory* pour les médailles.
//
// Le frontend traduit EXACTEMENT ces clés via le manifeste
// apps/web/src/lib/i18n/manifests/citations.toml (clés `citations.category.<clé>`,
// résolues par features/citations/labels.ts) ; toute valeur hors de ce set
// retomberait sur la clé brute affichée telle quelle. La normalisation ici est
// donc la frontière title-agnostic : aucun libellé humain (FR ou EN) ne doit
// franchir la couche API.
//
// Valeurs brutes constatées en base (2026-08-04) :
//   - Halo 5, commendation_definitions.category (champ category.name de l'API
//     Metadata officielle, EN majuscules) : MULTIPLAYER, GAME MODE, WEAPON,
//     VEHICLE, ENEMY. Source externe non maîtrisée : la normalisation de ces
//     libellés est PERMANENTE.
//   - Halo Infinite, citation_mappings.category : les clés canoniques ci-dessous
//     depuis le 2026-08-04 (le seed écrivait auparavant des libellés FR ; migration
//     de données `normalize_citation_mappings_category_keys`).
//
// Les deux titres décrivent la même taxonomie dans deux langues : une seule
// table de normalisation les couvre.
const (
	CommendationCategoryMultiplayer      = "multiplayer"
	CommendationCategoryGameMode         = "game_mode"
	CommendationCategoryWeapon           = "weapon"
	CommendationCategoryVehicle          = "vehicle"
	CommendationCategoryEnemy            = "enemy"
	CommendationCategorySpartanCompanies = "spartan_companies"
	CommendationCategoryOther            = "other"
)

// allCommendationCategories : ordre d'AFFICHAGE produit, du plus général au plus
// spécifique, « other » en dernier. Cet ordre remplace l'ancien tri alphabétique
// sur les libellés bruts, qui dépendait de la langue de la donnée (« Arme » en
// premier côté Infinite, « ENEMY » côté Halo 5) — donc instable entre titres.
var allCommendationCategories = []string{
	CommendationCategoryMultiplayer,
	CommendationCategoryGameMode,
	CommendationCategoryWeapon,
	CommendationCategoryVehicle,
	CommendationCategoryEnemy,
	CommendationCategorySpartanCompanies,
	CommendationCategoryOther,
}

// AllCommendationCategories retourne la liste exhaustive des catégories
// canoniques, dans l'ordre d'affichage. Chaque clé DOIT avoir une entrée
// [citations.category.<clé>] dans citations.toml.
func AllCommendationCategories() []string {
	out := make([]string, len(allCommendationCategories))
	copy(out, allCommendationCategories)
	return out
}

// commendationCategoryRanks : clé -> rang d'affichage (index dans l'ordre ci-dessus).
var commendationCategoryRanks = func() map[string]int {
	m := make(map[string]int, len(allCommendationCategories))
	for i, k := range allCommendationCategories {
		m[k] = i
	}
	return m
}()

// CommendationCategoryRank retourne le rang d'affichage d'une catégorie
// canonique. Une clé inconnue (ne devrait pas exister, la normalisation renvoie
// toujours une clé du set) est classée après « other » pour rester déterministe.
func CommendationCategoryRank(key string) int {
	if r, ok := commendationCategoryRanks[key]; ok {
		return r
	}
	return len(allCommendationCategories)
}

// NormalizeCommendationCategory ramène une catégorie de citation brute (peu
// importe le titre et la langue d'origine) vers l'une des clés canoniques
// multiplayer|game_mode|weapon|vehicle|enemy|spartan_companies|other.
//
// Idempotente et insensible à la casse : elle accepte les libellés EN de l'API
// Halo 5 (« GAME MODE » — source externe, tolérance PERMANENTE) et les clés déjà
// normalisées (« game_mode »). Le sentinelle SQL historique « misc » (COALESCE de
// citation_mappings.category) et la chaîne vide retombent sur « other », comme
// toute valeur inconnue.
//
// TOLÉRANCE DE TRANSITION — les libellés FR du seed Infinite (« Mode de jeu »,
// « Véhicule », …) : le seed et la migration de données
// `normalize_citation_mappings_category_keys` écrivent des clés stables depuis le
// 2026-08-04. Cette tolérance couvre les bases non encore migrées.
// Critère de retrait : plus aucune ligne de citation_mappings.category hors du set
// canonique sur les bases prod (SELECT DISTINCT category FROM citation_mappings).
// Date cible de retrait : 2026-11-04. Garde-rail : commendation_category_test.go.
func NormalizeCommendationCategory(raw string) string {
	switch normalizeCommendationCategoryKey(raw) {
	case "multiplayer", "multijoueur":
		return CommendationCategoryMultiplayer
	case "game mode", "game_mode", "gamemode", "mode de jeu", "mode_de_jeu", "modedejeu":
		return CommendationCategoryGameMode
	case "weapon", "weapons", "arme", "armes":
		return CommendationCategoryWeapon
	case "vehicle", "vehicles", "vehicule", "vehicules":
		return CommendationCategoryVehicle
	case "enemy", "enemies", "ennemi", "ennemis":
		return CommendationCategoryEnemy
	case "spartan companies", "spartan_companies", "spartancompanies", "spartan company":
		return CommendationCategorySpartanCompanies
	default:
		return CommendationCategoryOther
	}
}

// normalizeCommendationCategoryKey met la valeur brute en forme comparable :
// minuscules, espaces de bord retirés, accents des libellés FR aplatis
// (« Véhicule » -> « vehicule ») pour éviter d'avoir à lister chaque variante
// accentuée dans le switch.
func normalizeCommendationCategoryKey(raw string) string {
	lowered := strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	b.Grow(len(lowered))
	for _, r := range lowered {
		switch r {
		case 'à', 'â', 'ä':
			b.WriteRune('a')
		case 'é', 'è', 'ê', 'ë':
			b.WriteRune('e')
		case 'î', 'ï':
			b.WriteRune('i')
		case 'ô', 'ö':
			b.WriteRune('o')
		case 'ù', 'û', 'ü':
			b.WriteRune('u')
		case 'ç':
			b.WriteRune('c')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
