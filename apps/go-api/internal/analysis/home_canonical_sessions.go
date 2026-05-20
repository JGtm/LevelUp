// Package analysis â€” home_canonical_sessions.go : BuildSessionSummary[ies]
// canonical (P4.3 finale). Agrege les matchs par session (solo/squad).
package analysis

import (
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// BuildSessionSummaryFromCanonical : full canonical (P4.3 finale).
// Filtre par IsWithFriends (squadMode), trouve la session la plus rÃ©cente
// par StartedAtUTC, agrÃ¨ge ses matchs en KPIs.
func BuildSessionSummaryFromCanonical(rows []canonical.PlayerMatchRow, squadMode bool, locale string) *domain.SessionSummaryItem {
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

	// Trouver le label de la session la plus rÃ©cente (par StartedAtUTC DESC).
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

	kpis := ComputeKPIsFromCanonical(sessionRows, len(sessionRows), locale)
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

// latestSessionLabelCanonical : trouve le label de la session la plus rÃ©cente.
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
// Liste des N derniÃ¨res sessions solo ou squad avec KPIs agrÃ©gÃ©s.
//
// Note ADR 0011 : legacymatch.HomeMatchRow.PairNameFR (composite Halo-only)
// n'a pas d'Ã©quivalent canonical. dominantMode est dÃ©rivÃ© de
// Summary.GameVariant.Labels["fr"] || DefaultLabel comme proxy.
func BuildSessionSummariesFromCanonical(rows []canonical.PlayerMatchRow, squadMode bool, limit int, locale string) []domain.SessionSummaryItem {
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

	// Labels distincts triÃ©s par StartedAtUTC DESC.
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

		// Performance Ã©quipe : uniquement en mode escouade.
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

		// Mode dominant : PairMode FR le plus joué (fallback GameVariant).
		// Bug #7 — PairMode expose pair_name_fr en DB.
		// Normalisation appliquée (strip " on Bazaar", extraction sous-mode
		// "Arena:Slayer" → "Slayer") pour aligner sur BuildRecentMatchesWith…
		// et fusionner les fréquences cross-map. Sans ça les sessions solo
		// (où pair_name_fr est souvent absent) affichaient l'EN brut.
		dominantMode := dominantNameFromRows(sessionRows, locale, func(r canonical.PlayerMatchRow) (string, string) {
			en, fr := modeLabels(r)
			mapEN, mapFR := assetLabels(r.Summary.Map)
			return normalizeHomeModeLabel(en, mapEN, mapFR), normalizeHomeModeLabel(fr, mapEN, mapFR)
		})

		// Playlist dominante (FR si dispo et locale=fr, sinon EN). Bug #2.
		dominantPlaylist := dominantNameFromRows(sessionRows, locale, func(r canonical.PlayerMatchRow) (string, string) {
			if r.Summary.Playlist == nil {
				return "", ""
			}
			return assetLabels(r.Summary.Playlist)
		})

		kpis := ComputeKPIsFromCanonical(sessionRows, len(sessionRows), locale)
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

// distinctSessionLabelsCanonical : labels distincts triÃ©s par StartedAtUTC DESC.
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

// latestEndTimeCanonical : end time estimÃ© du dernier match (start + duration).
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
