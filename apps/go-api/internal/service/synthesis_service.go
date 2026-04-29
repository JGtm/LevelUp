// Package service — synthesis_service.go : orchestration de la page Synthèse.
//
// Sprint 55 D1 : extrait de squad_service.go — SynthesisService devient autonome,
// implémente port.SynthesisService.
//
// Sprint 55 D2 : period et filters du SynthesisRequest sont réellement appliqués.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// SynthesisService orchestre les données de la page Synthèse.
type SynthesisService struct {
	repo port.SynthesisRepository
	// dataAdapter (optionnel, Phase 2 plan finition multi-titres) :
	// quand fourni, GetSynthesisPage mesure la capability match.history pour
	// loguer une éventuelle dégradation.
	dataAdapter games.TitleDataAdapter
	// playerMatchesRepo (P4.1+P4.3, ADR 0011) : source canonical-aware. Quand
	// fournie avec titleSlug+gamertag, GetSynthesisPage charge directement
	// `[]canonical.PlayerMatchRow` et appelle les analyses *FromCanonical sans
	// converter. Le path legacy (s.repo.LoadSynthesisMatches) reste pour
	// rétrocompatibilité tant que la DI cabling n'est pas mise à jour partout.
	playerMatchesRepo port.PlayerMatchesRepository
	// titleSlug est nécessaire pour appeler PlayerMatchesRepo.LoadPlayerMatches.
	// Si "" et playerMatchesRepo != nil, fallback sur le repo legacy.
	titleSlug string
	gamertag  string
}

// NewSynthesisService crée un SynthesisService avec le repository injecté.
func NewSynthesisService(repo port.SynthesisRepository) *SynthesisService {
	return &SynthesisService{repo: repo}
}

// WithDataAdapter injecte un games.TitleDataAdapter optionnel (Phase 2 plan
// finition multi-titres). Permet de logger le statut des capabilities et
// d'amorcer la bascule fonctionnelle vers la couche canonique.
func (s *SynthesisService) WithDataAdapter(a games.TitleDataAdapter) *SynthesisService {
	s.dataAdapter = a
	return s
}

// WithPlayerMatchesRepo (P4.1+P4.3, ADR 0011) injecte le loader canonical-aware.
// Quand fourni avec titleSlug+gamertag, GetSynthesisPage charge depuis le
// loader unifié et appelle les analyses *FromCanonical (pas de converter).
func (s *SynthesisService) WithPlayerMatchesRepo(
	repo port.PlayerMatchesRepository,
	titleSlug, gamertag string,
) *SynthesisService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	s.gamertag = gamertag
	return s
}

