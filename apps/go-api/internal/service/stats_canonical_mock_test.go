// Package service â€” stats_canonical_mock_test.go : mock partagÃ©
// PlayerMatchesRepository pour tests Stats / Timeseries / SessionCompare /
// SessionPage. P4.3 finale.
//
// Convertit []legacymatch.StatsMatchRow vers []canonical.PlayerMatchRow pour
// exercer le path canonical (le seul actif aprÃ¨s suppression du legacy
// fallback).
package service

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

type mockStatsPlayerMatches struct {
	rows []legacymatch.StatsMatchRow
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

// newStatsMockFromRows construit un mock PlayerMatchesRepository Ã  partir
// de rows StatsMatchRow legacy. Helper partagÃ© entre les 4 services.
func newStatsMockFromRows(rows []legacymatch.StatsMatchRow, err error) *mockStatsPlayerMatches {
	return &mockStatsPlayerMatches{rows: rows, err: err}
}

// mockSynthPlayerMatches : mock PlayerMatchesRepository qui convertit
// []SynthesisMatchRow vers []canonical.PlayerMatchRow. Pour squad/teammates
// services qui consomment SynthesisMatchRow.
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

// newSynthMockFromRows construit un mock canonical pour squad/teammates.
func newSynthMockFromRows(rows []legacymatch.SynthesisMatchRow, err error) *mockSynthPlayerMatches {
	return &mockSynthPlayerMatches{rows: rows, err: err}
}
