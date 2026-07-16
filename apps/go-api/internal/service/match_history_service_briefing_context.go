// Package service — match_history_service_briefing_context.go : module contexte
// solo/escouade du bandeau de briefing de l'Explorer (item 6, P-5).
//
// Le split se calcule côté backend sur les raw rows du scope (signal
// IsWithFriends) car le briefing agrège sur TOUT le scope filtré, pas sur la
// page de table visible : le front ne peut pas reconstituer les deux
// sous-groupes depuis les lignes paginées. Aucun gating capability (P-7 :
// IsWithFriends est disponible tous titres) — dégradation par omission si un
// sous-groupe est trop petit ou vide.
package service

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// minContextSplitMatches est le seuil (par sous-groupe) d'affichage de la carte
// solo/escouade (D-B : défaut 10, aligné MinDimensionGroupMatches — même
// fiabilité minimale qu'un groupe de dimension). La carte n'apparaît que si
// CHAQUE contexte atteint ce seuil.
const minContextSplitMatches = 10

// buildBriefingContextSplit partitionne le scope sur IsWithFriends et agrège
// chaque contexte via aggregateRawStats. Retourne nil si l'un des deux
// sous-groupes est sous minContextSplitMatches (couvre aussi le scope
// mono-contexte : un sous-groupe vide est < seuil) — omission propre.
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
	if len(solo) < minContextSplitMatches || len(squad) < minContextSplitMatches {
		return nil
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
