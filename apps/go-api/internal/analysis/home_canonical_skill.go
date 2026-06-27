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
//   - urlResolver : résolveur d'URL de badge INJECTÉ (title-aware). Signature
//     (tierEN capitalisé, subTier 0..6 ; 0 = Onyx) → URL ; "" → pas d'URL.
//     nil → aucune URL (le package analysis reste pur, sans template HINF figé).
//     Le câblage HINF/Halo 5 vit dans la couche service/boot qui connaît le titre.

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
	// Label-only : pas besoin de l'URL → résolveur nil.
	label, _ := buildCanonicalSkillBadge(tierDisplay, *tierCode, subTier, nil)
	return label
}

// csrTierENtoFR : tier CSR anglais (player_csr_snapshots.alltime_tier / API Halo)
// → libellé FR. Dupliqué volontairement de sync.tierENtoFR (le package analysis
// ne dépend pas de sync) ; 6 paliers stables. Tier inconnu → laissé en EN.
var csrTierENtoFR = map[string]string{
	"Bronze":   "Bronze",
	"Silver":   "Argent",
	"Gold":     "Or",
	"Platinum": "Platine",
	"Diamond":  "Diamant",
	"Onyx":     "Onyx",
}

// BuildCSRTierLabelFromEN construit le libellé CSR localisé ("Diamant III") depuis
// le tier ANGLAIS brut (ex. player_csr_snapshots.alltime_tier) + son sous-palier.
// frPreferred → nom FR (CSR FR-first, comme sync.formatCSRTierLabel) ; sinon EN.
// Onyx n'a pas de sous-palier. nil si tier vide. La casse d'entrée est normalisée
// (diamond / DIAMOND / Diamond → "Diamant").
func BuildCSRTierLabelFromEN(tierEN string, subTier *int, frPreferred bool) *string {
	t := strings.TrimSpace(tierEN)
	if t == "" {
		return nil
	}
	key := strings.ToUpper(t[:1]) + strings.ToLower(t[1:])
	var frPtr *string
	if fr := csrTierENtoFR[key]; fr != "" {
		frPtr = &fr
	}
	return BuildSkillTierLabel(&t, frPtr, subTier, frPreferred)
}

func buildCanonicalSkillBadge(tierDisplay, tierCodeEN string, subTier *int, urlResolver func(tierEN string, subTier int) string) (*string, *string) {
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
	// st : sous-palier normalisé pour le résolveur (0 = Onyx, 1..6 sinon).
	var st int

	if strings.EqualFold(tierEN, "onyx") {
		label = display
		st = 0
	} else {
		if subTier != nil {
			st = *subTier
		}
		if st < 1 || st > 6 {
			return nil, nil
		}
		label = fmt.Sprintf("%s %s", display, csrSubTierRoman[st])
	}

	// URL : déléguée au résolveur injecté (title-aware). Aucun template HINF
	// figé dans ce package — analysis reste pur et title-agnostic.
	if urlResolver == nil {
		return &label, nil
	}
	urlStr := urlResolver(tierENcap, st)
	if urlStr == "" {
		return &label, nil
	}
	return &label, &urlStr
}

// csrSubTiersPerTier : chaque palier non-Onyx (Bronze..Diamant) compte 6
// sous-paliers (I..VI).
const csrSubTiersPerTier = 6

// SkillTierBand calcule le remplissage de la barre, de façon ORDINALE via le
// sous-palier stocké (I..VI) — INDÉPENDANTE de l'échelle de valeur. Nécessaire
// car CSR et LUSR n'utilisent pas la même échelle (LUSR « Or IV » ≈ 1500, là où
// 1500 = Onyx côté CSR), ET parce que le LUSR définit ses sous-paliers sur l'échelle
// μ interne (TrueSkill), absente de la valeur affichée → un remplissage « dans le
// sous-palier » n'est pas dérivable. On reste donc à la granularité sous-palier :
//
//   - Onyx (sommet) → barre pleine (100 %).
//   - sous-palier 1..6 → n/6 : I ≈ 17 % … VI = 100 % (sommet du palier).
//   - ok=false sinon (placement / sans rang) → pas de barre.
func SkillTierBand(tierEN string, subTier int) (progressPct float64, ok bool) {
	if strings.EqualFold(strings.TrimSpace(tierEN), "Onyx") {
		return 100, true
	}
	if subTier < 1 || subTier > csrSubTiersPerTier {
		return 0, false
	}
	return float64(subTier) / float64(csrSubTiersPerTier) * 100, true
}

// csrTierOrderEN : ordre canonique des paliers Halo (ordinal croissant). SOURCE
// UNIQUE de l'ordre des paliers — utilisée pour le palier suivant (nextTierEN) ET
// la sélection du meilleur CSR (CSRTierOrdinal, cf. home_repo_skill_peak). Champion
// = sommet H5 ("Champion #N", au-dessus d'Onyx) ; absent d'Infinite mais inclus
// pour ranker un pic Champion. Ordinal → indépendant de l'échelle de valeur : un
// titre tier-only (qui ne fournit PAS la valeur numérique CSR) reste classable par
// palier. nextTierEN(Onyx)→Champion est inoffensif : NextSubTierLabel court-circuite
// déjà Onyx→nil en amont.
var csrTierOrderEN = []string{"Bronze", "Silver", "Gold", "Platinum", "Diamond", "Onyx", "Champion"}

// CSRTierOrdinal : rang ordinal du palier EN (Bronze=1 … Champion=7 ; 0 si vide/
// inconnu). Source UNIQUE pour comparer/sélectionner des paliers CSR, y compris
// pour les titres tier-only où la valeur numérique est absente (sélection par
// palier puis sous-palier). Insensible à la casse.
func CSRTierOrdinal(tierEN string) int {
	t := strings.TrimSpace(tierEN)
	for i, name := range csrTierOrderEN {
		if strings.EqualFold(name, t) {
			return i + 1
		}
	}
	return 0
}

// NextSubTierLabel : libellé localisé du SOUS-PALIER suivant (extrémité droite de
// la barre). Intelligent en sortie de palier :
//   - Or I → "Or II" … Or V → "Or VI"
//   - Or VI → "Platine I" (premier sous-palier du palier suivant)
//   - Diamant VI → "Onyx" (sommet)
//   - Onyx → nil (déjà au sommet, pas de suivant).
func NextSubTierLabel(tierEN string, subTier int, frPreferred bool) *string {
	if strings.EqualFold(strings.TrimSpace(tierEN), "Onyx") {
		return nil
	}
	if subTier >= 1 && subTier < csrSubTiersPerTier {
		next := subTier + 1
		return BuildCSRTierLabelFromEN(tierEN, &next, frPreferred)
	}
	if subTier == csrSubTiersPerTier {
		if nt := nextTierEN(tierEN); nt != "" {
			one := 1
			return BuildCSRTierLabelFromEN(nt, &one, frPreferred) // Onyx → "Onyx" (sous-palier ignoré)
		}
	}
	return nil
}

// nextTierEN : palier EN au-dessus de tierEN, ou "" si dernier / inconnu.
func nextTierEN(tierEN string) string {
	t := strings.TrimSpace(tierEN)
	if t == "" {
		return ""
	}
	key := strings.ToUpper(t[:1]) + strings.ToLower(t[1:])
	for i, name := range csrTierOrderEN {
		if name == key && i+1 < len(csrTierOrderEN) {
			return csrTierOrderEN[i+1]
		}
	}
	return ""
}