// GetSynthesisPage construit la réponse de la page Synthèse.
// Sprint 55 D2 : applique period et filters depuis le SynthesisRequest.
func (s *SynthesisService) GetSynthesisPage(
	ctx context.Context,
	playerXUID string,
	req domain.SynthesisRequest,
) (*domain.SynthesisPageV2Response, error) {
	period := req.Period
	if period == "" {
		period = "all"
	}

	// Phase 2 plan finition multi-titres : log de la capability match.history
	// quand un DataAdapter est injecté. Sert à mesurer la dégradation potentielle
	// avant la bascule fonctionnelle (le Synthesis lit aujourd'hui depuis le repo
	// legacy car canonical.PlayerStats ne couvre pas encore SynthesisMatch).
	if s.dataAdapter != nil {
		caps := s.dataAdapter.Capabilities()
		if !caps.Has(games.CapMatchHistory) {
			slog.WarnContext(ctx, "capability_not_supported",
				"title_slug", s.dataAdapter.TitleSlug(),
				"capability", string(games.CapMatchHistory),
				"caller", "synthesis_service.GetSynthesisPage",
			)
		}
	}

	// P4.3 (ADR 0011) : chemin canonical pur quand playerMatchesRepo est
	// injecté. Plus de converter — toutes les analyses synthesis ont leur
	// variante *FromCanonical. Le path legacy (s.repo.LoadSynthesisMatches)
	// reste pour la rétrocompatibilité tant que la DI n'est pas câblée.
	var (
		filtered       []domain.SynthesisMatchRow
		filteredCanon  []canonical.PlayerMatchRow
		filtersApplied []string
		filtersIgnored []string
		useCanonical   bool
	)
	if s.playerMatchesRepo != nil && s.titleSlug != "" && s.gamertag != "" {
		canonicalRows, e := s.playerMatchesRepo.LoadPlayerMatches(
			ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
		)
		if e != nil {
			return nil, e
		}
		slog.DebugContext(ctx, "synthesis: loaded canonical",
			"rows", len(canonicalRows), "title_slug", s.titleSlug)
		filteredCanon, filtersApplied, filtersIgnored = filterSynthesisByPeriodCanonical(canonicalRows, period, req.Filters)
		useCanonical = true
	} else {
		synthMatches, err := s.repo.LoadSynthesisMatches(ctx, playerXUID)
		if err != nil {
			return nil, err
		}
		// D2 : filtrer par période (legacy path)
		filtered, filtersApplied, filtersIgnored = filterSynthesisByPeriod(synthMatches, period, req.Filters)
	}

	var soloKPIs, squadKPIs domain.SynthesisKPIs
	var topWeeks []domain.TopWeekEntry
	var heatmap []domain.TemporalHeatmapCell
	var overview domain.SynthesisOverview
	var highlights domain.SynthesisHighlightsPreview
	var matchCount int
	if useCanonical {
		soloKPIs = analysis.ComputeSynthesisKPIsFromCanonical(filteredCanon, false)
		squadKPIs = analysis.ComputeSynthesisKPIsFromCanonical(filteredCanon, true)
		topWeeks = analysis.ComputeSynthesisTopWeeksFromCanonical(filteredCanon)
		heatmap = analysis.ComputeTemporalHeatmapFromCanonical(filteredCanon)
		overview = buildSynthesisOverviewCanonical(filteredCanon, soloKPIs)
		highlights = buildHighlightsPreviewCanonical(filteredCanon)
		matchCount = len(filteredCanon)
	} else {
		soloKPIs = analysis.ComputeSynthesisKPIs(filtered, false)
		squadKPIs = analysis.ComputeSynthesisKPIs(filtered, true)
		topWeeks = analysis.ComputeSynthesisTopWeeks(filtered)
		heatmap = analysis.ComputeTemporalHeatmap(filtered)
		overview = buildSynthesisOverview(filtered, soloKPIs)
		highlights = buildHighlightsPreview(filtered)
		matchCount = len(filtered)
	}
	comparison := analysis.ComputeComparisonMetrics(soloKPIs, squadKPIs)

	// D6 : rivalries — encounters depuis shared (requête séparée)
	encounters, _ := s.repo.LoadEncounters(ctx, playerXUID) // erreur non fatale
	rivalries := buildRivalriesPreview(encounters)

	// D7 : breakdowns map/mode depuis la heatmap (requête séparée)
	heatmapRows, _ := s.repo.LoadSynthesisHeatmap(ctx, playerXUID) // erreur non fatale
	breakdowns := buildBreakdowns(heatmapRows)

	scope := domain.SynthesisScope{
		Period:         period,
		MatchCount:     matchCount,
		FiltersApplied: filtersApplied,
		FiltersIgnored: filtersIgnored,
		Description:    buildScopeDescription(period, matchCount),
		ComputedAt:     time.Now().UTC(),
	}

	return &domain.SynthesisPageV2Response{
		Scope:             scope,
		Overview:          overview,
		SoloKPIs:          soloKPIs,
		SquadKPIs:         squadKPIs,
		ComparisonMetrics: comparison,
		HeatmapData:       heatmap,
		TopWeeks:          topWeeks,
		HighlightsPreview: highlights,
		RivalriesPreview:  rivalries,
		Breakdowns:        breakdowns,
	}, nil
}

// =============================================================================
// Helpers internes
// =============================================================================

// filterSynthesisByPeriod filtre les matchs Synthèse selon la période demandée.
// Retourne les matchs filtrés, les filtres appliqués et ceux ignorés.
func filterSynthesisByPeriod(
	rows []domain.SynthesisMatchRow,
	period string,
	_ domain.FilterContextInput, // filtres avancés — à implémenter après backfill de map/mode
) ([]domain.SynthesisMatchRow, []string, []string) {
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

	filtered := make([]domain.SynthesisMatchRow, 0, len(rows))
	for _, r := range rows {
		if !r.StartTime.Before(*cutoff) {
			filtered = append(filtered, r)
		}
	}
	return filtered, applied, ignored
}

