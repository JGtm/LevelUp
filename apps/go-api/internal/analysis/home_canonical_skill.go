// Package analysis — home_canonical_skill.go : skill history inference +
// construction du badge CSR (label localisé + URL d'image statique).
package analysis

import (
	"fmt"
	"strings"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
)

// InferHomeSkillHistoryFromCanonical est la variante canonical-aware de l'helper
// privé inferHomeSkillHistory du service home. Retourne (hasRanked, hasUnranked).
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
// "Diamant V", "Onyx") à partir du code de tier EN d'un SkillSnapshot : le nom du palier vient du
// catalogue des messages (skill.toml) dans la langue `loc` ; un palier hors catalogue garde son
// code. Retourne nil si pas de tier exploitable (placement / non-rankée). Même formule que la home —
// exposé pour que la page session affiche le palier comme l'Explorer (et non la valeur brute).
func BuildSkillTierLabel(tierCode *string, subTier *int, loc i18n.Locale) *string {
	if tierCode == nil || *tierCode == "" {
		return nil
	}
	tierDisplay := tierDisplayName(loc, *tierCode)
	// Label-only : pas besoin de l'URL → résolveur nil.
	label, _ := buildCanonicalSkillBadge(tierDisplay, *tierCode, subTier, nil)
	return label
}

// Noms canoniques EN des paliers, en minuscules : clés de recherche du catalogue des messages.
const (
	tierKeyBronze   = "bronze"
	tierKeySilver   = "silver"
	tierKeyGold     = "gold"
	tierKeyPlatinum = "platinum"
	tierKeyDiamond  = "diamond"
	tierKeyOnyx     = "onyx"
)

// tierDisplayName retourne le nom du palier `tierEN` (casse libre : diamond / DIAMOND / Diamond)
// dans la langue `loc`, ou `tierEN` tel quel quand le catalogue ne connaît pas ce palier.
func tierDisplayName(loc i18n.Locale, tierEN string) string {
	t := strings.TrimSpace(tierEN)
	if t == "" {
		return tierEN
	}
	switch strings.ToLower(t) {
	case tierKeyBronze:
		return i18n.Msg(loc, "skill.tier.bronze", nil)
	case tierKeySilver:
		return i18n.Msg(loc, "skill.tier.silver", nil)
	case tierKeyGold:
		return i18n.Msg(loc, "skill.tier.gold", nil)
	case tierKeyPlatinum:
		return i18n.Msg(loc, "skill.tier.platinum", nil)
	case tierKeyDiamond:
		return i18n.Msg(loc, "skill.tier.diamond", nil)
	case tierKeyOnyx:
		return i18n.Msg(loc, "skill.tier.onyx", nil)
	}
	return tierEN
}

// BuildCSRTierLabel construit le libellé CSR localisé ("Diamant III") depuis
// le tier ANGLAIS brut (ex. player_csr_snapshots.alltime_tier) + son sous-palier, dans la langue
// `loc`. Onyx n'a pas de sous-palier. nil si tier vide. La casse d'entrée est normalisée
// (diamond / DIAMOND / Diamond → "Diamant").
func BuildCSRTierLabel(tierEN string, subTier *int, loc i18n.Locale) *string {
	t := strings.TrimSpace(tierEN)
	if t == "" {
		return nil
	}
	return BuildSkillTierLabel(&t, subTier, loc)
}

// normalizeSkillBadgeParts normalise le couple (tier EN brut, sous-palier) vers la
// forme attendue par les résolveurs d'URL de badge : tier CAPITALISÉ
// ("Bronze".."Diamond", "Onyx") + sous-palier 0 pour Onyx / 1..6 sinon.
// ok=false quand rien n'est constructible (tier vide, sous-palier hors bornes).
//
// SOURCE UNIQUE de cette normalisation : le format d'URL des badges dépend de la
// capitalisation exacte du tier ("120px-HINF-CSR_Gold3" côté HINF). Deux
// consommateurs — buildCanonicalSkillBadge (badge de la home) et SkillBadgeURL
// (lignes Explorer/historique) ; toute 3e copie doit passer par ici.
func normalizeSkillBadgeParts(tierCodeEN string, subTier *int) (tierENcap string, st int, ok bool) {
	tierEN := strings.ToLower(strings.TrimSpace(tierCodeEN))
	if tierEN == "" {
		return "", 0, false
	}
	tierENcap = strings.ToUpper(tierEN[:1]) + tierEN[1:]
	if strings.EqualFold(tierEN, "onyx") {
		return tierENcap, 0, true // Onyx : palier unique, pas de sous-palier
	}
	if subTier != nil {
		st = *subTier
	}
	if st < 1 || st > 6 {
		return "", 0, false
	}
	return tierENcap, st, true
}

