// Package service — match_history_service_briefing_streaks.go : modules « Séries »
// et « Moments forts » du bandeau de briefing de l'Explorer (items 8/9, P-9).
//
// Les séries se calculent côté backend sur TOUT le scope filtré (jamais depuis la
// frise outcome_sequence, cappée à maxOutcomeSequencePoints) : le front ne voit
// que la page paginée du tableau et ne peut pas reconstituer la plus longue série.
// Extrait du fichier principal du briefing pour rester sous le seuil de 500 lignes
// (CLAUDE.md §5), à l'image des modules ranked / context.
package service

import (
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// buildBriefingStreaks calcule la meilleure série de victoires et la pire série
// de défaites du scope (P-9). Tri chronologique par StartTime ; les rows sans
// date sont écartées (non ordonnables, comme la frise). Une série est rompue par
// TOUT autre outcome. Retourne nil si aucune row datée ; un segment à zéro reste
// à 0 (omitempty) et le front l'omet.
func buildBriefingStreaks(scope []domain.MatchHistoryRawRow) *domain.ExplorerBriefingStreaks {
	dated := make([]domain.MatchHistoryRawRow, 0, len(scope))
	for _, r := range scope {
		if r.StartTime != nil {
			dated = append(dated, r)
		}
	}
	if len(dated) == 0 {
		return nil
	}
	sort.SliceStable(dated, func(i, j int) bool {
		return dated[i].StartTime.Before(*dated[j].StartTime)
	})
	return &domain.ExplorerBriefingStreaks{
		BestWinStreak:   longestOutcomeRun(dated, domain.OutcomeWin),
		WorstLossStreak: longestOutcomeRun(dated, domain.OutcomeLoss),
	}
}

// longestOutcomeRun retourne la plus longue série de rows consécutives (dans
// l'ordre fourni) dont l'outcome vaut want. Rompue par TOUT autre outcome (P-9).
func longestOutcomeRun(rows []domain.MatchHistoryRawRow, want int) int {
	best, cur := 0, 0
	for _, r := range rows {
		if r.Outcome == want {
			cur++
			if cur > best {
				best = cur
			}
		} else {
			cur = 0
		}
	}
	return best
}

// buildBriefingDominance compte les DominanceFlag 1..5 du scope (P-9). Retourne
// nil si tous les compteurs sont à zéro (dégradation par omission). Constantes
// nommées analysis.DominanceFlag* (pas de magic number, CLAUDE.md §6).
func buildBriefingDominance(scope []domain.MatchHistoryRawRow) *domain.ExplorerBriefingDominance {
	var d domain.ExplorerBriefingDominance
	for _, r := range scope {
		switch r.DominanceFlag {
		case analysis.DominanceFlagDomination:
			d.Dominations++
		case analysis.DominanceFlagHumiliation:
			d.Humiliations++
		case analysis.DominanceFlagRemontada:
			d.Remontadas++
		case analysis.DominanceFlagDebacle:
			d.Debandades++
		case analysis.DominanceFlagContreRemontada:
			d.ContreRemontadas++
		}
	}
	if d.Dominations == 0 && d.Humiliations == 0 && d.Remontadas == 0 &&
		d.Debandades == 0 && d.ContreRemontadas == 0 {
		return nil
	}
	return &d
}