// buildSynthesisOverview calcule les cumuls, moyennes et pics depuis les matchs filtrés.
func buildSynthesisOverview(rows []domain.SynthesisMatchRow, soloKPIs domain.SynthesisKPIs) domain.SynthesisOverview {
	var totalKills, totalDeaths, totalWins, totalLosses int
	var bestKills int
	var bestKDA float64
	var winStreak, maxStreak int

	for _, r := range rows {
		totalKills += r.Kills
		totalDeaths += r.Deaths
		if r.Outcome == 2 { // WIN
			totalWins++
			winStreak++
			if winStreak > maxStreak {
				maxStreak = winStreak
			}
		} else {
			totalLosses++
			winStreak = 0
		}
		if r.Kills > bestKills {
			bestKills = r.Kills
		}
		if r.KDA != nil && *r.KDA > bestKDA {
			bestKDA = *r.KDA
		}
	}

	n := len(rows)
	ov := domain.SynthesisOverview{
		TotalMatches:     n,
		TotalWins:        totalWins,
		TotalLosses:      totalLosses,
		TotalKills:       totalKills,
		TotalDeaths:      totalDeaths,
		WinRate:          soloKPIs.WinRate,
		LongestWinStreak: maxStreak,
	}
	if soloKPIs.KDRatio != nil {
		ov.AvgKDA = soloKPIs.KDRatio
	}
	if n > 0 {
		avgKills := float64(totalKills) / float64(n)
		avgDeaths := float64(totalDeaths) / float64(n)
		ov.AvgKills = &avgKills
		ov.AvgDeaths = &avgDeaths
		// TotalKDR canonique (P2.5, ADR 0006) — debloque suppression du
		// recompute SynthesisPage.tsx:139-141 (B3, sum/sum mathematiquement faux).
		totalKDR := analysis.KDR(totalKills, totalDeaths)
		ov.TotalKDR = &totalKDR
	}
	if soloKPIs.PerformanceScore != nil {
		ov.AvgPerfScore = soloKPIs.PerformanceScore
	}
	if bestKills > 0 {
		ov.BestKillsMatch = &bestKills
	}
	if bestKDA > 0 {
		ov.BestKDAMatch = &bestKDA
	}
	return ov
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
	return fmt.Sprintf("%d matchs — %s", matchCount, label)
}

// =============================================================================
// Helpers previews D5/D6/D7
// =============================================================================

const highlightTopN = 5

