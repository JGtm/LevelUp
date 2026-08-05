// Package service - synthesis_service_legacy.go : builders legacy
// (SynthesisMatchRow / SynthesisHeatmapRow) pour la page Synthese.
// Decoupe de synthesis_service.go (god-file split, refactor 2026-05-27).
package service

import (
	"fmt"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

func filterSynthesisByPeriod(
	rows []legacymatch.SynthesisMatchRow,
	period string,
	_ domain.FilterContextInput, // filtres avancÃ©s â€" Ã  implÃ©menter aprÃ¨s backfill de map/mode
) ([]legacymatch.SynthesisMatchRow, []string, []string) {
	applied := []string{}
	ignored := []string{}

	var cutoff *time.Time
	now := time.Now().UTC()

	switch period {
	case "1w":
		t := now.AddDate(0, 0, -7)
		cutoff = &t
		applied = append(applied, fmt.Sprintf("pÃ©riode=%s", period))
	case "1m":
		t := now.AddDate(0, -1, 0)
		cutoff = &t
		applied = append(applied, fmt.Sprintf("pÃ©riode=%s", period))
	case "1y":
		t := now.AddDate(-1, 0, 0)
		cutoff = &t
		applied = append(applied, fmt.Sprintf("pÃ©riode=%s", period))
	case "2y":
		t := now.AddDate(-2, 0, 0)
		cutoff = &t
		applied = append(applied, fmt.Sprintf("pÃ©riode=%s", period))
	default:
		// "all" â€" pas de filtre temporel
	}

	if cutoff == nil {
		return rows, applied, ignored
	}

	filtered := make([]legacymatch.SynthesisMatchRow, 0, len(rows))
	for _, r := range rows {
		if !r.StartTime.Before(*cutoff) {
			filtered = append(filtered, r)
		}
	}
	return filtered, applied, ignored
}

func buildScopeDescription(period string, matchCount int) string {
	labels := map[string]string{
		"all": "tous les matchs",
		"1w":  "7 derniers jours",
		"1m":  "30 derniers jours",
		"1y":  "12 derniers mois",
		"2y":  "2 derniÃ¨res annÃ©es",
	}
	label, ok := labels[period]
	if !ok {
		label = period
	}
	return fmt.Sprintf("%d matchs - %s", matchCount, label)
}

// =============================================================================
// Helpers previews D5/D6/D7
// =============================================================================

const highlightTopN = 5

// buildHighlightsPreview construit les top/pire matchs depuis les matchs filtrÃ©s.
// Tri en place sur des copies â€" pas de mutation des slices partagÃ©s.
func buildHighlightsPreview(rows []legacymatch.SynthesisMatchRow) domain.SynthesisHighlightsPreview {
	if len(rows) == 0 {
		return domain.SynthesisHighlightsPreview{}
	}
	toHighlight := func(r legacymatch.SynthesisMatchRow) domain.SynthesisMatchHighlight {
		return domain.SynthesisMatchHighlight{
			MatchID:   r.MatchID,
			Kills:     r.Kills,
			Deaths:    r.Deaths,
			KDA:       r.KDA,
			Outcome:   r.Outcome,
			PerfScore: r.PerformanceScore,
		}
	}

	topByKills := topNByFunc(rows, highlightTopN, func(a, b legacymatch.SynthesisMatchRow) bool {
		return a.Kills > b.Kills
	})
	topByKDA := topNByFunc(rows, highlightTopN, func(a, b legacymatch.SynthesisMatchRow) bool {
		av := 0.0
		if a.KDA != nil {
			av = *a.KDA
		}
		bv := 0.0
		if b.KDA != nil {
			bv = *b.KDA
		}
		return av > bv
	})
	worstByDeaths := topNByFunc(rows, highlightTopN, func(a, b legacymatch.SynthesisMatchRow) bool {
		return a.Deaths > b.Deaths
	})

	toSlice := func(src []legacymatch.SynthesisMatchRow) []domain.SynthesisMatchHighlight {
		out := make([]domain.SynthesisMatchHighlight, len(src))
		for i, r := range src {
			out[i] = toHighlight(r)
		}
		return out
	}
	return domain.SynthesisHighlightsPreview{
		TopByKills:    toSlice(topByKills),
		TopByKDA:      toSlice(topByKDA),
		WorstByDeaths: toSlice(worstByDeaths),
	}
}

// topNByFunc retourne les N premiers Ã©lÃ©ments selon la fonction de comparaison less(a,b).
func topNByFunc(rows []legacymatch.SynthesisMatchRow, n int, less func(a, b legacymatch.SynthesisMatchRow) bool) []legacymatch.SynthesisMatchRow {
	cp := make([]legacymatch.SynthesisMatchRow, len(rows))
	copy(cp, rows)
	// tri partiel : sÃ©lectionner les N premiers
	for i := 0; i < n && i < len(cp); i++ {
		minIdx := i
		for j := i + 1; j < len(cp); j++ {
			if less(cp[j], cp[minIdx]) {
				minIdx = j
			}
		}
		cp[i], cp[minIdx] = cp[minIdx], cp[i]
	}
	if n > len(cp) {
		n = len(cp)
	}
	return cp[:n]
}

// buildBreakdowns agrÃ¨ge les donnÃ©es heatmap en breakdowns carte et mode.
func buildBreakdowns(rows []domain.SynthesisHeatmapRow) domain.SynthesisBreakdowns {
	if len(rows) == 0 {
		return domain.SynthesisBreakdowns{
			TopMaps:  []domain.SynthesisMapEntry{},
			TopModes: []domain.SynthesisModeEntry{},
		}
	}

	mapAgg := map[string][2]int{}  // map_name â†' [match_count, wins]
	modeAgg := map[string][2]int{} // mode_name â†' [match_count, wins]
	for _, r := range rows {
		m := mapAgg[r.MapName]
		m[0] += r.MatchCount
		m[1] += r.Wins
		mapAgg[r.MapName] = m

		mo := modeAgg[r.ModeName]
		mo[0] += r.MatchCount
		mo[1] += r.Wins
		modeAgg[r.ModeName] = mo
	}

	mapEntries := make([]domain.SynthesisMapEntry, 0, len(mapAgg))
	for name, v := range mapAgg {
		wr := 0.0
		if v[0] > 0 {
			wr = float64(v[1]) / float64(v[0]) * 100
		}
		mapEntries = append(mapEntries, domain.SynthesisMapEntry{
			MapName:    name,
			MatchCount: v[0],
			Wins:       v[1],
			WinRate:    wr,
		})
	}
	modeEntries := make([]domain.SynthesisModeEntry, 0, len(modeAgg))
	for name, v := range modeAgg {
		wr := 0.0
		if v[0] > 0 {
			wr = float64(v[1]) / float64(v[0]) * 100
		}
		modeEntries = append(modeEntries, domain.SynthesisModeEntry{
			ModeName:   name,
			MatchCount: v[0],
			Wins:       v[1],
			WinRate:    wr,
		})
	}
	// tri par MatchCount desc (sÃ©lection partielle des top 10)
	sortMapEntries(mapEntries)
	sortModeEntries(modeEntries)
	if len(mapEntries) > 10 {
		mapEntries = mapEntries[:10]
	}
	if len(modeEntries) > 10 {
		modeEntries = modeEntries[:10]
	}
	return domain.SynthesisBreakdowns{TopMaps: mapEntries, TopModes: modeEntries}
}

// buildBreakdownsFromCanonical agrège les rows canoniques filtrés en breakdowns
// carte et mode, avec décompte complet des outcomes (wins/losses/ties/unfinished).
// Remplace buildBreakdowns(heatmapRows) pour être period-aware.
func buildBreakdownsFromCanonical(rows []canonical.PlayerMatchRow) domain.SynthesisBreakdowns {
	type entry struct{ wins, losses, ties, unfinished, total int }
	maps := map[string]*entry{}
	modes := map[string]*entry{}

	for _, r := range rows {
		mapName := ""
		if m := r.Summary.Map; m != nil {
			// Préférer le label FR si disponible (noms de carte identiques EN/FR pour Halo Infinite).
			if fr := m.Labels["fr"]; fr != "" {
				mapName = fr
			} else {
				mapName = m.DefaultLabel
			}
		}
		modeName := ""
		if p := r.Summary.PairMode; p != nil {
			// Même priorité que ResolveModeUI : COALESCE(pair_name_fr, pair_name).
			// pair_name_fr donne le label FR après NormalizeModeLabel (ex. "Arène : Slayer" → "Slayer" en FR).
			src := p.Labels["fr"]
			if src == "" {
				src = p.DefaultLabel
			}
			modeName = analysis.NormalizeModeLabel(src)
		}
		if mapName != "" {
			if maps[mapName] == nil {
				maps[mapName] = &entry{}
			}
			e := maps[mapName]
			e.total++
			switch r.Self.Outcome {
			case canonical.OutcomeWin:
				e.wins++
			case canonical.OutcomeLoss:
				e.losses++
			case canonical.OutcomeTie:
				e.ties++
			default:
				e.unfinished++
			}
		}
		if modeName != "" {
			if modes[modeName] == nil {
				modes[modeName] = &entry{}
			}
			e := modes[modeName]
			e.total++
			switch r.Self.Outcome {
			case canonical.OutcomeWin:
				e.wins++
			case canonical.OutcomeLoss:
				e.losses++
			case canonical.OutcomeTie:
				e.ties++
			default:
				e.unfinished++
			}
		}
	}

	mapEntries := make([]domain.SynthesisMapEntry, 0, len(maps))
	for name, e := range maps {
		wr := 0.0
		if e.total > 0 {
			wr = float64(e.wins) / float64(e.total)
		}
		mapEntries = append(mapEntries, domain.SynthesisMapEntry{
			MapName:    name,
			MatchCount: e.total,
			Wins:       e.wins,
			Losses:     e.losses,
			Ties:       e.ties,
			Unfinished: e.unfinished,
			WinRate:    wr,
		})
	}
	modeEntries := make([]domain.SynthesisModeEntry, 0, len(modes))
	for name, e := range modes {
		wr := 0.0
		if e.total > 0 {
			wr = float64(e.wins) / float64(e.total)
		}
		modeEntries = append(modeEntries, domain.SynthesisModeEntry{
			ModeName:   name,
			MatchCount: e.total,
			Wins:       e.wins,
			Losses:     e.losses,
			Ties:       e.ties,
			Unfinished: e.unfinished,
			WinRate:    wr,
		})
	}
	sortMapEntries(mapEntries)
	sortModeEntries(modeEntries)
	if mapEntries == nil {
		mapEntries = []domain.SynthesisMapEntry{}
	}
	if modeEntries == nil {
		modeEntries = []domain.SynthesisModeEntry{}
	}
	return domain.SynthesisBreakdowns{TopMaps: mapEntries, TopModes: modeEntries}
}

// sortMapEntries / sortModeEntries — tri par nombre de matchs décroissant, avec
// DÉPARTAGE alphabétique sur le nom. Les entrées sont construites par itération
// sur une map Go (ordre aléatoire par construction) : sans clé de départage, deux
// cartes/modes à nombre de matchs égal permutaient d'une réponse d'API à l'autre,
// ce qui rendait les classements de la page Synthèse non déterministes (dérive
// visuelle observée sur le harnais de régression, cf. e2e/visual/app-pages —
// canvas « classements cartes et modes »). Le nom est la clé stable disponible
// (identifiant d'agrégation de l'entrée).
func sortMapEntries(s []domain.SynthesisMapEntry) {
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].MatchCount != s[j].MatchCount {
			return s[i].MatchCount > s[j].MatchCount
		}
		return s[i].MapName < s[j].MapName
	})
}

func sortModeEntries(s []domain.SynthesisModeEntry) {
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].MatchCount != s[j].MatchCount {
			return s[i].MatchCount > s[j].MatchCount
		}
		return s[i].ModeName < s[j].ModeName
	})
}

// =============================================================================
// P4.3 (ADR 0011) : helpers canonical (le converter SynthesisMatchRow est retirÃ©)
// =============================================================================

// filterSynthesisByPeriodCanonical filtre les lignes canoniques selon :
// - period : preset string (1w, 1m, 1y, 2y, all)
// - startDate / endDate : plage ISO YYYY-MM-DD (envoyee depuis PeriodePill/SaisonPill)
