package modelabel

import (
	"regexp"
	"strings"
)

// category.go - LA règle de catégorie parente d'un pair_name Halo Infinite, et il n'y en a
// qu'une dans le dépôt. Elle sert l'interface (filtres, en-tête de match, taxonomie injectée
// dans analysis.ModeTaxonomy par games/halo_infinite), l'écrivain de
// match_registry.mode_category (sync) et la migration qui répare cette colonne (migration) :
// tous passent ici, aucun ne recopie la table des préfixes. Garde-rail :
// sync/mode_category_single_rule_test.go.
//
// Deux niveaux ORTHOGONAUX pour la sémantique d'un pair_name :
//
//  1. SOUS-MODE (libellé affiché) : "Arena:Slayer on Bazaar" -> "Slayer" (strip.go).
//  2. CATÉGORIE PARENTE (filtre Mode, regroupements) : "Arena:Slayer on Bazaar" -> "Assassin".
//
// Une catégorie regroupe plusieurs préfixes de pair_name :
//
//	Assassin     : Arena, Tactical, Assault, Community
//	Fiesta       : Fiesta, Castle Wars
//	Super Fiesta : Super Fiesta
//	Husky Raid   : Husky Raid, Super Husky Raid
//	BTB          : BTB, BTB Heavies
//	Ranked       : Ranked
//	Firefight    : Firefight, Gruntpocalypse
//	Other        : Event, et tout préfixe inconnu (y compris un pair_name qui vaut encore son
//	               identifiant d'asset : la catégorie se recalcule quand le nom se résout)
//
// Super Fiesta et Husky Raid sont des catégories distinctes (le cockpit Python les regroupait
// sous Fiesta) : ce sont des rotations d'événement que les joueurs identifient.
//
// Source de référence Python : `git show v7/cockpit:src/analysis/mode_categories.py`.

// Catégories canoniques retournées par InferCategory. Stables : étiquettes de l'interface et
// valeurs stockées dans match_registry.mode_category.
const (
	CategoryAssassin    = "Assassin"
	CategoryFiesta      = "Fiesta"
	CategorySuperFiesta = "Super Fiesta"
	CategoryHuskyRaid   = "Husky Raid"
	CategoryBTB         = "BTB"
	CategoryRanked      = "Ranked"
	CategoryFirefight   = "Firefight"
	CategoryOther       = "Other"
)

// Préfixes de playlist qui apparaissent à plusieurs endroits (clés de prefixToCategory et de
// prefixCaseMap).
const (
	PrefixCastleWars     = "Castle Wars"
	PrefixBTBHeavies     = "BTB Heavies"
	PrefixSuperHuskyRaid = "Super Husky Raid"
)

// prefixToCategory : préfixe (gauche du ":" d'un pair_name, casse normalisée) -> catégorie.
// Firefight et Gruntpocalypse y sont des CATÉGORIES, pas des conteneurs de grammaire
// (« Gruntpocalypse:Fiesta » a Gruntpocalypse pour mode) : la liste des conteneurs de
// container.go est une autre notion, ne pas en ajouter une troisième.
var prefixToCategory = map[string]string{
	"Arena":              CategoryAssassin,
	"Tactical":           CategoryAssassin,
	"Assault":            CategoryAssassin,
	"Community":          CategoryAssassin,
	"Fiesta":             CategoryFiesta,
	CategorySuperFiesta:  CategorySuperFiesta,
	CategoryHuskyRaid:    CategoryHuskyRaid,
	PrefixSuperHuskyRaid: CategoryHuskyRaid,
	PrefixCastleWars:     CategoryFiesta,
	"BTB":                CategoryBTB,
	PrefixBTBHeavies:     CategoryBTB,
	"Ranked":             CategoryRanked,
	"Firefight":          CategoryFirefight,
	"Gruntpocalypse":     CategoryFirefight,
	"Event":              CategoryOther,
}

// prefixCaseMap : préfixes dont la casse canonique n'est pas la capitalisation par mot.
var prefixCaseMap = map[string]string{
	"btb heavies":      PrefixBTBHeavies,
	"btb":              "BTB",
	"super fiesta":     CategorySuperFiesta,
	"super husky raid": PrefixSuperHuskyRaid,
	"husky raid":       CategoryHuskyRaid,
	"castle wars":      PrefixCastleWars,
}

var idSuffixRe = regexp.MustCompile(`(?i)^(.*?)(?:\s*[\-–—]\s*[0-9A-Za-z]{8,})$`)

// stripForCategory retire le suffixe " on Carte" et un éventuel suffixe d'identifiant technique
// (8+ caractères alphanumériques après " - ").
func stripForCategory(s string) string {
	if i := strings.Index(s, " on "); i >= 0 {
		s = s[:i]
	}
	if m := idSuffixRe.FindStringSubmatch(s); m != nil {
		s = strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(s)
}

// normalizePrefixCase normalise la casse d'un préfixe pour la recherche dans prefixToCategory :
// acronymes et multi-mots via prefixCaseMap, tout-majuscules conservé, sinon capitalisation par
// mot.
func normalizePrefixCase(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return ""
	}
	if v, ok := prefixCaseMap[strings.ToLower(prefix)]; ok {
		return v
	}
	if prefix == strings.ToUpper(prefix) {
		return prefix
	}
	parts := strings.Fields(prefix)
	for i, p := range parts {
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, " ")
}

// InferCategory rend la catégorie parente d'un pair_name brut. Gère le format normal
// ("Arena:Slayer on Bazaar" -> Assassin), le format inversé quand la gauche est inconnue
// ("CTF:Arena" -> Assassin) et l'absence de séparateur ("Husky Raid" -> Husky Raid,
// "Sniper Slayer" -> Other). Un pair_name vide ou inconnu vaut Other.
func InferCategory(pairName string) string {
	raw := stripForCategory(strings.TrimSpace(pairName))
	if raw == "" {
		return CategoryOther
	}
	if !strings.Contains(raw, ":") {
		if cat, ok := prefixToCategory[normalizePrefixCase(raw)]; ok {
			return cat
		}
		return CategoryOther
	}
	left, right, _ := strings.Cut(raw, ":")
	leftCanon := normalizePrefixCase(left)
	rightCanon := normalizePrefixCase(right)
	_, leftIsPrefix := prefixToCategory[leftCanon]
	_, rightIsPrefix := prefixToCategory[rightCanon]
	prefix := leftCanon
	if rightIsPrefix && !leftIsPrefix {
		prefix = rightCanon
	}
	if cat, ok := prefixToCategory[prefix]; ok {
		return cat
	}
	return CategoryOther
}

// PrefixesForCategory rend les préfixes EN rangés dans la catégorie : de quoi construire le
// WHERE d'un filtre (`pair_name = p OR pair_name LIKE 'p:%'`). nil pour Other ou "" : l'appelant
// utilise KnownPrefixes pour un NOT IN.
func PrefixesForCategory(category string) []string {
	if category == "" || category == CategoryOther {
		return nil
	}
	var out []string
	for prefix, cat := range prefixToCategory {
		if cat == category {
			out = append(out, prefix)
		}
	}
	return out
}

// KnownPrefixes rend tous les préfixes rangés dans une catégorie autre que Other.
func KnownPrefixes() []string {
	out := make([]string, 0, len(prefixToCategory))
	for prefix, cat := range prefixToCategory {
		if cat == CategoryOther {
			continue
		}
		out = append(out, prefix)
	}
	return out
}
