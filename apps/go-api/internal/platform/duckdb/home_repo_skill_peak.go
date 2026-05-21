// Package duckdb — home_repo_skill_peak.go : skill peak Home (CSR/LUSR).
//
// Charge le meilleur rating CSR ou LUSR pour la home, avec gestion de la
// phase de placement (10 matchs par playlist_group côté LUSR, 10 matchs
// côté Microsoft pour CSR via player_csr_snapshots). Inclut le builder
// d'URL de badge de rang (utilisé aussi par career_repo / match_history).
//
// Sous-module de home_repo.go (split god-file 2026-05-21).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path"
	"strconv"
	"strings"

	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite"
)

// Labels rating_type (UPPERCASE pour cohérence DB : match_skill_rank.rating_type).
// canonical.RatingType{CSR,LUSR} reste lowercase pour l'API publique.
const (
	ratingTypeLUSR = "LUSR"
	ratingTypeCSR  = "CSR"
)

// Tier names canoniques Halo Infinite (PascalCase pour l'affichage UI).
// Utilisés par canonicalHomeSkillTierName + tests de classification rang.
const (
	tierBronze  = "Bronze"
	tierGold    = "Gold"
	tierDiamond = "Diamond"
)

// peakRow : scratch interne pour la classification CSR/LUSR + best per group.
type peakRow struct {
	matchID       string
	playlistGroup string // "_unknown" si NULL en DB
	ratingValue   float64
	ratingType    string // raw msr.rating_type
	tier          string
	subTier       int
	tierLabel     string
	recency       sql.NullTime
}

// peakRegistryInfo : projection Phase B match_registry pour classification CSR/LUSR.
type peakRegistryInfo struct {
	isRanked     bool
	playlistName string
	pairName     string
}

// loadHomeSkillPeak lit le meilleur rating CSR ou LUSR pour la home, avec
// gestion de la phase de placement (10 matchs par playlist_group côté LUSR,
// 10 matchs côté Microsoft pour CSR via player_csr_snapshots).
//
// Comportement :
//   - CSR : priorité à player_csr_snapshots.alltime_value (officiel Waypoint).
//     Si vide, fallback sur Q26eHomeSkillPeakByType qui calcule via les
//     match_skill_rank classés CSR (heuristique playlist_name).
//   - LUSR : Q26eHomeSkillPeakByType uniquement.
//   - En placement (placement_remaining > 0) : retourne un row avec
//     BadgeImageURL=unranked_(10-remaining).png et MeasurementMatchesRemaining
//     non-nil ; le front affichera "En placement" sans inventer.
//   - Matured (placement_remaining = 0) : retourne rating + tier badge habituel.
func (r *HomeRepo) loadHomeSkillPeak(ctx context.Context, ratingType string) *domain.HomeSkillPeakRow {
	if r == nil || r.pdb == nil || r.pdb.Player == nil {
		return nil
	}

	// Pour CSR, lire depuis player_csr_snapshots (alltime officiel Waypoint).
	// Fallback sur match_skill_rank si la table est vide ou absente.
	if ratingType == "CSR" {
		if peak := r.loadCSRAlltimePeak(ctx); peak != nil {
			return peak
		}
	}

	playerRows, err := r.loadPeakPhaseA(ctx)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "loadHomeSkillPeak: match_skill_rank missing",
				"rating_type", ratingType, "xuid", r.pdb.XUID, "err", err)
			return nil
		}
		slog.WarnContext(ctx, "loadHomeSkillPeak: Phase A failed (silent drop)",
			"rating_type", ratingType, "xuid", r.pdb.XUID, "err", err)
		return nil
	}
	if len(playerRows) == 0 {
		return nil
	}
	matchIDs := make([]string, 0, len(playerRows))
	for _, pr := range playerRows {
		matchIDs = append(matchIDs, pr.matchID)
	}
	registryByMatch := r.loadPeakPhaseB(ctx, matchIDs)
	return r.assemblePeak(playerRows, registryByMatch, ratingType)
}

