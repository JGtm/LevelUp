// Package service â€” TimeseriesService : endpoint POST /pages/timeseries (contrat FastAPI).
//
// Sprint 33 : adapte les donnÃ©es StatsService vers le contrat TimeseriesPageResponse.
//
// Architecture data-only : le Go ne gÃ©nÃ¨re pas de figures Plotly. Le frontend
// React reconstruit les visualisations via les wrappers ECharts Ã  partir des
// data points bruts fournis dans les onglets (cumulative_kd, ewma_kd_points,
// kda_buckets, correlation_points, heatmap_data, etc.).
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// TimeseriesService construit la rÃ©ponse timeseries au format FastAPI.
type TimeseriesService struct {
	statsRepo port.StatsRepository
	// dataAdapter (optionnel, Phase C+ multi-titres) : point d'extension pour
	// router LoadTimeseries via la couche canonique. Ã€ ce jour, le service
	// utilise le repo direct car canonical.MetricSeries ne couvre pas encore
	// la totalitÃ© du payload (5 onglets : win_loss/accuracy/objective/form/
	// lusr). Le hook est en place pour permettre une bascule incrÃ©mentale.
	dataAdapter games.TitleDataAdapter
	// playerMatchesRepo (P4.1, ADR 0011) : loader canonical-aware optionnel.
	// Quand fourni avec titleSlug + gamertag, GetPage charge canonical et
	// convertit via statsMatchRowFromCanonical (partagÃ© avec stats_service).
	// TODO P4.3 : retirer le converter quand les analyses timeseries
	// (buildCumulTab, computeRegressionStats, etc.) consommeront canonical.
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string
	gamertag          string
	// weaponKillsRepo (chart .04 Top weapons) : optionnel, dégradation gracieuse.
	// Si nil, TopWeapons reste vide.
	weaponKillsRepo port.WeaponKillsRepository
	// highlightEventsRepo (chart .11 Premier événement) : optionnel, dégradation
	// gracieuse. Si nil ou xuid vide, FirstEvents reste nil.
	//
	// Interface locale ciblée — le service n'a besoin que de Load(filters), pas
	// de la signature multi-titres LoadHighlightEvents(slug, filters) du port
	// (slug déjà fixé par PlayerDB).
	highlightEventsRepo highlightEventsLoader
	playerXUID          string
}

// highlightEventsLoader expose la sous-API du HighlightEventsRepo per-player
// utilisée par TimeseriesService (cf. port pattern, segregation).
type highlightEventsLoader interface {
	Load(ctx context.Context, filters port.HighlightEventFilters) ([]canonical.HighlightEvent, error)
}

// NewTimeseriesService crÃ©e un TimeseriesService.
func NewTimeseriesService(repo port.StatsRepository) *TimeseriesService {
	return &TimeseriesService{statsRepo: repo}
}

// WithDataAdapter injecte le DataAdapter multi-titres pour activer une
// future bascule LoadTimeseries. DÃ©gradation gracieuse si nil.
func (s *TimeseriesService) WithDataAdapter(a games.TitleDataAdapter) *TimeseriesService {
	s.dataAdapter = a
	return s
}

// WithPlayerMatchesRepo (P4.1, ADR 0011) injecte le loader canonical-aware.
func (s *TimeseriesService) WithPlayerMatchesRepo(repo port.PlayerMatchesRepository, titleSlug, gamertag string) *TimeseriesService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	s.gamertag = gamertag
	return s
}

// WithWeaponKillsRepo injecte le repo weapon_kills pour alimenter le chart .04
// (Top weapons by kills). Optionnel : si non câblé, TopWeapons reste vide.
func (s *TimeseriesService) WithWeaponKillsRepo(repo port.WeaponKillsRepository) *TimeseriesService {
	s.weaponKillsRepo = repo
	return s
}

// WithHighlightEventsRepo injecte le repo highlight_events pour alimenter le
// chart .11 (Premier événement). xuid est le XUID du joueur principal — sert à
// distinguer kills/deaths dans narrative.ComputeFirstEventsPerMatch.
func (s *TimeseriesService) WithHighlightEventsRepo(repo highlightEventsLoader, xuid string) *TimeseriesService {
	s.highlightEventsRepo = repo
	s.playerXUID = xuid
	return s
}

// GetPage construit la rÃ©ponse complÃ¨te avec 5 onglets.
func (s *TimeseriesService) GetPage(
	ctx context.Context,
	req domain.TimeseriesQueryRequest,
) (domain.TimeseriesPageResponse, error) {
	defer func(start time.Time) {
		observability.RecordDurationMS("timeseries_get_page", time.Since(start).Milliseconds())
	}(time.Now())
	// P4.3 finale (ADR 0011) : path canonical exclusif.
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return domain.TimeseriesPageResponse{}, fmt.Errorf("TimeseriesService: PlayerMatchesRepo non cÃ¢blÃ© (P4.3 finale exige le wiring DI)")
	}
	canonicalRows, err := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	if err != nil {
		slog.ErrorContext(ctx, "timeseries: chargement canonical", "error", err)
		return domain.TimeseriesPageResponse{}, fmt.Errorf("TimeseriesService: %w", err)
	}
	slog.DebugContext(ctx, "timeseries: loaded canonical",
		"rows", len(canonicalRows), "title_slug", s.titleSlug)
	allMatches := analysis.StatsMatchRowsFromCanonical(canonicalRows)

	matches := filterStatsMatchRows(allMatches, req.Filters)
	slog.DebugContext(ctx, "timeseries: matches chargÃ©s",
		"total", len(allMatches),
		"apres_filtres", len(matches),
	)

	// Population historique (mode solo, sans filtre cascade/period/sessions) —
	// sert à enrichir le map_breakdown avec les références "Historique".
	historicalSolo := filterStatsMatchRowsByContext(allMatches, req.Filters.MatchContext)

	resp := domain.TimeseriesPageResponse{
		TotalMatches:     len(matches),
		MatchRows:        buildMatchRows(matches),
		SummaryTab:       buildTimeseriesSummaryTab(matches),
		CumulTab:         buildCumulTab(matches),
		FormTab:          buildTimeseriesFormTab(matches),
		IntensityTab:     buildIntensityTab(matches),
		DistributionsTab: buildDistributionsTab(matches),
		OutcomesOverTime: buildOutcomesOverTime(matches),
		MapBreakdown:     buildSoloMapBreakdown(matches, historicalSolo),
		SoloSessionPerf:  buildSoloSessionPerf(historicalSolo),
		TopWeapons:       []domain.TimeseriesWeaponKill{},
	}

	// Top weapons (chart .04) : dégradation gracieuse si weaponKillsRepo nil
	// ou si le titre ne supporte pas la capability.
	if s.weaponKillsRepo != nil && len(matches) > 0 && s.gamertag != "" {
		matchIDs := make([]string, 0, len(matches))
		for _, m := range matches {
			matchIDs = append(matchIDs, m.MatchID)
		}
		filters := port.WeaponKillFilters{
			MatchIDs:            matchIDs,
			Gamertag:            s.gamertag,
			IncludeGrenadeMelee: false,
		}
		if err := filters.Validate(); err == nil {
			rows, err := s.weaponKillsRepo.LoadWeaponKillsAggregated(ctx, s.titleSlug, filters)
			if err != nil {
				slog.WarnContext(ctx, "timeseries: top weapons load failed", "err", err)
			} else {
				resp.TopWeapons = buildTopWeapons(rows, 10)
			}
		}
	}

	// First events distribution (chart .11) + Intensity heatmap : dégradation
	// gracieuse si repo nil ou xuid manquant. Une seule charge events.
	if s.highlightEventsRepo != nil && s.playerXUID != "" && len(matches) > 0 {
		matchIDs := make([]string, 0, len(matches))
		for _, m := range matches {
			matchIDs = append(matchIDs, m.MatchID)
		}
		events, err := s.loadHighlightEvents(ctx, matchIDs)
		if err != nil {
			slog.WarnContext(ctx, "timeseries: highlight events load failed", "err", err)
		} else if len(events) > 0 {
			resp.FirstEvents = buildFirstEventsDistribution(
				narrative.ComputeFirstEventsPerMatch(events, s.playerXUID, matchIDs),
			)
			resp.IntensityRows = buildIntensityRows(events, matches, s.playerXUID)
		}
	}

	// BriefingKPIs : KPIs sur les rows canoniques filtres (memes match_ids que
	// matches). Alimente le composant <SessionBriefing> en mode solo. Reutilise
	// ComputeKPIStats sans re-filtrer les filtres metier.
	if filtered := filterCanonicalByMatchIDs(canonicalRows, matches); len(filtered) > 0 {
		briefingKPIs := analysis.ComputeKPIStats(filtered)
		resp.BriefingKPIs = &briefingKPIs
	}

	return resp, nil
}

