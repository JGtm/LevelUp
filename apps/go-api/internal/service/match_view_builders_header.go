// Package service — builders pour le header + rank de la Match View.
//
// Extrait de match_view_service.go (audit #1 god files).
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// ---------------------------------------------------------------------------
// Header
// ---------------------------------------------------------------------------

func buildMatchHeader(
	ctx context.Context,
	matchID string,
	meta *domain.MatchMetaRaw,
	stats *domain.PlayerMatchStatsRaw,
	enrich *domain.MatchEnrichmentRaw,
	scoreboard []domain.ScoreboardRaw,
	assetURL games.TitleAssetURLAdapter,
	isFavorite bool,
) domain.MatchViewHeader {
	h := domain.MatchViewHeader{
		MatchID:      matchID,
		OutcomeLabel: "-",
		OutcomeColor: mvHexOutcomeUnknown,
		PerfDisplay:  "-",
		IsFavorite:   isFavorite,
	}

	if meta == nil {
		return h
	}

	applyMatchHeaderMetaLabels(&h, meta)
	applyMatchHeaderMapImage(ctx, &h, matchID, meta, assetURL)
	h.PlayableDurationSeconds = meta.PlayableDurationSeconds
	h.IsRanked = meta.IsRanked
	if meta.MapAssetID != nil {
		h.WaypointURL = fmt.Sprintf("https://www.halowaypoint.com/halo-infinite/matches/%s", matchID)
	}

	applyMatchHeaderOutcome(&h, meta, stats)
	applyMatchHeaderEnrichment(&h, stats, enrich)

	return h
}

// applyMatchHeaderMetaLabels renseigne StartTime, MapUI, MapID, ModeUI, PlaylistLabel.
func applyMatchHeaderMetaLabels(h *domain.MatchViewHeader, meta *domain.MatchMetaRaw) {
	h.StartTime = meta.StartTime
	if meta.StartTime != nil {
		h.StartTimeLabel = formatDateFRLong(*meta.StartTime)
	}
	if meta.MapNameFR != nil && *meta.MapNameFR != "" {
		h.MapUI = *meta.MapNameFR
	} else if meta.MapName != nil {
		h.MapUI = *meta.MapName
	}
	if meta.MapAssetID != nil {
		h.MapID = *meta.MapAssetID
	}
	// ModeNameFR est normalement déjà le résultat de analysis.ResolveModeUI
	// côté repo. Fallback défense-en-profondeur via le même helper si jamais
	// un caller externe construit un MatchMetaRaw sans pré-résoudre.
	modeUI := meta.ModeNameFR
	if modeUI == nil || *modeUI == "" {
		modeUI = analysis.ResolveModeUI(meta.PairName, meta.PairNameFR)
	}
	if modeUI != nil {
		h.ModeUI = *modeUI
	}
	if meta.PlaylistNameFR != nil && *meta.PlaylistNameFR != "" {
		h.PlaylistLabel = *meta.PlaylistNameFR
	} else if meta.PlaylistName != nil {
		h.PlaylistLabel = *meta.PlaylistName
	}
}

// applyMatchHeaderMapImage applique la cascade MapImageURL :
//  1. map_images_registry (meta.MapImageURL).
//  2. AssetURLAdapter avec nom EN résolu (asset_translations en-US).
//  3. AssetURLAdapter avec map_name brut (legacy).
func applyMatchHeaderMapImage(
	ctx context.Context, h *domain.MatchViewHeader, matchID string,
	meta *domain.MatchMetaRaw, assetURL games.TitleAssetURLAdapter,
) {
	if meta.MapImageURL != nil && *meta.MapImageURL != "" {
		h.MapImageURL = meta.MapImageURL
		return
	}
	if assetURL == nil {
		return
	}
	nameForAdapter := ""
	if meta.MapNameEN != nil && *meta.MapNameEN != "" {
		nameForAdapter = *meta.MapNameEN
	} else if meta.MapName != nil && *meta.MapName != "" {
		nameForAdapter = *meta.MapName
	}
	if nameForAdapter == "" {
		return
	}
	if url := assetURL.MapImageURL(nameForAdapter); url != "" {
		h.MapImageURL = &url
		return
	}
	slog.WarnContext(ctx, "match_header: map image missing",
		"match_id", matchID,
		"map_name_used", nameForAdapter,
		"map_name_raw", strDeref(meta.MapName),
		"map_name_en", strDeref(meta.MapNameEN))
}

// applyMatchHeaderOutcome remplit OutcomeCode/Label/Color + ScoreLabel.
func applyMatchHeaderOutcome(h *domain.MatchViewHeader, meta *domain.MatchMetaRaw, stats *domain.PlayerMatchStatsRaw) {
	if stats == nil || stats.OutcomeCode == 0 {
		return
	}
	code := stats.OutcomeCode
	h.OutcomeCode = &code
	h.OutcomeLabel = outcomeLabel(code)
	h.OutcomeColor = outcomeColor(code)
	h.OutcomeColorToken = outcomeColorToken(code)
	h.ScoreLabel = buildScoreLabelFromMeta(meta, stats)
}

