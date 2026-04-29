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
	"sort"
	"strconv"
	"time"

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
// Entry-points canonical-aware (P4.3b → P4.3 finale)
// =============================================================================

// ComputeKPIsFromCanonical est la variante canonical-aware de ComputeKPIs
// (P4.3 finale). Logique strictement identique ; lit depuis Self/Enrichment.
func ComputeKPIsFromCanonical(rows []canonical.PlayerMatchRow, totalMatches int) domain.HeroKPIs {
	if len(rows) == 0 {
		return domain.HeroKPIs{}
	}
	var wins, losses, draws, dnfs int
	var ratioSum, ratioCount float64
	var kdaSum, kdaCount float64
	var accSum, accCount float64
	var totalPlaytime int
	var offSum, offCount float64
	var defSum, defCount float64
	playlistCounts := make(map[string]int)
	playlistNames := make(map[string]string)

	for _, r := range rows {
		switch r.Self.Outcome {
		case canonical.OutcomeWin:
			wins++
		case canonical.OutcomeLoss:
			losses++
		case canonical.OutcomeTie:
			draws++
		case canonical.OutcomeDNF:
			dnfs++
		}
		if r.Self.Deaths != nil && *r.Self.Deaths > 0 && r.Self.Kills != nil {
			ratioSum += float64(*r.Self.Kills) / float64(*r.Self.Deaths)
			ratioCount++
		}
		if r.Self.KDA != nil {
			kdaSum += *r.Self.KDA
			kdaCount++
		}
		if r.Self.Accuracy != nil {
			accSum += *r.Self.Accuracy
			accCount++
		}
		if r.Self.TimePlayed != nil {
			totalPlaytime += *r.Self.TimePlayed
		}
		if r.Self.DamageDealt != nil && r.Self.DamageTaken != nil {
			k := derefIntZero(r.Self.Kills)
			a := derefIntZero(r.Self.Assists)
			d := derefIntZero(r.Self.Deaths)
			cy := ComputeCombatYield(k, a, float64(*r.Self.DamageDealt), float64(*r.Self.DamageTaken), d)
			if cy.OffensiveConversion > 0 {
				offSum += cy.OffensiveConversion
				offCount++
			}
			if cy.DefensiveResistance > 0 {
				defSum += cy.DefensiveResistance
				defCount++
			}
		}
		if r.Summary.Playlist != nil {
			id := r.Summary.Playlist.ID
			name := r.Summary.Playlist.DefaultLabel
			if v, ok := r.Summary.Playlist.Labels["fr"]; ok && v != "" {
				name = v
			}
			if name != "" && !homeUUIDRe.MatchString(name) && id != "" {
				playlistCounts[id]++
				if _, seen := playlistNames[id]; !seen {
					playlistNames[id] = name
				}
			}
		}
	}

	total := len(rows)
	kpis := domain.HeroKPIs{
		WinRate:           WinRate(wins, total),
		TotalMatches:      totalMatches,
		Wins:              wins,
		Draws:             draws,
		DNFs:              dnfs,
		Losses:            losses,
		TotalPlaytimeSecs: totalPlaytime,
	}
	if ratioCount > 0 {
		v := round2(ratioSum / ratioCount)
		kpis.GlobalRatio = &v
	}
	if kdaCount > 0 {
		v := round2(kdaSum / kdaCount)
		kpis.AvgKDA = &v
	}
	if accCount > 0 {
		v := round1(accSum / accCount)
		kpis.AvgAccuracy = &v
	}
	if offCount > 0 {
		v := round2(offSum / offCount)
		kpis.AvgOffensiveConversion = &v
	}
	if defCount > 0 {
		v := round2(defSum / defCount)
		kpis.AvgDefensiveResistance = &v
	}
	if bestID, bestCount := dominantKey(playlistCounts); bestID != "" {
		kpis.FavoritePlaylistName = playlistNames[bestID]
		kpis.FavoritePlaylistCount = bestCount
	}
	return kpis
}

