// Package duckdb — ExplorerRepo : matchs communs entre deux joueurs.
//
// Port Go de apps/api/app/routers/explorer.py + services/explorer.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"levelup/go-api/internal/domain"
)

// ExplorerRepo implémente port.ExplorerRepository sur DuckDB.
type ExplorerRepo struct {
	pdb  *PlayerDB
	xuid string
}

// NewExplorerRepo crée un ExplorerRepo.
func NewExplorerRepo(pdb *PlayerDB, xuid string) *ExplorerRepo {
	return &ExplorerRepo{pdb: pdb, xuid: xuid}
}

// GetCommonMatches retourne les matchs communs entre xuid1 et xuid2 (max 100).
// Q19 retourne 10 colonnes : match_id, start_time, map_ui, mode_ui,
// player1_team_id, player2_team_id, player1_outcome, player1_kills, player1_deaths, player1_kda.
// Exécutée sur SharedReader (ADR 0016, shared-only).
func (r *ExplorerRepo) GetCommonMatches(ctx context.Context, xuid1, xuid2 string) ([]domain.CommonMatchRaw, error) {
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetCommonMatches: shared reader: %w", err)
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, Q19CommonMatches, xuid1, xuid2)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetCommonMatches: query: %w", err)
	}
	defer rows.Close()

	var results []domain.CommonMatchRaw
	for rows.Next() {
		var m domain.CommonMatchRaw
		if err := rows.Scan(
			&m.MatchID,
			&m.StartTime,
			&m.MapUI,
			&m.ModeUI,
			&m.Player1TeamID,
			&m.Player2TeamID,
			&m.Player1Outcome,
			&m.Player1Kills,
			&m.Player1Deaths,
			&m.Player1KDA,
		); err != nil {
			return nil, fmt.Errorf("ExplorerRepo.GetCommonMatches: scan: %w", err)
		}
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetCommonMatches: rows: %w", err)
	}
	return results, nil
}

// GetKillerVictimBetween retourne les kills croisés agrégés entre xuid1 et xuid2 (Q19b).
// Exécutée sur SharedReader (ADR 0016, shared-only).
func (r *ExplorerRepo) GetKillerVictimBetween(ctx context.Context, xuid1, xuid2 string) (domain.KillerVictimAggregate, error) {
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return domain.KillerVictimAggregate{}, fmt.Errorf("ExplorerRepo.GetKillerVictimBetween: shared reader: %w", err)
	}
	defer release()
	row := sharedDB.QueryRowContext(ctx, Q19bKillerVictimBetween, xuid1, xuid2, xuid2, xuid1)
	var agg domain.KillerVictimAggregate
	if err := row.Scan(&agg.KillsDealt, &agg.DeathsSuffered); err != nil {
		return domain.KillerVictimAggregate{}, fmt.Errorf("ExplorerRepo.GetKillerVictimBetween: %w", err)
	}
	return agg, nil
}