// ---------------------------------------------------------------------------
// First events distribution (chart .11)
// ---------------------------------------------------------------------------

// loadHighlightEvents charge les events bruts (kill / death / first_kill /
// first_death) pour les match_ids fournis. Source unique partagée par chart
// .11 (premier événement) et le heatmap d'intensité.
func (s *TimeseriesService) loadHighlightEvents(
	ctx context.Context, matchIDs []string,
) ([]canonical.HighlightEvent, error) {
	filters := port.HighlightEventFilters{
		MatchIDs: matchIDs,
		EventTypes: []canonical.HighlightEventType{
			canonical.EventKill,
			canonical.EventDeath,
			canonical.EventFirstKill,
			canonical.EventFirstDeath,
		},
	}
	if err := filters.Validate(); err != nil {
		return nil, err
	}
	return s.highlightEventsRepo.Load(ctx, filters)
}

// buildIntensityRows agrège les frags du joueur (events où KillerXUID == xuid)
// en 10 buckets normalisés [0..1] par match. Réutilise narrative.ComputeMatchIntensityProfiles
// + NormalizeIntensityBuckets sur les events filtrés solo.
//
// Le label affiché côté front est "Map — dd/MM" (lookup depuis matches).
// matchOrder : préserve l'ordre des matches du scope (latest first comme
// LoadPlayerMatches), idem squad SquadIntensityProfile.
func buildIntensityRows(
	events []canonical.HighlightEvent,
	matches []legacymatch.StatsMatchRow,
	playerXUID string,
) []domain.IntensityMatchRow {
	// Filtrer les events où le joueur est tueur (frags du joueur uniquement).
	playerKills := make([]canonical.HighlightEvent, 0, len(events))
	for _, ev := range events {
		killer := ""
		if ev.KillerXUID != nil {
			killer = *ev.KillerXUID
		} else {
			killer = ev.XUID // fallback legacy : XUID = tueur sur kill events
		}
		if killer == playerXUID && (ev.EventType == string(canonical.EventKill) ||
			ev.EventType == string(canonical.EventFirstKill)) {
			playerKills = append(playerKills, ev)
		}
	}
	if len(playerKills) == 0 {
		return nil
	}
	profiles := narrative.ComputeMatchIntensityProfiles(playerKills, 10)
	if len(profiles) == 0 {
		return nil
	}

	// Index match_rows pour récupérer label (map_name + date).
	type matchMeta struct {
		label    string
		startUTC time.Time
	}
	metaByID := make(map[string]matchMeta, len(matches))
	for _, m := range matches {
		mapName := m.MapNameFR
		if mapName == "" {
			mapName = m.MapName
		}
		date := m.StartTime.Format("02/01")
		metaByID[m.MatchID] = matchMeta{
			label:    fmt.Sprintf("%s — %s", mapName, date),
			startUTC: m.StartTime,
		}
	}

	out := make([]domain.IntensityMatchRow, 0, len(profiles))
	for _, p := range profiles {
		normalized := narrative.NormalizeIntensityBuckets(p.Buckets)
		var phases [10]float64
		for i := 0; i < 10 && i < len(normalized); i++ {
			phases[i] = normalized[i]
		}
		row := domain.IntensityMatchRow{
			MatchID: p.MatchID,
			Phases:  phases,
		}
		if meta, ok := metaByID[p.MatchID]; ok {
			row.Label = meta.label
		} else {
			row.Label = p.MatchID
		}
		out = append(out, row)
	}
	// Tri chronologique (plus récent en premier — DESC) cohérent avec
	// match_rows et LoadPlayerMatches (ORDER BY start_time DESC).
	sort.SliceStable(out, func(i, j int) bool {
		mi := metaByID[out[i].MatchID]
		mj := metaByID[out[j].MatchID]
		return mi.startUTC.After(mj.startUTC)
	})
	return out
}

