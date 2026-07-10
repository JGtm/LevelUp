// Package duckdb — career_repo_lusr.go : LUSR (skill rating) history pour la
// page Carrière. Découpé de career_repo.go (god-file split, refactor 2026-05-27).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// GetLUSRHistory retourne les checkpoints LUSR.
//
// Split cross-DB en 2 phases (ADR 0016) :
//   - Phase A : match_skill_rank (player) sur pdb.Player.
//   - Phase B : match_registry (shared) pour start_time + playlist via
//     SharedReader avec WHERE match_id IN (...).
//   - Phase C : tri par start_time ASC + calcul rating_delta côté Go via
//     LAG manuel par (rating_type, playlist_group).
//
// lusrPlayerRow projette une ligne match_skill_rank côté player.
type lusrPlayerRow struct {
	MatchID       string
	RatingType    string
	RatingValue   float64
	TierLabel     *string
	PlaylistGroup *string
	Tier          sql.NullString
	SubTier       sql.NullInt16
}

// lusrRegistryInfo agrège start_time + playlist depuis shared.match_registry.
type lusrRegistryInfo struct {
	RecordedAt   *time.Time
	PlaylistName string
	PlaylistID   string
}

func (r *CareerRepo) GetLUSRHistory(ctx context.Context) ([]domain.LUSRCheckpointDTO, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	playerRows, matchIDs, err := r.loadLUSRPlayerRows(ctx)
	if err != nil {
		return nil, err
	}
	if len(playerRows) == 0 {
		return nil, nil
	}

	registryByMatch, err := r.loadLUSRRegistryInfo(ctx, matchIDs)
	if err != nil {
		return nil, err
	}

	results := assembleLUSRResults(r.titleSlug(), playerRows, registryByMatch)
	sortLUSRResultsByRecordedAt(results)
	computeLUSRRatingDeltas(results)

	r.enrichLUSRPlaylistNames(ctx, results)
	return results, nil
}

// loadLUSRPlayerRows charge match_skill_rank du joueur (phase A).
func (r *CareerRepo) loadLUSRPlayerRows(ctx context.Context) ([]lusrPlayerRow, []string, error) {
	rows, err := r.pdb.Player.QueryRecovered(ctx, Q8LUSRHistoryPlayer)
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetLUSRHistory: phase A: %w", err)
	}
	defer rows.Close()

	var playerRows []lusrPlayerRow
	matchIDs := make([]string, 0)
	for rows.Next() {
		var p lusrPlayerRow
		if err := rows.Scan(&p.MatchID, &p.RatingType, &p.RatingValue, &p.TierLabel,
			&p.PlaylistGroup, &p.Tier, &p.SubTier); err != nil {
			return nil, nil, fmt.Errorf("CareerRepo.GetLUSRHistory scan A: %w", err)
		}
		playerRows = append(playerRows, p)
		matchIDs = append(matchIDs, p.MatchID)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return playerRows, matchIDs, nil
}

// loadLUSRRegistryInfo enrichit match_registry (phase B) via SharedReader.
func (r *CareerRepo) loadLUSRRegistryInfo(ctx context.Context, matchIDs []string) (map[string]lusrRegistryInfo, error) {
	out := make(map[string]lusrRegistryInfo, len(matchIDs))
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetLUSRHistory: shared reader: %w", err)
	}
	defer release()
	query := fmt.Sprintf(Q8LUSRHistoryRegistryTpl, Placeholders(len(matchIDs)))
	regRows, err := sharedDB.QueryContext(ctx, query, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetLUSRHistory: phase B: %w", err)
	}
	defer regRows.Close()
	for regRows.Next() {
		var mid string
		var info lusrRegistryInfo
		var ts sql.NullTime
		if err := regRows.Scan(&mid, &ts, &info.PlaylistName, &info.PlaylistID); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetLUSRHistory scan B: %w", err)
		}
		if ts.Valid {
			t := ts.Time
			info.RecordedAt = &t
		}
		out[mid] = info
	}
	return out, nil
}

// assembleLUSRResults compose les DTOs en croisant phases A et B.
// slug est le slug du titre du joueur (cf. CareerRepo.titleSlug()).
func assembleLUSRResults(slug string, playerRows []lusrPlayerRow, registryByMatch map[string]lusrRegistryInfo) []domain.LUSRCheckpointDTO {
	results := make([]domain.LUSRCheckpointDTO, 0, len(playerRows))
	for _, p := range playerRows {
		cp := domain.LUSRCheckpointDTO{
			MatchID:       p.MatchID,
			RatingType:    p.RatingType,
			RatingValue:   p.RatingValue,
			TierLabel:     p.TierLabel,
			PlaylistGroup: p.PlaylistGroup,
		}
		if info, ok := registryByMatch[p.MatchID]; ok {
			cp.RecordedAt = info.RecordedAt
			cp.PlaylistName = info.PlaylistName
			cp.PlaylistID = info.PlaylistID
		}
		tierLabel := ""
		if cp.TierLabel != nil {
			tierLabel = *cp.TierLabel
		}
		cp.BadgeImageURL = buildHomeSkillPeakBadgeURL(
			optionalNullStringValue(p.Tier),
			tierLabel,
			optionalNullInt16Value(p.SubTier),
			slug,
			0,
		)
		results = append(results, cp)
	}
	return results
}

// sortLUSRResultsByRecordedAt trie par recorded_at ASC (NULLS LAST).
func sortLUSRResultsByRecordedAt(results []domain.LUSRCheckpointDTO) {
	sort.SliceStable(results, func(i, j int) bool {
		ai, aj := results[i].RecordedAt, results[j].RecordedAt
		if ai == nil && aj == nil {
			return false
		}
		if ai == nil {
			return false
		}
		if aj == nil {
			return true
		}
		return ai.Before(*aj)
	})
}

// computeLUSRRatingDeltas calcule rating_delta = current - prev par (rating_type, playlist_group).
func computeLUSRRatingDeltas(results []domain.LUSRCheckpointDTO) {
	type lagKey struct {
		ratingType    string
		playlistGroup string
	}
	prev := make(map[lagKey]float64)
	for i := range results {
		key := lagKey{ratingType: results[i].RatingType}
		if results[i].PlaylistGroup != nil {
			key.playlistGroup = *results[i].PlaylistGroup
		}
		if p, ok := prev[key]; ok {
			delta := results[i].RatingValue - p
			results[i].RatingDelta = &delta
		}
		prev[key] = results[i].RatingValue
	}
}

// enrichLUSRPlaylistNames résout les noms de playlists via asset_translations
// selon la locale de requête (GH2-B3, même famille que enrichCSRPlaylistNames :
// la cascade PreferredLangsForLocale retombe sur le FR si l'EN manque).
// Lookup par playlist_id (UUID). Best-effort : silencieux si Metadata absent.
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
	names, err := metaRepo.ResolveAssetNamesBulk(ctx, "playlist", ids, PreferredLangsForLocale(ctxkeys.Locale(ctx)))
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