// GetParticipantStatsForMatches agrège les stats brutes du joueur cible
// (xuid) sur la liste de matchs fournie. Lecture sur shared.match_participants.
//
// Win/loss/draw sont dérivés du champ `outcome` (1=tie, 2=win, 3=loss,
// 4=DNF). Les DNF ne comptent ni en wins ni en losses ni en draws — convention
// alignée sur le reste du produit (cf. compare_repo, match_view).
//
// Retourne nil si matchIDs est vide.
func (r *ExplorerRepo) GetParticipantStatsForMatches(
	ctx context.Context, xuid string, matchIDs []string,
) (*domain.ParticipantStatsAggregate, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(matchIDs)), ",")
	q := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(kills), 0)               AS kills,
			COALESCE(SUM(deaths), 0)              AS deaths,
			COALESCE(SUM(assists), 0)             AS assists,
			COALESCE(SUM(CASE WHEN outcome = 2 THEN 1 ELSE 0 END), 0) AS wins,
			COALESCE(SUM(CASE WHEN outcome = 3 THEN 1 ELSE 0 END), 0) AS losses,
			COALESCE(SUM(CASE WHEN outcome = 1 THEN 1 ELSE 0 END), 0) AS draws,
			COALESCE(SUM(shots_fired), 0)         AS shots_fired,
			COALESCE(SUM(shots_hit), 0)           AS shots_hit,
			COALESCE(SUM(damage_dealt), 0.0)      AS damage_dealt,
			COALESCE(SUM(damage_taken), 0.0)      AS damage_taken,
			COALESCE(SUM(headshot_kills), 0)      AS headshot_kills,
			COALESCE(SUM(melee_kills), 0)         AS melee_kills,
			COALESCE(SUM(power_weapon_kills), 0)  AS power_weapon_kills,
			COALESCE(SUM(grenade_kills), 0)       AS grenade_kills,
			COALESCE(SUM(time_played_seconds), 0) AS time_played_seconds,
			COALESCE(SUM(personal_score), 0)      AS personal_score
		FROM match_participants
		WHERE xuid = ? AND match_id IN (%s)
	`, placeholders)

	args := make([]any, 0, 1+len(matchIDs))
	args = append(args, xuid)
	for _, mid := range matchIDs {
		args = append(args, mid)
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetParticipantStatsForMatches: shared reader: %w", err)
	}
	defer release()

	row := db.QueryRowContext(ctx, q, args...)
	var agg domain.ParticipantStatsAggregate
	err = row.Scan(
		&agg.Kills, &agg.Deaths, &agg.Assists,
		&agg.Wins, &agg.Losses, &agg.Draws,
		&agg.ShotsFired, &agg.ShotsHit,
		&agg.DamageDealt, &agg.DamageTaken,
		&agg.HeadshotKills, &agg.MeleeKills,
		&agg.PowerWeaponKills, &agg.GrenadeKills,
		&agg.TimePlayedSeconds, &agg.PersonalScore,
	)
	if err == sql.ErrNoRows {
		// Aucune ligne : le joueur n'a aucun participants row sur ces matchs.
		// On retourne un agrégat zéro (sampleSize sera 0 côté service).
		return &domain.ParticipantStatsAggregate{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetParticipantStatsForMatches: scan: %w", err)
	}
	return &agg, nil
}

// GetMedalCountsForMatches retourne le total d'occurrences (SUM(count)) et le
// nombre de types distincts de médailles gagnées par le joueur sur la liste
// de matchs fournie. Lecture sur shared.medals_earned.
//
// Retourne nil si matchIDs est vide.
func (r *ExplorerRepo) GetMedalCountsForMatches(
	ctx context.Context, xuid string, matchIDs []string,
) (*domain.MedalCountsAggregate, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(matchIDs)), ",")
	q := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(count), 0)                  AS total,
			COALESCE(COUNT(DISTINCT medal_name_id), 0) AS unique_count,
			-- Frags parfaits = médaille "Perfect" (medal_name_id 1512363953),
			-- même approche que Q12MatchScoreboard / Q30 (queries_squad.go).
			COALESCE(SUM(CASE WHEN medal_name_id = 1512363953 THEN count ELSE 0 END), 0) AS perfect_kills
		FROM medals_earned
		WHERE xuid = ? AND match_id IN (%s)
	`, placeholders)

	args := make([]any, 0, 1+len(matchIDs))
	args = append(args, xuid)
	for _, mid := range matchIDs {
		args = append(args, mid)
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetMedalCountsForMatches: shared reader: %w", err)
	}
	defer release()

	row := db.QueryRowContext(ctx, q, args...)
	var agg domain.MedalCountsAggregate
	if err := row.Scan(&agg.Total, &agg.Unique, &agg.PerfectKills); err != nil {
		if err == sql.ErrNoRows {
			return &domain.MedalCountsAggregate{}, nil
		}
		return nil, fmt.Errorf("ExplorerRepo.GetMedalCountsForMatches: scan: %w", err)
	}
	return &agg, nil
}

// ResolveXUIDByGamertag résout un gamertag en xuid via shared.v_gamertag_lookup (ILIKE).
//
// Source : la vue v_gamertag_lookup (cascade xuid_aliases ∪ match_participants
// avec fallback bots officiels). Plus robuste que la table xuid_aliases seule
// car elle capture aussi les joueurs qui sont apparus dans match_participants
// avant d'être synchronisés dans xuid_aliases. Bots filtrés (xuid 'bid(...)').
func (r *ExplorerRepo) ResolveXUIDByGamertag(ctx context.Context, gamertag string) (string, error) {
	const q = `
		SELECT xuid FROM v_gamertag_lookup
		WHERE gamertag ILIKE ? AND xuid NOT LIKE 'bid(%'
		LIMIT 1
	`
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return "", fmt.Errorf("ExplorerRepo.ResolveXUIDByGamertag(%q): %w", gamertag, err)
	}
	defer release()

	var xuid string
	if err := db.QueryRowContext(ctx, q, gamertag).Scan(&xuid); err != nil {
		return "", fmt.Errorf("ExplorerRepo.ResolveXUIDByGamertag(%q): %w", gamertag, err)
	}
	return xuid, nil
}
