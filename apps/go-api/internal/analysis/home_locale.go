// Package analysis â€” home_locale.go : constantes outcome/color/tone, helpers
// locale (FR/EN), labels d'outcome, normalisation des modes, badges narratifs,
// score label, regex UUID.
//
// Ces helpers sont partagÃ©s entre la projection legacy (home.go, home_*.go)
// et la projection canonique (home_canonical_*.go).
package analysis

import (
	"regexp"
	"strings"

	"levelup/go-api/internal/legacymatch"
)

// ---------------------------------------------------------------------------
// Constantes outcome (codes numÃ©riques Halo Infinite)
// ---------------------------------------------------------------------------

const (
	homeOutcomeWin                = 2
	homeOutcomeLoss               = 3
	homeOutcomeTie                = 1
	homeOutcomeDNF                = 4
	homeDominanceDomination       = 1
	homeDominanceHumiliation      = 2
	homeDominanceRemontada        = 3
	homeDominanceDebacle          = 4
	homeDominanceCounterRemontada = 5
)

// Codes couleur sÃ©mantiques utilisÃ©s dans les blocs JSON du Home (highlights).
const (
	homeColorPositive = "positive"
	homeColorNeutral  = "neutral"
	homeColorNegative = "negative"
)

// homeOutcomeLabelFallback est le label retourné quand l'outcome code n'est pas
// reconnu (defaut FR/EN identique : "Match").
const homeOutcomeLabelFallback = "Match"

// Tones d'outcome partagÃ©s entre la projection JSON (home) et les filtres
// (match_filter). DÃ©clarÃ©s ici car le package n'a pas de fichier de constantes
// partagÃ©es et home_locale.go est le point d'entrÃ©e des codes outcome.
const (
	OutcomeToneWin  = "win"
	OutcomeToneLoss = "loss"
	OutcomeToneDraw = "draw"
	OutcomeToneTie  = "tie"
	OutcomeToneDNF  = "dnf"
)

var homeOutcomeLabels = map[int]string{
	homeOutcomeWin:  "Victoire",
	homeOutcomeLoss: "DÃ©faite",
	homeOutcomeTie:  "Ã‰galitÃ©",
	homeOutcomeDNF:  "Abandon",
}

var homeOutcomeLabelsEN = map[int]string{
	homeOutcomeWin:  "Victory",
	homeOutcomeLoss: "Defeat",
	homeOutcomeTie:  "Tie",
	homeOutcomeDNF:  "DNF",
}

var homeOutcomeTones = map[int]string{
	homeOutcomeWin:  OutcomeToneWin,
	homeOutcomeLoss: OutcomeToneLoss,
	homeOutcomeTie:  OutcomeToneTie,
	homeOutcomeDNF:  OutcomeToneDNF,
}

var homeUUIDRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// IsRawAssetUUID indique si s est un UUID v4 brut (asset id non traduit par la
// metadata), donc à masquer côté UI. Source unique de la détection, partagée
// par cleanAssetLabel (projection home) et la résolution du mode Escouade
// (teammates_service_assets.go) — évite qu'un pair_name non résolu fuie à l'UI.
func IsRawAssetUUID(s string) bool {
	return homeUUIDRe.MatchString(strings.TrimSpace(s))
}

// labelFR retourne fr si non vide, sinon en.
func labelFR(fr, en string) string {
	if fr != "" {
		return fr
	}
	return en
}

func normalizeHomeLocale(locale string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "en") {
		return "en"
	}
	return "fr"
}

func labelForLocale(locale, fr, en string) string {
	if normalizeHomeLocale(locale) == "en" {
		if strings.TrimSpace(en) != "" {
			return en
		}
		return fr
	}
	return labelFR(fr, en)
}

func outcomeLabelForLocale(outcome int, locale string) string {
	if normalizeHomeLocale(locale) == "en" {
		if label, ok := homeOutcomeLabelsEN[outcome]; ok {
			return label
		}
		return homeOutcomeLabelFallback
	}
	if label, ok := homeOutcomeLabels[outcome]; ok {
		return label
	}
	return homeOutcomeLabelFallback
}

func outcomeLabel(code int) string {
	if l, ok := homeOutcomeLabels[code]; ok {
		return l
	}
	return "DNF"
}

func outcomeTone(code int) string {
	if t, ok := homeOutcomeTones[code]; ok {
		return t
	}
	return OutcomeToneDNF
}

// buildHomeScoreLabel — chemin LEGACY de l'accueil. Ne fabrique plus le libellé lui-même :
// il délègue à TeamScoreLabel, source unique depuis le 2026-08-29 (cf.
// team_score_display.go). Les manches ne sont pas encore portées par HomeMatchRow — nil
// donc, et la lecture reste celle des points, à l'identique.
func buildHomeScoreLabel(match legacymatch.HomeMatchRow) *string {
	left, right := match.Team0Score, match.Team1Score
	if match.TeamID == 1 {
		left, right = match.Team1Score, match.Team0Score
	}
	label := TeamScoreLabel(TeamScoreInput{MyPoints: &left, EnemyPoints: &right})
	if label == "" {
		return nil
	}
	return &label
}

func buildHomeNarrativeBadges(dominanceFlag int) []string {
	switch dominanceFlag {
	case homeDominanceDomination:
		return []string{"dominant"}
	case homeDominanceHumiliation:
		return []string{"humiliation"}
	case homeDominanceRemontada:
		return []string{"remontada"}
	case homeDominanceDebacle:
		return []string{"debacle"}
	case homeDominanceCounterRemontada:
		return []string{"contre_remontada"}
	default:
		return nil
	}
}

// normalizeHomeModeLabel est un alias interne vers NormalizeModeLabel.
// ConservÃ© pour ne pas casser les appelants internes au package.
func normalizeHomeModeLabel(raw string, mapLabels ...string) string {
	return NormalizeModeLabel(raw, mapLabels...)
}

func copyOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
