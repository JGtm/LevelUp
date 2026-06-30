// Package service - TimeseriesService : endpoint POST /pages/timeseries (contrat FastAPI).
//
// Sprint 33 : adapte les donnees StatsService vers le contrat TimeseriesPageResponse.
//
// Architecture data-only : le Go ne genere pas de figures Plotly. Le frontend
// React reconstruit les visualisations via les wrappers ECharts a partir des
// data points bruts fournis dans les onglets (cumulative_kd, ewma_kd_points,
// kda_buckets, correlation_points, heatmap_data, etc.).
//
// Le code est decoupe en fichiers thematiques pour respecter la limite des
// 500 lignes par fichier (CLAUDE.md). Ce fichier contient le type service,
// le constructeur, les Withers et GetPage (entry point). Les autres
// responsabilites vivent dans :
//
//   - timeseries_service_events.go       : highlight events + intensity rows +
//     first events distribution
//   - timeseries_service_aggregations.go : map breakdown, sessions, top
//     weapons, outcomes over time, filtres canonical
//   - timeseries_service_tabs.go         : builders Summary/Cumul/Intensity/
//     Distributions + buildMatchRows
//   - timeseries_service_buckets.go      : builders distribution buckets
//     (Accuracy, ScorePerMin, Life, etc.)
package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// Cles metriques canoniques utilisees dans MetricXKey/MetricYKey + KpiCards.
const (
	tsMetricKeyKills    = "kills"
	tsMetricKeyAccuracy = "accuracy"
	tsLabelUnknown      = "Unknown"
)

// TimeseriesService construit la reponse timeseries au format FastAPI.
type TimeseriesService struct {
	statsRepo port.StatsRepository
	// dataAdapter (optionnel, Phase C+ multi-titres) : point d'extension pour
	// router LoadTimeseries via la couche canonique. A ce jour, le service
	// utilise le repo direct car canonical.MetricSeries ne couvre pas encore
	// la totalite du payload (5 onglets : win_loss/accuracy/objective/form/
	// lusr). Le hook est en place pour permettre une bascule incrementale.
	dataAdapter games.TitleDataAdapter
	// playerMatchesRepo (P4.1, ADR 0011) : loader canonical-aware optionnel.
	// Quand fourni avec titleSlug + gamertag, GetPage charge canonical et
	// convertit via statsMatchRowFromCanonical (partage avec stats_service).
	// TODO P4.3 : retirer le converter quand les analyses timeseries
	// (buildCumulTab, computeRegressionStats, etc.) consommeront canonical.
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string
	gamertag          string
	// weaponKillsRepo (chart .04 Top weapons) : optionnel, degradation gracieuse.
	// Si nil, TopWeapons reste vide.
	weaponKillsRepo port.WeaponKillsRepository
	// highlightEventsRepo (chart .11 Premier evenement) : optionnel, degradation
	// gracieuse. Si nil ou xuid vide, FirstEvents reste nil.
	//
	// Interface locale ciblee - le service n'a besoin que de Load(filters), pas
	// de la signature multi-titres LoadHighlightEvents(slug, filters) du port
	// (slug deja fixe par PlayerDB).
	highlightEventsRepo highlightEventsLoader
	playerXUID          string
}

// highlightEventsLoader expose la sous-API du HighlightEventsRepo per-player
// utilisee par TimeseriesService (cf. port pattern, segregation).
type highlightEventsLoader interface {
	Load(ctx context.Context, filters port.HighlightEventFilters) ([]canonical.HighlightEvent, error)
}

// NewTimeseriesService cree un TimeseriesService.
func NewTimeseriesService(repo port.StatsRepository) *TimeseriesService {
	return &TimeseriesService{statsRepo: repo}
}

// WithDataAdapter injecte le DataAdapter multi-titres pour activer une
// future bascule LoadTimeseries. Degradation gracieuse si nil.
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
// (Top weapons by kills). Optionnel : si non cable, TopWeapons reste vide.
func (s *TimeseriesService) WithWeaponKillsRepo(repo port.WeaponKillsRepository) *TimeseriesService {
	s.weaponKillsRepo = repo
	return s
}

// WithHighlightEventsRepo injecte le repo highlight_events pour alimenter le
// chart .11 (Premier evenement). xuid est le XUID du joueur principal - sert a
// distinguer kills/deaths dans narrative.ComputeFirstEventsPerMatch.
func (s *TimeseriesService) WithHighlightEventsRepo(repo highlightEventsLoader, xuid string) *TimeseriesService {
	s.highlightEventsRepo = repo
	s.playerXUID = xuid
	return s
}

