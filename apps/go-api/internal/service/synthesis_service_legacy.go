// Package service - synthesis_service_legacy.go : builders legacy
// (SynthesisMatchRow) pour la page Synthese.
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
	_ domain.FilterContextInput, // filtres avancés — à implémenter après backfill de map/mode
) ([]legacymatch.SynthesisMatchRow, []string, []string) {
	applied := []string{}
	ignored := []string{}

	var cutoff *time.Time
	now := time.Now().UTC()

	switch period {
	case "1w":
		t := now.AddDate(0, 0, -7)
		cutoff = &t
		applied = append(applied, fmt.Sprintf("période=%s", period))
	case "1m":
		t := now.AddDate(0, -1, 0)
		cutoff = &t
		applied = append(applied, fmt.Sprintf("période=%s", period))
	case "1y":
		t := now.AddDate(-1, 0, 0)
		cutoff = &t
		applied = append(applied, fmt.Sprintf("période=%s", period))
	case "2y":
		t := now.AddDate(-2, 0, 0)
		cutoff = &t
		applied = append(applied, fmt.Sprintf("période=%s", period))
	default:
		// "all" — pas de filtre temporel
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
		"2y":  "2 dernières années",
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

// buildHighlightsPreview construit les top/pire matchs depuis les matchs filtrés.
// Tri en place sur des copies — pas de mutation des slices partagés.
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

// topNByFunc retourne les N premiers éléments selon la fonction de comparaison less(a,b).
func topNByFunc(rows []legacymatch.SynthesisMatchRow, n int, less func(a, b legacymatch.SynthesisMatchRow) bool) []legacymatch.SynthesisMatchRow {
	cp := make([]legacymatch.SynthesisMatchRow, len(rows))
	copy(cp, rows)
	// tri partiel : sélectionner les N premiers
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

// breakdownTally cumule, pour une ligne carte ou mode, les issues des matchs et
// la somme des scores de performance des seuls matchs notés.
type breakdownTally struct {
	wins, losses, ties, unfinished, total int
	scored                                int
	scoreSum                              float64
}

// tallyBreakdown ajoute le match r à la ligne e : une issue (toute issue qui
// n'est ni victoire, ni défaite, ni égalité compte comme non terminée) et, si le
// match porte un score de performance, ce score à la moyenne de la ligne.
func tallyBreakdown(e *breakdownTally, r canonical.PlayerMatchRow) {
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
	if r.Enrichment.PerformanceScore != nil {
		e.scored++
		e.scoreSum += *r.Enrichment.PerformanceScore
	}
}

// tallyBreakdownInto cumule r sous la clé name de m (créée au besoin).
func tallyBreakdownInto(m map[string]*breakdownTally, name string, r canonical.PlayerMatchRow) {
	e := m[name]
	if e == nil {
		e = &breakdownTally{}
		m[name] = e
	}
	tallyBreakdown(e, r)
}

// winRate est la part de victoires de la ligne (0 si aucun match).
func (e *breakdownTally) winRate() float64 {
	if e.total == 0 {
		return 0
	}
	return float64(e.wins) / float64(e.total)
}

// performance rend la moyenne des scores des matchs notés de la ligne et son
// palier canonique (analysis.PerfTier) ; (nil, 0) si aucun match n'est noté.
func (e *breakdownTally) performance() (*float64, int) {
	if e.scored == 0 {
		return nil, 0
	}
	avg := e.scoreSum / float64(e.scored)
	return &avg, int(analysis.PerfTier(avg))
}

// breakdownMapLabel est le libellé de carte d'un match (vide si inconnu).
func breakdownMapLabel(r canonical.PlayerMatchRow) string {
	m := r.Summary.Map
	if m == nil {
		return ""
	}
	// Préférer le label FR si disponible (noms de carte identiques EN/FR pour Halo Infinite).
	if fr := m.Labels["fr"]; fr != "" {
		return fr
	}
	return m.DefaultLabel
}

// breakdownModeLabel est le libellé de mode d'un match (vide si inconnu).
func breakdownModeLabel(r canonical.PlayerMatchRow) string {
	p := r.Summary.PairMode
	if p == nil {
		return ""
	}
	// Même priorité que ResolveModeUI : COALESCE(pair_name_fr, pair_name).
	// pair_name_fr donne le label FR après NormalizeModeLabel (ex. "Arène : Slayer" → "Slayer" en FR).
	src := p.Labels["fr"]
	if src == "" {
		src = p.DefaultLabel
	}
	return analysis.NormalizeModeLabel(src)
}

// buildBreakdownsFromCanonical agrège les rows canoniques filtrés en breakdowns
// carte et mode, avec décompte complet des outcomes (wins/losses/ties/unfinished)
// et score de performance moyen des matchs notés de chaque ligne.
func buildBreakdownsFromCanonical(rows []canonical.PlayerMatchRow) domain.SynthesisBreakdowns {
	maps := map[string]*breakdownTally{}
	modes := map[string]*breakdownTally{}
	for _, r := range rows {
		if name := breakdownMapLabel(r); name != "" {
			tallyBreakdownInto(maps, name, r)
		}
		if name := breakdownModeLabel(r); name != "" {
			tallyBreakdownInto(modes, name, r)
		}
	}

	mapEntries := make([]domain.SynthesisMapEntry, 0, len(maps))
	for name, e := range maps {
		avg, tier := e.performance()
		mapEntries = append(mapEntries, domain.SynthesisMapEntry{
			MapName:        name,
			MatchCount:     e.total,
			Wins:           e.wins,
			Losses:         e.losses,
			Ties:           e.ties,
			Unfinished:     e.unfinished,
			WinRate:        e.winRate(),
			PerformanceAvg: avg,
			PerfTier:       tier,
		})
	}
	modeEntries := make([]domain.SynthesisModeEntry, 0, len(modes))
	for name, e := range modes {
		avg, tier := e.performance()
		modeEntries = append(modeEntries, domain.SynthesisModeEntry{
			ModeName:       name,
			MatchCount:     e.total,
			Wins:           e.wins,
			Losses:         e.losses,
			Ties:           e.ties,
			Unfinished:     e.unfinished,
			WinRate:        e.winRate(),
			PerformanceAvg: avg,
			PerfTier:       tier,
		})
	}
	sortMapEntries(mapEntries)
	sortModeEntries(modeEntries)
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
// P4.3 (ADR 0011) : helpers canonical (le converter SynthesisMatchRow est retiré)
// =============================================================================

// filterSynthesisByPeriodCanonical filtre les lignes canoniques selon :
// - period : preset string (1w, 1m, 1y, 2y, all)
// - startDate / endDate : plage ISO YYYY-MM-DD (envoyee depuis PeriodePill/SaisonPill)
