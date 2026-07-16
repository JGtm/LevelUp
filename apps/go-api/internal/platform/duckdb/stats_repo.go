// Package duckdb â€” StatsRepo : chargement des mÃ©triques pour le performance score et le LUSR.
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
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

// LoadStatsMatches charge tous les matchs avec leurs métriques analytiques (Q23).
// Paramètre : xuid du joueur.
// Ordre de retour : start_time ASC.
//
// Split cross-DB (ADR 0016) : Phase A shared (Q23StatsMatchesShared) via
// SharedReader, Phase B player_match_enrichment, merge Go.
func (r *StatsRepo) LoadStatsMatches(ctx context.Context) ([]legacymatch.StatsMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	// Phase A : shared (mp + r) via SharedReader.
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatsRepo.LoadStatsMatches: shared reader: %w", err)
	}
	defer release()
	// Baseline rendement/résistance title-aware (225 Infinite, 115 h5) — liée 2×
	// (offensive_conversion + defensive_resistance) AVANT le xuid de la clause WHERE.
	hp := games.EffectiveHpToKill(r.pdb.TitleSlug)
	statsQ := resolveCampaignExclusion(Q23StatsMatchesShared, r.pdb.TitleSlug, "r")
	rows, err := sharedDB.QueryContext(ctx, statsQ, hp, hp, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("StatsRepo.LoadStatsMatches: %w", err)
	}
	defer rows.Close()

	var results []legacymatch.StatsMatchRow
	matchIDs := make([]string, 0)
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
			&m.OffensiveConversion,
			&m.DefensiveResistance,
		); err != nil {
			return nil, fmt.Errorf("StatsRepo.LoadStatsMatches scan: %w", err)
		}
		results = append(results, m)
		matchIDs = append(matchIDs, m.MatchID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("StatsRepo.LoadStatsMatches rows: %w", err)
	}

	// Phase B : player_match_enrichment via pdb.Player.
	if len(matchIDs) == 0 {
		return results, nil
	}
	if err := r.mergeStatsMatchesPME(ctx, results, matchIDs); err != nil {
		return nil, err
	}
	return results, nil
}

// mergeStatsMatchesPME hydrate les champs pme (performance_score, session_id,
// session_label) dans results pour les match_ids passés.
func (r *StatsRepo) mergeStatsMatchesPME(ctx context.Context, results []legacymatch.StatsMatchRow, matchIDs []string) error {
	query := fmt.Sprintf(Q23StatsMatchesPlayerEnrichTpl, Placeholders(len(matchIDs)))
	// QueryRecovered (Phase 5 ART) : retry après Reopen si la handle est invalidée.
	rows, err := r.pdb.Player.QueryRecovered(ctx, query, ToAnySlice(matchIDs)...)
	if err != nil {
		return fmt.Errorf("StatsRepo.LoadStatsMatches pme: %w", err)
	}
	defer rows.Close()

	type pmeRow struct {
		perfScore    *float64
		sessionID    *string
		sessionLabel *string
	}
	pmeByMatch := make(map[string]pmeRow, len(matchIDs))
	for rows.Next() {
		var mid string
		var pme pmeRow
		if err := rows.Scan(&mid, &pme.perfScore, &pme.sessionID, &pme.sessionLabel); err != nil {
			return fmt.Errorf("StatsRepo.LoadStatsMatches pme scan: %w", err)
		}
		pmeByMatch[mid] = pme
	}
	for i := range results {
		pme, ok := pmeByMatch[results[i].MatchID]
		if !ok {
			continue
		}
		results[i].PerfScoreComputed = pme.perfScore
		results[i].SessionID = pme.sessionID
		results[i].SessionLabel = pme.sessionLabel
	}
	return nil
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
