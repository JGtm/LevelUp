// Package duckdb — MatchViewRepo : données pour la vue détail d'un match.
package duckdb

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// MatchViewRepo implémente port.MatchViewRepository.
type MatchViewRepo struct {
	pdb  *PlayerDB
	xuid string
}

// NewMatchViewRepo crée un MatchViewRepo.
func NewMatchViewRepo(pdb *PlayerDB, xuid string) *MatchViewRepo {
	return &MatchViewRepo{pdb: pdb, xuid: xuid}
}

// GetMatchMeta retourne les métadonnées du match (Q13).
func (r *MatchViewRepo) GetMatchMeta(ctx context.Context, matchID string) (*domain.MatchMetaRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var row domain.MatchMetaRaw
	err := r.pdb.Player.QueryRow(ctx, Q13MatchMeta, matchID).Scan(
		&row.MatchID,
		&row.StartTime,
		&row.DurationSeconds,
		&row.MapName,
		&row.PairName,
		&row.PlaylistName,
		&row.IsFirefight,
		&row.IsRanked,
	)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchMeta: %w", err)
	}
	return &row, nil
}

// GetPlayerMatchStats retourne les stats du joueur pour ce match (Q17).
func (r *MatchViewRepo) GetPlayerMatchStats(ctx context.Context, xuid, matchID string) (*domain.PlayerMatchStatsRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var s domain.PlayerMatchStatsRaw
	err := r.pdb.Player.QueryRow(ctx, Q17PlayerMatchStats, matchID, xuid).Scan(
		&s.OutcomeCode,
		&s.TeamID,
		&s.RankInTeam,
		&s.Kills,
		&s.Deaths,
		&s.Assists,
		&s.KDA,
		&s.Accuracy,
		&s.PersonalScore,
		&s.AvgLifeSeconds,
		&s.TimePlayedSeconds,
		&s.ShotsFired,
		&s.ShotsHit,
		&s.DamageDealt,
		&s.DamageTaken,
	)
	if err != nil {
		// Le joueur peut ne pas avoir participé → retourner une stats vide
		return &domain.PlayerMatchStatsRaw{}, nil
	}
	return &s, nil
}

// GetMatchEnrichment retourne l'enrichissement pour ce match (Q18).
func (r *MatchViewRepo) GetMatchEnrichment(ctx context.Context, matchID string) (*domain.MatchEnrichmentRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var e domain.MatchEnrichmentRaw
	err := r.pdb.Player.QueryRow(ctx, Q18MatchEnrichment, matchID).Scan(
		&e.PerformanceScore,
		&e.IsWithFriends,
		&e.IsExcluded,
	)
	if err != nil {
		// Pas d'enrichissement → retourner vide
		return &domain.MatchEnrichmentRaw{}, nil
	}
	return &e, nil
}

// GetMatchScoreboard retourne les stats de tous les joueurs (Q12).
func (r *MatchViewRepo) GetMatchScoreboard(ctx context.Context, matchID string) ([]domain.ScoreboardRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.Query(ctx, Q12MatchScoreboard, matchID)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchScoreboard: %w", err)
	}
	defer rows.Close()

	var results []domain.ScoreboardRaw
	for rows.Next() {
		var s domain.ScoreboardRaw
		if err := rows.Scan(
			&s.XUID,
			&s.Gamertag,
			&s.TeamID,
			&s.RankInTeam,
			&s.OutcomeCode,
			&s.PersonalScore,
			&s.Kills,
			&s.Deaths,
			&s.Assists,
			&s.KDA,
			&s.Accuracy,
			&s.TimePlayed,
			&s.TeamMMR,
			&s.EnemyMMR,
			&s.ShotsFired,
			&s.ShotsHit,
			&s.DamageDealt,
			&s.DamageTaken,
			&s.AvgLifeSeconds,
			&s.HeadshotKills,
			&s.MaxKillingSpree,
			&s.GrenadeKills,
			&s.MeleeKills,
			&s.PowerWeaponKills,
		); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchScoreboard scan: %w", err)
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

// GetMatchMedals retourne les médailles du joueur dans ce match (Q14).
func (r *MatchViewRepo) GetMatchMedals(ctx context.Context, xuid, matchID string) ([]domain.MedalRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.Query(ctx, Q14MatchMedals, xuid, matchID)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchMedals: %w", err)
	}
	defer rows.Close()

	var results []domain.MedalRaw
	var medalIDs []int64
	for rows.Next() {
		var m domain.MedalRaw
		if err := rows.Scan(&m.MedalID, &m.Count); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchMedals scan: %w", err)
		}
		results = append(results, m)
		medalIDs = append(medalIDs, m.MedalID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	labels := r.lookupMedalLabels(ctx, medalIDs)
	for index := range results {
		if label, ok := labels[results[index].MedalID]; ok {
			results[index].Label = label
			continue
		}
		results[index].Label = strconv.FormatInt(results[index].MedalID, 10)
	}
	return results, nil
}

// GetMatchEvents retourne les events highlight du match (Q21).
func (r *MatchViewRepo) GetMatchEvents(ctx context.Context, matchID string) ([]domain.EventRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.Query(ctx, Q21MatchEventsWithXUID, matchID)
	if err != nil {
		// La table peut être absente sur certains matchs → retourner vide
		return nil, nil
	}
	defer rows.Close()

	var results []domain.EventRaw
	for rows.Next() {
		var e domain.EventRaw
		var tsUTC interface{} // timestamp_utc ignoré (3e colonne)
		if err := rows.Scan(&e.EventType, &e.TickCount, &tsUTC, &e.XUID); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchEvents scan: %w", err)
		}
		results = append(results, e)
	}
	return results, rows.Err()
}

