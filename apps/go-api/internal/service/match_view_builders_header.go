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
	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// ---------------------------------------------------------------------------
// Header
// ---------------------------------------------------------------------------

// buildMatchHeader assemble la section header de la Match View depuis meta, stats,
// enrich, scoreboard et assetURL.
//
// l'en-tête (cf. roadmap Q3 2026). Aujourd'hui non lu, signature stable.
//
//nolint:unparam // scoreboard maintenu pour future intégration MVP/LVP badges dans
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

	applyMatchHeaderMetaLabels(&h, meta, ctxkeys.Locale(ctx))
	applyMatchHeaderMapImage(ctx, &h, matchID, meta, assetURL)
	h.PlayableDurationSeconds = headerGameplayDurationSeconds(meta)
	h.IsRanked = meta.IsRanked
	// Lien vers la page publique du match (Waypoint pour Infinite). Via l'adapter
	// du titre (F3) : un titre sans page publique (H5) → "" → pas de lien mort.
	if meta.MapAssetID != nil && assetURL != nil {
		h.WaypointURL = assetURL.MatchWebURL(matchID)
	}

	applyMatchHeaderOutcome(&h, meta, stats)
	applyMatchHeaderEnrichment(&h, stats, enrich)

	return h
}

// headerGameplayDurationSeconds retourne la VRAIE durée de gameplay du match
// (countdown retranché) : duration_seconds − T0/1000 quand disponible, source la
// plus fiable. Fallback sur playable_duration_seconds (API) si duration absente.
func headerGameplayDurationSeconds(meta *domain.MatchMetaRaw) *int64 {
	if meta.DurationSeconds == nil {
		return meta.PlayableDurationSeconds
	}
	gp := int64(*meta.DurationSeconds)
	if meta.T0Ms != nil {
		gp -= *meta.T0Ms / 1000
	}
	if gp < 0 {
		gp = 0
	}
	return &gp
}

