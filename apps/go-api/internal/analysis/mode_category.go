// Package analysis — mode_category.go : portage Go de
// `src.analysis.mode_categories` / `src.analysis.mode_display` (branche v7/cockpit Python).
//
// =============================================================================
// 2 NIVEAUX ORTHOGONAUX pour la sémantique des modes Halo Infinite :
// =============================================================================
//
//  1. SOUS-MODE (label affiché individuellement) → cf. mode_label.go
//     "Arena:Slayer on Bazaar" → "Slayer" (puis traduit "Assassin" via mode_name_tr)
//     Implémenté par NormalizeModeLabel().
//
//  2. CATÉGORIE PARENTE (filtre Mode dans la galerie média, regroupements UI)
//     "Arena:Slayer on Bazaar" → "Assassin" (catégorie qui regroupe Arena/Tactical/Assault/Community)
//     Implémenté par InferModeCategoryFromPairName() ci-dessous.
//
// NE PAS DUPLIQUER : selon le besoin, choisir l'une ou l'autre fonction.
// =============================================================================
//
// Une `mode_category` custom regroupe plusieurs préfixes de pair_name :
//
//	Assassin  : Arena, Tactical, Assault, Community
//	Fiesta    : Fiesta, Super Fiesta, Husky Raid, Super Husky Raid, Castle Wars
//	BTB       : BTB, BTB Heavies
//	Ranked    : Ranked
//	Firefight : Firefight, Gruntpocalypse
//	Other     : tout le reste (Event, et tout préfixe inconnu)
//
// Source de vérité Python (consulter en cas de doute) :
//
//	git show v7/cockpit:src/analysis/mode_display.py     (_PREFIX_RULES)
//	git show v7/cockpit:src/analysis/mode_categories.py  (PREFIX_TO_CATEGORY)
package analysis

import (
	"regexp"
	"strings"
)

// ModeCategoryAssassin et autres : valeurs canoniques retournées par
// InferModeCategoryFromPairName. Stables — utilisées comme labels dans l'UI.
//
// DIVERGENCE PYTHON v7/cockpit : Super Fiesta et Husky Raid sont promus en
// catégories distinctes (Python les regroupait sous "Fiesta"). Justification :
// ce sont des playlists temporaires Halo Infinite identifiables par les joueurs
// (rotations event), donc les masquer derrière "Fiesta" rend le filtre opaque.
const (
	ModeCategoryAssassin    = "Assassin"
	ModeCategoryFiesta      = "Fiesta"
	ModeCategorySuperFiesta = "Super Fiesta"
	ModeCategoryHuskyRaid   = "Husky Raid"
	ModeCategoryBTB         = "BTB"
	ModeCategoryRanked      = "Ranked"
	ModeCategoryFirefight   = "Firefight"
	ModeCategoryOther       = "Other"
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
	"Super Fiesta":     ModeCategorySuperFiesta, // promu (cf. divergence Python)
	"Husky Raid":       ModeCategoryHuskyRaid,   // promu (cf. divergence Python)
	"Super Husky Raid": ModeCategoryHuskyRaid,
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