// GetMatchWeaponKills retourne les kills par arme du joueur (Q16).
func (r *MatchViewRepo) GetMatchWeaponKills(ctx context.Context, xuid, matchID string) ([]domain.WeaponKillRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.Query(ctx, Q16WeaponKills, xuid, matchID)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	var results []domain.WeaponKillRaw
	var weaponIDs []int64
	for rows.Next() {
		var w domain.WeaponKillRaw
		if err := rows.Scan(&w.WeaponID, &w.Kills); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchWeaponKills scan: %w", err)
		}
		results = append(results, w)
		weaponIDs = append(weaponIDs, w.WeaponID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	labels := r.lookupWeaponLabels(ctx, weaponIDs)
	for index := range results {
		if label, ok := labels[results[index].WeaponID]; ok {
			results[index].WeaponLabel = label
			continue
		}
		results[index].WeaponLabel = strconv.FormatInt(results[index].WeaponID, 10)
	}
	return results, nil
}

func (r *MatchViewRepo) lookupMedalLabels(ctx context.Context, medalIDs []int64) map[int64]string {
	return lookupLabelsByID(
		ctx,
		r.pdb.Metadata,
		`SELECT medal_id, citation_name_display
		 FROM citation_mappings
		 WHERE medal_id IN (%s)
		   AND citation_name_display IS NOT NULL
		   AND citation_name_display <> ''`,
		medalIDs,
	)
}

func (r *MatchViewRepo) lookupWeaponLabels(ctx context.Context, weaponIDs []int64) map[int64]string {
	return lookupLabelsByID(
		ctx,
		r.pdb.Metadata,
		`SELECT weapon_id, COALESCE(label_fr, label_en, CAST(weapon_id AS VARCHAR)) AS weapon_label
		 FROM weapon_labels
		 WHERE weapon_id IN (%s)`,
		weaponIDs,
	)
}

func lookupLabelsByID(ctx context.Context, db *DB, queryTemplate string, ids []int64) map[int64]string {
	labels := map[int64]string{}
	query, args, ok := buildLookupQuery(queryTemplate, ids)
	if !ok || db == nil {
		return labels
	}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return labels
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var label string
		if err := rows.Scan(&id, &label); err == nil && label != "" {
			labels[id] = label
		}
	}
	return labels
}

func buildLookupQuery(queryTemplate string, ids []int64) (string, []interface{}, bool) {
	uniqueIDs := uniqueInt64s(ids)
	if len(uniqueIDs) == 0 {
		return "", nil, false
	}

	placeholders := make([]string, 0, len(uniqueIDs))
	args := make([]interface{}, 0, len(uniqueIDs))
	for _, id := range uniqueIDs {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	return fmt.Sprintf(queryTemplate, strings.Join(placeholders, ",")), args, true
}

func uniqueInt64s(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

// GetMatchKVPairs retourne les paires killer→victim du match (Q20).
func (r *MatchViewRepo) GetMatchKVPairs(ctx context.Context, matchID string) ([]domain.KVPairRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.Query(ctx, Q20KVPairs, matchID)
	if err != nil {
		// Vue v_killer_victim_full peut être absente dans certaines DBs → vide
		return nil, nil
	}
	defer rows.Close()

	var results []domain.KVPairRaw
	for rows.Next() {
		var kv domain.KVPairRaw
		if err := rows.Scan(
			&kv.KillerXUID,
			&kv.KillerGT,
			&kv.VictimXUID,
			&kv.VictimGT,
			&kv.KillCount,
			&kv.TimeMS,
		); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchKVPairs scan: %w", err)
		}
		results = append(results, kv)
	}
	return results, rows.Err()
}
