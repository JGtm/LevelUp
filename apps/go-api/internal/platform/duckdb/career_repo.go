// Package duckdb — CareerRepo : données de progression de carrière.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// CareerRepo implémente port.CareerRepository.
type CareerRepo struct {
	pdb *PlayerDB
}

// NewCareerRepo crée un CareerRepo depuis un PlayerDB.
func NewCareerRepo(pdb *PlayerDB) *CareerRepo {
	return &CareerRepo{pdb: pdb}
}

// GetLatestRank retourne la dernière entrée de progression de rang.
func (r *CareerRepo) GetLatestRank(ctx context.Context) (*domain.CareerRankData, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var row domain.CareerRankData
	err := r.pdb.ReadDB().QueryRow(ctx, Q6CareerLatestRank).Scan(
		&row.RankNumber,
		&row.CurrentXP,
		&row.RecordedAt,
		&row.RankLabel,
		&row.RankName,
		&row.RankTier,
		&row.XPForNextRank,
		&row.XPTotal,
		&row.IsMaxRank,
	)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetLatestRank: %w", err)
	}
	return &row, nil
}

// GetXPHistory retourne l'historique XP complet.
func (r *CareerRepo) GetXPHistory(ctx context.Context) ([]domain.XPHistoryPoint, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q7CareerXPHistory)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetXPHistory: %w", err)
	}
	defer rows.Close()

	var results []domain.XPHistoryPoint
	for rows.Next() {
		var p domain.XPHistoryPoint
		if err := rows.Scan(&p.RecordedAt, &p.Rank, &p.CurrentXP, &p.XPTotal); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetXPHistory scan: %w", err)
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

// GetLUSRHistory retourne les checkpoints LUSR.
func (r *CareerRepo) GetLUSRHistory(ctx context.Context) ([]domain.LUSRCheckpointDTO, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q8LUSRHistory)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetLUSRHistory: %w", err)
	}
	defer rows.Close()

	var results []domain.LUSRCheckpointDTO
	for rows.Next() {
		var cp domain.LUSRCheckpointDTO
		var tier sql.NullString
		var subTier sql.NullInt16
		if err := rows.Scan(
			&cp.MatchID, &cp.RatingType, &cp.RatingValue, &cp.TierLabel, &cp.PlaylistGroup, &cp.RecordedAt,
			&cp.RatingDelta, &cp.PlaylistName, &cp.PlaylistID, &tier, &subTier,
		); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetLUSRHistory scan: %w", err)
		}
		tierLabel := ""
		if cp.TierLabel != nil {
			tierLabel = *cp.TierLabel
		}
		cp.BadgeImageURL = buildHomeSkillPeakBadgeURL(
			optionalNullStringValue(tier),
			tierLabel,
			optionalNullInt16Value(subTier),
			homeStaticTitleSlug,
		)
		results = append(results, cp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	r.enrichLUSRPlaylistNames(ctx, results)
	return results, nil
}

// enrichLUSRPlaylistNames résout les noms de playlists FR via asset_translations.
// Même pattern que applyMatchHistoryFRTranslations : lookup par playlist_id (UUID).
// Best-effort : silencieux si Metadata absent ou résolution échoue.
func (r *CareerRepo) enrichLUSRPlaylistNames(ctx context.Context, cps []domain.LUSRCheckpointDTO) {
	if r.pdb.Metadata == nil || len(cps) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(cps))
	var ids []string
	for _, cp := range cps {
		id := strings.TrimSpace(cp.PlaylistID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return
	}
	metaRepo := NewMetadataRepoFromDB(r.pdb.Metadata)
	names, err := metaRepo.ResolveAssetNamesBulk(ctx, "playlist", ids, PreferredLangsForLocale("fr"))
	if err != nil || len(names) == 0 {
		return
	}
	for i := range cps {
		id := strings.TrimSpace(cps[i].PlaylistID)
		if id == "" {
			continue
		}
		if name := strings.TrimSpace(names[id]); name != "" {
			cps[i].PlaylistName = name
		}
	}
}

// GetTopMatches retourne les N meilleurs matchs par performance_score.
func (r *CareerRepo) GetTopMatches(ctx context.Context) ([]domain.TopMatchRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q9TopMatches, r.pdb.XUID, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetTopMatches: %w", err)
	}
	defer rows.Close()

	var results []domain.TopMatchRawRow
	for rows.Next() {
		var m domain.TopMatchRawRow
		var _section int
		if err := rows.Scan(
			&m.MatchID, &m.PerformanceScore, &m.StartTime,
			&m.MapName, &m.PairName, &m.PlaylistName,
			&m.Outcome, &m.Kills, &m.Deaths, &m.KDA,
			&m.TeamMMR, &m.EnemyMMR, &m.DominanceFlag, &_section,
		); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetTopMatches scan: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// GetEncounters retourne les adversaires/coéquipiers fréquents.
func (r *CareerRepo) GetEncounters(ctx context.Context) ([]domain.EncounterRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := r.pdb.Shared.Query(ctx, Q10Encounters, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetEncounters: %w", err)
	}
	defer rows.Close()

	var results []domain.EncounterRawRow
	for rows.Next() {
		var e domain.EncounterRawRow
		if err := rows.Scan(
			&e.XUID, &e.Gamertag, &e.MatchCount, &e.AsTeammate, &e.AsEnemy, &e.AvgKDA,
		); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetEncounters scan: %w", err)
		}
		results = append(results, e)
	}
	return results, rows.Err()
}
