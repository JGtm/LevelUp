// Package service — session_page_match_scores.go : SCORE ET DOMINANCE DES LIGNES DE MATCH de la page
// Sessions (plan `.ai/V7.5/PLAN_SESSIONS_EMPRISE_2026-10-06.md`, D15) — la bande de résultats, l'encoche
// de dominance et les en-têtes de la grille match par match les lisent.
package service

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// appliquerScoresEtDominance pose `score_label` (source unique analysis.ScoreLabelCanonical : manches
// pour une variante déclarée dans `roundsDecide`, points sinon) et `dominance_flag` sur les lignes de
// la session affichée et de la session comparée. Un match sans ligne canonique reste sans l'un ni
// l'autre.
func appliquerScoresEtDominance(resp *domain.SessionPageResponse, canon []canonical.PlayerMatchRow, roundsDecide map[string]bool) {
	parID := make(map[string]*canonical.PlayerMatchRow, len(canon))
	for i := range canon {
		parID[canon[i].Summary.MatchID] = &canon[i]
	}
	for _, rows := range [][]domain.SessionDetailMatchRow{resp.Matches, resp.CompareMatches} {
		for i := range rows {
			r, ok := parID[rows[i].MatchID]
			if !ok {
				continue
			}
			if label := analysis.ScoreLabelCanonical(*r, roundsDecide); label != nil {
				rows[i].ScoreLabel = *label
			}
			rows[i].DominanceFlag = int(r.Enrichment.DominanceFlag)
		}
	}
}
