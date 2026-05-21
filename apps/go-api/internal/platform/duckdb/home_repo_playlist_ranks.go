// Package duckdb — home_repo_playlist_ranks.go : 3 dernières playlists
// distinctes jouées avec leur dernier rang compétitif connu (Q26g).
//
// Sprint P7 / ADR 0016 : 3 phases SQL (Phase B shared, Phase A1 MSR player,
// Phase A2 snapshot player).
//
// Sous-module de home_repo.go (split god-file 2026-05-21).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/domain"
)

// LoadRecentPlaylistRanks retourne les 3 dernières playlists distinctes jouées avec leur
// dernier rang compétitif connu (Q26g). Retourne (nil, nil) si aucune donnée.
// Le nom de playlist est résolu depuis asset_translations (metadata) puis adapté à la locale.
func (r *HomeRepo) LoadRecentPlaylistRanks(ctx context.Context, locale string) ([]domain.HomePlaylistRank, error) {
	if r == nil || r.pdb == nil || r.pdb.Player == nil || r.pdb.SharedReader == nil {
		return nil, nil
	}

	// Sprint P7 / ADR 0016 : 3 phases.
	phaseB, err := r.loadPlaylistPhaseB(ctx)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(phaseB) == 0 {
		return nil, nil
	}
	matchIDs := make([]string, 0, len(phaseB))
	plIDs := make([]string, 0, len(phaseB))
	for _, p := range phaseB {
		if p.lastMatchID != "" {
			matchIDs = append(matchIDs, p.lastMatchID)
		}
		if p.playlistID != "" {
			plIDs = append(plIDs, p.playlistID)
		}
	}
	msrByMatch := r.loadPlaylistPhaseAMSR(ctx, matchIDs)
	snapshotByPlaylist := r.loadPlaylistPhaseASnapshot(ctx, plIDs)

	raws := make([]playlistRawItem, 0, len(phaseB))
	for _, p := range phaseB {
		item := buildHomePlaylistRankItem(p, msrByMatch, snapshotByPlaylist)
		raws = append(raws, playlistRawItem{playlistID: p.playlistID, playlistName: p.playlistName, item: item})
	}

	// Enrichissement FR depuis asset_translations (même source que les tuiles de matchs).
	playlistIDs := make([]string, 0, len(raws))
	for _, raw := range raws {
		if raw.playlistID != "" {
			playlistIDs = append(playlistIDs, raw.playlistID)
		}
	}
	assetNames := r.resolveAssetNames(ctx, "playlist", playlistIDs, "fr")

	result := make([]domain.HomePlaylistRank, 0, len(raws))
	for _, raw := range raws {
		nameFR := strings.TrimSpace(assetNames[raw.playlistID])
		raw.item.PlaylistName = resolvePlaylistNameForLocale(locale, nameFR, raw.playlistName)
		result = append(result, raw.item)
	}
	return result, nil
}

// playlistRawItem : tuple (playlistID, playlistName brut, item construit).
type playlistRawItem struct {
	playlistID   string
	playlistName string
	item         domain.HomePlaylistRank
}

// buildHomePlaylistRankItem assemble une HomePlaylistRank à partir d'un row Phase B
// + les MSR/snapshot Phase A. Si rang MSR connu → badge ranked ; sinon mode placement
// (badge unranked + matches_remaining) ; sinon item nu.
func buildHomePlaylistRankItem(
	p playlistPhaseBRow,
	msrByMatch map[string]playlistMSRRow,
	snapshotByPlaylist map[string]int,
) domain.HomePlaylistRank {
	item := domain.HomePlaylistRank{
		PlaylistName: p.playlistName,
		IsRanked:     p.isRanked,
	}
	if msr, ok := msrByMatch[p.lastMatchID]; ok {
		fillRankedMSRItem(&item, p.isRanked, msr)
		return item
	}
	if p.isRanked {
		fillPlacementItem(&item, p.playlistID, snapshotByPlaylist)
	}
	return item
}

// fillRankedMSRItem renseigne RatingType, RatingValue, TierLabel et BadgeImageURL
// depuis un MSR connu (ranked LUSR/CSR).
func fillRankedMSRItem(item *domain.HomePlaylistRank, isRanked bool, msr playlistMSRRow) {
	ratingType := ratingTypeLUSR
	if isRanked {
		ratingType = ratingTypeCSR
	}
	item.RatingType = &ratingType
	ratingValueCopy := msr.ratingValue
	item.RatingValue = &ratingValueCopy
	if msr.tierLabel != "" {
		item.TierLabel = stringPtr(msr.tierLabel)
	}
	item.BadgeImageURL = buildHomeSkillPeakBadgeURL(msr.tier, msr.tierLabel, msr.subTier, homeStaticTitleSlug, 0)
}

