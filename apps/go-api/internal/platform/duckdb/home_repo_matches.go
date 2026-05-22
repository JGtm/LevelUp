// Package duckdb — home_repo_matches.go : chargement des matchs Home (Q26) +
// sessions (Q27) + médias récents (Q28) + count total (Q26b).
//
// Sous-module de home_repo.go (split god-file 2026-05-21).
package duckdb

import (
	"context"
	"database/sql"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// LoadHomeMatches charge tous les matchs du joueur (Q26).
func (r *HomeRepo) LoadHomeMatches(ctx context.Context) ([]legacymatch.HomeMatchRow, error) {
	rows, err := r.pdb.ReadDB().Query(ctx, Q26HomeMatches, r.pdb.XUID, r.pdb.XUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []legacymatch.HomeMatchRow
	for rows.Next() {
		var row legacymatch.HomeMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.MapID,
			&row.MapName,
			&row.MapNameFR,
			&row.PairID,
			&row.PairName,
			&row.PairNameFR,
			&row.GameVariantID,
			&row.GameVariantName,
			&row.PlaylistID,
			&row.PlaylistName,
			&row.PlaylistNameFR,
			&row.IsFirefight,
			&row.IsRanked,
			&row.SessionLabel,
			&row.IsWithFriends,
			&row.Outcome,
			&row.TeamID,
			&row.Team0Score,
			&row.Team1Score,
			&row.DominanceFlag,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.KDA,
			&row.Ratio,
			&row.Accuracy,
			&row.AvgLifeSeconds,
			&row.TimePlayedSecs,
			&row.DamageDealt,
			&row.DamageTaken,
			&row.TeamMMR,
			&row.EnemyMMR,
			&row.PerformanceScore,
			&row.SkillRatingValue,
			&row.SkillRatingType,
			&row.SkillTier,
			&row.SkillSubTier,
			&row.SkillTierLabel,
			&row.SkillRatingDelta,
			&row.SkillPlaylistGroup,
			&row.RankInTeam,
			&row.HeadshotKills,
			&row.PerfectKills,
			&row.MaxKillingSpree,
		); err != nil {
			return nil, err
		}
		tier := ""
		if row.SkillTier != nil {
			tier = *row.SkillTier
		}
		tierLabel := ""
		if row.SkillTierLabel != nil {
			tierLabel = *row.SkillTierLabel
		}
		row.SkillRankImageURL = buildHomeSkillPeakBadgeURL(tier, tierLabel, row.SkillSubTier, homeStaticTitleSlug, 0)
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	r.enrichHomeMatchTranslations(ctx, result)
	return result, nil
}

// CountPlayerMatches retourne le nombre total de matchs du joueur (Q26b).
//
// Phase 3 plan stabilisation 2026-05-22 : migré de pdb.ReadDB() (player conn)
// vers SharedReader.Get(). match_participants vit dans shared_matches_v2 et
// l'ATTACH shared sur la player conn a été retiré au sprint P0→P7 (ADR 0016).
// La query référence désormais `match_participants` sans préfixe `shared.`
// (cf. Q26bCountPlayerMatches dans queries_home_citations.go).
func (r *HomeRepo) CountPlayerMatches(ctx context.Context) (int, error) {
	if r.pdb.SharedReader == nil {
		return 0, nil
	}
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return 0, err
	}
	defer release()
	var count int
	err = sharedDB.QueryRowContext(ctx, Q26bCountPlayerMatches, r.pdb.XUID).Scan(&count)
	return count, err
}

// LoadHomeSessions charge les sessions avec label depuis player_match_enrichment (Q27).
func (r *HomeRepo) LoadHomeSessions(ctx context.Context) ([]legacymatch.HomeSessionRow, error) {
	rows, err := r.pdb.ReadDB().Query(ctx, Q27HomeSessions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []legacymatch.HomeSessionRow
	for rows.Next() {
		var row legacymatch.HomeSessionRow
		if err := rows.Scan(
			&row.MatchID,
			&row.SessionID,
			&row.SessionLabel,
			&row.IsWithFriends,
			&row.StartTime,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadRecentMedia charge les médias récents du joueur (Q28).
// Retourne une liste vide si la table media_files n'existe pas.
//
// Phase 3 plan stabilisation 2026-05-22 : migré de pdb.ReadDB() (player conn)
// vers pdb.SharedSocial. La table media_files (et media_match_associations)
// a été déplacée vers shared_social.duckdb via migration
// drop_media_from_player_db (cf. steps_player.go:370). Avant ce fix, la
// query lançait "Catalog Error: Table with name media_files does not exist"
// sur la player conn — cf. AUDIT_DUCKDB_ATTACH_2026-05-21 §2.
func (r *HomeRepo) LoadRecentMedia(ctx context.Context, limit int) ([]domain.HomeMediaRow, error) {
	if r.pdb.SharedSocial == nil {
		return nil, nil
	}
	rows, err := r.pdb.SharedSocial.Query(ctx, Q28RecentMedia, limit)
	if err != nil {
		// La table media_files peut ne pas exister — dégradation silencieuse.
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var result []domain.HomeMediaRow
	for rows.Next() {
		var row domain.HomeMediaRow
		var matchID sql.NullString
		var matchStartTime sql.NullTime
		if err := rows.Scan(&row.FileName, &matchID, &matchStartTime); err != nil {
			return nil, err
		}
		if matchID.Valid {
			row.MatchID = &matchID.String
		}
		if matchStartTime.Valid {
			row.MatchStartTime = &matchStartTime.Time
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
