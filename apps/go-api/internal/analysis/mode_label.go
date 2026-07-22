// Package analysis — mode_label.go : normalisation canonique des labels de mode de jeu.
//
// Port Go de src.analysis.mode_display (branche Python v7/cockpit).
// Logique unifiée utilisée par la home, l'historique de matchs et les filtres.
// Aucun accès DB — function pure.
//
// COMPLÉMENTAIRE de mode_category.go : ce fichier extrait le SOUS-MODE
// ("Arena:Slayer" → "Slayer"), tandis que mode_category.go infère la
// CATÉGORIE PARENTE ("Arena:Slayer" → "Assassin"). Voir l'en-tête de
// mode_category.go pour le détail des 2 niveaux orthogonaux.
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

// Labels d'identité de playlists Halo Infinite — partagés avec mode_category.go
// (côté games/halo_infinite). Re-déclarés ici pour éviter un cycle d'import
// analysis ← halo_infinite ← analysis.
const (
	modeLabelSuperFiesta    = "Super Fiesta"
	modeLabelHuskyRaid      = "Husky Raid"
	modeLabelSuperHuskyRaid = "Super Husky Raid"
)

// playlistIdentityPrefixes : préfixes pair_name qui SONT le label utilisateur,
// donc à conserver tels quels au lieu d'extraire le sous-mode après ":".
//
// Exemple : "Super Fiesta:Slayer on Forbidden - Forge" doit afficher
// "Super Fiesta" (et non "Slayer", qui serait traduit FR en "Assassin" via
// mode_name_tr et noierait l'identité de la playlist).
//
// Liste alignée sur les catégories promues côté `halo_infinite/mode_category.go`
// (ModeCategorySuperFiesta, ModeCategoryHuskyRaid). Les "containers"
// (Arena/Tactical/Assault/Community) restent extraits au sous-mode.
//
// Comparaison case-insensitive — les pair_name peuvent arriver avec une casse
// inconsistante depuis l'API Halo.
var playlistIdentityPrefixes = map[string]string{
	"super fiesta":     modeLabelSuperFiesta,
	"husky raid":       modeLabelHuskyRaid,
	"super husky raid": modeLabelSuperHuskyRaid,
}

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
		// SAUF si le préfixe gauche est lui-même l'identité de la playlist
		// (Super Fiesta, Husky Raid, Super Husky Raid) — auquel cas on garde
		// le préfixe pour ne pas afficher "Slayer/Assassin" sur une tuile
		// Super Fiesta. Cf. thought_log 2026-05-08.
		left := strings.TrimSpace(normalized[:idx])
		if canonical, ok := playlistIdentityPrefixes[strings.ToLower(left)]; ok {
			normalized = canonical
		} else {
			normalized = strings.TrimSpace(normalized[idx+1:])
		}
	}

	// Étape 3 — strip générique " sur/on <carte>" résiduel
	normalized = modeLabelStripMapRe.ReplaceAllString(normalized, "")

	// Étape 4 — strip suffixes Forge / Ranked
	normalized = modeLabelForgeRe.ReplaceAllString(normalized, "")
	normalized = modeLabelRankedRe.ReplaceAllString(normalized, "")

	return strings.TrimSpace(normalized)
}

// ResolveModeUI applique la formule canonique de résolution du libellé de mode
// affiché côté UI (home tiles, match-view header, historique).
//
// Source unique de vérité : COALESCE(pair_name_fr, pair_name) → NormalizeModeLabel.
// Tout caller qui a besoin d'un libellé de mode propre (sans suffixe map, sans
// préfixe technique "Arena:") doit appeler ce helper plutôt que d'inventer
// sa propre cascade — la divergence home/match-view qui produisait
// "Slayer on Streets" sur la vue détail est née d'une cascade ad hoc.
//
// Retourne nil si les deux sources sont vides après normalisation.
func ResolveModeUI(pairName, pairNameFR *string) *string {
	src := ""
	if pairNameFR != nil && *pairNameFR != "" {
		src = *pairNameFR
	} else if pairName != nil {
		src = *pairName
	}
	out := NormalizeModeLabel(src)
	if out == "" {
		return nil
	}
	return &out
}

