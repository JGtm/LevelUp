// Package analysis â€” home_canonical_skill.go : skill history inference +
// construction du badge CSR (label localisÃ© + URL d'image statique).
package analysis

import (
	"fmt"
	"strings"

	"levelup/go-api/internal/games/canonical"
)

// InferHomeSkillHistoryFromCanonical est la variante canonical-aware de l'helper
// privÃ© inferHomeSkillHistory du service home. Retourne (hasRanked, hasUnranked).
// PvE matchs sont exclus (Summary.IsPvE).
func InferHomeSkillHistoryFromCanonical(rows []canonical.PlayerMatchRow) (bool, bool) {
	hasRanked := false
	hasUnranked := false
	for _, r := range rows {
		if r.Summary.IsPvE != nil && *r.Summary.IsPvE {
			continue
		}
		isRanked := false
		if r.Summary.IsRanked != nil {
			isRanked = *r.Summary.IsRanked
		}
		if isRanked {
			hasRanked = true
		} else {
			hasUnranked = true
		}
		if hasRanked && hasUnranked {
			break
		}
	}
	return hasRanked, hasUnranked
}

// buildCanonicalSkillBadge construit le label localisé et l'URL du badge.
//
//   - tierDisplay : nom du tier dans la locale cible (ex: "Or", "gold") — pour le label
//   - tierCodeEN  : nom du tier en anglais lowercase (ex: "gold") — pour l'URL de badge
//   - subTier     : 1..6, nil pour Onyx
//
// URL templates pour les badges CSR (rank images statiques).
// Format : /static/ranks/halo_infinite/120px-HINF-CSR_{Tier}{SubTier}.png
// (même format que halo_infinite.AssetURLAdapter.CSRRankImageURL, sans import
// cyclique — duplication intentionnelle, alignée par convention).
const (
	csrRankImageBasePath = "/static/ranks/halo_infinite/"
	csrRankImageOnyxURL  = csrRankImageBasePath + "120px-HINF-CSR_Onyx.png"
	csrRankImageTemplate = csrRankImageBasePath + "120px-HINF-CSR_%s%d.png"
)

var csrSubTierRoman = [7]string{"", "I", "II", "III", "IV", "V", "VI"}

// BuildSkillTierLabel construit le libellé localisé du palier (ex. "Or III",
// "Diamant V", "Onyx") à partir des codes de tier d'un SkillSnapshot. frPreferred
// → nom FR (tierCodeFR si présent), sinon EN (tierCode). Retourne nil si pas de
// tier exploitable (placement / non-rankée). Même formule que la home — exposé
// pour que la page session affiche le palier comme l'Explorer (et non la valeur brute).
func BuildSkillTierLabel(tierCode, tierCodeFR *string, subTier *int, frPreferred bool) *string {
	if tierCode == nil || *tierCode == "" {
		return nil
	}
	tierDisplay := *tierCode
	if frPreferred && tierCodeFR != nil && *tierCodeFR != "" {
		tierDisplay = *tierCodeFR
	}
	label, _ := buildCanonicalSkillBadge(tierDisplay, *tierCode, subTier)
	return label
}

func buildCanonicalSkillBadge(tierDisplay, tierCodeEN string, subTier *int) (*string, *string) {
	tierEN := strings.ToLower(strings.TrimSpace(tierCodeEN))
	if tierEN == "" {
		return nil, nil
	}
	// Capitalize first letter pour l'URL (Bronze, Silver, Gold, Platinum, Diamond, Onyx).
	tierENcap := strings.ToUpper(tierEN[:1]) + tierEN[1:]

	// Label : capitalize first letter du nom localisé.
	display := strings.TrimSpace(tierDisplay)
	if display == "" {
		display = tierENcap
	} else {
		display = strings.ToUpper(display[:1]) + display[1:]
	}

	var label string
	var urlStr string

	if strings.EqualFold(tierEN, "onyx") {
		label = display
		urlStr = csrRankImageOnyxURL
	} else {
		st := 0
		if subTier != nil {
			st = *subTier
		}
		if st < 1 || st > 6 {
			return nil, nil
		}
		label = fmt.Sprintf("%s %s", display, csrSubTierRoman[st])
		urlStr = fmt.Sprintf(csrRankImageTemplate, tierENcap, st)
	}

	return &label, &urlStr
}
