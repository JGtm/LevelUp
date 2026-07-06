package teammates

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// mockSynthPlayerMatches : mock PlayerMatchesRepository qui convertit
// []SynthesisMatchRow vers []canonical.PlayerMatchRow. Dupliqué de
// service/stats_canonical_mock_test.go (K3b : packages de test disjoints).
type mockSynthPlayerMatches struct {
	rows []legacymatch.SynthesisMatchRow
	err  error
}

func (m *mockSynthPlayerMatches) LoadPlayerMatches(_ context.Context, _, _ string, _ port.PlayerMatchFilters) ([]canonical.PlayerMatchRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := make([]canonical.PlayerMatchRow, len(m.rows))
	for i, r := range m.rows {
		k, d := r.Kills, r.Deaths
		var outcome canonical.Outcome
		switch r.Outcome {
		case domain.OutcomeWin:
			outcome = canonical.OutcomeWin
		case domain.OutcomeLoss:
			outcome = canonical.OutcomeLoss
		case domain.OutcomeDraw:
			outcome = canonical.OutcomeTie
		case domain.OutcomeDNF:
			outcome = canonical.OutcomeDNF
		}
		isRanked := r.IsRanked
		isPvE := r.IsFirefight
		out[i] = canonical.PlayerMatchRow{
			Summary: canonical.MatchSummary{
				MatchID:      r.MatchID,
				StartedAtUTC: r.StartTime,
				IsRanked:     &isRanked,
				IsPvE:        &isPvE,
				Outcome:      outcome,
				Playlist:     &canonical.AssetReference{Kind: "playlist", DefaultLabel: r.PlaylistName},
			},
			Self: canonical.MatchParticipant{
				Kills: &k, Deaths: &d, KDA: r.KDA, Outcome: outcome,
				Accuracy: r.Accuracy, TimePlayed: r.TimePlayedSecs,
			},
			Enrichment: canonical.PlayerMatchEnrichment{
				IsWithFriends:    r.IsWithFriends,
				PerformanceScore: r.PerformanceScore,
				SessionLabel:     r.SessionLabel,
			},
		}
	}
	return out, nil
}

func (m *mockSynthPlayerMatches) InvalidatePlayer(_, _ string) {}

// newSynthMockFromRows construit un mock canonical pour les tests teammates.
func newSynthMockFromRows(rows []legacymatch.SynthesisMatchRow, err error) *mockSynthPlayerMatches {
	return &mockSynthPlayerMatches{rows: rows, err: err}
}
