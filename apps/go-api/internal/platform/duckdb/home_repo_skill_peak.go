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
// Phase 6 : threshold paramétré (CSR=lookup season ou default, LUSR=10).
func (r *HomeRepo) assemblePeak(playerRows []peakRow, registryByMatch map[string]peakRegistryInfo, ratingType string) *domain.HomeSkillPeakRow {
	want := strings.ToUpper(strings.TrimSpace(ratingType))
	// LUSR garde son seuil interne de 10 (algorithme local). CSR utilise la
	// saison courante du HomeRepo (configurée via WithCSRThresholds) ou le
	// default S3+ (=5) si non câblé.
	threshold := 10
	if want == "CSR" {
		threshold = r.csrThreshold(r.currentCSRSID)
	}

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

	var chosen *groupBest
	for _, gb := range byGroup {
		switch {
		case chosen == nil:
			chosen = gb
		case gb.matchCount >= threshold && chosen.matchCount < threshold:
			chosen = gb
		case (gb.matchCount >= threshold) == (chosen.matchCount >= threshold):
			if gb.row.ratingValue > chosen.row.ratingValue {
				chosen = gb
			}
		}
	}
	remaining := threshold - chosen.matchCount
	if remaining < 0 {
		remaining = 0
	}

	peak := &domain.HomeSkillPeakRow{RatingValue: chosen.row.ratingValue}
	totalCopy := threshold
	peak.PlacementTotal = &totalCopy
	if remaining > 0 {
		peak.BadgeImageURL = buildHomeSkillPeakBadgeURLForThreshold("", "", 0, homeStaticTitleSlug, remaining, threshold)
		remCopy := remaining
		peak.MeasurementMatchesRemaining = &remCopy
		return peak
	}
	peak.BadgeImageURL = buildHomeSkillPeakBadgeURLForThreshold(chosen.row.tier, chosen.row.tierLabel, chosen.row.subTier, homeStaticTitleSlug, 0, threshold)
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
			return "CSR"
		}
		return "LUSR"
	}
	rt := strings.ToUpper(strings.TrimSpace(pr.ratingType))
	if rt == "CSR" {
		return "CSR"
	}
	return "LUSR"
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
	// Phase 6 : threshold de la saison courante pour les calculs CSR.
	threshold := r.csrThreshold(r.currentCSRSID)

	var value sql.NullFloat64
	var tier sql.NullString
	var subTier sql.NullInt16
	if err := r.pdb.ReadDB().QueryRow(ctx, Q26csrAlltimePeak).Scan(&value, &tier, &subTier); err == nil && value.Valid {
		peak := &domain.HomeSkillPeakRow{RatingValue: value.Float64}
		tierStr := optionalNullStringValue(tier)
		subTierInt := optionalNullInt16Value(subTier)
		peak.BadgeImageURL = buildHomeSkillPeakBadgeURLForThreshold(tierStr, "", subTierInt, homeStaticTitleSlug, 0, threshold)
		if tierStr != "" {
			peak.TierLabel = stringPtr(tierStr)
		}
		zero := 0
		peak.MeasurementMatchesRemaining = &zero
		totalCopy := threshold
		peak.PlacementTotal = &totalCopy
		return peak
	}

	// Pas d'alltime : tenter de récupérer l'état de placement le plus avancé.
	// MIN(current_measurement_remaining) = playlist la plus proche de la fin
	// du placement (threshold → 0). Si pas de snapshot du tout : retourner nil
	// pour laisser Q26e fallback prendre la suite.
	//
	// NOTE : à terme, on pourrait lire le season_id ici aussi pour appliquer le
	// threshold de cette saison-là spécifiquement. Pour l'instant on prend le
	// threshold de la saison courante — acceptable dans 99% des cas (joueur en
	// placement = saison récente).
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
	peak.BadgeImageURL = buildHomeSkillPeakBadgeURLForThreshold("", "", 0, homeStaticTitleSlug, remaining, threshold)
	peak.MeasurementMatchesRemaining = &remaining
	totalCopy := threshold
	peak.PlacementTotal = &totalCopy
	return peak
}

// unrankedBadgeURL retourne l'URL du badge unranked_N.png pour un placement à
// seuil 10 (compat historique). Préfèrer unrankedBadgeURLForThreshold pour les
// nouveaux callers qui veulent supporter le seuil dynamique par saison.
func unrankedBadgeURL(placementsCompleted int, titleSlug string) *string {
	return unrankedBadgeURLForThreshold(placementsCompleted, 10, titleSlug)
}

// unrankedBadgeURLForThreshold retourne l'URL du badge unranked en mappant
// proportionnellement la progression sur les 10 images disponibles
// (unranked_0.png .. unranked_9.png).
//
// Phase 6 du plan pipeline CSR : depuis Season 3 (2023-03-07) Halo utilise
// un seuil 5 au lieu de 10. On recycle les images existantes via un mapping
// régulier :
//
//	threshold=10 : completed * 10 / 10 = identité (0,1,2,3,4,5,6,7,8,9)
//	threshold=5  : completed * 10 / 5  = 0,2,4,6,8 (5 images utilisées)
//
// N est ensuite clampé [0, 9] pour les bornes (completed négatif ou ≥ threshold).
func unrankedBadgeURLForThreshold(placementsCompleted, threshold int, titleSlug string) *string {
	if threshold <= 0 {
		threshold = 10 // garde-fou
	}
	// Mapping proportionnel : completed * 10 / threshold.
	n := (placementsCompleted * 10) / threshold
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

// buildHomeSkillPeakBadgeURL construit l'URL du badge de rang (compat seuil 10).
// Wrapper de buildHomeSkillPeakBadgeURLForThreshold. Préfèrer la version
// "ForThreshold" pour les nouveaux callers conscients du seuil dynamique.
func buildHomeSkillPeakBadgeURL(tier string, tierLabel string, subTier int, titleSlug string, measurementMatchesRemaining int) *string {
	return buildHomeSkillPeakBadgeURLForThreshold(tier, tierLabel, subTier, titleSlug, measurementMatchesRemaining, 10)
}

// buildHomeSkillPeakBadgeURLForThreshold construit l'URL du badge avec seuil
// dynamique pour le calcul de l'image placement. Phase 6 du plan pipeline CSR.
func buildHomeSkillPeakBadgeURLForThreshold(tier string, tierLabel string, subTier int, titleSlug string, measurementMatchesRemaining, threshold int) *string {
	normalizedTier, normalizedSubTier := normalizeHomeSkillPeakBadgeParts(tier, tierLabel, subTier)
	if normalizedTier == "" {
		if measurementMatchesRemaining > 0 {
			if threshold <= 0 {
				threshold = 10
			}
			completed := threshold - measurementMatchesRemaining
			return unrankedBadgeURLForThreshold(completed, threshold, titleSlug)
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
		return "Bronze"
	case "silver":
		return "Silver"
	case "gold":
		return "Gold"
	case "platinum":
		return "Platinum"
	case "diamond":
		return "Diamond"
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