// buildHighlightsPreview construit les top/pire matchs depuis les matchs filtrés.
// Tri en place sur des copies — pas de mutation des slices partagés.
func buildHighlightsPreview(rows []domain.SynthesisMatchRow) domain.SynthesisHighlightsPreview {
	if len(rows) == 0 {
		return domain.SynthesisHighlightsPreview{}
	}
	toHighlight := func(r domain.SynthesisMatchRow) domain.SynthesisMatchHighlight {
		return domain.SynthesisMatchHighlight{
			MatchID:   r.MatchID,
			Kills:     r.Kills,
			Deaths:    r.Deaths,
			KDA:       r.KDA,
			Outcome:   r.Outcome,
			PerfScore: r.PerformanceScore,
		}
	}

	topByKills := topNByFunc(rows, highlightTopN, func(a, b domain.SynthesisMatchRow) bool {
		return a.Kills > b.Kills
	})
	topByKDA := topNByFunc(rows, highlightTopN, func(a, b domain.SynthesisMatchRow) bool {
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
	worstByDeaths := topNByFunc(rows, highlightTopN, func(a, b domain.SynthesisMatchRow) bool {
		return a.Deaths > b.Deaths
	})

	toSlice := func(src []domain.SynthesisMatchRow) []domain.SynthesisMatchHighlight {
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
func topNByFunc(rows []domain.SynthesisMatchRow, n int, less func(a, b domain.SynthesisMatchRow) bool) []domain.SynthesisMatchRow {
	cp := make([]domain.SynthesisMatchRow, len(rows))
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

// buildRivalriesPreview construit les previews encounters depuis les données brutes.
func buildRivalriesPreview(rows []domain.EncounterRawRow) domain.SynthesisRivalriesPreview {
	if len(rows) == 0 {
		return domain.SynthesisRivalriesPreview{}
	}
	toPreview := func(r domain.EncounterRawRow) domain.SynthesisEncounterPreview {
		return domain.SynthesisEncounterPreview{
			XUID:       r.XUID,
			Gamertag:   r.Gamertag,
			MatchCount: r.MatchCount,
			AsTeammate: r.AsTeammate,
			AsEnemy:    r.AsEnemy,
			AvgKDA:     r.AvgKDA,
		}
	}

	var teammates, enemies []domain.SynthesisEncounterPreview
	for _, r := range rows {
		if r.AsTeammate > r.AsEnemy {
			teammates = append(teammates, toPreview(r))
		} else {
			enemies = append(enemies, toPreview(r))
		}
		if len(teammates) >= 5 && len(enemies) >= 5 {
			break
		}
	}
	if len(teammates) > 5 {
		teammates = teammates[:5]
	}
	if len(enemies) > 5 {
		enemies = enemies[:5]
	}
	return domain.SynthesisRivalriesPreview{
		TopTeammates: teammates,
		TopEnemies:   enemies,
		Total:        len(rows),
	}
}

// buildBreakdowns agrège les données heatmap en breakdowns carte et mode.
func buildBreakdowns(rows []domain.SynthesisHeatmapRow) domain.SynthesisBreakdowns {
	if len(rows) == 0 {
		return domain.SynthesisBreakdowns{}
	}

	mapAgg := map[string][2]int{}  // map_name → [match_count, wins]
	modeAgg := map[string][2]int{} // mode_name → [match_count, wins]
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
	// tri par MatchCount desc (sélection partielle des top 10)
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

func sortMapEntries(s []domain.SynthesisMapEntry) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j].MatchCount > s[i].MatchCount {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

func sortModeEntries(s []domain.SynthesisModeEntry) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j].MatchCount > s[i].MatchCount {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// =============================================================================
// P4.3 (ADR 0011) : helpers canonical (le converter SynthesisMatchRow est retiré)
// =============================================================================

// filterSynthesisByPeriodCanonical est la variante canonical de
// filterSynthesisByPeriod. Logique strictement identique.
func filterSynthesisByPeriodCanonical(
	rows []canonical.PlayerMatchRow,
	period string,
	_ domain.FilterContextInput,
) ([]canonical.PlayerMatchRow, []string, []string) {
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

	filtered := make([]canonical.PlayerMatchRow, 0, len(rows))
	for _, r := range rows {
		if !r.Summary.StartedAtUTC.Before(*cutoff) {
			filtered = append(filtered, r)
		}
	}
	return filtered, applied, ignored
}

// buildSynthesisOverviewCanonical est la variante canonical de
// buildSynthesisOverview. Lit Self.Kills/Deaths/Outcome/KDA depuis
// canonical au lieu de SynthesisMatchRow.{Kills,Deaths,Outcome,KDA}.
func buildSynthesisOverviewCanonical(rows []canonical.PlayerMatchRow, soloKPIs domain.SynthesisKPIs) domain.SynthesisOverview {
	var totalKills, totalDeaths, totalWins, totalLosses int
	var bestKills int
	var bestKDA float64
	var winStreak, maxStreak int

	for _, r := range rows {
		k := 0
		if r.Self.Kills != nil {
			k = *r.Self.Kills
		}
		d := 0
		if r.Self.Deaths != nil {
			d = *r.Self.Deaths
		}
		totalKills += k
		totalDeaths += d
		if r.Self.Outcome == canonical.OutcomeWin {
			totalWins++
			winStreak++
			if winStreak > maxStreak {
				maxStreak = winStreak
			}
		} else {
			totalLosses++
			winStreak = 0
		}
		if k > bestKills {
			bestKills = k
		}
		if r.Self.KDA != nil && *r.Self.KDA > bestKDA {
			bestKDA = *r.Self.KDA
		}
	}

	n := len(rows)
	ov := domain.SynthesisOverview{
		TotalMatches:     n,
		TotalWins:        totalWins,
		TotalLosses:      totalLosses,
		TotalKills:       totalKills,
		TotalDeaths:      totalDeaths,
		WinRate:          soloKPIs.WinRate,
		LongestWinStreak: maxStreak,
	}
	if soloKPIs.KDRatio != nil {
		ov.AvgKDA = soloKPIs.KDRatio
	}
	if n > 0 {
		avgKills := float64(totalKills) / float64(n)
		avgDeaths := float64(totalDeaths) / float64(n)
		ov.AvgKills = &avgKills
		ov.AvgDeaths = &avgDeaths
		totalKDR := analysis.KDR(totalKills, totalDeaths)
		ov.TotalKDR = &totalKDR
	}
	if soloKPIs.PerformanceScore != nil {
		ov.AvgPerfScore = soloKPIs.PerformanceScore
	}
	if bestKills > 0 {
		ov.BestKillsMatch = &bestKills
	}
	if bestKDA > 0 {
		ov.BestKDAMatch = &bestKDA
	}
	return ov
}

// buildHighlightsPreviewCanonical est la variante canonical de
// buildHighlightsPreview. Top/pire matchs sur les mêmes critères
// (kills DESC, KDA DESC, deaths DESC).
func buildHighlightsPreviewCanonical(rows []canonical.PlayerMatchRow) domain.SynthesisHighlightsPreview {
	if len(rows) == 0 {
		return domain.SynthesisHighlightsPreview{}
	}
	toHighlight := func(r canonical.PlayerMatchRow) domain.SynthesisMatchHighlight {
		k, d := 0, 0
		if r.Self.Kills != nil {
			k = *r.Self.Kills
		}
		if r.Self.Deaths != nil {
			d = *r.Self.Deaths
		}
		// Outcome canonical → int Halo pour le DTO inchangé.
		var outcome int
		switch r.Self.Outcome {
		case canonical.OutcomeWin:
			outcome = domain.OutcomeWin
		case canonical.OutcomeLoss:
			outcome = domain.OutcomeLoss
		case canonical.OutcomeTie:
			outcome = domain.OutcomeDraw
		case canonical.OutcomeDNF:
			outcome = domain.OutcomeDNF
		}
		return domain.SynthesisMatchHighlight{
			MatchID:   r.Summary.MatchID,
			Kills:     k,
			Deaths:    d,
			KDA:       r.Self.KDA,
			Outcome:   outcome,
			PerfScore: r.Enrichment.PerformanceScore,
		}
	}

	topByKills := topNByFuncCanonical(rows, highlightTopN, func(a, b canonical.PlayerMatchRow) bool {
		ak, bk := 0, 0
		if a.Self.Kills != nil {
			ak = *a.Self.Kills
		}
		if b.Self.Kills != nil {
			bk = *b.Self.Kills
		}
		return ak > bk
	})
	topByKDA := topNByFuncCanonical(rows, highlightTopN, func(a, b canonical.PlayerMatchRow) bool {
		av := 0.0
		if a.Self.KDA != nil {
			av = *a.Self.KDA
		}
		bv := 0.0
		if b.Self.KDA != nil {
			bv = *b.Self.KDA
		}
		return av > bv
	})
	worstByDeaths := topNByFuncCanonical(rows, highlightTopN, func(a, b canonical.PlayerMatchRow) bool {
		ad, bd := 0, 0
		if a.Self.Deaths != nil {
			ad = *a.Self.Deaths
		}
		if b.Self.Deaths != nil {
			bd = *b.Self.Deaths
		}
		return ad > bd
	})

	toSlice := func(src []canonical.PlayerMatchRow) []domain.SynthesisMatchHighlight {
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

// topNByFuncCanonical est la variante canonical de topNByFunc.
func topNByFuncCanonical(rows []canonical.PlayerMatchRow, n int, less func(a, b canonical.PlayerMatchRow) bool) []canonical.PlayerMatchRow {
	cp := make([]canonical.PlayerMatchRow, len(rows))
	copy(cp, rows)
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
