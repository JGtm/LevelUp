// Package service — calcul des options Explorer-spécifiques avec count cascade-aware.
//
// Ce fichier expose 5 fonctions qui calculent, pour chaque dimension Explorer
// (outcome, perf_tier, skill_tier, ranked_context, squad_scope), la liste des
// options disponibles avec leur count "OR au sein de la dimension, AND entre
// dimensions". Le front utilise ces counts pour griser les valeurs à 0 et
// afficher "Win (42)" / "Defeat (12)" / etc.
//
// Sémantique OR : pour chaque valeur X de la dimension D, count(X) = nb de rows
// si on AJOUTE X à la sélection courante de D, en gardant tous les autres
// filtres (autres dims + Explorer-cascade : date/exp/playlist/map/mode/scope/id)
// inchangés. Pour les single-select (ranked_context, squad_scope), count(X) =
// nb de rows si on FORCE D = X (remplace la sélection actuelle).
//
// Cohérent avec buildAvailableOptions du shell (filters_options.go) — même
// logique appliquée à des dimensions différentes.
package service

import (
	"strconv"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// perfTierLabel retourne le label FR canonique pour un palier perf (1..5).
// Aligné avec apps/web/src/lib/i18n/manifests/explorer.toml perf_tier_*.
func perfTierLabel(t int) string {
	switch t {
	case 1:
		return "Excellent"
	case 2:
		return "Bon"
	case 3:
		return "Correct"
	case 4:
		return "Faible"
	case 5:
		return "Mauvais"
	default:
		return strconv.Itoa(t)
	}
}

// Tiers CSR canoniques (réutilisés dans skillTierLabel + computeAvailableSkillTiers).
const (
	csrTierBronze   = "Bronze"
	csrTierSilver   = "Silver"
	csrTierGold     = "Gold"
	csrTierPlatinum = "Platinum"
	csrTierDiamond  = "Diamond"
	csrTierOnyx     = "Onyx"
)

// skillTierLabel retourne le label FR du tier ranked depuis sa clé EN.
func skillTierLabel(en string) string {
	switch en {
	case csrTierBronze:
		return "Bronze"
	case csrTierSilver:
		return "Argent"
	case csrTierGold:
		return "Or"
	case csrTierPlatinum:
		return "Platine"
	case csrTierDiamond:
		return "Diamant"
	case csrTierOnyx:
		return csrTierOnyx
	default:
		return en
	}
}

// computeAvailableOutcomes : count par outcome (Win/Loss/Tie/DNF), sémantique OR.
//
// base = rows post-cascade post-PickedSoloSessions (avant les filtres Explorer).
// Pour chaque outcome X candidat, on simule selected ∪ {X} et on applique tous
// les autres filtres (ranked, skill, perf + Explorer-cascade).
func computeAvailableOutcomes(
	base []domain.MatchHistoryRawRow, req domain.MatchHistoryQueryRequest, replays port.ReplayAvailability,
) []domain.LabelValue {
	candidates := []int{2, 3, 1, 4}
	selected := intSliceToSet(req.OutcomeFilter)
	out := make([]domain.LabelValue, 0, len(candidates))
	for _, o := range candidates {
		sim := unionWithInt(selected, o)
		rs := filterByRankedContext(base, req.RankedContext)
		rs = filterByOutcome(rs, sim)
		rs = filterBySkillTier(rs, req.SkillTiers, req.RankedContext)
		rs = filterByPerfTiers(rs, req.PerfTiers)
		rs = applyExplorerMatchFilters(rs, req, replays)
		out = append(out, domain.LabelValue{
			Label: outcomeLabel(o),
			Value: strconv.Itoa(o),
			Count: len(rs),
		})
	}
	return out
}

// computeAvailablePerfTiers : count par palier perf (1..5), sémantique OR.
func computeAvailablePerfTiers(
	base []domain.MatchHistoryRawRow, req domain.MatchHistoryQueryRequest, replays port.ReplayAvailability,
) []domain.LabelValue {
	candidates := []int{1, 2, 3, 4, 5}
	selected := intSliceToSet(req.PerfTiers)
	out := make([]domain.LabelValue, 0, len(candidates))
	for _, t := range candidates {
		sim := unionWithInt(selected, t)
		rs := filterByRankedContext(base, req.RankedContext)
		rs = filterByOutcome(rs, req.OutcomeFilter)
		rs = filterBySkillTier(rs, req.SkillTiers, req.RankedContext)
		rs = filterByPerfTiers(rs, sim)
		rs = applyExplorerMatchFilters(rs, req, replays)
		out = append(out, domain.LabelValue{
			Label: perfTierLabel(t),
			Value: strconv.Itoa(t),
			Count: len(rs),
		})
	}
	return out
}

// computeAvailableSkillTiers : count par tier ranked (Bronze..Onyx), sémantique OR.
//
// Si RankedContext est vide, tous les counts valent 0 (skill_tier nécessite
// un contexte ranked/unranked pour être appliqué — voir filterBySkillTier).
func computeAvailableSkillTiers(
	base []domain.MatchHistoryRawRow, req domain.MatchHistoryQueryRequest, replays port.ReplayAvailability,
) []domain.LabelValue {
	candidates := []string{csrTierBronze, csrTierSilver, csrTierGold, csrTierPlatinum, csrTierDiamond, csrTierOnyx}
	selected := stringSliceToSet(req.SkillTiers)
	out := make([]domain.LabelValue, 0, len(candidates))
	for _, t := range candidates {
		var count int
		if req.RankedContext != "" && req.RankedContext != "all" {
			sim := unionWith(selected, t) // []string déjà
			rs := filterByRankedContext(base, req.RankedContext)
			rs = filterByOutcome(rs, req.OutcomeFilter)
			rs = filterBySkillTier(rs, sim, req.RankedContext)
			rs = filterByPerfTiers(rs, req.PerfTiers)
			rs = applyExplorerMatchFilters(rs, req, replays)
			count = len(rs)
		}
		out = append(out, domain.LabelValue{
			Label: skillTierLabel(t),
			Value: t,
			Count: count,
		})
	}
	return out
}

// computeAvailableRankedContexts : count par contexte (all/ranked/unranked).
//
// Single-select : pour chaque contexte X, count = nb rows si on FORCE D = X
// (pas d'union avec sélection courante puisqu'il n'y a qu'une valeur active).
// Pour "ranked"/"unranked" on remet aussi les filtres skill_tier (qui en
// dépendent) ; pour "" (all) on désactive le filtre skill_tier.
func computeAvailableRankedContexts(
	base []domain.MatchHistoryRawRow, req domain.MatchHistoryQueryRequest, replays port.ReplayAvailability,
) []domain.LabelValue {
	contexts := []struct {
		Value string
		Label string
	}{
		{"", "Tous"},
		{scopeRanked, "Ranked (CSR)"},
		{scopeUnranked, "Non-ranked (LUSR)"},
	}
	out := make([]domain.LabelValue, 0, len(contexts))
	for _, c := range contexts {
		rs := filterByRankedContext(base, c.Value)
		rs = filterByOutcome(rs, req.OutcomeFilter)
		// skill_tiers ne s'applique que si un contexte ranked est défini
		rs = filterBySkillTier(rs, req.SkillTiers, c.Value)
		rs = filterByPerfTiers(rs, req.PerfTiers)
		rs = applyExplorerMatchFilters(rs, req, replays)
		out = append(out, domain.LabelValue{
			Label: c.Label,
			Value: c.Value,
			Count: len(rs),
		})
	}
	return out
}

// computeAvailableSquadScopes : count par scope squad (all/solo/squad).
//
// Single-select : applique tous les filtres + force squad_scope = X.
func computeAvailableSquadScopes(
	base []domain.MatchHistoryRawRow, req domain.MatchHistoryQueryRequest, replays port.ReplayAvailability,
) []domain.LabelValue {
	scopes := []struct {
		Value string
		Label string
	}{
		{"", "Tous"},
		{scopeSolo, "Solo"},
		{scopeSquad, "Escouade"},
	}
	// Construire une req sans squad_scope pour appliquer tous les autres filtres
	// Explorer puis forcer le scope candidat séparément.
	reqNoScope := req
	reqNoScope.SquadScope = ""
	out := make([]domain.LabelValue, 0, len(scopes))
	for _, s := range scopes {
		rs := filterByRankedContext(base, req.RankedContext)
		rs = filterByOutcome(rs, req.OutcomeFilter)
		rs = filterBySkillTier(rs, req.SkillTiers, req.RankedContext)
		rs = filterByPerfTiers(rs, req.PerfTiers)
		rs = applyExplorerMatchFilters(rs, reqNoScope, replays)
		rs = filterByExplorerSquadScope(rs, s.Value)
		out = append(out, domain.LabelValue{
			Label: s.Label,
			Value: s.Value,
			Count: len(rs),
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Helpers set/conversions (int — la version string vit dans filters_options.go)
// ---------------------------------------------------------------------------

func intSliceToSet(s []int) map[int]struct{} {
	out := make(map[int]struct{}, len(s))
	for _, v := range s {
		out[v] = struct{}{}
	}
	return out
}

// unionWithInt : retourne []int contenant set ∪ {v}. v ajouté si absent.
func unionWithInt(set map[int]struct{}, v int) []int {
	out := make([]int, 0, len(set)+1)
	for k := range set {
		out = append(out, k)
	}
	if _, ok := set[v]; !ok {
		out = append(out, v)
	}
	return out
}