// buildFirstEventsDistribution agrège les premiers timings en buckets de 10
// secondes + calcule les moyennes (markLines).
func buildFirstEventsDistribution(rows []narrative.FirstEventsRow) *domain.FirstEventDistribution {
	const binWidthSec = 10.0
	type acc struct {
		kills, deaths int
	}
	buckets := make(map[int]*acc)
	var killSum, deathSum float64
	var killN, deathN int
	for _, r := range rows {
		if r.FirstKillMS != nil {
			sec := float64(*r.FirstKillMS) / 1000.0
			idx := int(sec / binWidthSec)
			if _, ok := buckets[idx]; !ok {
				buckets[idx] = &acc{}
			}
			buckets[idx].kills++
			killSum += sec
			killN++
		}
		if r.FirstDeathMS != nil {
			sec := float64(*r.FirstDeathMS) / 1000.0
			idx := int(sec / binWidthSec)
			if _, ok := buckets[idx]; !ok {
				buckets[idx] = &acc{}
			}
			buckets[idx].deaths++
			deathSum += sec
			deathN++
		}
	}
	if len(buckets) == 0 {
		return nil
	}
	idxs := make([]int, 0, len(buckets))
	for i := range buckets {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)
	out := make([]domain.FirstEventBucket, 0, len(idxs))
	for _, i := range idxs {
		out = append(out, domain.FirstEventBucket{
			LowerSeconds: float64(i) * binWidthSec,
			UpperSeconds: float64(i+1) * binWidthSec,
			FirstKills:   buckets[i].kills,
			FirstDeaths:  buckets[i].deaths,
		})
	}
	dist := &domain.FirstEventDistribution{Buckets: out}
	if killN > 0 {
		v := math.Round((killSum/float64(killN))*10) / 10
		dist.MeanFirstKillSeconds = &v
	}
	if deathN > 0 {
		v := math.Round((deathSum/float64(deathN))*10) / 10
		dist.MeanFirstDeathSeconds = &v
	}
	return dist
}

// ---------------------------------------------------------------------------
// Map breakdown (charts teammates.02 + teammates.13 — page Stats Solo)
// ---------------------------------------------------------------------------

// filterStatsMatchRowsByContext applique uniquement le filtre match_context
// (solo/squad/all) sur des StatsMatchRow — sans cascade, period ni sessions.
// Sert à dériver la population "Historique" pour le map_breakdown.
func filterStatsMatchRowsByContext(rows []legacymatch.StatsMatchRow, ctx string) []legacymatch.StatsMatchRow {
	switch ctx {
	case domain.MatchContextSolo:
		out := make([]legacymatch.StatsMatchRow, 0, len(rows))
		for _, r := range rows {
			if !r.IsWithFriends {
				out = append(out, r)
			}
		}
		return out
	case domain.MatchContextSquad:
		out := make([]legacymatch.StatsMatchRow, 0, len(rows))
		for _, r := range rows {
			if r.IsWithFriends {
				out = append(out, r)
			}
		}
		return out
	}
	return rows
}

// buildSoloMapBreakdown agrège les stats par carte sur la session courante
// (matches filtrés) et enrichit chaque ligne avec les références historiques
// (historicalSolo : tous les matchs solo, sans cascade/period/sessions).
//
// Symétrie avec computeMapBreakdown + enrichMapBreakdownWithSquadStats du
// service teammates, adapté à StatsMatchRow + sans notion d'escouade.
//
// Clé d'agrégation : MapNameFR (priorité) puis MapName (fallback EN). Une
// carte présente dans la session sans donnée historique reste sans
// historical_* (la cellule front affiche alors "—" ou se masque).
func buildSoloMapBreakdown(current, historicalSolo []legacymatch.StatsMatchRow) []domain.MapBreakdownRow {
	if len(current) == 0 {
		return []domain.MapBreakdownRow{}
	}
	currentByMap := aggregateMapStats(current)
	historicalByMap := aggregateMapStats(historicalSolo)

	out := make([]domain.MapBreakdownRow, 0, len(currentByMap))
	for key, s := range currentByMap {
		if s.count == 0 {
			continue
		}
		row := domain.MapBreakdownRow{
			MapID:      key,
			MapUI:      s.label,
			MatchCount: s.count,
			WinRate:    round2(float64(s.wins) / float64(s.count)),
		}
		if s.perfCount > 0 {
			avg := round2(s.perfSum / float64(s.perfCount))
			row.PerformanceAvg = &avg
		}
		if h, ok := historicalByMap[key]; ok && h.count > 0 {
			hwr := round2(float64(h.wins) / float64(h.count))
			row.HistoricalWinRate = &hwr
			if h.perfCount > 0 {
				hperf := round2(h.perfSum / float64(h.perfCount))
				row.HistoricalPerformanceAvg = &hperf
			}
		}
		out = append(out, row)
	}
	// Tri stable par count desc puis label asc — confort UX (top maps en haut).
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].MatchCount != out[j].MatchCount {
			return out[i].MatchCount > out[j].MatchCount
		}
		return out[i].MapUI < out[j].MapUI
	})
	return out
}

type mapAgg struct {
	label       string
	count, wins int
	perfSum     float64
	perfCount   int
}

func aggregateMapStats(rows []legacymatch.StatsMatchRow) map[string]*mapAgg {
	m := make(map[string]*mapAgg, 16)
	for _, r := range rows {
		label := r.MapNameFR
		if label == "" {
			label = r.MapName
		}
		if label == "" {
			label = "Unknown"
		}
		// Clé = label affiché (StatsMatchRow n'expose pas de map_id).
		// Cohérent avec le fallback de computeMapBreakdown côté squad.
		key := label
		if _, ok := m[key]; !ok {
			m[key] = &mapAgg{label: label}
		}
		m[key].count++
		if r.Outcome != nil && *r.Outcome == analysis.OutcomeWin {
			m[key].wins++
		}
		if r.PerfScoreComputed != nil {
			m[key].perfSum += *r.PerfScoreComputed
			m[key].perfCount++
		}
	}
	return m
}

// ---------------------------------------------------------------------------
// Solo session performance (Synthèse — agrégat par session sur tout l'historique)
// ---------------------------------------------------------------------------

// buildSoloSessionPerf agrège les matchs solo (sans filtre period/session/
// cascade) par session_label, puis re-agrège en semaines/mois si la densité
// devient illisible.
//
// Granularité adaptative :
//   - ≤ 30 sessions : session par session (label original)
//   - 31..30 semaines : groupe par semaine ISO (label "2026-S12")
//   - sinon : groupe par mois (label "Jan 26")
func buildSoloSessionPerf(rows []legacymatch.StatsMatchRow) *domain.SoloSessionPerfBlock {
	if len(rows) == 0 {
		return nil
	}
	const sessionCap = 30
	const weekCap = 30

	sessionPts := aggregateSessions(rows)
	if len(sessionPts) == 0 {
		return nil
	}
	if len(sessionPts) <= sessionCap {
		return &domain.SoloSessionPerfBlock{Granularity: "session", Points: sessionPts}
	}
	weekPts := rollupSessionPoints(sessionPts, "week")
	if len(weekPts) <= weekCap {
		return &domain.SoloSessionPerfBlock{Granularity: "week", Points: weekPts}
	}
	monthPts := rollupSessionPoints(sessionPts, "month")
	return &domain.SoloSessionPerfBlock{Granularity: "month", Points: monthPts}
}

