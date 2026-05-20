// Package analysis â€” home_locale.go : constantes outcome/color/tone, helpers
// locale (FR/EN), labels d'outcome, normalisation des modes, badges narratifs,
// score label, regex UUID.
//
// Ces helpers sont partagÃ©s entre la projection legacy (home.go, home_*.go)
// et la projection canonique (home_canonical_*.go).
package analysis

import (
	"fmt"
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
		return "Match"
	}
	if label, ok := homeOutcomeLabels[outcome]; ok {
		return label
	}
	return "Match"
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

func buildHomeScoreLabel(match legacymatch.HomeMatchRow) *string {
	if match.Team0Score < 0 || match.Team1Score < 0 {
		return nil
	}

	leftScore := match.Team0Score
	rightScore := match.Team1Score
	if match.TeamID == 1 {
		leftScore = match.Team1Score
		rightScore = match.Team0Score
	}

	label := fmt.Sprintf("%d-%d", leftScore, rightScore)
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