// GetPage construit la reponse complete avec 5 onglets.
func (s *TimeseriesService) GetPage(
	ctx context.Context,
	req domain.TimeseriesQueryRequest,
) (domain.TimeseriesPageResponse, error) {
	defer func(start time.Time) {
		observability.RecordDurationMS("timeseries_get_page", time.Since(start).Milliseconds())
	}(time.Now())
	// P4.3 finale (ADR 0011) : path canonical exclusif.
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return domain.TimeseriesPageResponse{}, fmt.Errorf("TimeseriesService: PlayerMatchesRepo non cable (P4.3 finale exige le wiring DI)")
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
	allMatches := analysis.StatsMatchRowsFromCanonical(canonicalRows, games.EffectiveHpToKill(s.titleSlug))

	matches := filterStatsMatchRows(allMatches, req.Filters)
	// Tri chronologique ASC (plus ancien -> plus récent). LoadPlayerMatches
	// renvoie DESC ; on inverse une fois ici pour que TOUS les builders et
	// charts aval (cumul, rolling, outcomes-over-time, MatchRows, séries
	// progression) soient cohérents oldest-first. Corrige aussi le cumul qui
	// était calculé à l'envers (accumulation depuis le match récent).
	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].StartTime.Before(matches[j].StartTime)
	})
	slog.DebugContext(ctx, "timeseries: matches charges",
		"total", len(allMatches),
		"apres_filtres", len(matches),
	)

	// Population historique (mode solo, sans filtre cascade/period/sessions) -
	// sert a enrichir le map_breakdown avec les references "Historique".
	historicalSolo := filterStatsMatchRowsByContext(allMatches, req.Filters.MatchContext)

	provideSpree := games.ProvidesMaxKillingSpree(s.titleSlug)

	// Events kill/death (chart .11 first events, chart .21 intensité, ET fallback
	// folie meurtrière) : UNE seule charge, réutilisée plus bas. Le repo synthétise
	// depuis killer_victim_pairs pour les titres sans kills natifs dans
	// highlight_events (Halo 5). Chargé AVANT la construction de resp pour que le
	// fallback spree (ci-dessous) soit visible des buckets Distributions + MatchRows.
	var (
		highlightEvents []canonical.HighlightEvent
		matchIDs        []string
	)
	if s.highlightEventsRepo != nil && s.playerXUID != "" && len(matches) > 0 {
		matchIDs = make([]string, 0, len(matches))
		for _, m := range matches {
			matchIDs = append(matchIDs, m.MatchID)
		}
		if evs, err := s.loadHighlightEvents(ctx, matchIDs); err != nil {
			slog.WarnContext(ctx, "timeseries: highlight events load failed", "err", err)
		} else {
			highlightEvents = evs
		}
	}
	// Fallback folie meurtrière (Halo 5) : la valeur native est absente du carnage →
	// on la calcule depuis les events. Gate par provideSpree (capability). NO-OP pour
	// les titres à valeur native (Infinite) — enrichMatchesMaxKillingSpree n'écrase
	// jamais une valeur déjà présente.
	if provideSpree {
		// Observabilité : un titre qui ANNONCE la spree (capability) mais sans events
		// (synthèse killer_victim_pairs infructueuse) laisse le donut « Folie meurtrière »
		// vide — exactement le symptôme à diagnostiquer. On le trace (≠ "titre sans spree").
		if len(highlightEvents) == 0 && len(matches) > 0 {
			slog.DebugContext(ctx, "timeseries: spree fallback sans events (donut Folie meurtrière vide)",
				"titleSlug", s.titleSlug, "player", s.gamertag, "matches", len(matches))
		}
		enrichMatchesMaxKillingSpree(matches, highlightEvents, s.playerXUID)
	}

	resp := domain.TimeseriesPageResponse{
		TotalMatches: len(matches),
		MatchRows:    buildMatchRows(matches, provideSpree),
		SummaryTab:   buildTimeseriesSummaryTab(matches),
		CumulTab:     buildCumulTab(matches),

		IntensityTab:     buildIntensityTab(matches),
		DistributionsTab: buildDistributionsTab(matches, provideSpree),
		OutcomesOverTime: buildOutcomesOverTime(matches),
		MapBreakdown:     buildSoloMapBreakdown(matches, historicalSolo),
		SoloSessionPerf:  buildSoloSessionPerf(historicalSolo),
		TopWeapons:       []domain.TimeseriesWeaponKill{},
		KillTypes:        buildTimeseriesKillTypes(matches),
	}

	// Top weapons (chart .04) : degradation gracieuse si weaponKillsRepo nil
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

	// First events distribution (chart .11) + Intensity heatmap : RÉUTILISE les events
	// déjà chargés (highlightEvents). Correction chronologie T0 ici (ramène les TimeMS
	// au référentiel gameplay) — distincte du fallback spree, qui reste order-based sur
	// les events bruts (invariant par décalage T0).
	if len(highlightEvents) > 0 {
		timelines := timeline.BuildTimelinesFromPlayerMatches(canonicalRows)
		corrected := timeline.CorrectEvents(highlightEvents, timelines)
		resp.FirstEvents = buildFirstEventsDistribution(
			narrative.ComputeFirstEventsPerMatch(corrected, s.playerXUID, matchIDs),
		)
		resp.IntensityRows = buildIntensityRows(corrected, matches, s.playerXUID, timeline.GameplayDurationsMS(timelines))
	}

	// BriefingKPIs : KPIs sur les rows canoniques filtres (memes match_ids que
	// matches). Alimente le composant <SessionBriefing> en mode solo. Reutilise
	// ComputeKPIStats sans re-filtrer les filtres metier.
	if filtered := filterCanonicalByMatchIDs(canonicalRows, matches); len(filtered) > 0 {
		briefingKPIs := analysis.ComputeKPIStats(filtered, games.EffectiveHpToKill(s.titleSlug))
		resp.BriefingKPIs = &briefingKPIs
		// Repartition des frags par type d'arme (donut Progression) : reutilise
		// l'agregateur canonical de Synthesis sur le meme set filtre.
		ds := buildSynthesisDetailedStatsFromCanonical(filtered, provideSpree)
		resp.DetailedStats = &ds
	}

	return resp, nil
}