// ResolveModeUIWithVariant retourne le libellé de MODE côté UI : pair prioritaire
// (FR sinon EN, normalisé via NormalizeModeLabel), sinon fallback game_variant
// (FR sinon EN, normalisé) pour les titres sans pair_name (Halo 5,
// games/halo_5/mapping.go PairMode nil). Source unique de la convention
// pair-sinon-variant — ne pas recopier ce pattern ailleurs. Retourne nil si
// aucune source (pair ET variant vides).
//
// Différence assumée avec ResolveModeUI : si une source non vide est sur-nettoyée
// par NormalizeModeLabel (résultat vide, ex. un brut réduit à un pur suffixe
// "- Forge"), on retombe sur la valeur brute trimée au lieu de perdre le libellé.
// Un pair_name / game_variant réel porte toujours un token de mode, donc ce repli
// ne se déclenche pas sur les données de production (iso-comportement effectif).
func ResolveModeUIWithVariant(pairName, pairNameFR, variantName, variantNameFR *string) *string {
	if v := resolveModeSource(pairNameFR, pairName); v != nil {
		return v
	}
	return resolveModeSource(variantNameFR, variantName)
}

// resolveModeSource prend la première source non vide (préférence au 1er argument
// = FR), applique NormalizeModeLabel, et retombe sur la valeur brute trimée si la
// normalisation vide le libellé. Retourne nil si les deux sources sont vides.
func resolveModeSource(preferred, fallback *string) *string {
	src := ""
	if preferred != nil && *preferred != "" {
		src = *preferred
	} else if fallback != nil && *fallback != "" {
		src = *fallback
	}
	src = strings.TrimSpace(src)
	if src == "" {
		return nil
	}
	if norm := NormalizeModeLabel(src); norm != "" {
		return &norm
	}
	return &src
}

// ExtractKnownMode retrouve un mode CANONIQUE connu à l'intérieur d'un label déjà
// normalisé mais non reconnu tel quel — typiquement une variante d'arme/saison
// ("Legacy Slayer BR" → "Slayer", "Tactical Slayer" → "Slayer"). Cherche le mode
// connu LE PLUS LONG apparaissant comme mot entier (case-insensitive).
//
// knownModesEN = noms EN canoniques (clés mode_en de mode_name_tr), chargés par le
// caller (repo). Retourne "" si aucun match → le caller garde le label d'origine.
// Fonction pure (aucun accès DB), à appliquer APRÈS NormalizeModeLabel.
func ExtractKnownMode(label string, knownModesEN []string) string {
	label = strings.TrimSpace(label)
	if label == "" || len(knownModesEN) == 0 {
		return ""
	}
	low := strings.ToLower(label)
	best := ""
	for _, m := range knownModesEN {
		m = strings.TrimSpace(m)
		if m == "" || len(m) <= len(best) {
			continue // garde le match le plus long (ex. "Super Fiesta" > "Fiesta")
		}
		if wholeWordIndex(low, strings.ToLower(m)) >= 0 {
			best = m
		}
	}
	return best
}

// wholeWordIndex retourne l'index de needle dans haystack en exigeant des frontières
// de mot (lettres/chiffres) de part et d'autre, sinon -1. Évite que "Slayer" matche
// au milieu d'un autre mot. haystack/needle doivent être déjà en minuscules ASCII.
func wholeWordIndex(haystack, needle string) int {
	for from := 0; from <= len(haystack)-len(needle); {
		i := strings.Index(haystack[from:], needle)
		if i < 0 {
			return -1
		}
		idx := from + i
		beforeOK := idx == 0 || !isWordChar(haystack[idx-1])
		end := idx + len(needle)
		afterOK := end >= len(haystack) || !isWordChar(haystack[end])
		if beforeOK && afterOK {
			return idx
		}
		from = idx + 1
	}
	return -1
}

// isWordChar : caractère de mot ASCII (a-z, 0-9, _). haystack est déjà en minuscules.
func isWordChar(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}
