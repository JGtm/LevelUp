// Package duckdb — career_repo_top_matches.go : GetTopMatches (top 10 WIN /
// top 10 LOSS par performance_score) pour la page Carrière. Découpé de
// career_repo.go (god-file split, refactor 2026-05-27).
package duckdb

import (
	"context"
	"fmt"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
)

// GetTopMatches retourne les 10 meilleurs (WIN) + 10 moins bons (LOSS) matchs
// par performance_score.
//
// Split cross-DB en 2 phases (ADR 0016) :
//   - Phase A : player_match_enrichment (pme) sur pdb.Player avec filtre
//     performance_score IS NOT NULL. had_bot_teammate est récupéré pour
//     l'asymétrie WIN/LOSS appliquée en Phase C.
//   - Phase B : match_participants + match_registry (shared) via SharedReader
//     avec filtres time_played>=180 + NOT is_firefight + IN match_ids.
//   - Phase C : merge + asymétrie bot (LOSS+bot skippé, WIN+bot conservé,
//     cf. GetHighlightMatchIDs / commit ec6efd2b) + sections WIN/LOSS + tri
//     par dominance flag (priorité section) + perf_score + top 10 chaque section.
//
// topMatchPMERow projette player_match_enrichment (perf + dominance + bot flag).
type topMatchPMERow struct {
	matchID        string
	perfScore      float64
	dominanceFlag  int
	hadBotTeammate bool
}

func (r *CareerRepo) GetTopMatches(ctx context.Context) ([]domain.TopMatchRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	pmes, err := r.loadTopMatchPMERows(ctx)
	if err != nil {
		return nil, err
	}
	if len(pmes) == 0 {
		return nil, nil
	}

	matchIDs := make([]string, 0, len(pmes))
	for id := range pmes {
		matchIDs = append(matchIDs, id)
	}

	enriched, err := r.loadTopMatchSharedRows(ctx, matchIDs, pmes)
	if err != nil {
		return nil, err
	}

	wins, losses := splitWinsLossesAndSortTopMatches(enriched)
	results := make([]domain.TopMatchRawRow, 0, len(wins)+len(losses))
	results = append(results, wins...)
	results = append(results, losses...)
	return results, nil
}

// loadTopMatchPMERows charge phase A : player_match_enrichment.
func (r *CareerRepo) loadTopMatchPMERows(ctx context.Context) (map[string]topMatchPMERow, error) {
	pmes := make(map[string]topMatchPMERow)
	pmeRows, err := r.pdb.Player.QueryRecovered(ctx, Q9TopMatchesPlayer)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetTopMatches: phase A: %w", err)
	}
	defer pmeRows.Close()
	for pmeRows.Next() {
		var p topMatchPMERow
		if err := pmeRows.Scan(&p.matchID, &p.perfScore, &p.dominanceFlag, &p.hadBotTeammate); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetTopMatches scan A: %w", err)
		}
		pmes[p.matchID] = p
	}
	return pmes, nil
}

// loadTopMatchSharedRows charge phase B (shared) + merge avec pmes.
func (r *CareerRepo) loadTopMatchSharedRows(
	ctx context.Context, matchIDs []string, pmes map[string]topMatchPMERow,
) ([]domain.TopMatchRawRow, error) {
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetTopMatches: shared reader: %w", err)
	}
	defer release()
	query := resolveCampaignExclusion(
		fmt.Sprintf(Q9TopMatchesSharedTpl, Placeholders(len(matchIDs))), r.titleSlug(), "r")
	sharedArgs := make([]any, 0, len(matchIDs)+1)
	sharedArgs = append(sharedArgs, r.pdb.XUID)
	sharedArgs = append(sharedArgs, ToAnySlice(matchIDs)...)
	sharedRows, err := sharedDB.QueryContext(ctx, query, sharedArgs...)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetTopMatches: phase B: %w", err)
	}
	defer sharedRows.Close()

	var enriched []domain.TopMatchRawRow
	for sharedRows.Next() {
		var m domain.TopMatchRawRow
		if err := sharedRows.Scan(
			&m.MatchID, &m.StartTime,
			&m.MapName, &m.PairName, &m.PlaylistName,
			&m.Outcome, &m.Kills, &m.Deaths, &m.KDA,
			&m.TeamMMR, &m.EnemyMMR,
		); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetTopMatches scan B: %w", err)
		}
		if pme, ok := pmes[m.MatchID]; ok {
			// Asymétrie bot teammate : LOSS avec bot dans MA team = handicap
			// 4v3, responsabilité du joueur non isolable → skip. WIN+bot reste
			// (perf personnelle méritoire). Cohérent avec GetHighlightMatchIDs.
			if m.Outcome == domain.OutcomeLoss && pme.hadBotTeammate {
				continue
			}
			m.PerformanceScore = pme.perfScore
			m.DominanceFlag = pme.dominanceFlag
			enriched = append(enriched, m)
		}
	}
	if err := sharedRows.Err(); err != nil {
		return nil, err
	}
	return enriched, nil
}

// splitWinsLossesAndSortTopMatches répartit en sections WIN/LOSS, trie chaque
// section et coupe au top 10.
//
// WIN : dominance ∈ (5,3,1) prioritaires (remontada/contre-remontada/domination), tri DESC.
// LOSS : dominance ∈ (4,2) prioritaires (débandade/humiliation), tri DESC.
// Tiebreak : perf_score DESC (WIN) ou ASC (LOSS, les moins bons en premier).
func splitWinsLossesAndSortTopMatches(enriched []domain.TopMatchRawRow) ([]domain.TopMatchRawRow, []domain.TopMatchRawRow) {
	var wins, losses []domain.TopMatchRawRow
	for _, m := range enriched {
		switch m.Outcome {
		case domain.OutcomeWin:
			wins = append(wins, m)
		case domain.OutcomeLoss:
			losses = append(losses, m)
		}
	}
	sort.SliceStable(wins, func(i, j int) bool {
		pi := topMatchDominancePriority(wins[i].DominanceFlag, []int{5, 3, 1})
		pj := topMatchDominancePriority(wins[j].DominanceFlag, []int{5, 3, 1})
		if pi != pj {
			return pi > pj
		}
		return wins[i].PerformanceScore > wins[j].PerformanceScore
	})
	sort.SliceStable(losses, func(i, j int) bool {
		pi := topMatchDominancePriority(losses[i].DominanceFlag, []int{4, 2})
		pj := topMatchDominancePriority(losses[j].DominanceFlag, []int{4, 2})
		if pi != pj {
			return pi > pj
		}
		return losses[i].PerformanceScore < losses[j].PerformanceScore
	})
	if len(wins) > 10 {
		wins = wins[:10]
	}
	if len(losses) > 10 {
		losses = losses[:10]
	}
	return wins, losses
}

// topMatchDominancePriority retourne la valeur du dominance_flag si présent
// dans le set prioritaire, sinon 0. Reproduit le `CASE WHEN ... THEN flag ELSE 0`
// du SQL historique Q9.
func topMatchDominancePriority(flag int, priorities []int) int {
	for _, p := range priorities {
		if flag == p {
			return flag
		}
	}
	return 0
}
