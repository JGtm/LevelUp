// Package analysis — home_canonical.go : entry-points canonical-aware pour la
// page Home (P4.3b, ADR 0011).
//
// **Stratégie pragmatique** : chaque `*FromCanonical` convertit canonical →
// `domain.HomeMatchRow` puis délègue à la version legacy. Ainsi :
//
//   - Le converter `HomeMatchRowFromCanonical` vit DANS le package analysis
//     (encapsulé) et n'est plus visible côté `service/home_service.go`.
//   - Le service consomme uniquement `analysis.BuildHeroCardFromCanonical(...)`,
//     plus de conversion à son niveau.
//   - La logique métier (ComputeKPIs, BuildHighlights, etc.) reste UNE source
//     de vérité côté legacy. Pas de duplication, pas de risque de drift.
//
// **TODO P4.3 finale** : porter les internals (ComputeKPIs, BuildHighlights,
// etc.) à canonical et retirer ces wrappers + les types legacy
// `domain.HomeMatchRow` / `domain.HomeSessionRow`. Bloqué tant que le repo
// `port.HomeRepository.LoadHomeMatches` retourne du legacy + tant que des
// callers parallèles (squad/teammates) consomment encore des types legacy.
package analysis

import (
	"strconv"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// =============================================================================
// Converters canonical → legacy (encapsulés)
// =============================================================================

// HomeMatchRowFromCanonical convertit canonical.PlayerMatchRow → HomeMatchRow.
//
// Mapping selon ADR 0011 :
//   - Données brutes (kills/deaths/MMR/Outcome/scores équipes/SkillSnapshot
//     data fields) : depuis canonical.
//   - Labels FR map/playlist/game_variant : via AssetReference.Labels["fr"]
//     avec fallback DefaultLabel.
//   - SkillTierLabel / SkillRankImageURL / PairName : laissés vides
//     (TitleSemanticAdapter / TitleAssetURLAdapter / composite Halo-only —
//     enrichissement P4.3 finale).
func HomeMatchRowFromCanonical(r canonical.PlayerMatchRow) domain.HomeMatchRow {
	out := domain.HomeMatchRow{
		MatchID:          r.Summary.MatchID,
		StartTime:        r.Summary.StartedAtUTC,
		SessionLabel:     r.Enrichment.SessionLabel,
		IsWithFriends:    r.Enrichment.IsWithFriends,
		Kills:            derefIntZero(r.Self.Kills),
		Deaths:           derefIntZero(r.Self.Deaths),
		Assists:          derefIntZero(r.Self.Assists),
		KDA:              r.Self.KDA,
		Accuracy:         r.Self.Accuracy,
		AvgLifeSeconds:   r.Self.AvgLifeSeconds,
		TimePlayedSecs:   r.Self.TimePlayed,
		TeamMMR:          r.Enrichment.TeamMMR,
		EnemyMMR:         r.Enrichment.EnemyMMR,
		PerformanceScore: r.Enrichment.PerformanceScore,
		HeadshotKills:    derefIntZero(r.Self.HeadshotKills),
		PerfectKills:     derefIntZero(r.Self.PerfectKills),
		MaxKillingSpree:  r.Self.MaxKillingSpree,
		DominanceFlag:    int(r.Enrichment.DominanceFlag),
	}
	if r.Self.TeamID != nil {
		out.TeamID = *r.Self.TeamID
	}
	if r.Self.RankInMatch != nil {
		v := *r.Self.RankInMatch
		out.RankInTeam = &v
	}
	if r.Self.DamageDealt != nil {
		v := float64(*r.Self.DamageDealt)
		out.DamageDealt = &v
	}
	if r.Self.DamageTaken != nil {
		v := float64(*r.Self.DamageTaken)
		out.DamageTaken = &v
	}
	if r.Summary.Map != nil {
		out.MapID = r.Summary.Map.ID
		out.MapName = r.Summary.Map.DefaultLabel
		if v, ok := r.Summary.Map.Labels["fr"]; ok && v != "" {
			out.MapNameFR = v
		} else {
			out.MapNameFR = r.Summary.Map.DefaultLabel
		}
	}
	if r.Summary.Playlist != nil {
		out.PlaylistID = r.Summary.Playlist.ID
		out.PlaylistName = r.Summary.Playlist.DefaultLabel
		if v, ok := r.Summary.Playlist.Labels["fr"]; ok && v != "" {
			out.PlaylistNameFR = v
		} else {
			out.PlaylistNameFR = r.Summary.Playlist.DefaultLabel
		}
	}
	if r.Summary.GameVariant != nil {
		out.GameVariantID = r.Summary.GameVariant.ID
		out.GameVariantName = r.Summary.GameVariant.DefaultLabel
		if v, ok := r.Summary.GameVariant.Labels["fr"]; ok && v != "" {
			out.GameVariantNameFR = v
		} else {
			out.GameVariantNameFR = r.Summary.GameVariant.DefaultLabel
		}
	}
	if r.Summary.IsRanked != nil {
		out.IsRanked = *r.Summary.IsRanked
	}
	if r.Summary.IsPvE != nil {
		out.IsFirefight = *r.Summary.IsPvE
	}
	switch r.Self.Outcome {
	case canonical.OutcomeWin:
		out.Outcome = domain.OutcomeWin
	case canonical.OutcomeLoss:
		out.Outcome = domain.OutcomeLoss
	case canonical.OutcomeTie:
		out.Outcome = domain.OutcomeDraw
	case canonical.OutcomeDNF:
		out.Outcome = domain.OutcomeDNF
	}
	for _, t := range r.Summary.Teams {
		if t.Score == nil {
			continue
		}
		switch t.TeamID {
		case 0:
			out.Team0Score = *t.Score
		case 1:
			out.Team1Score = *t.Score
		}
	}
	if r.Enrichment.SkillSnapshot != nil {
		out.SkillRatingValue = r.Enrichment.SkillSnapshot.RatingValue
		out.SkillRatingType = string(r.Enrichment.SkillSnapshot.RatingType)
		out.SkillTier = r.Enrichment.SkillSnapshot.TierCode
		if r.Enrichment.SkillSnapshot.SubTier != nil {
			out.SkillSubTier = *r.Enrichment.SkillSnapshot.SubTier
		}
		out.SkillRatingDelta = r.Enrichment.SkillSnapshot.Delta
		out.SkillPlaylistGroup = r.Enrichment.SkillSnapshot.PlaylistGroup
	}
	if r.Self.Deaths != nil && *r.Self.Deaths > 0 && r.Self.Kills != nil {
		v := float64(*r.Self.Kills) / float64(*r.Self.Deaths)
		out.Ratio = &v
	}
	return out
}

// HomeMatchRowsFromCanonical : version slice.
func HomeMatchRowsFromCanonical(rows []canonical.PlayerMatchRow) []domain.HomeMatchRow {
	out := make([]domain.HomeMatchRow, len(rows))
	for i, r := range rows {
		out[i] = HomeMatchRowFromCanonical(r)
	}
	return out
}

// HomeSessionsFromCanonical dérive []HomeSessionRow depuis canonical.
// SessionID : canonical *string → home *int via strconv.
func HomeSessionsFromCanonical(rows []canonical.PlayerMatchRow) []domain.HomeSessionRow {
	out := make([]domain.HomeSessionRow, 0, len(rows))
	for _, r := range rows {
		entry := domain.HomeSessionRow{
			MatchID:       r.Summary.MatchID,
			SessionLabel:  r.Enrichment.SessionLabel,
			IsWithFriends: r.Enrichment.IsWithFriends,
		}
		if r.Enrichment.SessionID != nil {
			if id, err := strconv.Atoi(*r.Enrichment.SessionID); err == nil {
				entry.SessionID = &id
			}
		}
		t := r.Summary.StartedAtUTC
		entry.StartTime = &t
		out = append(out, entry)
	}
	return out
}

// derefIntZero retourne *p ou 0 si p est nil. Utilitaire interne.
func derefIntZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// =============================================================================
// Entry-points canonical-aware (P4.3b)
// =============================================================================

// BuildHeroCardFromCanonical est la variante canonical-aware de BuildHeroCard.
// TODO P4.3 finale : porter ComputeKPIs et ComputeTrend à canonical pour
// retirer la conversion intermédiaire.
func BuildHeroCardFromCanonical(rows []canonical.PlayerMatchRow, gamertag string, totalMatches int) domain.HomeHeroCard {
	return BuildHeroCard(HomeMatchRowsFromCanonical(rows), gamertag, totalMatches)
}

// BuildHighlightsFromCanonical est la variante canonical-aware de BuildHighlights.
// TODO P4.3 finale : porter les helpers (selectHighlightWindow,
// buildMaitriseHighlight, etc.) à canonical pour retirer la conversion.
func BuildHighlightsFromCanonical(rows []canonical.PlayerMatchRow) []domain.HighlightItem {
	return BuildHighlights(HomeMatchRowsFromCanonical(rows))
}

// BuildRecentMatchesWithFavoritesFromCanonical est la variante canonical-aware
// de BuildRecentMatchesWithFavoritesForLocale.
// TODO P4.3 finale : porter les conversions de mode/map/outcome label à
// canonical (consommer Summary.Map.Labels et Summary.Playlist.Labels
// directement plutôt que via HomeMatchRow.MapNameFR/PlaylistNameFR).
func BuildRecentMatchesWithFavoritesFromCanonical(
	rows []canonical.PlayerMatchRow,
	limit int,
	favoriteIDs map[string]bool,
	locale string,
) []domain.RecentMatchItem {
	return BuildRecentMatchesWithFavoritesForLocale(
		HomeMatchRowsFromCanonical(rows), limit, favoriteIDs, locale,
	)
}

// BuildSessionSummaryFromCanonical est la variante canonical-aware de BuildSessionSummary.
// Sessions sont dérivées des mêmes rows canonical (HomeSessionRow = projection
// matches.{MatchID, Enrichment.SessionID, SessionLabel, IsWithFriends, StartTime}).
// TODO P4.3 finale : porter BuildSessionSummary à canonical (consommer
// Enrichment.SessionLabel/IsWithFriends + Summary.StartedAtUTC sans
// HomeMatchRow/HomeSessionRow intermédiaires).
func BuildSessionSummaryFromCanonical(rows []canonical.PlayerMatchRow, isSquad bool) *domain.SessionSummaryItem {
	return BuildSessionSummary(
		HomeMatchRowsFromCanonical(rows),
		HomeSessionsFromCanonical(rows),
		isSquad,
	)
}

// BuildSessionSummariesFromCanonical est la variante canonical-aware de
// BuildSessionSummaries (liste des N dernières sessions solo ou squad).
// TODO P4.3 finale : porter BuildSessionSummaries à canonical.
func BuildSessionSummariesFromCanonical(rows []canonical.PlayerMatchRow, isSquad bool, limit int) []domain.SessionSummaryItem {
	return BuildSessionSummaries(
		HomeMatchRowsFromCanonical(rows),
		HomeSessionsFromCanonical(rows),
		isSquad,
		limit,
	)
}

// InferHomeSkillHistoryFromCanonical est la variante canonical-aware de l'helper
// privé inferHomeSkillHistory du service home. Retourne (hasRanked, hasUnranked).
// PvE matchs sont exclus (Summary.IsPvE).
func InferHomeSkillHistoryFromCanonical(rows []canonical.PlayerMatchRow) (bool, bool) {
	hasRanked := false
	hasUnranked := false
	for _, r := range rows {
		if r.Summary.IsPvE != nil && *r.Summary.IsPvE {
			continue
		}
		isRanked := false
		if r.Summary.IsRanked != nil {
			isRanked = *r.Summary.IsRanked
		}
		if isRanked {
			hasRanked = true
		} else {
			hasUnranked = true
		}
		if hasRanked && hasUnranked {
			break
		}
	}
	return hasRanked, hasUnranked
}
