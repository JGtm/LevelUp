// Package service - teammates_squad_charts_sessions_maps.go : builders
// teammates.04 (squad timeline par session) + teammates.03 (heatmap
// player x map). Decoupe de teammates_squad_charts.go (god-file split,
// refactor 2026-05-27).
package teammates

import (
	"context"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// buildCompositionSessionLabels dérive les sessions où la composition EXACTE a
// joué ensemble, à partir des rows d'intersection NON filtrées par session
// (allSquadRowsForTimeline = historique complet de la composition). Une entrée
// par SessionLabel non vide, bornes calculées depuis StartTime, triée par
// StartedAt DESC (la plus récente en tête). Dédup par match_id.
//
// Tri via les timestamps (StartTime), pas par parsing de label, pour éviter tout
// décalage TZ/DST (cohérent buildSquadSessionTimeline). Alimente le
// SessionMultiSelect ET le ré-ancrage front (sortie[0] = LatestCompositionSession).
func buildCompositionSessionLabels(rows []domain.SquadMatchRow) []domain.SessionLabelEntry {
	type bounds struct {
		started, ended time.Time
		count          int
	}
	byLabel := make(map[string]*bounds)
	seen := make(map[string]struct{}, len(rows))
	for _, m := range rows {
		if _, dup := seen[m.MatchID]; dup {
			continue
		}
		seen[m.MatchID] = struct{}{}
		if m.SessionLabel == nil || *m.SessionLabel == "" {
			continue
		}
		label := *m.SessionLabel
		b, ok := byLabel[label]
		if !ok {
			byLabel[label] = &bounds{started: m.StartTime, ended: m.StartTime, count: 1}
			continue
		}
		// MatchCount = matchs de la session « commencés ensemble » (population du
		// roster) : c'est CE nombre que le sélecteur de sessions affiche, et non le
		// suffixe « (N) » du label (figé au sync, solo+escouade confondus).
		b.count++
		if m.StartTime.Before(b.started) {
			b.started = m.StartTime
		}
		if m.StartTime.After(b.ended) {
			b.ended = m.StartTime
		}
	}

	out := make([]domain.SessionLabelEntry, 0, len(byLabel))
	for label, b := range byLabel {
		out = append(out, domain.SessionLabelEntry{
			Label:      label,
			StartedAt:  b.started,
			EndedAt:    b.ended,
			MatchCount: b.count,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	return out
}

func buildSquadSessionTimeline(matches []domain.SquadMatchRow) []domain.SquadSessionPoint {
	type agg struct {
		count                  int
		wins, losses           int
		perfSum                float64
		perfCount              int
		mmrSum                 float64
		mmrCount               int
		firstSeenStartTimeUnix int64
	}

	buckets := make(map[string]*agg)
	seen := make(map[string]struct{}, len(matches)) // dédup par match_id (peut apparaître via plusieurs teammates)
	for _, m := range matches {
		if _, dup := seen[m.MatchID]; dup {
			continue
		}
		seen[m.MatchID] = struct{}{}
		label := "(no session)"
		if m.SessionLabel != nil && *m.SessionLabel != "" {
			label = *m.SessionLabel
		}
		b, ok := buckets[label]
		if !ok {
			b = &agg{firstSeenStartTimeUnix: m.StartTime.Unix()}
			buckets[label] = b
		}
		if ts := m.StartTime.Unix(); ts < b.firstSeenStartTimeUnix {
			b.firstSeenStartTimeUnix = ts
		}
		b.count++
		switch m.Outcome {
		case analysis.OutcomeWin:
			b.wins++
		case analysis.OutcomeLoss:
			b.losses++
		}
		if m.PerformanceScore != nil {
			b.perfSum += *m.PerformanceScore
			b.perfCount++
		}
		if m.TeamMMR > 0 {
			b.mmrSum += m.TeamMMR
			b.mmrCount++
		}
	}

	points := make([]domain.SquadSessionPoint, 0, len(buckets))
	for label, b := range buckets {
		p := domain.SquadSessionPoint{
			SessionLabel: label,
			MatchCount:   b.count,
			Wins:         b.wins,
			Losses:       b.losses,
		}
		if b.perfCount > 0 {
			p.SquadPerf = round2(b.perfSum / float64(b.perfCount))
		}
		if b.count > 0 {
			wr := round2(float64(b.wins) / float64(b.count))
			p.WinRate = &wr
		}
		if b.mmrCount > 0 {
			mmr := round2(b.mmrSum / float64(b.mmrCount))
			p.TeamMMRAvg = &mmr
		}
		points = append(points, p)
	}

	// Tri par firstSeen ASC (chronologique).
	type sortable struct {
		point domain.SquadSessionPoint
		first int64
	}
	sortables := make([]sortable, 0, len(points))
	for _, p := range points {
		sortables = append(sortables, sortable{p, buckets[p.SessionLabel].firstSeenStartTimeUnix})
	}
	sort.Slice(sortables, func(i, j int) bool { return sortables[i].first < sortables[j].first })

	// Fenêtre adaptative au rythme du joueur : on ne garde que les sessions les
	// plus récentes, l'horizon (en jours) étant dimensionné par l'écart médian
	// entre sessions (cf. analysis.SquadSessionWindowKeep).
	times := make([]int64, len(sortables))
	for i, s := range sortables {
		times[i] = s.first
	}
	keep := analysis.SquadSessionWindowKeep(times, analysis.DefaultSquadSessionWindow())
	start := len(sortables) - keep
	if start < 0 {
		start = 0
	}
	windowed := sortables[start:]

	out := make([]domain.SquadSessionPoint, len(windowed))
	for i, s := range windowed {
		out[i] = s.point
	}
	return out
}

// ---------------------------------------------------------------------------
// teammates.03 — Heatmap performance player × map (toutes cartes jouées)
// ---------------------------------------------------------------------------

// buildSquadMapHeatmap construit la matrice perf joueur × carte. Le main
// player est la première ligne ; ses données viennent de allSquadRows
// (PerformanceScore = perf du main). Pour chaque coéquipier, on charge ses
// matchs canoniques via squadLoader (résolution dynamique par gamertag) et
// on filtre sur l'union des match_ids escouade.
//
// Si LoadFor échoue pour un teammate, sa ligne reste vide (cells nil) et
// un warn est loggé.
//
//nolint:funlen // chart-builder cohésif (compute counts → aggregate → cells matrix).
func (s *TeammatesService) buildSquadMapHeatmap(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	selectedGamertags []string,
	issues *dataIssues,
) *domain.SquadMapHeatmap {
	if len(allSquadRows) == 0 {
		return nil
	}

	// 1. Toutes les cartes jouées en escouade (dédup match_id).
	// Clé interne = MapID (UUID, language-agnostic) si dispo, sinon MapUI.
	// mapIDToUI assure la correspondance UUID → label d'affichage (FR).
	// firstSeen = plus ancien StartTime de la carte (StartTime = timestamp
	// canonique UTC, chargé via StartTimeCanonicalSQL côté repo).
	type mapStats struct {
		mapUI     string
		firstSeen time.Time
	}
	mapFirstSeen := make(map[string]time.Time)
	mapIDToUI := make(map[string]string)
	matchIDByID := make(map[string]struct{}, len(allSquadRows))
	for _, m := range allSquadRows {
		if _, dup := matchIDByID[m.MatchID]; dup {
			continue
		}
		matchIDByID[m.MatchID] = struct{}{}
		key := m.MapID
		if key == "" {
			key = m.MapUI
		}
		if key == "" {
			continue
		}
		if prev, ok := mapFirstSeen[key]; !ok || m.StartTime.Before(prev) {
			mapFirstSeen[key] = m.StartTime
		}
		if m.MapUI != "" {
			mapIDToUI[key] = m.MapUI
		}
	}
	if len(mapFirstSeen) == 0 {
		return nil
	}
	all := make([]mapStats, 0, len(mapFirstSeen))
	for k, first := range mapFirstSeen {
		ui := mapIDToUI[k]
		if ui == "" {
			ui = k
		}
		all = append(all, mapStats{mapUI: ui, firstSeen: first})
	}
	// Ordre CHRONOLOGIQUE de première apparition (carte jouée en premier en tête).
	// Verbatim utilisateur : « pas dans l'ordre et je veux pas de regroupement » —
	// on abandonne le tri par fréquence décroissante. Tie-break mapUI (déterminisme).
	sort.Slice(all, func(i, j int) bool {
		if !all[i].firstSeen.Equal(all[j].firstSeen) {
			return all[i].firstSeen.Before(all[j].firstSeen)
		}
		return all[i].mapUI < all[j].mapUI
	})
	// mapsTopKeys = clés internes (UUID ou MapUI), mapsTop = labels FR pour le frontend.
	mapsTopKeys := make([]string, 0, len(all))
	mapsTop := make([]string, 0, len(all))
	mapSet := make(map[string]bool, len(all))
	for _, ms := range all {
		// Retrouver la clé interne depuis le label d'affichage
		key := ms.mapUI
		for k, v := range mapIDToUI {
			if v == ms.mapUI {
				key = k
				break
			}
		}
		mapsTopKeys = append(mapsTopKeys, key)
		mapsTop = append(mapsTop, ms.mapUI)
		mapSet[key] = true
	}

	// 2. Joueurs : moi en tête + coéquipiers sélectionnés (dans l'ordre).
	players := append([]string{s.gamertag}, selectedGamertags...)

	// 3. Cellules. Main player : agrège PerformanceScore depuis allSquadRows.
	cells := make([]domain.SquadMapHeatmapCell, 0, len(players)*len(mapsTop))
	type cellAgg struct {
		sum    float64
		count  int
		nMatch int
	}
	mainAgg := make(map[string]*cellAgg)
	mainSeen := make(map[string]struct{}, len(allSquadRows))
	for _, m := range allSquadRows {
		if _, dup := mainSeen[m.MatchID]; dup {
			continue
		}
		mainSeen[m.MatchID] = struct{}{}
		key := m.MapID
		if key == "" {
			key = m.MapUI
		}
		if !mapSet[key] {
			continue
		}
		if _, ok := mainAgg[key]; !ok {
			mainAgg[key] = &cellAgg{}
		}
		mainAgg[key].nMatch++
		if m.PerformanceScore != nil {
			mainAgg[key].sum += *m.PerformanceScore
			mainAgg[key].count++
		}
	}
	for i, mapKey := range mapsTopKeys {
		mapUI := mapsTop[i]
		c := domain.SquadMapHeatmapCell{Player: s.gamertag, MapUI: mapUI}
		if a, ok := mainAgg[mapKey]; ok && a.nMatch > 0 {
			c.MatchCount = a.nMatch
			if a.count > 0 {
				p := round2(a.sum / float64(a.count))
				c.PerfAvg = &p
			}
		}
		cells = append(cells, c)
	}

	// 4. Coéquipiers : LoadPlayerMatches par gamertag, restreint aux match_ids de
	// la population escouade reçue (matchIDByID). Aucun re-filtrage privé ici :
	// allSquadRows porte DÉJÀ les filtres de session/période/cascade, et un second
	// filtre local (sessionMatchIDs, calculé sur les matchs du joueur principal)
	// donnait aux lignes coéquipiers une population plus étroite que la ligne du
	// main — une des sources du désaccord entre compteurs (retiré le 2026-08-02).
	matchIDsAllowed := matchIDByID

	for _, gt := range selectedGamertags {
		if s.squadLoader == nil {
			for _, mapUI := range mapsTop {
				cells = append(cells, domain.SquadMapHeatmapCell{Player: gt, MapUI: mapUI})
			}
			continue
		}
		mateRows, err := s.squadLoader.LoadFor(
			ctx, s.titleSlug, gt, port.PlayerMatchFilters{},
		)
		if err != nil {
			// Ligne de cartes vide pour ce coéquipier : l'UI doit signaler que la
			// heatmap est partielle plutôt que d'afficher des cellules muettes.
			issues.add(ctx, domain.DataIssueHeatmapTeammate, gt, err)
			for _, mapUI := range mapsTop {
				cells = append(cells, domain.SquadMapHeatmapCell{Player: gt, MapUI: mapUI})
			}
			continue
		}
		mateAgg := make(map[string]*cellAgg)
		for _, r := range mateRows {
			if _, ok := matchIDsAllowed[r.Summary.MatchID]; !ok {
				continue
			}
			if r.Summary.Map == nil {
				continue
			}
			// Clé interne = UUID (même source que MapID dans SquadMatchRow).
			key := r.Summary.Map.ID
			if key == "" {
				key = r.Summary.Map.DefaultLabel
			}
			if !mapSet[key] {
				continue
			}
			if _, ok := mateAgg[key]; !ok {
				mateAgg[key] = &cellAgg{}
			}
			mateAgg[key].nMatch++
			if r.Enrichment.PerformanceScore != nil {
				mateAgg[key].sum += *r.Enrichment.PerformanceScore
				mateAgg[key].count++
			}
		}
		for i, mapKey := range mapsTopKeys {
			mapUI := mapsTop[i]
			c := domain.SquadMapHeatmapCell{Player: gt, MapUI: mapUI}
			if a, ok := mateAgg[mapKey]; ok && a.nMatch > 0 {
				c.MatchCount = a.nMatch
				if a.count > 0 {
					p := round2(a.sum / float64(a.count))
					c.PerfAvg = &p
				}
			}
			cells = append(cells, c)
		}
	}

	return &domain.SquadMapHeatmap{
		Players:  players,
		MapsTopN: mapsTop,
		Cells:    cells,
	}
}

// ---------------------------------------------------------------------------
// teammates.07 — Impact scoreboard (8 badges)
// ---------------------------------------------------------------------------

// Badge keys canoniques (parité avec analysis.ComputeMatchImpactFull).
const (
	impactBadgeFirstBlood      = "first_blood"
	impactBadgeClutchFinisher  = "clutch_finisher"
	impactBadgeLastCasualty    = "last_casualty"
	impactBadgeLastGroupKill   = "last_group_kill"
	impactBadgeFirstGroupDeath = "first_group_death"
	impactBadgeSilentHero      = "silent_hero"
	impactBadgeFalseBrother    = "false_brother"
	impactBadgeKamikaze        = "kamikaze"
	impactBadgeTopKiller       = "top_killer"
)

// impactBadgeOrd est l'ordre canonique des colonnes agrégat du scoreboard.
var impactBadgeOrd = []string{
	impactBadgeFirstBlood, impactBadgeClutchFinisher, impactBadgeLastCasualty, impactBadgeLastGroupKill,
	impactBadgeFirstGroupDeath, impactBadgeSilentHero, impactBadgeFalseBrother, impactBadgeKamikaze, impactBadgeTopKiller,
}

// impactScoreWeights mappe chaque badge à son poids dans le score global du
// joueur (cf. .ai/charts_specs/teammates/07_impact_taquinerie.yaml constants).
var impactScoreWeights = map[string]float64{
	impactBadgeClutchFinisher:  2.0,
	impactBadgeFirstBlood:      2.0,
	impactBadgeLastCasualty:    -2.0,
	impactBadgeSilentHero:      1.5,
	impactBadgeFalseBrother:    -1.5,
	impactBadgeLastGroupKill:   -1.0,
	impactBadgeFirstGroupDeath: -1.0,
	impactBadgeKamikaze:        -1.0,
	impactBadgeTopKiller:       1.0,
}

// buildSquadImpactMatrix charge les events highlight + participants des matchs
// escouade, calcule les 8 badges via analysis.ComputeMatchImpactFull, et
// construit la matrice scoreboard. Filtre les matchs sans aucun event.
//
// Restreint le set de joueurs à : main + coéquipiers sélectionnés. Les badges
// d'autres joueurs (adversaires) sont ignorés.
//
//nolint:funlen // chart-builder cohésif (load events → compute badges → matrix).
