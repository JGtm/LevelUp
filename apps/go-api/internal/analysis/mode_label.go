// Package analysis — mode_label.go : normalisation canonique des labels de mode de jeu.
//
// Port Go de src.analysis.mode_display (branche Python v7/cockpit).
// Logique unifiée utilisée par la home, l'historique de matchs et les filtres.
// Aucun accès DB — function pure.
package analysis

import (
	"regexp"
	"strings"
)

// Regex partagées pour la normalisation des modes.
var (
	// Strip " sur NomCarte" ou " on MapName" (FR + EN) — générique.
	modeLabelStripMapRe = regexp.MustCompile(`(?i)\s+(?:on|sur)\s+.+$`)
	// Strip " - Forge" et " - Ranked" (suffixes Halo Infinite).
	modeLabelForgeRe  = regexp.MustCompile(`(?i)\s*-\s*Forge\b`)
	modeLabelRankedRe = regexp.MustCompile(`(?i)\s*-\s*Ranked\b`)
)

// NormalizeModeLabel normalise un label brut de mode de jeu Halo Infinite.
//
// Logique (alignée sur Python resolve_display_mode + translate_pair_name) :
//  1. Strip map-label connu : " sur {map}" / " on {map}" → retiré en priorité.
//  2. Extraction du mode depuis le format pair_name :
//     - Format FR avec séparateur espacé " : " → prend la partie avant
//     ("Assassin : Classé" → "Assassin").
//     - Format technique "Prefix:Mode" → prend la partie après le dernier ":"
//     ("Arena:Slayer" → "Slayer", "BTB:CTF" → "CTF").
//  3. Strip générique " sur .+" / " on .+" (FR + EN) si non retiré à l'étape 1.
//  4. Strip " - Forge" et " - Ranked".
//
// mapLabels est optionnel : les noms de map connus pour l'étape 1.
func NormalizeModeLabel(raw string, mapLabels ...string) string {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return ""
	}

	// Étape 1 — strip map-label connu (prioritaire, avant extraction du préfixe)
	for _, mapLabel := range mapLabels {
		trimmedMap := strings.TrimSpace(mapLabel)
		if trimmedMap == "" {
			continue
		}
		mapSpecificRe := regexp.MustCompile(`(?i)\s+(?:on|sur)\s+` + regexp.QuoteMeta(trimmedMap) + `$`)
		updated := mapSpecificRe.ReplaceAllString(normalized, "")
		if updated != normalized {
			normalized = strings.TrimSpace(updated)
			break
		}
	}

	// Étape 2 — extraction du mode depuis le format pair_name
	// Format FR : "Assassin : Classé" → "Assassin" (prend avant " : ")
	if idx := strings.Index(normalized, " : "); idx > 0 {
		normalized = strings.TrimSpace(normalized[:idx])
	} else if idx := strings.LastIndex(normalized, ":"); idx >= 0 && idx < len(normalized)-1 {
		// Format technique "Arena:Slayer" ou "BTB:CTF" → prend après ":"
		normalized = strings.TrimSpace(normalized[idx+1:])
	}

	// Étape 3 — strip générique " sur/on <carte>" résiduel
	normalized = modeLabelStripMapRe.ReplaceAllString(normalized, "")

	// Étape 4 — strip suffixes Forge / Ranked
	normalized = modeLabelForgeRe.ReplaceAllString(normalized, "")
	normalized = modeLabelRankedRe.ReplaceAllString(normalized, "")

	return strings.TrimSpace(normalized)
}