// ComputeTrendFromCanonical est la variante canonical-aware de ComputeTrend.
func ComputeTrendFromCanonical(rows []canonical.PlayerMatchRow, window int) *domain.HeroTrend {
	if len(rows) < window+1 {
		return nil
	}
	recent := rows[:window]
	prev := rows[window : window+window]
	if len(prev) == 0 {
		return nil
	}
	trend := &domain.HeroTrend{}
	rCurr, rPrev := meanRatioCanonical(recent), meanRatioCanonical(prev)
	if rCurr != nil && rPrev != nil {
		v := round3(*rCurr - *rPrev)
		trend.RatioDelta = &v
	}
	aCurr, aPrev := meanAccuracyCanonical(recent), meanAccuracyCanonical(prev)
	if aCurr != nil && aPrev != nil {
		v := round2(*aCurr - *aPrev)
		trend.AccuracyDelta = &v
	}
	wrCurr := winRateCanonical(recent)
	wrPrev := winRateCanonical(prev)
	v := round4(wrCurr - wrPrev)
	trend.WinRateDelta = &v
	return trend
}

func meanRatioCanonical(rows []canonical.PlayerMatchRow) *float64 {
	var sum float64
	var count int
	for _, r := range rows {
		if r.Self.Deaths != nil && *r.Self.Deaths > 0 && r.Self.Kills != nil {
			sum += float64(*r.Self.Kills) / float64(*r.Self.Deaths)
			count++
		}
	}
	if count == 0 {
		return nil
	}
	v := sum / float64(count)
	return &v
}

func meanAccuracyCanonical(rows []canonical.PlayerMatchRow) *float64 {
	var sum float64
	var count int
	for _, r := range rows {
		if r.Self.Accuracy != nil {
			sum += *r.Self.Accuracy
			count++
		}
	}
	if count == 0 {
		return nil
	}
	v := sum / float64(count)
	return &v
}

