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

	"levelup/go-api/internal/analysis"
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
	// Fix S5 : on charge le MSR de TOUS les matchs récents (cap 30/playlist), pas
	// seulement du dernier — pour retomber sur le dernier match exploitable.
	matchIDSet := make(map[string]struct{}, len(phaseB)*8)
	matchIDs := make([]string, 0, len(phaseB)*8)
	plIDs := make([]string, 0, len(phaseB))
	for _, p := range phaseB {
		if p.lastMatchID != "" {
			if _, ok := matchIDSet[p.lastMatchID]; !ok {
				matchIDSet[p.lastMatchID] = struct{}{}
				matchIDs = append(matchIDs, p.lastMatchID)
			}
		}
		for _, mid := range p.recentMatchIDs {
			if mid == "" {
				continue
			}
			if _, ok := matchIDSet[mid]; !ok {
				matchIDSet[mid] = struct{}{}
				matchIDs = append(matchIDs, mid)
			}
		}
		if p.playlistID != "" {
			plIDs = append(plIDs, p.playlistID)
		}
	}
	msrByMatch := r.loadPlaylistPhaseAMSR(ctx, matchIDs)
	snapshotByPlaylist := r.loadPlaylistPhaseASnapshot(ctx, plIDs)

	// Locale → préférence FR pour le libellé du sous-palier suivant (même
	// convention que resolvePlaylistNameForLocale : tout sauf "en*" = FR).
	frPreferred := !strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "en")
	raws := make([]playlistRawItem, 0, len(phaseB))
	for _, p := range phaseB {
		// Phase 6 : threshold lookup par saison du dernier match. Fallback à
		// CSRPlacementThresholdDefault si season_id absent (matchs anciens
		// non backfillés ou playlists sociales).
		threshold := r.csrThreshold(p.lastSeasonID)
		raws = append(raws, playlistRawItem{
			playlistID:   p.playlistID,
			playlistName: p.playlistName,
			item:         buildPlaylistRankItem(r.titleSlug(), p, msrByMatch, snapshotByPlaylist, threshold, frPreferred),
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
		// Libellé d'affichage via le chokepoint unique (strip + override) — même
		// résolution que les tuiles de match : « Super Fiesta Fête » → « Super Fiesta ».
		nameFR := r.playlistDisplay.Display(strings.TrimSpace(assetNames[raw.playlistID]))
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
//
//nolint:unused // WIP refacto Phase B home_repo playlist ranks — péremption 2026-06-22 (à câbler ou supprimer).
func buildHomePlaylistRankItem(
	slug string,
	p playlistPhaseBRow,
	msrByMatch map[string]playlistMSRRow,
	snapshotByPlaylist map[string]int,
) domain.HomePlaylistRank {
	item := domain.HomePlaylistRank{
		PlaylistName: p.playlistName,
		IsRanked:     p.isRanked,
	}
	if msr, ok := msrByMatch[p.lastMatchID]; ok {
		fillRankedMSRItem(slug, &item, p.isRanked, msr)
		return item
	}
	if p.isRanked {
		fillPlacementItem(slug, &item, p.playlistID, snapshotByPlaylist)
	}
	return item
}

// fillRankedMSRItem renseigne RatingType, RatingValue, TierLabel et BadgeImageURL
// depuis un MSR connu (ranked LUSR/CSR).
//
//nolint:unused // WIP refacto Phase B home_repo playlist ranks — péremption 2026-06-22 (à câbler ou supprimer).
func fillRankedMSRItem(slug string, item *domain.HomePlaylistRank, isRanked bool, msr playlistMSRRow) {
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
	item.BadgeImageURL = buildHomeSkillPeakBadgeURL(msr.tier, msr.tierLabel, msr.subTier, slug, 0)
}

// fillPlacementItem renseigne BadgeImageURL (unranked) + MeasurementMatchesRemaining
// pour le mode placement (10 matchs avant rang).
//
//nolint:unused // WIP refacto Phase B home_repo playlist ranks — péremption 2026-06-22 (à câbler ou supprimer).
func fillPlacementItem(slug string, item *domain.HomePlaylistRank, playlistID string, snapshotByPlaylist map[string]int) {
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
	item.BadgeImageURL = unrankedBadgeURL(completed, slug)
	if rem, ok := snapshotByPlaylist[playlistID]; ok {
		remCopy := rem
		item.MeasurementMatchesRemaining = &remCopy
	}
}

// playlistPhaseBRow : projection Phase B (shared) — playlist + dernier match.
// lastSeasonID est l'identifiant CSR de la saison du dernier match (ex.
// "CsrSeason13-1") — peut être vide pour les matchs antérieurs au backfill
// season_id (Phase 1).
type playlistPhaseBRow struct {
	playlistID   string
	playlistName string
	isRanked     bool
	lastMatchID  string
	lastSeasonID string
	// recentMatchIDs : match_ids récents de la playlist (plus récent → plus
	// ancien, cap 30). Fix S5 : sert à retomber sur le dernier match QUI a une
	// ligne MSR exploitable quand le tout dernier n'en a pas (H5 : dernier match
	// avec seulement un placeholder CSR=0, sans LUSR).
	recentMatchIDs []string
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

	q := resolveCampaignExclusion(Q26gPlaylistPhaseBShared, r.titleSlug(), "r")
	rows, err := sharedDB.QueryContext(ctx, q, r.pdb.XUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []playlistPhaseBRow
	for rows.Next() {
		var p playlistPhaseBRow
		var lastPlayed sql.NullTime
		var lastSeasonID, recentMatchIDs sql.NullString
		if err := rows.Scan(&p.playlistID, &p.playlistName, &p.isRanked, &lastPlayed, &p.lastMatchID, &lastSeasonID, &recentMatchIDs); err != nil {
			return nil, err
		}
		p.lastSeasonID = optionalNullStringValue(lastSeasonID)
		p.recentMatchIDs = splitCSV(optionalNullStringValue(recentMatchIDs))
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
	rows, err := r.pdb.Player.QueryRecovered(ctx, query, ToAnySlice(matchIDs)...)
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
	rows, err := r.pdb.Player.QueryRecovered(ctx, query, ToAnySlice(playlistIDs)...)
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
// PlacementTotal=threshold. Assemblage Phase B des items par-playlist : branches
// isPlacement vs hasMSR vs fallback unranked, chacune avec sa logique
// tier/badge/value. La complexité reflète le nombre d'états légitimes du modèle.
//
//nolint:gocyclo // branches multiples par état métier, splitter perd la cohésion.
func buildPlaylistRankItem(
	slug string,
	p playlistPhaseBRow,
	msrByMatch map[string]playlistMSRRow,
	snapshotByPlaylist map[string]int,
	threshold int,
	frPreferred bool,
) domain.HomePlaylistRank {
	if threshold <= 0 {
		threshold = CSRPlacementThresholdDefault
	}
	item := domain.HomePlaylistRank{
		PlaylistName: p.playlistName,
		IsRanked:     p.isRanked,
	}
	msr, hasMSR := resolvePlaylistMSR(p, msrByMatch)
	snapRem, hasSnap := snapshotByPlaylist[p.playlistID]

	msrIsPlacement := hasMSR && (msr.tier == placementTier || strings.HasPrefix(msr.tierLabel, placementTier))
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
		item.BadgeImageURL = unrankedBadgeURLForThreshold(completed, threshold, slug)
		remCopy := remaining
		item.MeasurementMatchesRemaining = &remCopy
		totalCopy := threshold
		item.PlacementTotal = &totalCopy
		ratingType := ratingTypeCSR
		item.RatingType = &ratingType
		// RatingValue / TierLabel laissés nil : signal explicite au front.

	case hasMSR:
		ratingType := ratingTypeLUSR
		if p.isRanked {
			ratingType = ratingTypeCSR
		}
		item.RatingType = &ratingType
		ratingValueCopy := msr.ratingValue
		item.RatingValue = &ratingValueCopy
		if msr.tierLabel != "" {
			item.TierLabel = stringPtr(msr.tierLabel)
		}
		item.BadgeImageURL = buildHomeSkillPeakBadgeURLForThreshold(msr.tier, msr.tierLabel, msr.subTier, slug, 0, threshold)
		// Bande de progression ORDINALE (extrémité droite du palier) + sous-palier
		// suivant — même calcul que le skill peak (analysis.SkillTierBand /
		// NextSubTierLabel), indépendant de l'échelle CSR vs LUSR ; Onyx → barre
		// pleine sans suivant ; sub_tier hors 1..6 → pas de barre.
		if pct, ok := analysis.SkillTierBand(msr.tier, msr.subTier); ok {
			item.TierProgressPct = &pct
			item.NextTierLabel = analysis.NextSubTierLabel(msr.tier, msr.subTier, frPreferred)
		}
		// Rang matured : on expose quand même le placement_total pour info front.
		if p.isRanked {
			totalCopy := threshold
			item.PlacementTotal = &totalCopy
		}
	}
	return item
}

// resolvePlaylistMSR retourne le MSR exploitable le plus récent de la playlist
// (fix S5). On parcourt recentMatchIDs (déjà trié plus récent → plus ancien) et
// on prend la première entrée présente dans msrByMatch — celui-ci ne contient
// déjà que des lignes exploitables (CSR=0 placeholder filtré par
// Q26gPlaylistPhaseAMSRTpl). Fallback sur lastMatchID si la liste récente est
// vide (parité avec l'ancien comportement). Title-agnostique : sur Infinite, le
// dernier match classé a son CSR réel → on retombe sur lastMatchID au 1er tour.
func resolvePlaylistMSR(p playlistPhaseBRow, msrByMatch map[string]playlistMSRRow) (playlistMSRRow, bool) {
	for _, mid := range p.recentMatchIDs {
		if msr, ok := msrByMatch[mid]; ok {
			return msr, true
		}
	}
	msr, ok := msrByMatch[p.lastMatchID]
	return msr, ok
}

// splitCSV découpe une liste STRING_AGG (séparateur ',') en slice, en ignorant
// les segments vides. Retourne nil si l'entrée est vide.
func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
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