// loadPeakPhaseA : query match_skill_rank sur pdb.Player (player-only).
func (r *HomeRepo) loadPeakPhaseA(ctx context.Context) ([]peakRow, error) {
	rows, err := r.pdb.Player.Query(ctx, Q26ePeakPhaseAPlayer)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []peakRow
	for rows.Next() {
		var (
			matchID    string
			playlist   sql.NullString
			rating     sql.NullFloat64
			ratingType sql.NullString
			tier       sql.NullString
			subTier    sql.NullInt16
			tierLabel  sql.NullString
			recency    sql.NullTime
		)
		if err := rows.Scan(&matchID, &playlist, &rating, &ratingType, &tier, &subTier, &tierLabel, &recency); err != nil {
			return nil, err
		}
		if !rating.Valid {
			continue
		}
		pr := peakRow{
			matchID:       matchID,
			playlistGroup: "_unknown",
			ratingValue:   rating.Float64,
			ratingType:    optionalNullStringValue(ratingType),
			tier:          optionalNullStringValue(tier),
			subTier:       optionalNullInt16Value(subTier),
			tierLabel:     optionalNullStringValue(tierLabel),
			recency:       recency,
		}
		if playlist.Valid && strings.TrimSpace(playlist.String) != "" {
			pr.playlistGroup = playlist.String
		}
		out = append(out, pr)
	}
	return out, rows.Err()
}

// loadPeakPhaseB : enrichit avec match_registry via SharedReader.
func (r *HomeRepo) loadPeakPhaseB(ctx context.Context, matchIDs []string) map[string]peakRegistryInfo {
	out := make(map[string]peakRegistryInfo, len(matchIDs))
	if len(matchIDs) == 0 || r.pdb.SharedReader == nil {
		return out
	}
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "loadHomeSkillPeak: Phase B SharedReader unavailable",
			"xuid", r.pdb.XUID, "err", err)
		return out
	}
	defer release()

	query := fmt.Sprintf(Q26ePeakPhaseBRegistryTpl, Placeholders(len(matchIDs)))
	rows, err := sharedDB.QueryContext(ctx, query, ToAnySlice(matchIDs)...)
	if err != nil {
		slog.WarnContext(ctx, "loadHomeSkillPeak: Phase B query failed",
			"xuid", r.pdb.XUID, "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var (
			matchID      string
			isRanked     bool
			playlistName string
			pairName     string
		)
		if err := rows.Scan(&matchID, &isRanked, &playlistName, &pairName); err != nil {
			continue
		}
		out[matchID] = peakRegistryInfo{isRanked: isRanked, playlistName: playlistName, pairName: pairName}
	}
	return out
}

// assemblePeak : filtre par effective_type, groupe, sélectionne le best matured.
func (r *HomeRepo) assemblePeak(playerRows []peakRow, registryByMatch map[string]peakRegistryInfo, ratingType string) *domain.HomeSkillPeakRow {
	want := strings.ToUpper(strings.TrimSpace(ratingType))
	type groupBest struct {
		row        peakRow
		matchCount int
	}
	byGroup := make(map[string]*groupBest)
	for _, pr := range playerRows {
		if classifyPeakType(pr, registryByMatch) != want {
			continue
		}
		gb, ok := byGroup[pr.playlistGroup]
		if !ok {
			gb = &groupBest{row: pr}
			byGroup[pr.playlistGroup] = gb
		}
		gb.matchCount++
		if isBetterPeak(pr, gb.row) {
			gb.row = pr
		}
	}
	if len(byGroup) == 0 {
		return nil
	}

	const placementThreshold = 10
	var chosen *groupBest
	for _, gb := range byGroup {
		switch {
		case chosen == nil:
			chosen = gb
		case gb.matchCount >= placementThreshold && chosen.matchCount < placementThreshold:
			chosen = gb
		case (gb.matchCount >= placementThreshold) == (chosen.matchCount >= placementThreshold):
			if gb.row.ratingValue > chosen.row.ratingValue {
				chosen = gb
			}
		}
	}
	remaining := placementThreshold - chosen.matchCount
	if remaining < 0 {
		remaining = 0
	}

	peak := &domain.HomeSkillPeakRow{RatingValue: chosen.row.ratingValue}
	if remaining > 0 {
		peak.BadgeImageURL = buildHomeSkillPeakBadgeURL("", "", 0, homeStaticTitleSlug, remaining)
		remCopy := remaining
		peak.MeasurementMatchesRemaining = &remCopy
		return peak
	}
	peak.BadgeImageURL = buildHomeSkillPeakBadgeURL(chosen.row.tier, chosen.row.tierLabel, chosen.row.subTier, homeStaticTitleSlug, 0)
	if strings.TrimSpace(chosen.row.tierLabel) != "" {
		peak.TierLabel = stringPtr(chosen.row.tierLabel)
	}
	zero := 0
	peak.MeasurementMatchesRemaining = &zero
	return peak
}