// applyMatchHeaderEnrichment renseigne PerfDisplay/Color, IsExcluded, DominanceFlag/Badge.
func applyMatchHeaderEnrichment(
	h *domain.MatchViewHeader, stats *domain.PlayerMatchStatsRaw, enrich *domain.MatchEnrichmentRaw,
) {
	if enrich == nil {
		return
	}
	isDNF := stats != nil && stats.OutcomeCode == 4
	if enrich.PerformanceScore != nil && !isDNF {
		perf := *enrich.PerformanceScore
		display := fmt.Sprintf("%.0f", perf)
		h.PerfDisplay = display
		color := perfColor(perf)
		h.PerfColor = &color
		h.PerfColorToken = perfColorToken(perf)
	}
	h.IsExcluded = enrich.IsExcluded

	flag := canonical.DominanceFlag(enrich.DominanceFlag)
	if badge := narrative.ResolveDominanceBadge(flag); badge != nil {
		h.DominanceFlag = true
		h.DominanceBadge = &domain.MatchViewDominanceBadge{
			Flag:       int(badge.Flag),
			LabelKey:   badge.LabelKey,
			ColorToken: badge.ColorToken,
		}
	}
}

// buildScoreLabelFromMeta construit "X-Y" depuis team_0_score/team_1_score de
// match_registry. L'équipe du joueur (stats.TeamID) est toujours affichée en
// premier (miroir de buildHomeScoreLabel dans analysis/home.go).
func buildScoreLabelFromMeta(meta *domain.MatchMetaRaw, stats *domain.PlayerMatchStatsRaw) string {
	if meta == nil || meta.Team0Score == nil || meta.Team1Score == nil {
		return ""
	}
	s0, s1 := int(*meta.Team0Score), int(*meta.Team1Score)
	if s0 < 0 || s1 < 0 {
		return ""
	}
	if stats != nil && stats.TeamID != nil && *stats.TeamID == 1 {
		return fmt.Sprintf("%d-%d", s1, s0)
	}
	return fmt.Sprintf("%d-%d", s0, s1)
}

// buildRankBlock construit le bloc rank depuis SkillRankRaw.
//
// IconURL est résolu via TitleAssetURLAdapter (CSRRankImageURL ou
// CSRRankImageURLOnyx selon le tier). Dégradation gracieuse :
//   - assetURL nil → IconURL = "" (front affiche fallback texte)
//   - rating_type CSR mais tier inconnu → IconURL = ""
//   - rating_type LUSR (custom games) → pas de badge officiel, IconURL = ""
func buildRankBlock(sr *domain.SkillRankRaw, assetURL games.TitleAssetURLAdapter) domain.MatchViewRank {
	if sr == nil {
		return domain.MatchViewRank{RatingType: "none"}
	}
	rank := domain.MatchViewRank{RatingType: sr.RatingType}
	if sr.TierLabel != nil {
		rank.TierLabel = sr.TierLabel
	}
	rank.NumericVal = sr.RatingValue
	rank.DeltaValue = sr.RatingDelta

	// ProgressPct : position dans le sous-tier (0.0–1.0).
	// CSR et LUSR Halo Infinite ont tous les deux des sous-tiers de 50 points.
	// Même constante que home_canonical.go (tierSize = 50).
	// Onyx : nil (pas de tier suivant défini).
	if sr.RatingValue != nil && sr.Tier != nil && !strings.EqualFold(*sr.Tier, "Onyx") {
		const tierSize = 50.0
		pts := math.Mod(*sr.RatingValue, tierSize)
		if pts < 0 {
			pts += tierSize
		}
		pct := pts / tierSize
		rank.ProgressPct = &pct
	}

	// Badge image — LUSR utilise les mêmes fichiers que CSR (même dossier static).
	// Onyx : pas de sub-tier → CSRRankImageURLOnyx().
	// Autres tiers (Bronze, Silver, Gold, Platinum, Diamond) : tier + sub-tier.
	// Sources : match_skill_rank.tier (EN, TitleCase) + match_skill_rank.sub_tier.
	if assetURL == nil || sr.Tier == nil || *sr.Tier == "" {
		return rank
	}
	tier := *sr.Tier
	subTier := 0
	if sr.SubTier != nil {
		subTier = *sr.SubTier
	}
	// Fallback : dériver sub_tier depuis tier_label quand sub_tier = 0 (défaut DB).
	if subTier <= 0 && sr.TierLabel != nil {
		parts := strings.Fields(strings.TrimSpace(*sr.TierLabel))
		if len(parts) > 1 {
			if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
				subTier = n
			}
		}
	}
	if strings.EqualFold(tier, "Onyx") {
		rank.IconURL = assetURL.CSRRankImageURLOnyx()
	} else if subTier >= 1 && subTier <= 6 {
		rank.IconURL = assetURL.CSRRankImageURL(tier, subTier)
	}
	return rank
}