// applyMatchHeaderMetaLabels renseigne StartTime, MapUI, MapID, ModeUI, PlaylistLabel,
// LOCALE-AWARE (locale de la requête). Sous UI EN, le header servait des libellés FR
// (map/mode/playlist) composés avec un joint « on » côté front → « Assassin en équipe
// on Bazaar » (GH-9). EN = noms canoniques API (MapNameEN, pair_name EN, PlaylistName
// brut) ; FR = traductions (comportement historique préservé à l'octet).
func applyMatchHeaderMetaLabels(h *domain.MatchViewHeader, meta *domain.MatchMetaRaw, locale string) {
	isEN := locale == "en"
	h.StartTime = meta.StartTime
	if meta.StartTime != nil {
		h.StartTimeLabel = formatDateLong(*meta.StartTime, locale)
	}
	// Map : EN canonique (asset_translations en-US) sous UI EN, FR sinon ; fallback brut.
	if isEN && meta.MapNameEN != nil && *meta.MapNameEN != "" {
		h.MapUI = *meta.MapNameEN
	} else if !isEN && meta.MapNameFR != nil && *meta.MapNameFR != "" {
		h.MapUI = *meta.MapNameFR
	} else if meta.MapName != nil {
		h.MapUI = *meta.MapName
	}
	if meta.MapAssetID != nil {
		h.MapID = *meta.MapAssetID
	}
	// Mode : sous UI EN on dérive du pair_name EN (pas de traduction FR injectée) ;
	// sinon ModeNameFR pré-résolu par le repo (fallback ResolveModeUI(pair, pairFR)).
	var modeUI *string
	if isEN {
		modeUI = analysis.ResolveModeUI(meta.PairName, nil)
	} else {
		modeUI = meta.ModeNameFR
		if modeUI == nil || *modeUI == "" {
			modeUI = analysis.ResolveModeUI(meta.PairName, meta.PairNameFR)
		}
	}
	if modeUI != nil {
		// Re-normalise SYSTÉMATIQUEMENT : le libellé peut arriver brut avec un nom de
		// map collé (pair_name = "Slayer on Forest"). NormalizeModeLabel strippe le
		// suffixe " on/sur <map>" — sinon le front recompose "Slayer on Forest sur Forêt".
		h.ModeUI = analysis.NormalizeModeLabel(*modeUI, h.MapUI)
	}
	// Playlist : EN brut (PlaylistName) sous UI EN, FR (PlaylistNameFR) sinon.
	if isEN && meta.PlaylistName != nil && *meta.PlaylistName != "" {
		h.PlaylistLabel = *meta.PlaylistName
	} else if !isEN && meta.PlaylistNameFR != nil && *meta.PlaylistNameFR != "" {
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

// applyMatchHeaderReplay renseigne ReplayAvailable — la présence de l'artefact de rejeu
// 2D du match. Appliqué APRÈS le header, comme le flag « Prolongation » : le service
// porte le service de rejeu, pas le builder (buildMatchHeader est déjà à la limite de
// paramètres du dépôt).
//
// svc nil (titre sans rejeu, registry non câblée) → false, jamais d'erreur : le front
// n'affiche simplement pas de lien.
func applyMatchHeaderReplay(
	ctx context.Context, h *domain.MatchViewHeader, matchID string, svc port.ReplayService,
) {
	if h == nil || svc == nil || matchID == "" {
		return
	}
	h.ReplayAvailable = svc.IsAvailable(ctx, matchID)
}

// applyMatchHeaderOvertime renseigne IsOvertime / OvertimeSeconds — deuxième
// marqueur narratif du header, à côté du badge dominance (les deux coexistent).
//
// regulation = table `game_variant_name → temps réglementaire` du titre courant
// (regulation.toml, injectée au boot). Nil/vide, variante inconnue ou durée non
// estimable → aucun flag, PAS d'erreur : un titre sans table (Halo 5) n'affiche
// simplement jamais ce badge.
func applyMatchHeaderOvertime(h *domain.MatchViewHeader, meta *domain.MatchMetaRaw, regulation map[string]int) {
	if h == nil || meta == nil {
		return
	}
	isOvertime, seconds := analysis.ResolveOvertime(
		strDeref(meta.GameVariantName), meta.ElapsedSeconds, regulation)
	h.IsOvertime = isOvertime
	h.OvertimeSeconds = seconds
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

// resolveSkillIconURL retourne l'URL du badge CSR/LUSR depuis tier + sub_tier.
// Extrait de buildRankBlock pour être réutilisé par le scoreboard (tous joueurs).
func resolveSkillIconURL(tier *string, subTier *int, tierLabel *string, assetURL games.TitleAssetURLAdapter) string {
	if assetURL == nil || tier == nil || *tier == "" {
		return ""
	}
	t := *tier
	st := 0
	if subTier != nil {
		st = *subTier
	}
	if st <= 0 && tierLabel != nil {
		parts := strings.Fields(strings.TrimSpace(*tierLabel))
		if len(parts) > 1 {
			if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
				st = n
			}
		}
	}
	if strings.EqualFold(t, "Onyx") {
		return assetURL.CSRRankImageURLOnyx()
	}
	if st >= 1 && st <= 6 {
		return assetURL.CSRRankImageURL(t, st)
	}
	return ""
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
	rank.DeltaValue = sr.RatingDelta

	// Placement : valeur numérique stockée à 0 (contrainte NOT NULL player DB)
	// mais sans signification — on n'envoie ni NumericVal ni ProgressPct pour
	// éviter "CSR 0" et la barre à 0% dans le header match view.
	isPlacement := sr.Tier != nil && strings.EqualFold(*sr.Tier, "Placement")
	if !isPlacement {
		rank.NumericVal = sr.RatingValue
	}

	// ProgressPct : position dans le sous-tier (0.0–1.0).
	// CSR : sous-paliers de 50 pts (échelle CSR propre). LUSR : rating_value continu
	// → position via la grille legacy (sous-paliers à largeurs variables ; l'ancien
	// mod-50 était faux pour LUSR). Onyx et Placement : nil.
	if !isPlacement && sr.RatingValue != nil && sr.Tier != nil && !strings.EqualFold(*sr.Tier, "Onyx") {
		if strings.EqualFold(sr.RatingType, "CSR") {
			const tierSize = 50.0
			pts := math.Mod(*sr.RatingValue, tierSize)
			if pts < 0 {
				pts += tierSize
			}
			pct := pts / tierSize
			rank.ProgressPct = &pct
		} else if pct, ok := skillv2.LegacyContinuousSubTierProgress(*sr.RatingValue); ok {
			rank.ProgressPct = &pct
		}
	}

	// Badge image — délégué au helper partagé avec le scoreboard.
	rank.IconURL = resolveSkillIconURL(sr.Tier, sr.SubTier, sr.TierLabel, assetURL)
	return rank
}
