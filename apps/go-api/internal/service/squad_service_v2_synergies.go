// Package service — squad_service_v2_synergies.go : helpers de l'onglet Synergies
// de la page Squad V2.
//
// Conformement au PLAN_SQUAD_GO_PORTAGE Phase P3 : 4 charts de synergies par
// carte construits depuis l'intersection des matchs partages.
//
//	Lollipop W/L par carte         : ChartSeries[ChartPointStacked]
//	Bullet winrate session vs hist : ChartSeries[ChartPointStacked]
//	Perf vs historique par carte   : ChartSeries[ChartPoint2D]
//	Heatmap escouade joueur x carte: ChartSeries[ChartPointHeatmap]
//
// Tous les helpers sont purs (aucun acces DB), consomment des
// canonical.PlayerMatchRow + agregats analysis/breakdown deja livres en Phase 0.
package service

import (
	"sort"

	"levelup/go-api/internal/analysis/breakdown"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// MapBreakdownLimit borne le nombre de cartes affichees (cf. audit § 3.1 : top 20
// dernieres cartes jouees). On garde 20 par defaut, override via parametre.
const MapBreakdownLimit = 20

// squadStatsToMapAggregates materialise les stats squad en []MapAggregate
// compatible avec breakdown.CompareToHistorical. Le label est repris depuis
// sessionAgg (meme MapID) pour preserver le rendu côté chart ; clef absente =
// MapAggregate avec label vide (le builder degrade gracieusement).
func squadStatsToMapAggregates(
	stats map[string]domain.MapSquadStats,
	sessionAgg []breakdown.MapAggregate,
) []breakdown.MapAggregate {
	if len(stats) == 0 {
		return nil
	}
	labelByID := make(map[string]string, len(sessionAgg))
	for _, s := range sessionAgg {
		if s.MapLabel != "" {
			labelByID[s.MapID] = s.MapLabel
		}
	}
	out := make([]breakdown.MapAggregate, 0, len(stats))
	for mapID, s := range stats {
		var wr float64
		if s.Total > 0 {
			wr = float64(s.Wins) / float64(s.Total)
		}
		out = append(out, breakdown.MapAggregate{
			MapID:    mapID,
			MapLabel: labelByID[mapID],
			Counts: breakdown.Counts{
				Played:  s.Total,
				Wins:    s.Wins,
				WinRate: wr,
			},
			AvgPerformanceScore: s.PerfAvg,
		})
	}
	return out
}

// rowsToBreakdownInputs convertit []canonical.PlayerMatchRow vers
// []breakdown.Row pour reutiliser analysis/breakdown.ByMap.
func rowsToBreakdownInputs(rows []canonical.PlayerMatchRow) []breakdown.Row {
	out := make([]breakdown.Row, 0, len(rows))
	for _, r := range rows {
		var perfScore *float64
		if r.Enrichment.PerformanceScore != nil {
			s := *r.Enrichment.PerformanceScore
			perfScore = &s
		}
		mapID := ""
		mapLabel := ""
		if r.Summary.Map != nil {
			mapID = r.Summary.Map.ID
			mapLabel = r.Summary.Map.DefaultLabel
		}
		out = append(out, breakdown.Row{
			Outcome:          r.Self.Outcome,
			MapID:            mapID,
			MapLabel:         mapLabel,
			PerformanceScore: perfScore,
		})
	}
	return out
}

// BuildMapBreakdownLollipop construit la serie de barres empilees W/L/T/DNF
// par carte (top N par nombre de matchs). Le wrapper <Lollipop> cote front
// rend ca en lollipop horizontal vert/rouge.
//
// Tri : par nombre de matchs joues descendant ; cap a limit (default 20).
func BuildMapBreakdownLollipop(
	rows []canonical.PlayerMatchRow,
	limit int,
) domain.ChartSeries[domain.ChartPointStacked] {
	if limit <= 0 {
		limit = MapBreakdownLimit
	}
	maps := breakdown.ByMap(rowsToBreakdownInputs(rows))
	// breakdown.ByMap trie par WinRate desc ; on re-trie par Played desc pour
	// privilegier les cartes les plus jouees.
	sort.SliceStable(maps, func(i, j int) bool {
		if maps[i].Played != maps[j].Played {
			return maps[i].Played > maps[j].Played
		}
		return maps[i].MapID < maps[j].MapID
	})
	if len(maps) > limit {
		maps = maps[:limit]
	}
	dps := make([]domain.ChartPointStacked, 0, len(maps))
	for _, m := range maps {
		category := m.MapLabel
		if category == "" {
			category = m.MapID
		}
		dps = append(dps, domain.ChartPointStacked{
			Category: category,
			Components: map[string]float64{
				"win":  float64(m.Wins),
				"loss": float64(m.Losses),
				"tie":  float64(m.Ties),
				"dnf":  float64(m.DNF),
			},
		})
	}
	return domain.ChartSeries[domain.ChartPointStacked]{
		Key:        "squad.synergies.map_breakdown",
		LabelKey:   "squad.synergies.lollipop_title",
		Datapoints: dps,
		Meta: map[string]any{
			"limit": limit,
		},
	}
}

// BuildBulletWinrate compose une serie bullet (3 valeurs par carte) :
//
//	"session"    : winrate sur les matchs partages (squad courante)
//	"historical" : winrate du joueur principal AVEC l'escouade strict sur
//	               tous les matchs joues ensemble sur cette carte (aucun
//	               filtre temporel).
//
// historical est alimente par SquadRepository.LoadMapStatsForSquad — clef =
// map_id, valeur = (Wins, Total, PerfAvg). Carte absente du map = pas de
// trace "historical" dans le bullet.
func BuildBulletWinrate(
	sessionRows []canonical.PlayerMatchRow,
	historical map[string]domain.MapSquadStats,
	limit int,
) domain.ChartSeries[domain.ChartPointStacked] {
	if limit <= 0 {
		limit = MapBreakdownLimit
	}
	sessionAgg := breakdown.ByMap(rowsToBreakdownInputs(sessionRows))

	// On re-trie sessionAgg par Played desc pour limiter aux N plus jouees.
	sort.SliceStable(sessionAgg, func(i, j int) bool {
		if sessionAgg[i].Played != sessionAgg[j].Played {
			return sessionAgg[i].Played > sessionAgg[j].Played
		}
		return sessionAgg[i].MapID < sessionAgg[j].MapID
	})
	if len(sessionAgg) > limit {
		sessionAgg = sessionAgg[:limit]
	}

	dps := make([]domain.ChartPointStacked, 0, len(sessionAgg))
	for _, s := range sessionAgg {
		category := s.MapLabel
		if category == "" {
			category = s.MapID
		}
		comp := map[string]float64{
			"session": s.WinRate,
		}
		if h, ok := historical[s.MapID]; ok && h.Total > 0 {
			comp["historical"] = float64(h.Wins) / float64(h.Total)
		}
		dps = append(dps, domain.ChartPointStacked{
			Category:   category,
			Components: comp,
		})
	}
	return domain.ChartSeries[domain.ChartPointStacked]{
		Key:        "squad.synergies.bullet_winrate",
		LabelKey:   "squad.synergies.bullet_title",
		Datapoints: dps,
	}
}

// BuildPerfVsHistorical compose une serie de barres delta perf_score
// (session - historique) par carte. Negatif = sous-performance, positif =
// sur-performance par rapport a l'historique du main AVEC l'escouade strict
// (aucun filtre temporel).
//
// historical est alimente par SquadRepository.LoadMapStatsForSquad. Les cartes
// sans AvgPerformanceScore session OU sans PerfAvg historique sont skippees.
func BuildPerfVsHistorical(
	sessionRows []canonical.PlayerMatchRow,
	historical map[string]domain.MapSquadStats,
	limit int,
) domain.ChartSeries[domain.ChartPoint2D] {
	if limit <= 0 {
		limit = MapBreakdownLimit
	}
	sessionAgg := breakdown.ByMap(rowsToBreakdownInputs(sessionRows))
	// CompareToHistorical attend []MapAggregate. On materialise une liste
	// minimale a partir du map (Played/Wins/WinRate/AvgPerformanceScore)
	// pour reutiliser la logique de delta.
	histAgg := squadStatsToMapAggregates(historical, sessionAgg)
	deltas := breakdown.CompareToHistorical(sessionAgg, histAgg)
	if len(deltas) > limit {
		deltas = deltas[:limit]
	}

	dps := make([]domain.ChartPoint2D, 0, len(deltas))
	for _, d := range deltas {
		if d.AvgPerformanceScoreDelta == nil {
			continue
		}
		category := d.MapLabel
		if category == "" {
			category = d.MapID
		}
		dps = append(dps, domain.ChartPoint2D{
			X: category,
			Y: *d.AvgPerformanceScoreDelta,
		})
	}
	return domain.ChartSeries[domain.ChartPoint2D]{
		Key:        "squad.synergies.perf_vs_historical",
		LabelKey:   "squad.synergies.perf_vs_historical_title",
		Datapoints: dps,
	}
}

// BuildHeatmapPlayerMap construit la heatmap 2D player x map.
//
// Pour chaque (joueur, carte) on calcule le AvgPerformanceScore (depuis les
// rows partages, pas l'historique solo). Les cellules sans donnee perf sont
// omises (le wrapper Heatmap2D rend ca en cellule vide).
//
// Axes :
//
//	X = noms de cartes (top N par matchs joues, deduits depuis la session du main)
//	Y = gamertags (tries alphabetique pour stabilite)
func BuildHeatmapPlayerMap(
	rowsByPlayer map[string][]canonical.PlayerMatchRow,
	limit int,
) domain.ChartSeries[domain.ChartPointHeatmap] {
	if limit <= 0 {
		limit = MapBreakdownLimit
	}
	if len(rowsByPlayer) == 0 {
		return domain.ChartSeries[domain.ChartPointHeatmap]{
			Key:      "squad.synergies.heatmap_player_map",
			LabelKey: "squad.synergies.heatmap_title",
		}
	}

	// Calculer top N maps depuis l'union des joueurs (par nombre de matchs).
	playedPerMap := make(map[string]int)
	labelByID := make(map[string]string)
	for _, rows := range rowsByPlayer {
		for _, r := range rows {
			if r.Summary.Map == nil || r.Summary.Map.ID == "" {
				continue
			}
			id := r.Summary.Map.ID
			playedPerMap[id]++
			if labelByID[id] == "" && r.Summary.Map.DefaultLabel != "" {
				labelByID[id] = r.Summary.Map.DefaultLabel
			}
		}
	}
	type mapEntry struct {
		ID, Label string
		Played    int
	}
	mapsRanked := make([]mapEntry, 0, len(playedPerMap))
	for id, n := range playedPerMap {
		label := labelByID[id]
		if label == "" {
			label = id
		}
		mapsRanked = append(mapsRanked, mapEntry{ID: id, Label: label, Played: n})
	}
	sort.Slice(mapsRanked, func(i, j int) bool {
		if mapsRanked[i].Played != mapsRanked[j].Played {
			return mapsRanked[i].Played > mapsRanked[j].Played
		}
		return mapsRanked[i].ID < mapsRanked[j].ID
	})
	if len(mapsRanked) > limit {
		mapsRanked = mapsRanked[:limit]
	}

	// Cle stable des joueurs.
	gts := make([]string, 0, len(rowsByPlayer))
	for gt := range rowsByPlayer {
		gts = append(gts, gt)
	}
	sort.Strings(gts)

	// Pour chaque (gt, mapID) calculer AvgPerformanceScore.
	dps := make([]domain.ChartPointHeatmap, 0, len(gts)*len(mapsRanked))
	for _, gt := range gts {
		perfByMap := make(map[string]struct {
			sum float64
			n   int
		})
		for _, r := range rowsByPlayer[gt] {
			if r.Summary.Map == nil || r.Summary.Map.ID == "" {
				continue
			}
			if r.Enrichment.PerformanceScore == nil {
				continue
			}
			cur := perfByMap[r.Summary.Map.ID]
			cur.sum += *r.Enrichment.PerformanceScore
			cur.n++
			perfByMap[r.Summary.Map.ID] = cur
		}
		for _, m := range mapsRanked {
			agg, ok := perfByMap[m.ID]
			if !ok || agg.n == 0 {
				continue
			}
			dps = append(dps, domain.ChartPointHeatmap{
				X:     m.Label,
				Y:     gt,
				Value: agg.sum / float64(agg.n),
				Detail: map[string]any{
					"matches": agg.n,
				},
			})
		}
	}
	return domain.ChartSeries[domain.ChartPointHeatmap]{
		Key:        "squad.synergies.heatmap_player_map",
		LabelKey:   "squad.synergies.heatmap_title",
		Datapoints: dps,
	}
}