// aggregateSessions agrège les rows par session_label (granularité finest).
func aggregateSessions(rows []legacymatch.StatsMatchRow) []domain.SoloSessionPerfPoint {
	type acc struct {
		label                            string
		earliest                         time.Time
		count, wins, perfCount, mmrCount int
		perfSum, mmrSum                  float64
	}
	bySession := make(map[string]*acc)
	for _, r := range rows {
		if r.SessionLabel == nil || *r.SessionLabel == "" {
			continue
		}
		lbl := *r.SessionLabel
		a, ok := bySession[lbl]
		if !ok {
			a = &acc{label: lbl, earliest: r.StartTime}
			bySession[lbl] = a
		}
		a.count++
		if r.Outcome != nil && *r.Outcome == domain.OutcomeWin {
			a.wins++
		}
		if r.PerfScoreComputed != nil {
			a.perfSum += *r.PerfScoreComputed
			a.perfCount++
		}
		if r.TeamMMR != nil {
			a.mmrSum += *r.TeamMMR
			a.mmrCount++
		}
		if r.StartTime.Before(a.earliest) {
			a.earliest = r.StartTime
		}
	}
	out := make([]domain.SoloSessionPerfPoint, 0, len(bySession))
	for _, a := range bySession {
		pt := domain.SoloSessionPerfPoint{
			SessionLabel: a.label,
			StartedAtUTC: a.earliest.Format(time.RFC3339),
			MatchCount:   a.count,
			Wins:         a.wins,
			WinRate:      round2(float64(a.wins) / float64(a.count)),
		}
		if a.perfCount > 0 {
			v := round2(a.perfSum / float64(a.perfCount))
			pt.PerfAvg = &v
		}
		if a.mmrCount > 0 {
			v := round2(a.mmrSum / float64(a.mmrCount))
			pt.TeamMMRAvg = &v
		}
		out = append(out, pt)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].StartedAtUTC < out[j].StartedAtUTC
	})
	return out
}

// rollupSessionPoints regroupe des SoloSessionPerfPoint par bucket temporel
// (week|month). Pondère perf/mmr par match_count pour préserver la
// représentativité des sessions denses.
func rollupSessionPoints(points []domain.SoloSessionPerfPoint, mode string) []domain.SoloSessionPerfPoint {
	type acc struct {
		label, startISO                          string
		earliest                                 time.Time
		count, wins, perfWeightedN, mmrWeightedN int
		perfWeightedSum, mmrWeightedSum          float64
	}
	keyFn := func(t time.Time) (string, string, time.Time) {
		switch mode {
		case "month":
			d := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			return d.Format("2006-01"), d.Format(time.RFC3339), d
		default: // "week"
			y, w := t.ISOWeek()
			jan4 := time.Date(y, 1, 4, 0, 0, 0, 0, time.UTC)
			start := jan4.AddDate(0, 0, -int(jan4.Weekday()-time.Monday)+(w-1)*7)
			return fmt.Sprintf("%d-S%02d", y, w), start.Format(time.RFC3339), start
		}
	}
	buckets := make(map[string]*acc)
	for _, p := range points {
		t, err := time.Parse(time.RFC3339, p.StartedAtUTC)
		if err != nil {
			continue
		}
		key, startISO, start := keyFn(t)
		a, ok := buckets[key]
		if !ok {
			a = &acc{label: key, startISO: startISO, earliest: start}
			buckets[key] = a
		}
		a.count += p.MatchCount
		a.wins += p.Wins
		if p.PerfAvg != nil && p.MatchCount > 0 {
			a.perfWeightedSum += *p.PerfAvg * float64(p.MatchCount)
			a.perfWeightedN += p.MatchCount
		}
		if p.TeamMMRAvg != nil && p.MatchCount > 0 {
			a.mmrWeightedSum += *p.TeamMMRAvg * float64(p.MatchCount)
			a.mmrWeightedN += p.MatchCount
		}
	}
	out := make([]domain.SoloSessionPerfPoint, 0, len(buckets))
	for _, a := range buckets {
		pt := domain.SoloSessionPerfPoint{
			SessionLabel: a.label,
			StartedAtUTC: a.startISO,
			MatchCount:   a.count,
			Wins:         a.wins,
			WinRate:      round2(float64(a.wins) / float64(a.count)),
		}
		if a.perfWeightedN > 0 {
			v := round2(a.perfWeightedSum / float64(a.perfWeightedN))
			pt.PerfAvg = &v
		}
		if a.mmrWeightedN > 0 {
			v := round2(a.mmrWeightedSum / float64(a.mmrWeightedN))
			pt.TeamMMRAvg = &v
		}
		out = append(out, pt)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].StartedAtUTC < out[j].StartedAtUTC
	})
	return out
}

// ---------------------------------------------------------------------------
// Top weapons (chart .04)
// ---------------------------------------------------------------------------

