// Package analysis — mode_category.go : portage Go de
// `src.analysis.mode_categories` / `src.analysis.mode_display` (branche v7/cockpit Python).
//
// Concept : une `mode_category` custom regroupe plusieurs préfixes de pair_name :
//
//	Assassin  : Arena, Tactical, Assault, Community
//	Fiesta    : Fiesta, Super Fiesta, Husky Raid, Super Husky Raid, Castle Wars
//	BTB       : BTB, BTB Heavies
//	Ranked    : Ranked
//	Firefight : Firefight, Gruntpocalypse
//	Other     : tout le reste (Event, etc.)
//
// Source : src/analysis/mode_display.go _PREFIX_RULES + src/analysis/mode_categories.PREFIX_TO_CATEGORY.
// Le filtre "Mode" exposé à l'UI utilise ces 6 catégories — pas les sous-modes
// (Slayer/CTF/KOTH/etc.) qui sont des spécialisations intra-catégorie.
package analysis

import (
	"regexp"
	"strings"
)

// ModeCategoryAssassin et autres : valeurs canoniques retournées par
// InferModeCategoryFromPairName. Stables — utilisées comme labels dans l'UI.
const (
	ModeCategoryAssassin  = "Assassin"
	ModeCategoryFiesta    = "Fiesta"
	ModeCategoryBTB       = "BTB"
	ModeCategoryRanked    = "Ranked"
	ModeCategoryFirefight = "Firefight"
	ModeCategoryOther     = "Other"
)

// modePrefixToCategory mappe le préfixe (gauche du ":" dans pair_name, casse normalisée)
// vers la catégorie custom. Miroir Go de _PREFIX_RULES Python.
//
// La traduction FR éventuelle (ex: "Arène", "Communauté", "Classé") est gérée
// côté inférence en testant les variantes via _CASE_MAP — ici on stocke les
// préfixes EN canoniques (cf. Python _normalize_case).
var modePrefixToCategory = map[string]string{
	"Arena":            ModeCategoryAssassin,
	"Tactical":         ModeCategoryAssassin,
	"Assault":          ModeCategoryAssassin,
	"Community":        ModeCategoryAssassin,
	"Fiesta":           ModeCategoryFiesta,
	"Super Fiesta":     ModeCategoryFiesta,
	"Husky Raid":       ModeCategoryFiesta,
	"Super Husky Raid": ModeCategoryFiesta,
	"Castle Wars":      ModeCategoryFiesta,
	"BTB":              ModeCategoryBTB,
	"BTB Heavies":      ModeCategoryBTB,
	"Ranked":           ModeCategoryRanked,
	"Firefight":        ModeCategoryFirefight,
	"Gruntpocalypse":   ModeCategoryFirefight,
	"Event":            ModeCategoryOther,
}

// modeCaseMap — préfixes spéciaux dont la casse doit être conservée (acronymes
// ou multi-mots usuels). Miroir Python _CASE_MAP.
var modeCaseMap = map[string]string{
	"btb heavies":      "BTB Heavies",
	"btb":              "BTB",
	"super fiesta":     "Super Fiesta",
	"super husky raid": "Super Husky Raid",
	"husky raid":       "Husky Raid",
	"castle wars":      "Castle Wars",
}

var modeMapSuffixRe = regexp.MustCompile(`(?i)^(.*?)(?:\s*[\-–—]\s*[0-9A-Za-z]{8,})$`)

