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
	"regexp"
	"strings"

	"levelup/go-api/internal/domain"
)

// placementRemainingRe extrait N depuis "Placement (N restant)" ou
// "Placement (N restants)" écrit par csr_writes.formatCSRTierLabel.
var placementRemainingRe = regexp.MustCompile(`\((\d+)\s*restant`)

// parsePlacementRemaining extrait le N d'un tier_label "Placement (N restants)".
// Retourne 10 si non parsable (joueur en placement, 0 match joué).
func parsePlacementRemaining(label string) int {
	m := placementRemainingRe.FindStringSubmatch(label)
	if len(m) < 2 {
		return 10
	}
	var n int
	if _, err := fmt.Sscanf(m[1], "%d", &n); err != nil {
		return 10
	}
	if n < 0 {
		return 0
	}
	if n > 10 {
		return 10
	}
	return n
}

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

	type rawItem struct {
		playlistID   string
		playlistName string
		item         domain.HomePlaylistRank
	}
	raws := make([]rawItem, 0, len(phaseB))
	for _, p := range phaseB {
		// Phase 6 : threshold lookup par saison du dernier match. Fallback à
		// CSRPlacementThresholdDefault si season_id absent (matchs anciens
		// non backfillés ou playlists sociales).
		threshold := r.csrThreshold(p.lastSeasonID)
		raws = append(raws, rawItem{
			playlistID:   p.playlistID,
			playlistName: p.playlistName,
			item:         buildPlaylistRankItem(p, msrByMatch, snapshotByPlaylist, threshold),
		})
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

// playlistPhaseBRow : projection Phase B (shared) — playlist + dernier match.
// lastSeasonID est l'identifiant CSR de la saison du dernier match (ex.
// "CsrSeason13-1") — peut être vide pour les matchs antérieurs au backfill
// season_id (Phase 1).
type playlistPhaseBRow struct {
	playlistID    string
	playlistName  string
	isRanked      bool
	lastMatchID   string
	lastSeasonID  string
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
		var lastSeasonID sql.NullString
		if err := rows.Scan(&p.playlistID, &p.playlistName, &p.isRanked, &lastPlayed, &p.lastMatchID, &lastSeasonID); err != nil {
			return nil, err
		}
		p.lastSeasonID = optionalNullStringValue(lastSeasonID)
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

// buildPlaylistRankItem assemble l'item domain.HomePlaylistRank pour une
// playlist Phase B en croisant MSR (Phase A1) et snapshot (Phase A2).
//
// Détection placement (parité avec resolveSkillPeakState côté front) :
//   - snapshot.current_measurement_remaining > 0, OU
//   - MSR avec tier="Placement" / tier_label commence par "Placement"
//     (sync écrit rating_value=0.0 en placement pour respecter NOT NULL du
//     schéma, cf. csr_writes.go).
//
// threshold : seuil de placement de la saison du match (5 depuis S3, 10 historique).
// Phase 6 du plan : passé par le caller via lookup CSRThresholdsRepo(p.lastSeasonID).
//
// En placement : RatingValue/TierLabel laissés nil, BadgeImageURL=unranked_N.png
// (mapping proportionnel via threshold), MeasurementMatchesRemaining=remaining,
// PlacementTotal=threshold.
func buildPlaylistRankItem(
	p playlistPhaseBRow,
	msrByMatch map[string]playlistMSRRow,
	snapshotByPlaylist map[string]int,
	threshold int,
) domain.HomePlaylistRank {
	if threshold <= 0 {
		threshold = CSRPlacementThresholdDefault
	}
	item := domain.HomePlaylistRank{
		PlaylistName: p.playlistName,
		IsRanked:     p.isRanked,
	}
	msr, hasMSR := msrByMatch[p.lastMatchID]
	snapRem, hasSnap := snapshotByPlaylist[p.playlistID]

	msrIsPlacement := hasMSR && (msr.tier == "Placement" || strings.HasPrefix(msr.tierLabel, "Placement"))
	snapIsPlacement := hasSnap && snapRem > 0
	// Une playlist classée sans MSR ET sans snapshot positif est traitée comme
	// placement à 0 match joué (parité avec l'ancien code `else if p.isRanked`).
	isPlacement := p.isRanked && (snapIsPlacement || msrIsPlacement || !hasMSR)

	switch {
	case isPlacement:
		remaining := threshold // défaut : 0 match de placement joué
		switch {
		case snapIsPlacement:
			remaining = snapRem
		case msrIsPlacement:
			remaining = parsePlacementRemaining(msr.tierLabel)
		}
		completed := threshold - remaining
		if completed < 0 {
			completed = 0
		}
		if completed >= threshold {
			completed = threshold - 1
		}
		item.BadgeImageURL = unrankedBadgeURLForThreshold(completed, threshold, homeStaticTitleSlug)
		remCopy := remaining
		item.MeasurementMatchesRemaining = &remCopy
		totalCopy := threshold
		item.PlacementTotal = &totalCopy
		ratingType := "CSR"
		item.RatingType = &ratingType
		// RatingValue / TierLabel laissés nil : signal explicite au front.

	case hasMSR:
		ratingType := "LUSR"
		if p.isRanked {
			ratingType = "CSR"
		}
		item.RatingType = &ratingType
		ratingValueCopy := msr.ratingValue
		item.RatingValue = &ratingValueCopy
		if msr.tierLabel != "" {
			item.TierLabel = stringPtr(msr.tierLabel)
		}
		item.BadgeImageURL = buildHomeSkillPeakBadgeURLForThreshold(msr.tier, msr.tierLabel, msr.subTier, homeStaticTitleSlug, 0, threshold)
		// Rang matured : on expose quand même le placement_total pour info front.
		if p.isRanked {
			totalCopy := threshold
			item.PlacementTotal = &totalCopy
		}
	}
	return item
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