func winRateCanonical(rows []canonical.PlayerMatchRow) float64 {
	if len(rows) == 0 {
		return 0
	}
	var wins, total int
	for _, r := range rows {
		if r.Self.Outcome == canonical.OutcomeWin || r.Self.Outcome == canonical.OutcomeLoss {
			total++
			if r.Self.Outcome == canonical.OutcomeWin {
				wins++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(wins) / float64(total)
}

// BuildHeroCardFromCanonical : entièrement canonical (P4.3 finale).
func BuildHeroCardFromCanonical(rows []canonical.PlayerMatchRow, gamertag string, totalMatches int) domain.HomeHeroCard {
	kpis := ComputeKPIsFromCanonical(rows, totalMatches)
	trend := ComputeTrendFromCanonical(rows, 5)
	return domain.HomeHeroCard{PlayerName: gamertag, KPIs: kpis, Trend: trend}
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

// BuildSessionSummaryFromCanonical : full canonical (P4.3 finale).
// Filtre par IsWithFriends (squadMode), trouve la session la plus récente
// par StartedAtUTC, agrège ses matchs en KPIs.
func BuildSessionSummaryFromCanonical(rows []canonical.PlayerMatchRow, squadMode bool) *domain.SessionSummaryItem {
	if len(rows) == 0 {
		return nil
	}

	// Filtrer les rows par squadMode + label non-nil.
	var filtered []canonical.PlayerMatchRow
	for _, r := range rows {
		if r.Enrichment.IsWithFriends == squadMode && r.Enrichment.SessionLabel != nil && *r.Enrichment.SessionLabel != "" {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	// Trouver le label de la session la plus récente (par StartedAtUTC DESC).
	latestLabel := latestSessionLabelCanonical(filtered)
	if latestLabel == "" {
		return nil
	}

	// Garder uniquement les rows de cette session.
	var sessionRows []canonical.PlayerMatchRow
	for _, r := range filtered {
		if r.Enrichment.SessionLabel != nil && *r.Enrichment.SessionLabel == latestLabel {
			sessionRows = append(sessionRows, r)
		}
	}
	if len(sessionRows) == 0 {
		return nil
	}

	kpis := ComputeKPIsFromCanonical(sessionRows, len(sessionRows))
	item := &domain.SessionSummaryItem{
		SessionLabel: latestLabel,
		MatchCount:   len(sessionRows),
		WinRate:      kpis.WinRate,
		GlobalRatio:  kpis.GlobalRatio,
	}
	if earliest := earliestStartTimeCanonical(sessionRows); earliest != nil {
		item.StartedAt = earliest
	}
	return item
}

// latestSessionLabelCanonical : trouve le label de la session la plus récente.
func latestSessionLabelCanonical(rows []canonical.PlayerMatchRow) string {
	sorted := make([]canonical.PlayerMatchRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Summary.StartedAtUTC.After(sorted[j].Summary.StartedAtUTC)
	})
	for _, r := range sorted {
		if r.Enrichment.SessionLabel != nil && *r.Enrichment.SessionLabel != "" {
			return *r.Enrichment.SessionLabel
		}
	}
	return ""
}

// earliestStartTimeCanonical : retourne le start_time le plus ancien.
func earliestStartTimeCanonical(rows []canonical.PlayerMatchRow) *time.Time {
	var earliest *time.Time
	for i := range rows {
		t := rows[i].Summary.StartedAtUTC
		if earliest == nil || t.Before(*earliest) {
			earliest = &t
		}
	}
	return earliest
}

// BuildSessionSummariesFromCanonical : full canonical (P4.3 finale).
// Liste des N dernières sessions solo ou squad avec KPIs agrégés.
//
// Note ADR 0011 : domain.HomeMatchRow.PairNameFR (composite Halo-only)
// n'a pas d'équivalent canonical. dominantMode est dérivé de
// Summary.GameVariant.Labels["fr"] || DefaultLabel comme proxy.
func BuildSessionSummariesFromCanonical(rows []canonical.PlayerMatchRow, squadMode bool, limit int) []domain.SessionSummaryItem {
	if len(rows) == 0 {
		return nil
	}

	// Filtrer par squadMode + label non-nil.
	var filtered []canonical.PlayerMatchRow
	for _, r := range rows {
		if r.Enrichment.IsWithFriends == squadMode && r.Enrichment.SessionLabel != nil && *r.Enrichment.SessionLabel != "" {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	// Labels distincts triés par StartedAtUTC DESC.
	labels := distinctSessionLabelsCanonical(filtered)

	resultCap := len(labels)
	if limit > 0 && limit < resultCap {
		resultCap = limit
	}
	result := make([]domain.SessionSummaryItem, 0, resultCap)
	for _, lbl := range labels {
		if limit > 0 && len(result) >= limit {
			break
		}
		var sessionRows []canonical.PlayerMatchRow
		for _, r := range filtered {
			if r.Enrichment.SessionLabel != nil && *r.Enrichment.SessionLabel == lbl {
				sessionRows = append(sessionRows, r)
			}
		}
		if len(sessionRows) == 0 {
			continue
		}

		var wins, losses, draws, dnfs int
		for _, r := range sessionRows {
			switch r.Self.Outcome {
			case canonical.OutcomeWin:
				wins++
			case canonical.OutcomeLoss:
				losses++
			case canonical.OutcomeTie:
				draws++
			default:
				dnfs++
			}
		}

		// Performance joueur : moyenne des PerformanceScore.
		var avgPlayerPerf *float64
		{
			var sum float64
			var count int
			for _, r := range sessionRows {
				if r.Enrichment.PerformanceScore != nil {
					sum += *r.Enrichment.PerformanceScore
					count++
				}
			}
			if count > 0 {
				v := round1(sum / float64(count))
				avgPlayerPerf = &v
			}
		}

		// Performance équipe : uniquement en mode escouade.
		var avgTeamPerf *float64
		if squadMode {
			var scores []*float64
			var winRates, kdas, kills []float64
			for _, r := range sessionRows {
				scores = append(scores, r.Enrichment.PerformanceScore)
				wr := 0.0
				switch r.Self.Outcome {
				case canonical.OutcomeWin:
					wr = 100.0
				case canonical.OutcomeTie:
					wr = 50.0
				}
				winRates = append(winRates, wr)
				kda := 0.0
				if r.Self.KDA != nil {
					kda = *r.Self.KDA
				}
				kdas = append(kdas, kda)
				k := 0
				if r.Self.Kills != nil {
					k = *r.Self.Kills
				}
				kills = append(kills, float64(k))
			}
			sq := ComputeSquadPerformanceScore(scores, winRates, kdas, kills)
			avgTeamPerf = sq.Score
		}

		// K/D moyen.
		var avgKDA *float64
		{
			var sum float64
			var count int
			for _, r := range sessionRows {
				if r.Self.KDA != nil {
					sum += *r.Self.KDA
					count++
				}
			}
			if count > 0 {
				v := round1(sum / float64(count))
				avgKDA = &v
			}
		}

		// Mode dominant : GameVariant FR le plus joué (proxy pour PairNameFR).
		var dominantMode *string
		{
			freq := make(map[string]int)
			for _, r := range sessionRows {
				if r.Summary.GameVariant == nil {
					continue
				}
				name := r.Summary.GameVariant.DefaultLabel
				if v, ok := r.Summary.GameVariant.Labels["fr"]; ok && v != "" {
					name = v
				}
				if name != "" {
					freq[name]++
				}
			}
			var best string
			var bestCount int
			for name, cnt := range freq {
				if cnt > bestCount || (cnt == bestCount && name < best) {
					best = name
					bestCount = cnt
				}
			}
			if best != "" {
				dominantMode = &best
			}
		}

		// Playlist dominante.
		var dominantPlaylist *string
		{
			freq := make(map[string]int)
			for _, r := range sessionRows {
				if r.Summary.Playlist == nil {
					continue
				}
				name := r.Summary.Playlist.DefaultLabel
				if v, ok := r.Summary.Playlist.Labels["fr"]; ok && v != "" {
					name = v
				}
				if name != "" {
					freq[name]++
				}
			}
			var best string
			var bestCount int
			for name, cnt := range freq {
				if cnt > bestCount || (cnt == bestCount && name < best) {
					best = name
					bestCount = cnt
				}
			}
			if best != "" {
				dominantPlaylist = &best
			}
		}

		kpis := ComputeKPIsFromCanonical(sessionRows, len(sessionRows))
		item := domain.SessionSummaryItem{
			SessionLabel:         lbl,
			MatchCount:           len(sessionRows),
			WinRate:              kpis.WinRate,
			GlobalRatio:          kpis.GlobalRatio,
			Wins:                 wins,
			Losses:               losses,
			Draws:                draws,
			DNFs:                 dnfs,
			AvgPlayerPerformance: avgPlayerPerf,
			AvgTeamPerformance:   avgTeamPerf,
			AvgKDA:               avgKDA,
			DominantPlaylist:     dominantPlaylist,
			DominantMode:         dominantMode,
		}
		if earliest := earliestStartTimeCanonical(sessionRows); earliest != nil {
			item.StartedAt = earliest
		}
		if ended := latestEndTimeCanonical(sessionRows); ended != nil {
			item.EndedAt = ended
		}
		result = append(result, item)
	}
	return result
}

// distinctSessionLabelsCanonical : labels distincts triés par StartedAtUTC DESC.
func distinctSessionLabelsCanonical(rows []canonical.PlayerMatchRow) []string {
	labelTimes := make(map[string]time.Time)
	for _, r := range rows {
		if r.Enrichment.SessionLabel == nil || *r.Enrichment.SessionLabel == "" {
			continue
		}
		lbl := *r.Enrichment.SessionLabel
		t := r.Summary.StartedAtUTC
		if existing, ok := labelTimes[lbl]; !ok || t.After(existing) {
			labelTimes[lbl] = t
		}
	}
	labels := make([]string, 0, len(labelTimes))
	for lbl := range labelTimes {
		labels = append(labels, lbl)
	}
	sort.Slice(labels, func(i, j int) bool {
		return labelTimes[labels[i]].After(labelTimes[labels[j]])
	})
	return labels
}

// latestEndTimeCanonical : end time estimé du dernier match (start + duration).
func latestEndTimeCanonical(rows []canonical.PlayerMatchRow) *time.Time {
	var latest *canonical.PlayerMatchRow
	for i := range rows {
		if latest == nil || rows[i].Summary.StartedAtUTC.After(latest.Summary.StartedAtUTC) {
			latest = &rows[i]
		}
	}
	if latest == nil {
		return nil
	}
	if latest.Self.TimePlayed != nil && *latest.Self.TimePlayed > 0 {
		t := latest.Summary.StartedAtUTC.Add(time.Duration(*latest.Self.TimePlayed) * time.Second)
		return &t
	}
	t := latest.Summary.StartedAtUTC
	return &t
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