// buildTopWeapons trie par kills desc et retourne le top N.
func buildTopWeapons(rows []port.WeaponKillRow, topN int) []domain.TimeseriesWeaponKill {
	if len(rows) == 0 {
		return []domain.TimeseriesWeaponKill{}
	}
	type agg struct {
		label string
		kills int
	}
	byID := make(map[int64]*agg, len(rows))
	for _, r := range rows {
		if r.IsGrenadeMelee {
			continue
		}
		a, ok := byID[r.WeaponID]
		if !ok {
			a = &agg{label: r.Label}
			byID[r.WeaponID] = a
		}
		if a.label == "" && r.Label != "" {
			a.label = r.Label
		}
		a.kills += r.Kills
	}
	out := make([]domain.TimeseriesWeaponKill, 0, len(byID))
	for id, a := range byID {
		out = append(out, domain.TimeseriesWeaponKill{
			WeaponID: id,
			Label:    a.label,
			Kills:    a.kills,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kills != out[j].Kills {
			return out[i].Kills > out[j].Kills
		}
		return out[i].WeaponID < out[j].WeaponID
	})
	if len(out) > topN {
		out = out[:topN]
	}
	return out
}

// ---------------------------------------------------------------------------
// Outcomes over time (chart .05)
// ---------------------------------------------------------------------------

// buildOutcomesOverTime agrège les outcomes par période. La granularité est
// choisie selon la durée totale du scope : <=14j → day, <=120j → week, sinon
// month. Les périodes vides ne sont pas émises.
func buildOutcomesOverTime(matches []legacymatch.StatsMatchRow) []domain.OutcomesPeriodPoint {
	if len(matches) == 0 {
		return []domain.OutcomesPeriodPoint{}
	}
	first := matches[0].StartTime
	last := matches[len(matches)-1].StartTime
	if last.Before(first) {
		first, last = last, first
	}
	totalDays := int(last.Sub(first).Hours()/24) + 1
	type bucket struct {
		startDate  time.Time
		label      string
		w, l, t, d int
	}
	buckets := make(map[string]*bucket)
	keyFn := func(t time.Time) (string, time.Time, string) {
		switch {
		case totalDays <= 14:
			d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			return d.Format("2006-01-02"), d, d.Format("02 Jan")
		case totalDays <= 120:
			y, w := t.ISOWeek()
			// Lundi de la semaine ISO
			jan4 := time.Date(y, 1, 4, 0, 0, 0, 0, time.UTC)
			startISO := jan4.AddDate(0, 0, -int(jan4.Weekday()-time.Monday)+(w-1)*7)
			label := fmt.Sprintf("%d-W%02d", y, w)
			return label, startISO, label
		default:
			d := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			return d.Format("2006-01"), d, d.Format("Jan 06")
		}
	}
	for _, m := range matches {
		key, start, label := keyFn(m.StartTime)
		b, ok := buckets[key]
		if !ok {
			b = &bucket{startDate: start, label: label}
			buckets[key] = b
		}
		if m.Outcome == nil {
			b.d++
			continue
		}
		switch *m.Outcome {
		case domain.OutcomeWin:
			b.w++
		case domain.OutcomeLoss:
			b.l++
		case domain.OutcomeDraw:
			b.t++
		default:
			b.d++
		}
	}
	out := make([]domain.OutcomesPeriodPoint, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, domain.OutcomesPeriodPoint{
			PeriodLabel: b.label,
			StartDate:   b.startDate.Format("2006-01-02"),
			Wins:        b.w,
			Losses:      b.l,
			Ties:        b.t,
			DNF:         b.d,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].StartDate < out[j].StartDate
	})
	return out
}