// fillPlacementItem renseigne BadgeImageURL (unranked) + MeasurementMatchesRemaining
// pour le mode placement (10 matchs avant rang).
func fillPlacementItem(item *domain.HomePlaylistRank, playlistID string, snapshotByPlaylist map[string]int) {
	completed := 0
	if rem, ok := snapshotByPlaylist[playlistID]; ok && rem > 0 {
		completed = 10 - rem
	}
	if completed < 0 {
		completed = 0
	}
	if completed > 9 {
		completed = 9
	}
	item.BadgeImageURL = unrankedBadgeURL(completed, homeStaticTitleSlug)
	if rem, ok := snapshotByPlaylist[playlistID]; ok {
		remCopy := rem
		item.MeasurementMatchesRemaining = &remCopy
	}
}

// playlistPhaseBRow : projection Phase B (shared) — playlist + dernier match.
type playlistPhaseBRow struct {
	playlistID   string
	playlistName string
	isRanked     bool
	lastMatchID  string
}

// playlistMSRRow : projection Phase A1 (player) — rating du last_match_id.
type playlistMSRRow struct {
	ratingValue float64
	tier        string
	tierFR      string
	subTier     int
	tierLabel   string
}

// loadPlaylistPhaseB : top 3 playlists pour xuid via SharedReader.
func (r *HomeRepo) loadPlaylistPhaseB(ctx context.Context) ([]playlistPhaseBRow, error) {
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "LoadRecentPlaylistRanks: SharedReader unavailable",
			"xuid", r.pdb.XUID, "err", err)
		return nil, nil //nolint:nilerr
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q26gPlaylistPhaseBShared, r.pdb.XUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []playlistPhaseBRow
	for rows.Next() {
		var p playlistPhaseBRow
		var lastPlayed sql.NullTime
		if err := rows.Scan(&p.playlistID, &p.playlistName, &p.isRanked, &lastPlayed, &p.lastMatchID); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// loadPlaylistPhaseAMSR : rating + tier des last_match_id depuis match_skill_rank.
func (r *HomeRepo) loadPlaylistPhaseAMSR(ctx context.Context, matchIDs []string) map[string]playlistMSRRow {
	out := make(map[string]playlistMSRRow, len(matchIDs))
	if len(matchIDs) == 0 {
		return out
	}
	query := fmt.Sprintf(Q26gPlaylistPhaseAMSRTpl, Placeholders(len(matchIDs)))
	rows, err := r.pdb.Player.Query(ctx, query, ToAnySlice(matchIDs)...)
	if err != nil {
		slog.DebugContext(ctx, "LoadRecentPlaylistRanks: Phase A1 (MSR) failed",
			"xuid", r.pdb.XUID, "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var matchID string
		var ratingValue sql.NullFloat64
		var tier, tierFR, tierLabel sql.NullString
		var subTier sql.NullInt16
		if err := rows.Scan(&matchID, &ratingValue, &tier, &tierFR, &subTier, &tierLabel); err != nil {
			continue
		}
		if !ratingValue.Valid {
			continue
		}
		out[matchID] = playlistMSRRow{
			ratingValue: ratingValue.Float64,
			tier:        optionalNullStringValue(tier),
			tierFR:      optionalNullStringValue(tierFR),
			subTier:     optionalNullInt16Value(subTier),
			tierLabel:   optionalNullStringValue(tierLabel),
		}
	}
	return out
}

// loadPlaylistPhaseASnapshot : current_measurement_remaining par playlist_id.
func (r *HomeRepo) loadPlaylistPhaseASnapshot(ctx context.Context, playlistIDs []string) map[string]int {
	out := make(map[string]int, len(playlistIDs))
	if len(playlistIDs) == 0 {
		return out
	}
	query := fmt.Sprintf(Q26gPlaylistPhaseASnapshotTpl, Placeholders(len(playlistIDs)))
	rows, err := r.pdb.Player.Query(ctx, query, ToAnySlice(playlistIDs)...)
	if err != nil {
		slog.DebugContext(ctx, "LoadRecentPlaylistRanks: Phase A2 (snapshot) failed",
			"xuid", r.pdb.XUID, "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var playlistID string
		var remaining sql.NullInt32
		if err := rows.Scan(&playlistID, &remaining); err != nil {
			continue
		}
		if remaining.Valid {
			out[playlistID] = int(remaining.Int32)
		}
	}
	return out
}

// resolvePlaylistNameForLocale retourne le nom de playlist adapté à la locale.
// Pour "en*" → préfère l'anglais ; sinon → préfère le français.
func resolvePlaylistNameForLocale(locale, fr, en string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "en") {
		if strings.TrimSpace(en) != "" {
			return en
		}
		return fr
	}
	if strings.TrimSpace(fr) != "" {
		return fr
	}
	return en
}
