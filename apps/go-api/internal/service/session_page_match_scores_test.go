package service

// session_page_match_scores_test.go — SCORE ET DOMINANCE DES LIGNES DE MATCH de la session (plan
// `.ai/V7.5/PLAN_SESSIONS_EMPRISE_2026-10-06.md`, D15) : le libellé de la source unique (manches pour
// une variante déclarée), la dominance de l'enrichissement, rien d'inventé sans ligne canonique.

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func ligneScoree(id, variante string, dominance canonical.DominanceFlag) canonical.PlayerMatchRow {
	p0, p1, r0, r1, total, team := 181, 186, 2, 1, 3, 0
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:     id,
			GameVariant: &canonical.AssetReference{Kind: "game_variant", DefaultLabel: variante},
			RoundsTotal: &total,
			Teams: []canonical.TeamSnapshot{
				{TeamID: 0, Score: &p0, RoundsWon: &r0},
				{TeamID: 1, Score: &p1, RoundsWon: &r1},
			},
		},
		Self:       canonical.MatchParticipant{TeamID: &team},
		Enrichment: canonical.PlayerMatchEnrichment{DominanceFlag: dominance},
	}
}

func TestAppliquerScoresEtDominance(t *testing.T) {
	resp := domain.SessionPageResponse{
		Matches:        []domain.SessionDetailMatchRow{{MatchID: "m1"}, {MatchID: "inconnu"}},
		CompareMatches: []domain.SessionDetailMatchRow{{MatchID: "m2"}},
	}
	canon := []canonical.PlayerMatchRow{
		ligneScoree("m1", "Arena:Oddball", canonical.DominanceRemontada),
		ligneScoree("m2", "CTF:Arena", canonical.DominanceNone),
	}
	appliquerScoresEtDominance(&resp, canon, map[string]bool{"Arena:Oddball": true})

	if got := resp.Matches[0]; got.ScoreLabel != "2 - 1" || got.DominanceFlag != int(canonical.DominanceRemontada) {
		t.Errorf("m1 = %q / %d, attendu « 2 - 1 » (manches) et la remontada", got.ScoreLabel, got.DominanceFlag)
	}
	if got := resp.CompareMatches[0]; got.ScoreLabel != "181 - 186" || got.DominanceFlag != 0 {
		t.Errorf("m2 = %q / %d, attendu les points (variante non déclarée), sans dominance", got.ScoreLabel, got.DominanceFlag)
	}
	if got := resp.Matches[1]; got.ScoreLabel != "" || got.DominanceFlag != 0 {
		t.Errorf("match sans ligne canonique = %+v, attendu rien d'inventé", got)
	}
}