// filterCanonicalByMatchIDs ne garde que les canonical rows dont le match_id
// figure dans le slice de StatsMatchRow filtre. Sert de pont entre la pipeline
// legacy (filterStatsMatchRows operant sur StatsMatchRow) et ComputeKPIStats
// (qui consomme du canonical.PlayerMatchRow).
func filterCanonicalByMatchIDs(canonicalRows []canonical.PlayerMatchRow, matches []legacymatch.StatsMatchRow) []canonical.PlayerMatchRow {
	if len(matches) == 0 || len(canonicalRows) == 0 {
		return nil
	}
	keep := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		keep[m.MatchID] = struct{}{}
	}
	out := make([]canonical.PlayerMatchRow, 0, len(matches))
	for _, r := range canonicalRows {
		if _, ok := keep[r.Summary.MatchID]; ok {
			out = append(out, r)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Onglet Summary
// ---------------------------------------------------------------------------

func buildTimeseriesSummaryTab(matches []legacymatch.StatsMatchRow) domain.TimeseriesSummaryTab {
	cards := make([]domain.TimeseriesKpiCard, 0, 6)
	n := len(matches)
	if n == 0 {
		return domain.TimeseriesSummaryTab{KpiCards: cards}
	}

	wins, totalKills, totalDeaths := 0, 0, 0
	accSum, accN := 0.0, 0
	for _, m := range matches {
		if m.Outcome != nil && *m.Outcome == analysis.OutcomeWin {
			wins++
		}
		totalKills += m.Kills
		totalDeaths += m.Deaths
		if m.Accuracy != nil {
			accSum += *m.Accuracy
			accN++
		}
	}

	// TODO P4 ADR 0006 : retirer *100 (convention API canonique 0..1).
	winRate := analysis.WinRate(wins, n) * 100
	kd := 0.0
	if totalDeaths > 0 {
		kd = float64(totalKills) / float64(totalDeaths)
	}

	cards = append(cards,
		domain.TimeseriesKpiCard{Key: "total_matches", Label: "Matchs", Value: fmt.Sprintf("%d", n)},
		domain.TimeseriesKpiCard{Key: "win_rate", Label: "Win Rate", Value: fmt.Sprintf("%.1f%%", winRate)},
		domain.TimeseriesKpiCard{Key: "kd_ratio", Label: "K/D", Value: fmt.Sprintf("%.2f", kd)},
		domain.TimeseriesKpiCard{Key: "kills_per_game", Label: "Kills/game", Value: fmt.Sprintf("%.1f", float64(totalKills)/float64(n))},
	)

	if accN > 0 {
		avgAcc := accSum / float64(accN) * 100
		cards = append(cards, domain.TimeseriesKpiCard{
			Key: "accuracy", Label: "PrÃ©cision", Value: fmt.Sprintf("%.1f%%", avgAcc),
		})
	}

	return domain.TimeseriesSummaryTab{KpiCards: cards}
}

// ---------------------------------------------------------------------------
// Onglet Cumul
// ---------------------------------------------------------------------------

func buildCumulTab(matches []legacymatch.StatsMatchRow) domain.TimeseriesCumulTab {
	n := len(matches)
	if n == 0 {
		return domain.TimeseriesCumulTab{}
	}

	cumulKD := make([]domain.CumulativePoint, 0, n)
	cumulNet := make([]domain.CumulativePoint, 0, n)
	rollingKD := make([]domain.CumulativePoint, 0, n)

	totalKills, totalDeaths, cumulNetVal := 0, 0, 0
	const rollingWindow = 5 // port Python compute_rolling_kd_polars(window=5)

	for i, m := range matches {
		totalKills += m.Kills
		totalDeaths += m.Deaths

		kd := 0.0
		if totalDeaths > 0 {
			kd = float64(totalKills) / float64(totalDeaths)
		}
		cumulKD = append(cumulKD, domain.CumulativePoint{
			Index: i, StartTime: m.StartTime, Value: math.Round(kd*100) / 100,
		})

		net := m.Kills - m.Deaths
		cumulNetVal += net
		cumulNet = append(cumulNet, domain.CumulativePoint{
			Index: i, StartTime: m.StartTime, Value: float64(cumulNetVal),
		})

		// Rolling K/D sur fenÃªtre glissante.
		start := i - rollingWindow + 1
		if start < 0 {
			start = 0
		}
		rk, rd := 0, 0
		for j := start; j <= i; j++ {
			rk += matches[j].Kills
			rd += matches[j].Deaths
		}
		rkd := 0.0
		if rd > 0 {
			rkd = float64(rk) / float64(rd)
		}
		rollingKD = append(rollingKD, domain.CumulativePoint{
			Index: i, StartTime: m.StartTime, Value: math.Round(rkd*100) / 100,
		})
	}

	return domain.TimeseriesCumulTab{
		CumulativeKD:  cumulKD,
		CumulativeNet: cumulNet,
		RollingKD:     rollingKD,
	}
}

// ---------------------------------------------------------------------------
// Onglet Form
// ---------------------------------------------------------------------------

func buildTimeseriesFormTab(matches []legacymatch.StatsMatchRow) domain.TimeseriesFormTab {
	regStats := computeRegressionStats(matches)

	// EWMA K/D (exponentially weighted moving average).
	// alpha = 0.20 â€” aligneÌ sur Python plot_ewma_kd (alpha=0.20).
	const alpha = 0.20
	ewmaPoints := make([]domain.CumulativePoint, 0, len(matches))
	var ewma float64
	for i, m := range matches {
		kd := 0.0
		if m.Deaths > 0 {
			kd = float64(m.Kills) / float64(m.Deaths)
		}
		if i == 0 {
			ewma = kd
		} else {
			ewma = alpha*kd + (1-alpha)*ewma
		}
		ewmaPoints = append(ewmaPoints, domain.CumulativePoint{
			Index: i, StartTime: m.StartTime,
			Value: math.Round(ewma*100) / 100,
		})
	}

	return domain.TimeseriesFormTab{
		RegressionStats: regStats,
		EWMAKDPoints:    ewmaPoints,
	}
}

// computeRegressionStats calcule les statistiques de rÃ©gression.
func computeRegressionStats(matches []legacymatch.StatsMatchRow) domain.TimeseriesRegressionStats {
	const minForTrend = 20

	n := len(matches)
	if n < minForTrend {
		return domain.TimeseriesRegressionStats{HasEnoughForTrend: false}
	}

	// RÃ©gression linÃ©aire simple sur le K/D.
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	count := 0
	for i, m := range matches {
		kd := 0.0
		if m.Deaths > 0 {
			kd = float64(m.Kills) / float64(m.Deaths)
		}
		x := float64(i)
		sumX += x
		sumY += kd
		sumXY += x * kd
		sumX2 += x * x
		count++
	}

	if count < minForTrend {
		return domain.TimeseriesRegressionStats{HasEnoughForTrend: false}
	}

	fn := float64(count)
	denom := fn*sumX2 - sumX*sumX
	if denom == 0 {
		return domain.TimeseriesRegressionStats{HasEnoughForTrend: false}
	}

	slope := (fn*sumXY - sumX*sumY) / denom
	meanY := sumY / fn

	// RÂ² approximation.
	ssTot, ssRes := 0.0, 0.0
	intercept := (sumY - slope*sumX) / fn
	for i, m := range matches {
		kd := 0.0
		if m.Deaths > 0 {
			kd = float64(m.Kills) / float64(m.Deaths)
		}
		predicted := intercept + slope*float64(i)
		ssRes += (kd - predicted) * (kd - predicted)
		ssTot += (kd - meanY) * (kd - meanY)
	}

	var r2 *float64
	if ssTot > 0 {
		v := math.Max(0, 1-ssRes/ssTot)
		r2 = &v
	}

	kdSlope := math.Round(slope*1000) / 1000

	// Trend detection.
	var trend *string
	if math.Abs(slope) < 0.001 {
		t := trendLabelStable
		trend = &t
	} else if slope > 0 {
		t := "improving"
		trend = &t
	} else {
		t := "declining"
		trend = &t
	}

	return domain.TimeseriesRegressionStats{
		KDSlope:           &kdSlope,
		RSquared:          r2,
		HasEnoughForTrend: true,
		Trend:             trend,
	}
}

// ---------------------------------------------------------------------------
// Onglet IntensitÃ© (Sprint 42)
// ---------------------------------------------------------------------------

// buildIntensityTab construit la heatmap jourÃ—heure et le score/min.
func buildIntensityTab(matches []legacymatch.StatsMatchRow) domain.TimeseriesIntensityTab {
	if len(matches) == 0 {
		return domain.TimeseriesIntensityTab{
			HeatmapData:     []domain.IntensityHeatmapPoint{},
			ScorePerMinData: []domain.CumulativePoint{},
		}
	}

	// Heatmap jour Ã— heure.
	type cell struct {
		kills, deaths, count int
	}
	heatmap := make(map[[2]int]*cell) // [day, hour]

	scorePerMin := make([]domain.CumulativePoint, 0, len(matches))

	for i, m := range matches {
		day := int(m.StartTime.Weekday())
		// Convertir Sunday=0 â†’ Monday=0..Sunday=6
		day = (day + 6) % 7
		hour := m.StartTime.Hour()
		key := [2]int{day, hour}
		c, ok := heatmap[key]
		if !ok {
			c = &cell{}
			heatmap[key] = c
		}
		c.kills += m.Kills
		c.deaths += m.Deaths
		c.count++

		// Score per minute.
		spm := 0.0
		if m.PersonalScore != nil && m.TimePlayedSeconds != nil && *m.TimePlayedSeconds > 0 {
			spm = float64(*m.PersonalScore) / (float64(*m.TimePlayedSeconds) / 60.0)
		}
		scorePerMin = append(scorePerMin, domain.CumulativePoint{
			Index: i, StartTime: m.StartTime,
			Value: math.Round(spm*100) / 100,
		})
	}

	points := make([]domain.IntensityHeatmapPoint, 0, len(heatmap))
	for key, c := range heatmap {
		avgKD := 0.0
		if c.deaths > 0 {
			avgKD = float64(c.kills) / float64(c.deaths)
		}
		points = append(points, domain.IntensityHeatmapPoint{
			DayOfWeek: key[0],
			Hour:      key[1],
			Count:     c.count,
			AvgKD:     math.Round(avgKD*100) / 100,
		})
	}

	return domain.TimeseriesIntensityTab{
		HeatmapData:     points,
		ScorePerMinData: scorePerMin,
	}
}

// ---------------------------------------------------------------------------
// Onglet Distributions (Sprint 42)
// ---------------------------------------------------------------------------

// buildDistributionsTab construit les histogrammes KDA/kills et les corrÃ©lations.
func buildDistributionsTab(matches []legacymatch.StatsMatchRow) domain.TimeseriesDistributionsTab {
	if len(matches) == 0 {
		return domain.TimeseriesDistributionsTab{
			KDABuckets:         []domain.DistributionBucket{},
			KillsBuckets:       []domain.DistributionBucket{},
			AccuracyBuckets:    []domain.DistributionBucket{},
			ScorePerMinBuckets: []domain.DistributionBucket{},
			RollingWRBuckets:   []domain.DistributionBucket{},
			CorrelationPoints:  []domain.CorrelationDataPair{},
		}
	}

	return domain.TimeseriesDistributionsTab{
		KDABuckets:             buildKDABuckets(matches),
		KillsBuckets:           buildKillsBuckets(matches),
		AccuracyBuckets:        buildAccuracyBuckets(matches),
		ScorePerMinBuckets:     buildScorePerMinBuckets(matches),
		RollingWRBuckets:       buildRollingWRBuckets(matches),
		LifeBuckets:            buildLifeBuckets(matches),
		PerfScoreBuckets:       buildPerfScoreBuckets(matches),
		PersonalScoreBuckets:   buildPersonalScoreBuckets(matches),
		MaxKillingSpreeBuckets: buildMaxKillingSpreeBuckets(matches),
		CorrelationPoints:      buildCorrelationPoints(matches),
	}
}

// buildKDABuckets crée des buckets pour la distribution FDA.
//
// Source : m.KDA — valeur synced depuis player_match_stats.kda (colonne BDD,
// inclut les assists). On ne recalcule pas kills/deaths côté Go : ADR 0006
// + revue 2026-04-29 — la canonique vient du sync. Si m.KDA est nil pour un
// match (donnée incomplète), il est skippé du bucketing.
func buildKDABuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const (
		binWidth = 0.25
		maxBin   = 5.0
	)
	numBins := int(maxBin / binWidth)
	counts := make([]int, numBins+1) // dernier = overflow

	for _, m := range matches {
		if m.KDA == nil {
			continue
		}
		kda := *m.KDA
		if kda < 0 {
			kda = 0
		}
		idx := int(kda / binWidth)
		if idx >= numBins {
			idx = numBins
		}
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, numBins+1)
	for i, c := range counts {
		if c == 0 {
			continue
		}
		start := float64(i) * binWidth
		end := start + binWidth
		if i == numBins {
			end = maxBin + binWidth
		}
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: math.Round(start*100) / 100,
			BucketUpper: math.Round(end*100) / 100,
			Count:       c,
		})
	}
	return buckets
}