// SkillBadgeURL résout l'URL de l'image du badge de palier depuis le tier EN brut
// (ex. match_skill_rank_latest.tier) et son sous-palier, via le résolveur
// title-aware INJECTÉ — même contrat que buildCanonicalSkillBadge : le résolveur
// reçoit (tier capitalisé, sous-palier 0=Onyx/1..6) et rend "" quand il ne sait
// pas construire d'URL.
//
// nil quand le palier n'est pas exploitable, que le résolveur est absent ou qu'il
// ne produit rien : l'appelant DÉGRADE alors sur le libellé texte du palier
// (aucune image manquante à l'écran).
func SkillBadgeURL(tierCodeEN string, subTier *int, urlResolver func(tierEN string, subTier int) string) *string {
	if urlResolver == nil {
		return nil
	}
	tierENcap, st, ok := normalizeSkillBadgeParts(tierCodeEN, subTier)
	if !ok {
		return nil
	}
	urlStr := urlResolver(tierENcap, st)
	if urlStr == "" {
		return nil
	}
	return &urlStr
}

func buildCanonicalSkillBadge(tierDisplay, tierCodeEN string, subTier *int, urlResolver func(tierEN string, subTier int) string) (*string, *string) {
	tierENcap, st, ok := normalizeSkillBadgeParts(tierCodeEN, subTier)
	if !ok {
		return nil, nil
	}

	// Label : capitalize first letter du nom localisé (repli sur le tier EN).
	display := strings.TrimSpace(tierDisplay)
	if display == "" {
		display = tierENcap
	} else {
		display = capitalizeFirstRune(display)
	}

	label := display
	if st > 0 { // non-Onyx → sous-palier en chiffres romains
		label = fmt.Sprintf("%s %s", display, csrSubTierRoman[st])
	}

	// URL : déléguée au résolveur injecté (title-aware) via le chokepoint unique.
	// Aucun template HINF figé ici — analysis reste pur et title-agnostic.
	return &label, SkillBadgeURL(tierCodeEN, subTier, urlResolver)
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
// UNIQUE de l'ordre des paliers — utilisée pour le palier suivant (nextTier) ET
// la sélection du meilleur CSR (CSRTierOrdinal, cf. home_repo_skill_peak). Champion
// = sommet H5 ("Champion #N", au-dessus d'Onyx) ; absent d'Infinite mais inclus
// pour ranker un pic Champion. Ordinal → indépendant de l'échelle de valeur : un
// titre tier-only (qui ne fournit PAS la valeur numérique CSR) reste classable par
// palier. nextTier(Onyx)→Champion est inoffensif : NextSubTierLabel court-circuite
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
func NextSubTierLabel(tierEN string, subTier int, loc i18n.Locale) *string {
	if strings.EqualFold(strings.TrimSpace(tierEN), "Onyx") {
		return nil
	}
	if subTier >= 1 && subTier < csrSubTiersPerTier {
		next := subTier + 1
		return BuildCSRTierLabel(tierEN, &next, loc)
	}
	if subTier == csrSubTiersPerTier {
		if nt := nextTier(tierEN); nt != "" {
			one := 1
			return BuildCSRTierLabel(nt, &one, loc) // Onyx → "Onyx" (sous-palier ignoré)
		}
	}
	return nil
}

// nextTier : palier EN au-dessus de tierEN, ou "" si dernier / inconnu.
func nextTier(tierEN string) string {
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