// stripMapSuffix retire le suffixe " on MapName" et un éventuel suffixe ID
// technique (8+ caractères alphanum après " - "). Miroir Python _strip_map_suffix.
func stripMapSuffix(s string) string {
	if i := strings.Index(s, " on "); i >= 0 {
		s = s[:i]
	}
	if m := modeMapSuffixRe.FindStringSubmatch(s); m != nil {
		s = strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(s)
}

// normalizeModeCase normalise la casse d'un préfixe pour le lookup dans
// modePrefixToCategory. Mêmes règles que Python _normalize_case :
//
//	"btb"           → "BTB"           (via modeCaseMap)
//	"super fiesta"  → "Super Fiesta"  (via modeCaseMap)
//	"ARENA"         → "ARENA"         (tout-majuscules conservé)
//	"team slayer"   → "Team Slayer"   (title case)
func normalizeModeCase(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return ""
	}
	if v, ok := modeCaseMap[strings.ToLower(prefix)]; ok {
		return v
	}
	if prefix == strings.ToUpper(prefix) {
		return prefix
	}
	parts := strings.Fields(prefix)
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, " ")
}

// InferModeCategoryFromPairName retourne la catégorie custom (Assassin/Fiesta/
// BTB/Ranked/Firefight/Other) inférée depuis un pair_name brut.
//
// Gère :
//   - Format normal : "Arena:Slayer on Bazaar" → "Assassin"
//   - Format inversé : "CTF:Arena" → "Assassin" (préfixe à droite si gauche est inconnu)
//   - Sans séparateur : "Husky Raid" → "Fiesta", "Sniper Slayer" → "Other"
//
// Miroir Python infer_custom_category_from_pair_name + infer_mode_category_from_pair_name.
func InferModeCategoryFromPairName(pairName string) string {
	if strings.TrimSpace(pairName) == "" {
		return ModeCategoryOther
	}
	raw := stripMapSuffix(strings.TrimSpace(pairName))
	if raw == "" {
		return ModeCategoryOther
	}

	// Pas de séparateur : tester si le label complet matche un préfixe connu
	// (ex: "Husky Raid" sans sous-mode → Fiesta)
	if !strings.Contains(raw, ":") {
		if cat, ok := modePrefixToCategory[normalizeModeCase(raw)]; ok {
			return cat
		}
		return ModeCategoryOther
	}

	left, right, _ := strings.Cut(raw, ":")
	leftCanon := normalizeModeCase(left)
	rightCanon := normalizeModeCase(right)
	_, leftIsPrefix := modePrefixToCategory[leftCanon]
	_, rightIsPrefix := modePrefixToCategory[rightCanon]
	prefix := leftCanon
	if rightIsPrefix && !leftIsPrefix {
		prefix = rightCanon
	}
	if cat, ok := modePrefixToCategory[prefix]; ok {
		return cat
	}
	return ModeCategoryOther
}

// PairNamePrefixesForCategory retourne la liste des préfixes EN qui sont
// rangés dans la catégorie donnée. Utilisé pour construire le WHERE inverse :
// quand l'utilisateur filtre "Fiesta", on génère
// `WHERE pair_name LIKE 'Fiesta:%' OR LIKE 'Super Fiesta:%' OR LIKE 'Husky Raid:%' OR ...`
// (et aussi "= 'Fiesta'", "= 'Husky Raid'" pour les modes sans `:`).
//
// Pour la catégorie "Other" retourne nil — l'appelant doit utiliser
// AllKnownPairNamePrefixes() pour construire un NOT IN.
func PairNamePrefixesForCategory(category string) []string {
	if category == "" || category == ModeCategoryOther {
		return nil
	}
	var out []string
	for prefix, cat := range modePrefixToCategory {
		if cat == category {
			out = append(out, prefix)
		}
	}
	return out
}

// AllKnownPairNamePrefixes retourne tous les préfixes EN rangés dans une
// catégorie connue (Assassin/Fiesta/BTB/Ranked/Firefight). Utilisé pour
// construire le filtre "Other" : NOT IN ces préfixes.
func AllKnownPairNamePrefixes() []string {
	out := make([]string, 0, len(modePrefixToCategory))
	for prefix, cat := range modePrefixToCategory {
		if cat == ModeCategoryOther {
			continue
		}
		out = append(out, prefix)
	}
	return out
}