// buildKillsBuckets crÃ©e des buckets pour la distribution des kills par match.
func buildKillsBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 5.0
	maxKills := 0
	for _, m := range matches {
		if m.Kills > maxKills {
			maxKills = m.Kills
		}
	}
	numBins := maxKills/int(binWidth) + 1
	counts := make([]int, numBins+1)

	for _, m := range matches {
		idx := m.Kills / int(binWidth)
		if idx > numBins {
			idx = numBins
		}
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, numBins+1)
	for i, c := range counts {
		if c == 0 {
			continue
		}
		start := float64(i) * binWidth
		end := start + binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start,
			BucketUpper: end,
			Count:       c,
		})
	}
	return buckets
}

// buildCorrelationPoints construit les paires de corrÃ©lation pour 5 types de scatter.
func buildCorrelationPoints(matches []legacymatch.StatsMatchRow) []domain.CorrelationDataPair {
	points := make([]domain.CorrelationDataPair, 0, len(matches)*5)
	for _, m := range matches {
		kd := 0.0
		if m.Deaths > 0 {
			kd = float64(m.Kills) / float64(m.Deaths)
		}
		// DurÃ©e de vie approchÃ©e : time_played / (deaths+1)
		lifespan := 0.0
		if m.TimePlayedSeconds != nil && *m.TimePlayedSeconds > 0 {
			lifespan = float64(*m.TimePlayedSeconds) / float64(m.Deaths+1)
		}

		// P7.1 (revue 2026-04-29) : Label composite ("kills_vs_kd") séparé en
		// MetricXKey/MetricYKey ; X/Y → XValue/YValue.
		points = append(points, domain.CorrelationDataPair{
			MetricXKey: "kills", MetricYKey: "kd_ratio",
			XValue: float64(m.Kills), YValue: math.Round(kd*100) / 100, Outcome: m.Outcome,
		})
		points = append(points, domain.CorrelationDataPair{
			MetricXKey: "lifespan", MetricYKey: "kills",
			XValue: math.Round(lifespan*10) / 10, YValue: float64(m.Kills), Outcome: m.Outcome,
		})
		if m.Accuracy != nil && m.KDA != nil {
			// Accuracy déjà en % 0..100 (sync/transforms.go:315) — round à 1 décimale
			// sans re-multiplier (bug historique du *100 qui sortait du domaine).
			points = append(points, domain.CorrelationDataPair{
				MetricXKey: "accuracy", MetricYKey: "kda",
				XValue: math.Round(*m.Accuracy*10) / 10, YValue: math.Round(*m.KDA*100) / 100, Outcome: m.Outcome,
			})
		}
		points = append(points, domain.CorrelationDataPair{
			MetricXKey: "lifespan", MetricYKey: "deaths",
			XValue: math.Round(lifespan*10) / 10, YValue: float64(m.Deaths), Outcome: m.Outcome,
		})
		points = append(points, domain.CorrelationDataPair{
			MetricXKey: "kills", MetricYKey: "deaths",
			XValue: float64(m.Kills), YValue: float64(m.Deaths), Outcome: m.Outcome,
		})
		if m.TeamMMR != nil && m.EnemyMMR != nil {
			points = append(points, domain.CorrelationDataPair{
				MetricXKey: "mmr_team", MetricYKey: "mmr_enemy",
				XValue: math.Round(*m.TeamMMR*100) / 100, YValue: math.Round(*m.EnemyMMR*100) / 100, Outcome: m.Outcome,
			})
		}
	}
	return points
}

// ---------------------------------------------------------------------------
// Lignes match brutes (pour les charts timeline React)
// ---------------------------------------------------------------------------

// buildMatchRows convertit StatsMatchRow en TimeseriesMatchRow (1 ligne = 1 match).
//
// KDA et KDRatio sont calcules par P2.5 (revue 2026-04-29 ADR 0006) â€” debloque
// la suppression du recompute K/D cote front (TimeseriesKdaBars.tsx:78, B3).
func buildMatchRows(matches []legacymatch.StatsMatchRow) []domain.TimeseriesMatchRow {
	rows := make([]domain.TimeseriesMatchRow, 0, len(matches))
	for i, m := range matches {
		// KDR canonique calcule a partir des compteurs (analysis.KDR).
		// Distinct du KDA pre-calcule cote sync (m.KDA inclut les assists).
		kdr := analysis.KDR(m.Kills, m.Deaths)
		rows = append(rows, domain.TimeseriesMatchRow{
			MatchID:           m.MatchID,
			Index:             i,
			StartTime:         m.StartTime,
			Kills:             m.Kills,
			Deaths:            m.Deaths,
			Assists:           m.Assists,
			KDA:               m.KDA,
			KDRatio:           &kdr,
			Accuracy:          m.Accuracy,
			Outcome:           m.Outcome,
			PersonalScore:     m.PersonalScore,
			DamageDealt:       m.DamageDealt,
			DamageTaken:       m.DamageTaken,
			PerfScore:         m.PerfScoreComputed,
			Rank:              m.Rank,
			PlaylistName:      m.PlaylistName,
			TimePlayedSeconds: m.TimePlayedSeconds,
			MaxKillingSpree:   m.MaxKillingSpree,
			HeadshotKills:     m.HeadshotKills,
			PerfectKills:      m.PerfectKills,
			MapName:           m.MapName,
			MapNameFR:         m.MapNameFR,
			SkillRatingValue:  m.SkillRatingValue,
			SkillRatingType:   m.SkillRatingType,
			SessionLabel:      m.SessionLabel,
			TeamMMR:           m.TeamMMR,
		})
	}
	return rows
}