// classifyPeakType : heuristique CSR/LUSR identique à Q26e historique.
func classifyPeakType(pr peakRow, registryByMatch map[string]peakRegistryInfo) string {
	if info, ok := registryByMatch[pr.matchID]; ok {
		if info.isRanked ||
			strings.Contains(strings.ToLower(info.playlistName), "ranked") ||
			strings.Contains(strings.ToLower(info.pairName), "ranked") {
			return ratingTypeCSR
		}
		return ratingTypeLUSR
	}
	rt := strings.ToUpper(strings.TrimSpace(pr.ratingType))
	if rt == ratingTypeCSR {
		return ratingTypeCSR
	}
	return ratingTypeLUSR
}

// isBetterPeak : ordre rating DESC, recency DESC, sub_tier DESC, match_id DESC.
func isBetterPeak(candidate, current peakRow) bool {
	if candidate.ratingValue != current.ratingValue {
		return candidate.ratingValue > current.ratingValue
	}
	if candidate.recency.Valid && current.recency.Valid && !candidate.recency.Time.Equal(current.recency.Time) {
		return candidate.recency.Time.After(current.recency.Time)
	}
	if candidate.recency.Valid != current.recency.Valid {
		return candidate.recency.Valid
	}
	if candidate.subTier != current.subTier {
		return candidate.subTier > current.subTier
	}
	return candidate.matchID > current.matchID
}

// loadCSRAlltimePeak lit le meilleur CSR alltime depuis player_csr_snapshots.
// Si aucun alltime_value > 0 n'existe (joueur en cours de placement sur sa
// première playlist ranked), on rend un row placement avec
// BadgeImageURL=unranked_N.png basé sur le MIN(current_measurement_remaining)
// (la playlist la plus avancée dans son placement) pour que la home affiche
// "En placement N/10" au lieu de "Aucune partie classée".
func (r *HomeRepo) loadCSRAlltimePeak(ctx context.Context) *domain.HomeSkillPeakRow {
	var value sql.NullFloat64
	var tier sql.NullString
	var subTier sql.NullInt16
	if err := r.pdb.ReadDB().QueryRow(ctx, Q26csrAlltimePeak).Scan(&value, &tier, &subTier); err == nil && value.Valid {
		peak := &domain.HomeSkillPeakRow{RatingValue: value.Float64}
		tierStr := optionalNullStringValue(tier)
		subTierInt := optionalNullInt16Value(subTier)
		peak.BadgeImageURL = buildHomeSkillPeakBadgeURL(tierStr, "", subTierInt, homeStaticTitleSlug, 0)
		if tierStr != "" {
			peak.TierLabel = stringPtr(tierStr)
		}
		zero := 0
		peak.MeasurementMatchesRemaining = &zero
		return peak
	}

	// Pas d'alltime : tenter de récupérer l'état de placement le plus avancé.
	// MIN(current_measurement_remaining) = playlist la plus proche de la fin
	// du placement (10 → 0). Si pas de snapshot du tout : retourner nil pour
	// laisser Q26e fallback prendre la suite.
	var minRemaining sql.NullInt32
	if err := r.pdb.ReadDB().QueryRow(ctx, `
		SELECT MIN(current_measurement_remaining)
		FROM player_csr_snapshots
		WHERE current_measurement_remaining IS NOT NULL
		  AND current_measurement_remaining > 0
	`).Scan(&minRemaining); err != nil || !minRemaining.Valid {
		return nil
	}
	remaining := int(minRemaining.Int32)
	peak := &domain.HomeSkillPeakRow{RatingValue: 0}
	peak.BadgeImageURL = buildHomeSkillPeakBadgeURL("", "", 0, homeStaticTitleSlug, remaining)
	peak.MeasurementMatchesRemaining = &remaining
	return peak
}

