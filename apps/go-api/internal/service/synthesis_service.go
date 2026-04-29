// Package service â€” synthesis_service.go : orchestration de la page SynthÃ¨se.
//
// Sprint 55 D1 : extrait de squad_service.go â€” SynthesisService devient autonome,
// implÃ©mente port.SynthesisService.
//
// Sprint 55 D2 : period et filters du SynthesisRequest sont rÃ©ellement appliquÃ©s.
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
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// SynthesisService orchestre les donnÃ©es de la page SynthÃ¨se.
type SynthesisService struct {
	repo port.SynthesisRepository
	// dataAdapter (optionnel, Phase 2 plan finition multi-titres) :
	// quand fourni, GetSynthesisPage mesure la capability match.history pour
	// loguer une Ã©ventuelle dÃ©gradation.
	dataAdapter games.TitleDataAdapter
	// playerMatchesRepo (P4.1+P4.3, ADR 0011) : source canonical-aware. Quand
	// fournie avec titleSlug+gamertag, GetSynthesisPage charge directement
	// `[]canonical.PlayerMatchRow` et appelle les analyses *FromCanonical sans
	// converter. Le path legacy (s.repo.LoadSynthesisMatches) reste pour
	// rÃ©trocompatibilitÃ© tant que la DI cabling n'est pas mise Ã  jour partout.
	playerMatchesRepo port.PlayerMatchesRepository
	// titleSlug est nÃ©cessaire pour appeler PlayerMatchesRepo.LoadPlayerMatches.
	// Si "" et playerMatchesRepo != nil, fallback sur le repo legacy.
	titleSlug string
	gamertag  string
}

// NewSynthesisService crÃ©e un SynthesisService avec le repository injectÃ©.
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
// loader unifiÃ© et appelle les analyses *FromCanonical (pas de converter).
func (s *SynthesisService) WithPlayerMatchesRepo(
	repo port.PlayerMatchesRepository,
	titleSlug, gamertag string,
) *SynthesisService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	s.gamertag = gamertag
	return s
}

// GetSynthesisPage construit la rÃ©ponse de la page SynthÃ¨se.
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
	// quand un DataAdapter est injectÃ©. Sert Ã  mesurer la dÃ©gradation potentielle
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

	// P4.3 finale (ADR 0011) : path canonical exclusif. Le legacy fallback
	// path a Ã©tÃ© supprimÃ© â€” playerMatchesRepo + titleSlug + gamertag sont
	// dÃ©sormais REQUIS (wirÃ©s universellement en DI via registry.go).
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return nil, fmt.Errorf("SynthesisService: PlayerMatchesRepo non cÃ¢blÃ© (P4.3 finale exige le wiring DI)")
	}
	canonicalRows, err := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	if err != nil {
		return nil, fmt.Errorf("SynthesisService load: %w", err)
	}
	slog.DebugContext(ctx, "synthesis: loaded canonical",
		"rows", len(canonicalRows), "title_slug", s.titleSlug)
	filteredCanon, filtersApplied, filtersIgnored := filterSynthesisByPeriodCanonical(canonicalRows, period, req.Filters)

	soloKPIs := analysis.ComputeSynthesisKPIsFromCanonical(filteredCanon, false)
	squadKPIs := analysis.ComputeSynthesisKPIsFromCanonical(filteredCanon, true)
	topWeeks := analysis.ComputeSynthesisTopWeeksFromCanonical(filteredCanon)
	heatmap := analysis.ComputeTemporalHeatmapFromCanonical(filteredCanon)
	overview := buildSynthesisOverviewCanonical(filteredCanon, soloKPIs)
	highlights := buildHighlightsPreviewCanonical(filteredCanon)
	matchCount := len(filteredCanon)
	comparison := analysis.ComputeComparisonMetrics(soloKPIs, squadKPIs)

	// D6 : rivalries â€” encounters depuis shared (requÃªte sÃ©parÃ©e)
	encounters, _ := s.repo.LoadEncounters(ctx, playerXUID) // erreur non fatale
	rivalries := buildRivalriesPreview(encounters)

	// D7 : breakdowns map/mode depuis la heatmap (requÃªte sÃ©parÃ©e)
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

// filterSynthesisByPeriod filtre les matchs SynthÃ¨se selon la pÃ©riode demandÃ©e.
// Retourne les matchs filtrÃ©s, les filtres appliquÃ©s et ceux ignorÃ©s.
func filterSynthesisByPeriod(
	rows []legacymatch.SynthesisMatchRow,
	period string,
	_ domain.FilterContextInput, // filtres avancÃ©s â€” Ã  implÃ©menter aprÃ¨s backfill de map/mode
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
		// "all" â€” pas de filtre temporel
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

// buildSynthesisOverview calcule les cumuls, moyennes et pics depuis les matchs filtrÃ©s.
func buildSynthesisOverview(rows []legacymatch.SynthesisMatchRow, soloKPIs domain.SynthesisKPIs) domain.SynthesisOverview {
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
		// TotalKDR canonique (P2.5, ADR 0006) â€” debloque suppression du
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
		"2y":  "2 derniÃ¨res annÃ©es",
	}
	label, ok := labels[period]
	if !ok {
		label = period
	}
	return fmt.Sprintf("%d matchs â€” %s", matchCount, label)
}

// =============================================================================
// Helpers previews D5/D6/D7
// =============================================================================

const highlightTopN = 5

// buildHighlightsPreview construit les top/pire matchs depuis les matchs filtrÃ©s.
// Tri en place sur des copies â€” pas de mutation des slices partagÃ©s.
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

// buildRivalriesPreview construit les previews encounters depuis les donnÃ©es brutes.
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

// buildBreakdowns agrÃ¨ge les donnÃ©es heatmap en breakdowns carte et mode.
func buildBreakdowns(rows []domain.SynthesisHeatmapRow) domain.SynthesisBreakdowns {
	if len(rows) == 0 {
		return domain.SynthesisBreakdowns{}
	}

	mapAgg := map[string][2]int{}  // map_name â†’ [match_count, wins]
	modeAgg := map[string][2]int{} // mode_name â†’ [match_count, wins]
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
// P4.3 (ADR 0011) : helpers canonical (le converter SynthesisMatchRow est retirÃ©)
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
		// "all" â€” pas de filtre temporel
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
// buildHighlightsPreview. Top/pire matchs sur les mÃªmes critÃ¨res
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
		// Outcome canonical â†’ int Halo pour le DTO inchangÃ©.
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