// ---------------------------------------------------------------------------
// Histogrammes supplÃ©mentaires (PrÃ©cision, Score/min, Win Rate glissant)
// ---------------------------------------------------------------------------

// buildAccuracyBuckets crée des buckets de 5 % pour la distribution de précision.
//
// Source : m.Accuracy stockée en pourcentage 0..100 (cf. sync/transforms.go:315
// — `acc = shots_hit / shots_fired * 100`). NE PAS re-multiplier par 100 ; le
// bug historique faisait coller toutes les valeurs au bin 100+ (un seul gros
// bâton sur le chart).
func buildAccuracyBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 5.0      // 5 % par bin
	counts := make([]int, 21) // bins 0-5, 5-10, …, 95-100, 100+

	for _, m := range matches {
		if m.Accuracy == nil {
			continue
		}
		pct := *m.Accuracy
		if pct < 0 {
			pct = 0
		}
		idx := int(pct / binWidth)
		if idx >= len(counts) {
			idx = len(counts) - 1
		}
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for i, c := range counts {
		if c == 0 {
			continue
		}
		start := float64(i) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	return buckets
}

// buildScorePerMinBuckets crÃ©e des buckets de 10 pts/min pour la distribution score/min.
func buildScorePerMinBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 10.0
	counts := make(map[int]int)

	for _, m := range matches {
		if m.PersonalScore == nil || m.TimePlayedSeconds == nil || *m.TimePlayedSeconds == 0 {
			continue
		}
		spm := float64(*m.PersonalScore) / (float64(*m.TimePlayedSeconds) / 60.0)
		idx := int(spm / binWidth)
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for idx, c := range counts {
		start := float64(idx) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].BucketLower < buckets[j].BucketLower })
	return buckets
}

// buildLifeBuckets crée des buckets de 5s pour la distribution de la durée
// de vie moyenne (time_played_seconds / (deaths + 1)). Match avec
// time_played non renseigné est skippé.
func buildLifeBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 5.0
	maxLife := 0.0
	for _, m := range matches {
		if m.TimePlayedSeconds == nil || *m.TimePlayedSeconds <= 0 {
			continue
		}
		life := float64(*m.TimePlayedSeconds) / float64(m.Deaths+1)
		if life > maxLife {
			maxLife = life
		}
	}
	if maxLife == 0 {
		return []domain.DistributionBucket{}
	}
	numBins := int(maxLife/binWidth) + 1
	counts := make(map[int]int)
	for _, m := range matches {
		if m.TimePlayedSeconds == nil || *m.TimePlayedSeconds <= 0 {
			continue
		}
		life := float64(*m.TimePlayedSeconds) / float64(m.Deaths+1)
		idx := int(life / binWidth)
		if idx > numBins {
			idx = numBins
		}
		counts[idx]++
	}
	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for idx, c := range counts {
		start := float64(idx) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].BucketLower < buckets[j].BucketLower })
	return buckets
}

// buildPersonalScoreBuckets crée des buckets de 250 points pour la
// distribution du score personnel par match (PersonalScore — synced).
func buildPersonalScoreBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 250.0
	counts := make(map[int]int)
	for _, m := range matches {
		if m.PersonalScore == nil {
			continue
		}
		score := float64(*m.PersonalScore)
		if score < 0 {
			score = 0
		}
		idx := int(score / binWidth)
		counts[idx]++
	}
	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for idx, c := range counts {
		start := float64(idx) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].BucketLower < buckets[j].BucketLower })
	return buckets
}

// buildMaxKillingSpreeBuckets crée des buckets entiers (binWidth=1) pour la
// distribution du max killing spree par match. Skip si nil ou ≤0.
func buildMaxKillingSpreeBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	counts := make(map[int]int)
	for _, m := range matches {
		if m.MaxKillingSpree == nil || *m.MaxKillingSpree <= 0 {
			continue
		}
		counts[*m.MaxKillingSpree]++
	}
	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for v, c := range counts {
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: float64(v), BucketUpper: float64(v + 1), Count: c,
		})
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].BucketLower < buckets[j].BucketLower })
	return buckets
}

// buildPerfScoreBuckets crée des buckets de 5 points pour la distribution du
// performance_score (PerfScoreComputed). Range attendue [0, 100]. Les matchs
// sans score (sync incomplet) sont skippés.
func buildPerfScoreBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 5.0
	counts := make([]int, 21) // 0-5, 5-10, …, 95-100, 100+
	for _, m := range matches {
		if m.PerfScoreComputed == nil {
			continue
		}
		score := *m.PerfScoreComputed
		if score < 0 {
			score = 0
		}
		idx := int(score / binWidth)
		if idx >= len(counts) {
			idx = len(counts) - 1
		}
		counts[idx]++
	}
	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for i, c := range counts {
		if c == 0 {
			continue
		}
		start := float64(i) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	return buckets
}

// buildRollingWRBuckets crÃ©e des buckets de 5 % pour la distribution du win-rate glissant (fenÃªtre 14).
func buildRollingWRBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const (
		window   = 14
		binWidth = 5.0
	)
	counts := make([]int, 21) // bins 0-5, 5-10, â€¦, 95-100

	for i := range matches {
		start := i - window + 1
		if start < 0 {
			start = 0
		}
		wins := 0
		for j := start; j <= i; j++ {
			if matches[j].Outcome != nil && *matches[j].Outcome == analysis.OutcomeWin {
				wins++
			}
		}
		total := i - start + 1
		// TODO P4 ADR 0006 : retirer *100 (convention API canonique 0..1).
		wr := analysis.WinRate(wins, total) * 100
		idx := int(wr / binWidth)
		if idx >= len(counts) {
			idx = len(counts) - 1
		}
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for i, c := range counts {
		if c == 0 {
			continue
		}
		start := float64(i) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	return buckets
}
