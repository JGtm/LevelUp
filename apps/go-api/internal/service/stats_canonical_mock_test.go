// Package service — stats_canonical_mock_test.go : mock partagé
// PlayerMatchesRepository pour tests Stats / Timeseries / SessionCompare /
// SessionPage. P4.3 finale.
//
// Convertit []domain.StatsMatchRow vers []canonical.PlayerMatchRow pour
// exercer le path canonical (le seul actif après suppression du legacy
// fallback).
package service

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

type mockStatsPlayerMatches struct {
	rows []domain.StatsMatchRow
	err  error
}

func (m *mockStatsPlayerMatches) LoadPlayerMatches(_ context.Context, _ string, _ string, _ port.PlayerMatchFilters) ([]canonical.PlayerMatchRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := make([]canonical.PlayerMatchRow, len(m.rows))
	for i, r := range m.rows {
		k, d, a := r.Kills, r.Deaths, r.Assists
		var outcome canonical.Outcome
		if r.Outcome != nil {
			switch *r.Outcome {
			case domain.OutcomeWin:
				outcome = canonical.OutcomeWin
			case domain.OutcomeLoss:
				outcome = canonical.OutcomeLoss
			case domain.OutcomeDraw:
				outcome = canonical.OutcomeTie
			case domain.OutcomeDNF:
				outcome = canonical.OutcomeDNF
			}
		}
		var dmgDealtPtr, dmgTakenPtr *int
		if r.DamageDealt != nil {
			v := int(*r.DamageDealt)
			dmgDealtPtr = &v
		}
		if r.DamageTaken != nil {
			v := int(*r.DamageTaken)
			dmgTakenPtr = &v
		}
		isRanked := r.IsRanked
		out[i] = canonical.PlayerMatchRow{
			Summary: canonical.MatchSummary{
				MatchID:      r.MatchID,
				StartedAtUTC: r.StartTime,
				IsRanked:     &isRanked,
				Outcome:      outcome,
				Playlist:     &canonical.AssetReference{Kind: "playlist", DefaultLabel: r.PlaylistName},
			},
			Self: canonical.MatchParticipant{
				Kills:         &k,
				Deaths:        &d,
				Assists:       &a,
				KDA:           r.KDA,
				Accuracy:      r.Accuracy,
				TimePlayed:    r.TimePlayedSeconds,
				PersonalScore: r.PersonalScore,
				RankInMatch:   r.Rank,
				TeamID:        r.TeamID,
				DamageDealt:   dmgDealtPtr,
				DamageTaken:   dmgTakenPtr,
				Outcome:       outcome,
			},
			Enrichment: canonical.PlayerMatchEnrichment{
				PerformanceScore: r.PerfScoreComputed,
				SessionID:        r.SessionID,
				SessionLabel:     r.SessionLabel,
				TeamMMR:          r.TeamMMR,
				EnemyMMR:         r.EnemyMMR,
				SkillSnapshot: &canonical.SkillSnapshot{
					KillsExpected:  r.KillsExpected,
					DeathsExpected: r.DeathsExpected,
				},
			},
		}
	}
	return out, nil
}

func (m *mockStatsPlayerMatches) InvalidatePlayer(_, _ string) {}

// newStatsMockFromRows construit un mock PlayerMatchesRepository à partir
// de rows StatsMatchRow legacy. Helper partagé entre les 4 services.
func newStatsMockFromRows(rows []domain.StatsMatchRow, err error) *mockStatsPlayerMatches {
	return &mockStatsPlayerMatches{rows: rows, err: err}
}
