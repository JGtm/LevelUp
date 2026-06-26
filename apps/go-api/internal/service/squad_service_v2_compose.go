// Package service — squad_service_v2_compose.go : orchestration finale des
// charts et tableaux de la page Squad V2 (chunk S11).
//
// Ce fichier appelle les 16 builders deja livres (S3-S9) en injectant les
// donnees pre-chargees par GetSquadPage. Il ne fait AUCUN acces DB direct —
// toutes les sources externes (events, weapons, medals) sont passees en
// parametre via input structs.
package service

import (
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// buildSquadChartsInput regroupe les inputs necessaires aux 17 charts.
//
// Si une source est absente (events nil pour capability non supportee), les
// charts dependants sont omis (pointer nil dans SquadCharts).
type buildSquadChartsInput struct {
	mainGT          string
	squadOrder      []string
	squadXUIDs      map[string]string
	rowsByPlayer    map[string][]canonical.PlayerMatchRow
	squadHistorical map[string]domain.MapSquadStats
	events          []canonical.HighlightEvent
	sharedMatches   []domain.SquadSharedMatch
	// provideSpree : false quand le titre ne porte pas le max killing spree (Halo 5)
	// → la série KillingSpreeMax est omise (nil) plutôt que fabriquée. Cf.
	// games.ProvidesMaxKillingSpree.
	provideSpree bool
}

// buildSquadCharts assemble tous les ChartSeries de la page V2.
func buildSquadCharts(in buildSquadChartsInput) *domain.SquadCharts {
	out := &domain.SquadCharts{}

	// Synergies par carte (S3) : utilisent les rows partages (intersection)
	// pour la session, et les stats du squad strict (LoadMapStatsForSquad)
	// pour la reference historique.
	mainSharedRows := in.rowsByPlayer[in.mainGT]
	if len(mainSharedRows) > 0 {
		lollipop := BuildMapBreakdownLollipop(mainSharedRows, MapBreakdownLimit)
		out.MapBreakdownLollipop = &lollipop

		bullet := BuildBulletWinrate(mainSharedRows, in.squadHistorical, MapBreakdownLimit)
		out.BulletWinrate = &bullet

		perfDelta := BuildPerfVsHistorical(mainSharedRows, in.squadHistorical, MapBreakdownLimit)
		out.PerfVsHistorical = &perfDelta
	}
	if len(in.rowsByPlayer) > 0 {
		heatmap := BuildHeatmapPlayerMap(in.rowsByPlayer, MapBreakdownLimit)
		out.HeatmapPlayerMap = &heatmap
	}

	// Timeline + Form Score (S4)
	out.TimelineMultiPlayer = BuildTimelineMultiPlayer(in.rowsByPlayer)
	if len(mainSharedRows) > 0 {
		formScore := BuildFormScore(mainSharedRows, 0.3)
		out.FormScore = &formScore
	}

	// Cadence + Intensite (S6) — events requis. Durées gameplay (countdown
	// retranché) par match pour fixer les buckets sur la vraie durée.
	if len(in.events) > 0 {
		gpDur := timeline.GameplayDurationsMS(sharedMatchTimelines(in.sharedMatches))
		cadence := BuildCadenceChart(in.events, in.squadXUIDs, DefaultCadencePhaseSeconds, gpDur)
		out.Cadence = &cadence

		intensity := BuildIntensityHeatmap(in.events, DefaultIntensityBuckets, gpDur)
		out.IntensityHeatmap = &intensity
	}

	// Impact 8 roles (S5) — events + outcomes requis
	if len(in.events) > 0 {
		matrix := BuildImpactRolesMatrix(in.events, in.squadOrder, in.squadXUIDs, in.sharedMatches)
		out.ImpactMatrix = &matrix
		out.ImpactRanking = BuildImpactRanking(matrix)
	}

	// Contributions (S7)
	if len(in.rowsByPlayer) > 0 {
		perMin := BuildPerMinuteStats(in.rowsByPlayer)
		out.PerMinuteStats = &perMin

		fragsDeaths := BuildFragsDeathsCombined(in.rowsByPlayer)
		out.FragsDeathsCombined = &fragsDeaths

		hsPk := BuildHsPkStacked(in.rowsByPlayer)
		out.HsPkStacked = &hsPk

		if in.provideSpree {
			// Native (Infinite) ou calculée depuis les events kill/death (Halo 5) — la
			// capability events-timeline du titre rend le calcul-fallback possible.
			out.KillingSpreeMax = BuildKillingSpreeMax(in.rowsByPlayer, in.events, in.squadXUIDs)
		}
		out.AssistsTimeseries = BuildAssistsChart(in.rowsByPlayer)
		out.KDATimeseries = BuildKDAChart(in.rowsByPlayer)
		out.AccuracyTimeseries = BuildAccuracyChart(in.rowsByPlayer)
		out.AvgLifeTimeseries = BuildAvgLifeChart(in.rowsByPlayer)
		out.PerformanceTimeseries = BuildPerformanceChart(in.rowsByPlayer)
	}

	// Radar 6 axes (S8) — derive la mode_family depuis la playlist majoritaire
	// des matchs partages. Pour Phase 1 pilote, "" (custom thresholds neutres).
	if len(in.rowsByPlayer) > 0 {
		radarSeries := BuildRadarSeries(in.rowsByPlayer, "")
		// Convert []RadarChartSeries vers []any pour serializer JSON.
		out.Radar = make([]any, 0, len(radarSeries))
		for _, rs := range radarSeries {
			out.Radar = append(out.Radar, rs)
		}
	}

	return out
}

// buildSquadTablesInput regroupe les inputs necessaires aux 3 tableaux.
type buildSquadTablesInput struct {
	sharedMatches []domain.SquadSharedMatch
	rowsByPlayer  map[string][]canonical.PlayerMatchRow
	squadOrder    []string
	squadXUIDs    map[string]string
	weapons       []port.WeaponKillRow
	medals        []port.MedalRow
}

// buildSquadTables assemble les 3 tableaux (history, weapons, medals).
func buildSquadTables(in buildSquadTablesInput) *domain.SquadTables {
	out := &domain.SquadTables{}

	// History : toujours buildable depuis les SharedMatches + rowsByPlayer.
	historyRows := BuildHistoryTable(in.sharedMatches, in.rowsByPlayer, in.squadOrder)
	if len(historyRows) > 0 {
		out.History = make([]any, 0, len(historyRows))
		for _, r := range historyRows {
			out.History = append(out.History, r)
		}
	}

	// xuidToGT : inverse de squadXUIDs pour rekey weapons/medals en gamertag.
	xuidToGT := make(map[string]string, len(in.squadXUIDs))
	for gt, xuid := range in.squadXUIDs {
		xuidToGT[xuid] = gt
	}

	if len(in.weapons) > 0 {
		weaponsTable := BuildWeaponsTable(in.weapons, xuidToGT, 0)
		out.Weapons = make([]any, 0, len(weaponsTable))
		for _, r := range weaponsTable {
			out.Weapons = append(out.Weapons, r)
		}
	}

	if len(in.medals) > 0 {
		matchOrder := matchIDsOf(in.sharedMatches)
		medalsGallery := BuildMedalsGallery(in.medals, xuidToGT, matchOrder)
		out.Medals = make([]any, 0, len(medalsGallery))
		for _, e := range medalsGallery {
			out.Medals = append(out.Medals, e)
		}
	}

	return out
}
