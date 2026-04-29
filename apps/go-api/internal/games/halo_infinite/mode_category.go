// Package halo_infinite â€” mode_category.go : portage Go de
// `src.analysis.mode_categories` / `src.analysis.mode_display` (branche v7/cockpit Python).
//
// =============================================================================
// 2 NIVEAUX ORTHOGONAUX pour la sÃ©mantique des modes Halo Infinite :
// =============================================================================
//
//  1. SOUS-MODE (label affichÃ© individuellement) â†’ cf. mode_label.go
//     "Arena:Slayer on Bazaar" â†’ "Slayer" (puis traduit "Assassin" via mode_name_tr)
//     ImplÃ©mentÃ© par NormalizeModeLabel().
//
//  2. CATÃ‰GORIE PARENTE (filtre Mode dans la galerie mÃ©dia, regroupements UI)
//     "Arena:Slayer on Bazaar" â†’ "Assassin" (catÃ©gorie qui regroupe Arena/Tactical/Assault/Community)
//     ImplÃ©mentÃ© par InferModeCategoryFromPairName() ci-dessous.
//
// NE PAS DUPLIQUER : selon le besoin, choisir l'une ou l'autre fonction.
// =============================================================================
//
// Une `mode_category` custom regroupe plusieurs prÃ©fixes de pair_name :
//
//	Assassin  : Arena, Tactical, Assault, Community
//	Fiesta    : Fiesta, Super Fiesta, Husky Raid, Super Husky Raid, Castle Wars
//	BTB       : BTB, BTB Heavies
//	Ranked    : Ranked
//	Firefight : Firefight, Gruntpocalypse
//	Other     : tout le reste (Event, et tout prÃ©fixe inconnu)
//
// Source de vÃ©ritÃ© Python (consulter en cas de doute) :
//
//	git show v7/cockpit:src/analysis/mode_display.py     (_PREFIX_RULES)
//	git show v7/cockpit:src/analysis/mode_categories.py  (PREFIX_TO_CATEGORY)
package halo_infinite

import (
	"regexp"
	"strings"
)

// ModeCategoryAssassin et autres : valeurs canoniques retournÃ©es par
// InferModeCategoryFromPairName. Stables â€” utilisÃ©es comme labels dans l'UI.
//
// DIVERGENCE PYTHON v7/cockpit : Super Fiesta et Husky Raid sont promus en
// catÃ©gories distinctes (Python les regroupait sous "Fiesta"). Justification :
// ce sont des playlists temporaires Halo Infinite identifiables par les joueurs
// (rotations event), donc les masquer derriÃ¨re "Fiesta" rend le filtre opaque.
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

// modePrefixToCategory mappe le prÃ©fixe (gauche du ":" dans pair_name, casse normalisÃ©e)
// vers la catÃ©gorie custom. Miroir Go de _PREFIX_RULES Python.
//
// La traduction FR Ã©ventuelle (ex: "ArÃ¨ne", "CommunautÃ©", "ClassÃ©") est gÃ©rÃ©e
// cÃ´tÃ© infÃ©rence en testant les variantes via _CASE_MAP â€” ici on stocke les
// prÃ©fixes EN canoniques (cf. Python _normalize_case).
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

// modeCaseMap â€” prÃ©fixes spÃ©ciaux dont la casse doit Ãªtre conservÃ©e (acronymes
// ou multi-mots usuels). Miroir Python _CASE_MAP.
var modeCaseMap = map[string]string{
	"btb heavies":      "BTB Heavies",
	"btb":              "BTB",
	"super fiesta":     "Super Fiesta",
	"super husky raid": "Super Husky Raid",
	"husky raid":       "Husky Raid",
	"castle wars":      "Castle Wars",
}

var modeMapSuffixRe = regexp.MustCompile(`(?i)^(.*?)(?:\s*[\-â€“â€”]\s*[0-9A-Za-z]{8,})$`)

// stripMapSuffix retire le suffixe " on MapName" et un Ã©ventuel suffixe ID
// technique (8+ caractÃ¨res alphanum aprÃ¨s " - "). Miroir Python _strip_map_suffix.
func stripMapSuffix(s string) string {
	if i := strings.Index(s, " on "); i >= 0 {
		s = s[:i]
	}
	if m := modeMapSuffixRe.FindStringSubmatch(s); m != nil {
		s = strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(s)
}

// normalizeModeCase normalise la casse d'un prÃ©fixe pour le lookup dans
// modePrefixToCategory. MÃªmes rÃ¨gles que Python _normalize_case :
//
//	"btb"           â†’ "BTB"           (via modeCaseMap)
//	"super fiesta"  â†’ "Super Fiesta"  (via modeCaseMap)
//	"ARENA"         â†’ "ARENA"         (tout-majuscules conservÃ©)
//	"team slayer"   â†’ "Team Slayer"   (title case)
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

// InferModeCategoryFromPairName retourne la catÃ©gorie custom (Assassin/Fiesta/
// BTB/Ranked/Firefight/Other) infÃ©rÃ©e depuis un pair_name brut.
//
// GÃ¨re :
//   - Format normal : "Arena:Slayer on Bazaar" â†’ "Assassin"
//   - Format inversÃ© : "CTF:Arena" â†’ "Assassin" (prÃ©fixe Ã  droite si gauche est inconnu)
//   - Sans sÃ©parateur : "Husky Raid" â†’ "Fiesta", "Sniper Slayer" â†’ "Other"
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

	// Pas de sÃ©parateur : tester si le label complet matche un prÃ©fixe connu
	// (ex: "Husky Raid" sans sous-mode â†’ Fiesta)
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

// PairNamePrefixesForCategory retourne la liste des prÃ©fixes EN qui sont
// rangÃ©s dans la catÃ©gorie donnÃ©e. UtilisÃ© pour construire le WHERE inverse :
// quand l'utilisateur filtre "Fiesta", on gÃ©nÃ¨re
// `WHERE pair_name LIKE 'Fiesta:%' OR LIKE 'Super Fiesta:%' OR LIKE 'Husky Raid:%' OR ...`
// (et aussi "= 'Fiesta'", "= 'Husky Raid'" pour les modes sans `:`).
//
// Pour la catÃ©gorie "Other" retourne nil â€” l'appelant doit utiliser
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

// AllKnownPairNamePrefixes retourne tous les prÃ©fixes EN rangÃ©s dans une
// catÃ©gorie connue (Assassin/Fiesta/BTB/Ranked/Firefight). UtilisÃ© pour
// construire le filtre "Other" : NOT IN ces prÃ©fixes.
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
