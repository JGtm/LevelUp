// Package analysis â€” home_sessions.go : rÃ©sumÃ©s de sessions (BuildSessionSummaries
// et BuildSessionSummary) + helpers de tri par label/temps.
package analysis

import (
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// ---------------------------------------------------------------------------
// BuildSessionSummaries / BuildSessionSummary â€” rÃ©sumÃ©s de sessions
// ---------------------------------------------------------------------------

// BuildSessionSummaries construit la liste des N derniÃ¨res sessions (solo ou escouade),
// triÃ©es de la plus rÃ©cente Ã  la plus ancienne.
func BuildSessionSummaries(
	matches []legacymatch.HomeMatchRow,
	sessions []legacymatch.HomeSessionRow,
	squadMode bool,
	limit int,
	effectiveHpToKill float64,
) []domain.SessionSummaryItem {
	if len(sessions) == 0 || len(matches) == 0 {
		return nil
	}

	// Filtrer par mode.
	var filtered []legacymatch.HomeSessionRow
	for _, s := range sessions {
		if s.IsWithFriends == squadMode && s.SessionLabel != nil {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	// Collecter les labels distincts triÃ©s par date dÃ©croissante.
	labels := distinctSessionLabels(filtered)

	// Construire le rÃ©sumÃ© pour chaque label, jusqu'Ã  la limite.
	resultCap := len(labels)
	if limit > 0 && limit < resultCap {
		resultCap = limit
	}
	result := make([]domain.SessionSummaryItem, 0, resultCap)
	for _, lbl := range labels {
		if limit > 0 && len(result) >= limit {
			break
		}
		// Rassembler les match_ids de ce label.
		matchIDSet := make(map[string]bool)
		for _, s := range filtered {
			if s.SessionLabel != nil && *s.SessionLabel == lbl {
				matchIDSet[s.MatchID] = true
			}
		}
		// Filtrer les matchs.
		var sessionMatches []legacymatch.HomeMatchRow
		for _, m := range matches {
			if matchIDSet[m.MatchID] {
				sessionMatches = append(sessionMatches, m)
			}
		}
		if len(sessionMatches) == 0 {
			continue
		}

		// Compter les outcomes.
		var wins, losses, draws, dnfs int
		for _, m := range sessionMatches {
			switch m.Outcome {
			case domain.OutcomeWin:
				wins++
			case domain.OutcomeLoss:
				losses++
			case domain.OutcomeDraw:
				draws++
			default:
				dnfs++
			}
		}

		// Performance joueur : toujours la moyenne des PerformanceScore personnels.
		var avgPlayerPerf *float64
		{
			var sum float64
			var count int
			for _, m := range sessionMatches {
				if m.PerformanceScore != nil {
					sum += *m.PerformanceScore
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
			for _, m := range sessionMatches {
				scores = append(scores, m.PerformanceScore)
				wr := 0.0
				switch m.Outcome {
				case domain.OutcomeWin:
					wr = 100.0
				case domain.OutcomeDraw:
					wr = 50.0
				}
				winRates = append(winRates, wr)
				kda := 0.0
				if m.KDA != nil {
					kda = *m.KDA
				}
				kdas = append(kdas, kda)
				kills = append(kills, float64(m.Kills))
			}
			sq := ComputeSquadPerformanceScore(scores, winRates, kdas, kills)
			avgTeamPerf = sq.Score
		}

		// K/D moyen sur la session.
		var avgKDA *float64
		{
			var sum float64
			var count int
			for _, m := range sessionMatches {
				if m.KDA != nil {
					sum += *m.KDA
					count++
				}
			}
			if count > 0 {
				v := round1(sum / float64(count))
				avgKDA = &v
			}
		}

		// Mode dominant : pair (map+mode) le plus jouÃ© sur la session (nom FR).
		var dominantMode *string
		{
			freq := make(map[string]int)
			for _, m := range sessionMatches {
				if m.PairNameFR != "" {
					freq[m.PairNameFR]++
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

		// Playlist dominante : playlist FR la plus jouÃ©e sur la session.
		var dominantPlaylist *string
		{
			freq := make(map[string]int)
			for _, m := range sessionMatches {
				name := m.PlaylistNameFR
				if name == "" {
					name = m.PlaylistName
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

		kpis := ComputeKPIs(sessionMatches, len(sessionMatches), effectiveHpToKill)
		item := domain.SessionSummaryItem{
			SessionLabel:         lbl,
			MatchCount:           len(sessionMatches),
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
		if earliest := earliestStartTime(sessionMatches); earliest != nil {
			item.StartedAt = earliest
		}
		if ended := latestEndTime(sessionMatches); ended != nil {
			item.EndedAt = ended
		}
		result = append(result, item)
	}
	return result
}

// distinctSessionLabels retourne les labels distincts triÃ©s par start_time DESC.
func distinctSessionLabels(sessions []legacymatch.HomeSessionRow) []string {
	// Calculer le start_time max par label.
	labelTimes := make(map[string]time.Time)
	for _, s := range sessions {
		if s.SessionLabel == nil || *s.SessionLabel == "" {
			continue
		}
		lbl := *s.SessionLabel
		if s.StartTime != nil {
			if t, ok := labelTimes[lbl]; !ok || s.StartTime.After(t) {
				labelTimes[lbl] = *s.StartTime
			}
		} else {
			if _, ok := labelTimes[lbl]; !ok {
				labelTimes[lbl] = time.Time{}
			}
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

// BuildSessionSummary construit le rÃ©sumÃ© de la derniÃ¨re session solo ou escouade.
func BuildSessionSummary(
	matches []legacymatch.HomeMatchRow,
	sessions []legacymatch.HomeSessionRow,
	squadMode bool,
	effectiveHpToKill float64,
) *domain.SessionSummaryItem {
	if len(sessions) == 0 || len(matches) == 0 {
		return nil
	}

	// Filtrer les sessions par mode.
	var filtered []legacymatch.HomeSessionRow
	for _, s := range sessions {
		if s.IsWithFriends == squadMode && s.SessionLabel != nil {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	// Trouver le label de la session la plus rÃ©cente.
	latestLabel := latestSessionLabel(filtered)
	if latestLabel == "" {
		return nil
	}

	// Rassembler les match_ids de cette session.
	matchIDSet := make(map[string]bool)
	for _, s := range filtered {
		if s.SessionLabel != nil && *s.SessionLabel == latestLabel {
			matchIDSet[s.MatchID] = true
		}
	}
	if len(matchIDSet) == 0 {
		return nil
	}

	// Filtrer les matchs de la session.
	var sessionMatches []legacymatch.HomeMatchRow
	for _, m := range matches {
		if matchIDSet[m.MatchID] {
			sessionMatches = append(sessionMatches, m)
		}
	}
	if len(sessionMatches) == 0 {
		return nil
	}

	kpis := ComputeKPIs(sessionMatches, len(sessionMatches), effectiveHpToKill)
	item := &domain.SessionSummaryItem{
		SessionLabel: latestLabel,
		MatchCount:   len(sessionMatches),
		WinRate:      kpis.WinRate,
		GlobalRatio:  kpis.GlobalRatio,
	}
	// Trouver le start_time le plus ancien de la session.
	if earliest := earliestStartTime(sessionMatches); earliest != nil {
		item.StartedAt = earliest
	}
	return item
}

func latestSessionLabel(sessions []legacymatch.HomeSessionRow) string {
	// Trier par start_time DESC, prendre le premier session_label.
	sorted := make([]legacymatch.HomeSessionRow, len(sessions))
	copy(sorted, sessions)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].StartTime == nil {
			return false
		}
		if sorted[j].StartTime == nil {
			return true
		}
		return sorted[i].StartTime.After(*sorted[j].StartTime)
	})
	for _, s := range sorted {
		if s.SessionLabel != nil && *s.SessionLabel != "" {
			return *s.SessionLabel
		}
	}
	return ""
}

func earliestStartTime(matches []legacymatch.HomeMatchRow) *time.Time {
	var earliest *time.Time
	for i := range matches {
		t := matches[i].StartTime
		if earliest == nil || t.Before(*earliest) {
			earliest = &t
		}
	}
	return earliest
}

// latestEndTime retourne l'heure de fin estimÃ©e du dernier match de la session.
func latestEndTime(matches []legacymatch.HomeMatchRow) *time.Time {
	var latest *legacymatch.HomeMatchRow
	for i := range matches {
		if latest == nil || matches[i].StartTime.After(latest.StartTime) {
			latest = &matches[i]
		}
	}
	if latest == nil {
		return nil
	}
	if latest.TimePlayedSecs != nil && *latest.TimePlayedSecs > 0 {
		t := latest.StartTime.Add(time.Duration(*latest.TimePlayedSecs) * time.Second)
		return &t
	}
	t := latest.StartTime
	return &t
}
