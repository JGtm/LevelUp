// Package service — match_history_service_briefing_context.go : module contexte
// solo/escouade du bandeau de briefing de l'Explorer (item 6, P-5).
//
// Le split se calcule côté backend sur les raw rows du scope (signal
// IsWithFriends) car le briefing agrège sur TOUT le scope filtré, pas sur la
// page de table visible : le front ne peut pas reconstituer les deux
// sous-groupes depuis les lignes paginées. Aucun gating capability (P-7 :
// IsWithFriends est disponible tous titres). Un scope mono-contexte (un côté à
// zéro match) n'émet pas la carte ; un côté sous le seuil l'émet en état vide
// (Insufficient) avec les deux comptes.
package service

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// minContextSplitMatches est le nombre minimal de matchs de CHAQUE contexte pour
// comparer solo et escouade (D2 : même fiabilité minimale qu'un groupe de
// dimension, d'où l'alignement sur MinDimensionGroupMatches).
const minContextSplitMatches = MinDimensionGroupMatches

// buildBriefingContextSplit partitionne le scope sur IsWithFriends et agrège
// chaque contexte via aggregateRawStats. Retourne nil si un côté n'a aucun match
// (scope mono-contexte : rien à comparer). Si les deux côtés existent mais que l'un
// est sous minContextSplitMatches, retourne un split Insufficient dont seuls les
// comptes Matches sont renseignés.
func buildBriefingContextSplit(scope []domain.MatchHistoryRawRow) *domain.ExplorerBriefingContextSplit {
	solo := make([]domain.MatchHistoryRawRow, 0, len(scope))
	squad := make([]domain.MatchHistoryRawRow, 0, len(scope))
	for _, r := range scope {
		if r.IsWithFriends {
			squad = append(squad, r)
		} else {
			solo = append(solo, r)
		}
	}
	if len(solo) == 0 || len(squad) == 0 {
		return nil
	}
	if len(solo) < minContextSplitMatches || len(squad) < minContextSplitMatches {
		return &domain.ExplorerBriefingContextSplit{
			Insufficient: true,
			Solo:         domain.ExplorerBriefingContextGroup{Matches: len(solo)},
			Squad:        domain.ExplorerBriefingContextGroup{Matches: len(squad)},
		}
	}
	return &domain.ExplorerBriefingContextSplit{
		Solo:  briefingContextGroup(solo),
		Squad: briefingContextGroup(squad),
	}
}

// briefingContextGroup agrège les compteurs socle d'un contexte (solo ou
// escouade). Symétrique de buildBriefingScope (mêmes unités ADR 0006).
func briefingContextGroup(rows []domain.MatchHistoryRawRow) domain.ExplorerBriefingContextGroup {
	a := aggregateRawStats(rows)
	return domain.ExplorerBriefingContextGroup{
		Matches: a.matches,
		WinRate: analysis.WinRate(a.wins, a.matches),
		KDA:     a.kda,
		AvgPerf: a.perf,
	}
}