// unrankedBadgeURL retourne l'URL du badge unranked_N.png.
// N = placementsCompleted (0-9) — total de 10 parties de placement pour CSR et LUSR.
func unrankedBadgeURL(placementsCompleted int, titleSlug string) *string {
	n := placementsCompleted
	if n < 0 {
		n = 0
	}
	if n > 9 {
		n = 9
	}
	slug := titleSlug
	if slug == "" {
		slug = homeStaticTitleSlug
	}
	url := static.URL(static.KindCSRRank, slug, fmt.Sprintf("unranked_%d", n), ".png")
	return &url
}

// buildHomeSkillPeakBadgeURL construit l'URL du badge de rang.
// titleSlug : slug de titre (ex "halo_infinite") pour les ratings game-specific (CSR).
// Passer "" pour les ratings cross-titre (LUSR) - l'URL n'inclut pas de slug.
// measurementMatchesRemaining > 0 → badge unranked_N.png (N = 10 - remaining, capped 0-9).
func buildHomeSkillPeakBadgeURL(tier string, tierLabel string, subTier int, titleSlug string, measurementMatchesRemaining int) *string {
	normalizedTier, normalizedSubTier := normalizeHomeSkillPeakBadgeParts(tier, tierLabel, subTier)
	if normalizedTier == "" {
		if measurementMatchesRemaining > 0 {
			completed := 10 - measurementMatchesRemaining
			return unrankedBadgeURL(completed, titleSlug)
		}
		return nil
	}
	// P5.4 (gap #9, ADR 0012) : déléguer à halo_infinite.AssetURLAdapter pour
	// le format `120px-HINF-CSR_*` (Halo-only). Évite la duplication du format.
	adapter := halo_infinite.NewAssetURLAdapter()
	var rawURL string
	if strings.EqualFold(normalizedTier, "Onyx") {
		rawURL = adapter.CSRRankImageURLOnyx()
	} else {
		if normalizedSubTier < 1 || normalizedSubTier > 6 {
			return nil
		}
		rawURL = adapter.CSRRankImageURL(normalizedTier, normalizedSubTier)
	}
	if rawURL == "" {
		return nil
	}
	// Quand titleSlug != adapter.TitleSlug() (LUSR cross-titre), recomposer
	// le path sans slug. L'adapter renvoie /static/ranks/<slug>/<id>.png.
	if titleSlug == "" {
		// Strip /static/ranks/halo_infinite/ → /static/ranks/
		prefix := static.MountPoint + "/" + static.Folder(static.KindCSRRank) + "/" + adapter.TitleSlug() + "/"
		if strings.HasPrefix(rawURL, prefix) {
			rawURL = path.Join(static.MountPoint, static.Folder(static.KindCSRRank), strings.TrimPrefix(rawURL, prefix))
		}
	}
	return &rawURL
}

func normalizeHomeSkillPeakBadgeParts(tier string, tierLabel string, subTier int) (string, int) {
	normalizedTier := canonicalHomeSkillTierName(tier)
	derivedSubTier := subTier
	if strings.TrimSpace(tierLabel) != "" {
		parts := strings.Fields(strings.TrimSpace(tierLabel))
		if normalizedTier == "" && len(parts) > 0 {
			normalizedTier = canonicalHomeSkillTierName(parts[0])
		}
		if derivedSubTier <= 0 && len(parts) > 1 {
			derivedSubTier = parseHomeSkillPeakSubTier(parts[len(parts)-1])
		}
	}
	return normalizedTier, derivedSubTier
}

func canonicalHomeSkillTierName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bronze":
		return tierBronze
	case "silver":
		return "Silver"
	case "gold":
		return tierGold
	case "platinum":
		return "Platinum"
	case "diamond":
		return tierDiamond
	case "onyx":
		return "Onyx"
	default:
		return ""
	}
}

func parseHomeSkillPeakSubTier(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	if numeric, err := strconv.Atoi(trimmed); err == nil {
		return numeric
	}
	switch strings.ToUpper(trimmed) {
	case "I":
		return 1
	case "II":
		return 2
	case "III":
		return 3
	case "IV":
		return 4
	case "V":
		return 5
	case "VI":
		return 6
	default:
		return 0
	}
}
