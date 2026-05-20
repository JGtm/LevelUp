// Package duckdb â€” StatsRepo : chargement des mÃ©triques pour le performance score et le LUSR.
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// StatsRepo charge les donnÃ©es analytics (Q23-Q25) depuis le PlayerDB.
type StatsRepo struct {
	pdb *PlayerDB
}

// NewStatsRepo crÃ©e un StatsRepo depuis un PlayerDB.
func NewStatsRepo(pdb *PlayerDB) *StatsRepo {
	return &StatsRepo{pdb: pdb}
}

// LoadStatsMatches charge tous les matchs avec leurs mÃ©triques analytiques (Q23).
// ParamÃ¨tre : xuid du joueur.
// Ordre de retour : start_time ASC.
func (r *StatsRepo) LoadStatsMatches(ctx context.Context) ([]legacymatch.StatsMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q23StatsMatches, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("StatsRepo.LoadStatsMatches: %w", err)
	}
	defer rows.Close()

	var results []legacymatch.StatsMatchRow
	for rows.Next() {
		var m legacymatch.StatsMatchRow
		if err := rows.Scan(
			&m.MatchID,
			&m.StartTime,
			&m.Outcome,
			&m.Kills,
			&m.Deaths,
			&m.Assists,
			&m.KDA,
			&m.Accuracy,
			&m.PersonalScore,
			&m.DamageDealt,
			&m.DamageTaken,
			&m.TimePlayedSeconds,
			&m.TeamMMR,
			&m.EnemyMMR,
			&m.KillsExpected,
			&m.DeathsExpected,
			&m.Rank,
			&m.IsRanked,
			&m.PlaylistName,
			&m.PairName,
			&m.TeamID,
			&m.PerfScoreComputed,
			&m.SessionID,
			&m.SessionLabel,
			&m.OffensiveConversion,
			&m.DefensiveResistance,
		); err != nil {
			return nil, fmt.Errorf("StatsRepo.LoadStatsMatches scan: %w", err)
		}
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("StatsRepo.LoadStatsMatches rows: %w", err)
	}
	return results, nil
}

// LoadLUSRHistory charge l'historique LUSR depuis match_skill_rank (Q24).
func (r *StatsRepo) LoadLUSRHistory(ctx context.Context) ([]domain.LUSRMatchRating, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q24LUSRHistory)
	if err != nil {
		return nil, fmt.Errorf("StatsRepo.LoadLUSRHistory: %w", err)
	}
	defer rows.Close()

	var results []domain.LUSRMatchRating
	for rows.Next() {
		var m domain.LUSRMatchRating
		if err := rows.Scan(
			&m.MatchID,
			&m.RatingValue,
			&m.RatingDeviation,
			&m.PlaylistGroup,
		); err != nil {
			return nil, fmt.Errorf("StatsRepo.LoadLUSRHistory scan: %w", err)
		}
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("StatsRepo.LoadLUSRHistory rows: %w", err)
	}
	return results, nil
}

// LoadMatchParticipants charge tous les participants des matchs du joueur (Q25).
// Utilisé pour l'estimation enemy strength dans le calcul LUSR.
// Exécutée sur SharedReader (ADR 0016, shared-only).
func (r *StatsRepo) LoadMatchParticipants(ctx context.Context) ([]domain.ParticipantRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatsRepo.LoadMatchParticipants: shared reader: %w", err)
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, Q25MatchParticipants, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("StatsRepo.LoadMatchParticipants: %w", err)
	}
	defer rows.Close()

	var results []domain.ParticipantRow
	for rows.Next() {
		var p domain.ParticipantRow
		if err := rows.Scan(
			&p.MatchID,
			&p.XUID,
			&p.TeamID,
			&p.KillsExpected,
			&p.DeathsExpected,
		); err != nil {
			return nil, fmt.Errorf("StatsRepo.LoadMatchParticipants scan: %w", err)
		}
		results = append(results, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("StatsRepo.LoadMatchParticipants rows: %w", err)
	}
	return results, nil
}
