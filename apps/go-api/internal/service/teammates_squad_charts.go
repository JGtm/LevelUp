// Package service — teammates_squad_charts.go : builders pour les charts
// teammates.03 (heatmap player×map), teammates.04 (squad timeline par session)
// et teammates.07 (impact scoreboard).
//
// Tous les builders consomment l'union des matchs escouade (allSquadRows) +
// éventuellement des chargements complémentaires (LoadPlayerMatches pour la
// perf des coéquipiers, LoadImpactEvents pour les events).
package service

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// canonicalOutcomeToInt convertit canonical.Outcome (string) vers le code int
// historique consomme par analysis.ParticipantSnap (2/3/1/4).
func canonicalOutcomeToInt(o canonical.Outcome) int {
	switch o {
	case canonical.OutcomeWin:
		return analysis.OutcomeWin
	case canonical.OutcomeLoss:
		return analysis.OutcomeLoss
	case canonical.OutcomeTie:
		return analysis.OutcomeTie
	case canonical.OutcomeDNF:
		return analysis.OutcomeDNF
	}
	return 0
}

// ---------------------------------------------------------------------------
// teammates.04 — Squad timeline par session
// ---------------------------------------------------------------------------

// buildSquadSessionTimeline regroupe allSquadRows par SessionLabel et calcule
// les agrégats (perf, win_rate, MMR moyen). Les matchs sans SessionLabel sont
// rangés dans le bucket "(hors session)". Tri final : par première occurrence
// de StartTime ASC dans chaque bucket.
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
	out := make([]domain.SquadSessionPoint, len(sortables))
	for i, s := range sortables {
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
func (s *TeammatesService) buildSquadMapHeatmap(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	selectedGamertags []string,
	sessionMatchIDs map[string]bool,
) *domain.SquadMapHeatmap {
	if len(allSquadRows) == 0 {
		return nil
	}

	// 1. Toutes les cartes jouées en escouade (dédup match_id).
	// Clé interne = MapID (UUID, language-agnostic) si dispo, sinon MapUI.
	// mapIDToUI assure la correspondance UUID → label d'affichage (FR).
	type mapStats struct {
		mapUI string
		count int
	}
	mapCounts := make(map[string]int)
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
		mapCounts[key]++
		if m.MapUI != "" {
			mapIDToUI[key] = m.MapUI
		}
	}
	if len(mapCounts) == 0 {
		return nil
	}
	all := make([]mapStats, 0, len(mapCounts))
	for k, c := range mapCounts {
		ui := mapIDToUI[k]
		if ui == "" {
			ui = k
		}
		all = append(all, mapStats{mapUI: ui, count: c})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].count != all[j].count {
			return all[i].count > all[j].count
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

	// 4. Coéquipiers : LoadPlayerMatches par gamertag, filtre sur match_ids escouade.
	matchIDsAllowed := matchIDByID
	if len(sessionMatchIDs) > 0 {
		filtered := make(map[string]struct{}, len(sessionMatchIDs))
		for id := range matchIDByID {
			if sessionMatchIDs[id] {
				filtered[id] = struct{}{}
			}
		}
		matchIDsAllowed = filtered
	}

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
			slog.WarnContext(ctx, "teammates_heatmap_load_failed",
				"gamertag", gt, "err", err.Error())
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

// impactBadgeOrd est l'ordre canonique des colonnes agrégat du scoreboard.
var impactBadgeOrd = []string{
	"first_blood", "clutch_finisher", "last_casualty", "last_group_kill",
	"first_group_death", "silent_hero", "false_brother", "top_killer",
}

// impactScoreWeights mappe chaque badge à son poids dans le score global du
// joueur (cf. .ai/charts_specs/teammates/07_impact_taquinerie.yaml constants).
var impactScoreWeights = map[string]float64{
	"clutch_finisher":   2.0,
	"first_blood":       2.0,
	"last_casualty":     -2.0,
	"silent_hero":       1.5,
	"false_brother":     -1.5,
	"last_group_kill":   -1.0,
	"first_group_death": -1.0,
	"top_killer":        1.0,
}

// buildSquadImpactMatrix charge les events highlight + participants des matchs
// escouade, calcule les 8 badges via analysis.ComputeMatchImpactFull, et
// construit la matrice scoreboard. Filtre les matchs sans aucun event.
//
// Restreint le set de joueurs à : main + coéquipiers sélectionnés. Les badges
// d'autres joueurs (adversaires) sont ignorés.
func (s *TeammatesService) buildSquadImpactMatrix(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainXUID string,
	mainGamertag string,
	selectedGamertags []string,
) *domain.SquadImpactMatrix {
	if len(allSquadRows) == 0 || len(selectedGamertags) == 0 {
		return nil
	}

	// 1. Match IDs uniques + outcome du main player. On trie ensuite par
	//    start_time ASC pour que la colonne #1 du scoreboard corresponde au
	//    match le plus ancien (Q30SquadMatches retourne DESC, et allSquadRows
	//    concatène plusieurs teammates donc l'ordre d'arrivée n'est pas
	//    globalement chronologique).
	mainOutcomeByMatch := make(map[string]int)
	startTimeByMatch := make(map[string]time.Time)
	matchIDOrder := make([]string, 0, len(allSquadRows))
	for _, m := range allSquadRows {
		if _, ok := mainOutcomeByMatch[m.MatchID]; ok {
			continue
		}
		mainOutcomeByMatch[m.MatchID] = m.Outcome
		startTimeByMatch[m.MatchID] = m.StartTime
		matchIDOrder = append(matchIDOrder, m.MatchID)
	}
	if len(matchIDOrder) == 0 {
		return nil
	}
	slices.SortStableFunc(matchIDOrder, func(a, b string) int {
		return startTimeByMatch[a].Compare(startTimeByMatch[b])
	})

	// 2. Charger les events impact pour tous les matchs.
	eventsByMatch := s.loadImpactEventsByMatch(ctx, matchIDOrder)

	// 3. Charger les participants de l'ÉQUIPE ALLIÉE complète du main pour
	//    chaque match (parité Python team_xuids dans compute_single_match_impact).
	//    On passera tous ces alliés à analysis.ComputeMatchImpactFull → les
	//    badges seront calculés en team-wide. Le filtre xuidToGT ci-dessous
	//    ne contient QUE les squad members (main + selected) → les badges qui
	//    tombent sur un allié non-squad sont silencieusement ignorés (cohérent
	//    avec la sémantique de la matrice scoreboard où il n'y a pas de
	//    ligne pour ces joueurs).
	allyByMatch := map[string][]domain.AllyParticipant{}
	if mainXUID != "" {
		allies, err := s.repo.LoadMainTeamParticipants(ctx, mainXUID, matchIDOrder)
		if err != nil {
			slog.WarnContext(ctx, "teammates_impact_load_team_failed",
				"main_xuid", mainXUID, "err", err.Error())
		}
		for _, a := range allies {
			allyByMatch[a.MatchID] = append(allyByMatch[a.MatchID], a)
		}
	}

	// xuid → gamertag des squad members uniquement (main + selected). Sert à
	// filtrer les badges affichés dans le scoreboard.
	xuidToGT := map[string]string{}
	gamertagSet := map[string]bool{mainGamertag: true}
	for _, gt := range selectedGamertags {
		gamertagSet[gt] = true
	}
	if mainXUID != "" {
		xuidToGT[mainXUID] = mainGamertag
	}
	for _, allies := range allyByMatch {
		for _, a := range allies {
			if a.XUID == "" {
				continue
			}
			if _, isSquad := gamertagSet[a.Gamertag]; isSquad {
				xuidToGT[a.XUID] = a.Gamertag
			}
		}
	}

	// 4. Pour chaque match, calculer les badges via analysis.ComputeMatchImpactFull
	//    et collecter les badges des joueurs de l'escouade uniquement.
	cells := []domain.SquadImpactCell{}
	keptMatchIDs := []string{}
	playerCounts := map[string]map[string]int{} // gamertag → badge_key → count
	playerScore := map[string]float64{}
	for gt := range gamertagSet {
		playerCounts[gt] = make(map[string]int, len(impactBadgeOrd))
		playerScore[gt] = 0
	}

	for _, mid := range matchIDOrder {
		evs := eventsByMatch[mid]
		allies := allyByMatch[mid]
		// Snaps = TOUS les alliés du main pour ce match (équipe alliée
		// complète, pas seulement squad). Les filtres internes de
		// ComputeMatchImpactFull (winXUIDs, lossXUIDs, squadXUIDs) en
		// découlent → calcul team-wide alliée.
		snaps := make([]analysis.ParticipantSnap, 0, len(allies))
		for _, a := range allies {
			snaps = append(snaps, analysis.ParticipantSnap{
				XUID: a.XUID, Outcome: a.Outcome, Kills: a.Kills, Deaths: a.Deaths, Assists: a.Assists,
			})
		}
		if len(snaps) == 0 && len(evs) == 0 {
			continue
		}
		badges := analysis.ComputeMatchImpactFull(analysis.MatchImpactInput{
			Events: evs, Participants: snaps,
		})
		// Filtrer aux badges des joueurs de l'escouade ET aux 8 badges du
		// scoreboard impact (parité Python : top_gun n'est pas inclus dans
		// la matrice impact même s'il est calculé). Les badges qui tombent
		// sur un allié non-squad sont droppés ici via xuidToGT.
		matchHadBadge := false
		cellByGT := map[string][]string{}
		for _, b := range badges {
			if _, scored := impactScoreWeights[b.BadgeKey]; !scored {
				continue
			}
			gt, ok := xuidToGT[b.PlayerXUID]
			if !ok {
				continue
			}
			cellByGT[gt] = append(cellByGT[gt], b.BadgeKey)
			playerCounts[gt][b.BadgeKey]++
			playerScore[gt] += impactScoreWeights[b.BadgeKey]
			matchHadBadge = true
		}
		if !matchHadBadge {
			continue
		}
		keptMatchIDs = append(keptMatchIDs, mid)
		for gt, keys := range cellByGT {
			cells = append(cells, domain.SquadImpactCell{
				Player: gt, MatchID: mid, BadgeKeys: keys,
			})
		}
	}

	if len(keptMatchIDs) == 0 {
		return nil
	}

	// 5. Build players summary triée par score DESC.
	playersOrdered := make([]string, 0, len(gamertagSet))
	for gt := range gamertagSet {
		playersOrdered = append(playersOrdered, gt)
	}
	slices.SortFunc(playersOrdered, func(a, b string) int {
		// score DESC ; tie-break alphabétique pour stabilité
		if c := cmp.Compare(playerScore[b], playerScore[a]); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})

	playerSummaries := make([]domain.SquadImpactPlayerSummary, 0, len(playersOrdered))
	for _, gt := range playersOrdered {
		counts := make([]domain.SquadImpactBadgeCount, 0, len(impactBadgeOrd))
		for _, badge := range impactBadgeOrd {
			counts = append(counts, domain.SquadImpactBadgeCount{
				BadgeKey: badge, Count: playerCounts[gt][badge],
			})
		}
		playerSummaries = append(playerSummaries, domain.SquadImpactPlayerSummary{
			Player: gt,
			Counts: counts,
			Score:  round2(playerScore[gt]),
		})
	}

	// 6. Build match headers (outcome du main).
	matchHeaders := make([]domain.SquadImpactMatchHeader, 0, len(keptMatchIDs))
	for _, mid := range keptMatchIDs {
		matchHeaders = append(matchHeaders, domain.SquadImpactMatchHeader{
			MatchID: mid, Outcome: mainOutcomeByMatch[mid],
		})
	}

	return &domain.SquadImpactMatrix{
		Matches:  matchHeaders,
		Players:  playerSummaries,
		Cells:    cells,
		BadgeOrd: append([]string{}, impactBadgeOrd...),
	}
}

// intPtrOrZero retourne *p si non nil, 0 sinon.
func intPtrOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// ---------------------------------------------------------------------------
// teammates.17 — Premier frag / première mort — butterfly chronologique
// ---------------------------------------------------------------------------

// firstEventsBinSize est la taille d'un bin temporel (15 s) — cf. spec.
const firstEventsBinSize = 15

// formatFirstEventsBinLabel formate la borne droite du bin selon le format spec :
//   - < 60s    → "Ns"   (ex "15s", "45s")
//   - >= 60s   → "MmSSs" (ex "1m00s", "2m15s")
func formatFirstEventsBinLabel(rightEdgeSec int) string {
	if rightEdgeSec < 60 {
		return fmt.Sprintf("%ds", rightEdgeSec)
	}
	m := rightEdgeSec / 60
	s := rightEdgeSec % 60
	return fmt.Sprintf("%dm%02ds", m, s)
}

// buildSquadFirstEvents charge les events highlight, calcule pour chaque
// (match_id, xuid de l'escouade) le first_kill_s et first_death_s, puis
// bucket en bins de 15 s. Retourne nil si aucun event ou aucun joueur résolu.
func (s *TeammatesService) buildSquadFirstEvents(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	teammates []domain.TeammateRow,
) *domain.SquadFirstEvents {
	if s.repo == nil || len(allSquadRows) == 0 {
		return nil
	}

	// 1. Match IDs uniques.
	matchIDs := make([]string, 0, len(allSquadRows))
	seen := make(map[string]struct{}, len(allSquadRows))
	for _, m := range allSquadRows {
		if _, ok := seen[m.MatchID]; ok {
			continue
		}
		seen[m.MatchID] = struct{}{}
		matchIDs = append(matchIDs, m.MatchID)
	}
	if len(matchIDs) == 0 {
		return nil
	}

	// 2. xuid → gamertag pour les joueurs de l'escouade (main + teammates).
	gtByXUID := make(map[string]string)
	playersOrdered := make([]string, 0, 1+len(teammates))
	if mainXUID != "" {
		gtByXUID[mainXUID] = mainGamertag
		playersOrdered = append(playersOrdered, mainGamertag)
	}
	for _, tm := range teammates {
		if tm.XUID == nil || *tm.XUID == "" {
			continue
		}
		gtByXUID[*tm.XUID] = tm.Gamertag
		playersOrdered = append(playersOrdered, tm.Gamertag)
	}
	if len(playersOrdered) == 0 {
		return nil
	}

	// 3. Charger les events.
	events, err := s.repo.LoadImpactEvents(ctx, matchIDs)
	if err != nil || len(events) == 0 {
		if err != nil {
			slog.WarnContext(ctx, "teammates_first_events_load_failed", "err", err.Error())
		}
		return nil
	}

	// 4. Pour chaque (match, xuid) calculer first_kill_s et first_death_s
	// (en secondes, time_ms / 1000).
	type firstTimes struct {
		firstKillS  int64 // -1 si absent
		firstDeathS int64
	}
	keyOf := func(matchID, xuid string) string { return matchID + "\x00" + xuid }
	firsts := make(map[string]*firstTimes)
	for _, e := range events {
		gt, ok := gtByXUID[e.XUID]
		if !ok {
			continue
		}
		_ = gt
		k := keyOf(e.MatchID, e.XUID)
		ft := firsts[k]
		if ft == nil {
			ft = &firstTimes{firstKillS: -1, firstDeathS: -1}
			firsts[k] = ft
		}
		secs := e.TimeMS / 1000
		switch e.EventType {
		case "kill":
			if ft.firstKillS == -1 || secs < ft.firstKillS {
				ft.firstKillS = secs
			}
		case "death":
			if ft.firstDeathS == -1 || secs < ft.firstDeathS {
				ft.firstDeathS = secs
			}
		}
	}
	if len(firsts) == 0 {
		return nil
	}

	// 5. Trouver le max bin nécessaire (max(first_kill_s, first_death_s) sur tous).
	maxSec := 0
	for _, ft := range firsts {
		if ft.firstKillS > 0 && int(ft.firstKillS) > maxSec {
			maxSec = int(ft.firstKillS)
		}
		if ft.firstDeathS > 0 && int(ft.firstDeathS) > maxSec {
			maxSec = int(ft.firstDeathS)
		}
	}
	if maxSec == 0 {
		// tous les events à t=0 ou aucun kill/death — bin minimal
		maxSec = firstEventsBinSize
	}
	nBins := (maxSec / firstEventsBinSize) + 1
	if nBins < 1 {
		nBins = 1
	}

	// 6. Bucketing : pour chaque (match, xuid), incrémenter le bin correspondant.
	binOf := func(secs int64) int {
		if secs < 0 {
			return -1
		}
		b := int(secs) / firstEventsBinSize
		if b >= nBins {
			b = nBins - 1
		}
		return b
	}

	// kill_counts[gt][bin], death_counts[gt][bin]
	killByGT := make(map[string][]int, len(playersOrdered))
	deathByGT := make(map[string][]int, len(playersOrdered))
	for _, gt := range playersOrdered {
		killByGT[gt] = make([]int, nBins)
		deathByGT[gt] = make([]int, nBins)
	}

	// Re-parcours du map firsts en retrouvant le gamertag depuis la clé.
	for k, ft := range firsts {
		// La clé est "matchID\x00xuid" — on extrait le xuid.
		sep := -1
		for i := 0; i < len(k); i++ {
			if k[i] == 0 {
				sep = i
				break
			}
		}
		if sep < 0 {
			continue
		}
		xuid := k[sep+1:]
		gt, ok := gtByXUID[xuid]
		if !ok {
			continue
		}
		if b := binOf(ft.firstKillS); b >= 0 {
			killByGT[gt][b]++
		}
		if b := binOf(ft.firstDeathS); b >= 0 {
			deathByGT[gt][b]++
		}
	}

	// 7. Vérifier qu'au moins une cellule non nulle.
	hasAny := false
	for _, gt := range playersOrdered {
		for i := 0; i < nBins; i++ {
			if killByGT[gt][i] > 0 || deathByGT[gt][i] > 0 {
				hasAny = true
				break
			}
		}
		if hasAny {
			break
		}
	}
	if !hasAny {
		return nil
	}

	// 8. Build labels (borne droite de chaque bin).
	binLabels := make([]string, nBins)
	for i := 0; i < nBins; i++ {
		binLabels[i] = formatFirstEventsBinLabel((i + 1) * firstEventsBinSize)
	}

	// 9. Build rows (1 par joueur dans l'ordre canonique).
	rows := make([]domain.SquadFirstEventsRow, 0, len(playersOrdered))
	for _, gt := range playersOrdered {
		rows = append(rows, domain.SquadFirstEventsRow{
			Player:      gt,
			KillCounts:  killByGT[gt],
			DeathCounts: deathByGT[gt],
		})
	}

	return &domain.SquadFirstEvents{
		BinSizeSeconds: firstEventsBinSize,
		BinLabels:      binLabels,
		Rows:           rows,
	}
}

// ---------------------------------------------------------------------------
// teammates.09 — Kills par arme — comparatif multi-joueurs
// ---------------------------------------------------------------------------

// buildSquadWeaponKills charge `LoadWeaponKills` via le squadLoader pour le
// main + chaque teammate (via leur xuid), agrège par weapon_id et trie ASC
// par total escouade (peu utilisées en haut).
//
// Renvoie nil si :
//   - squadLoader == nil (DI non câblée)
//   - aucun match partagé
//   - aucun joueur avec xuid résolu
//   - le repo ne renvoie aucune donnée (capability absente ou tables vides)
func (s *TeammatesService) buildSquadWeaponKills(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	teammates []domain.TeammateRow,
) *domain.SquadWeaponKills {
	if s.squadLoader == nil || len(allSquadRows) == 0 || len(teammates) == 0 {
		return nil
	}

	// 1. Matchs partagés (intersection — un match qui apparaît N fois dans
	// allSquadRows = présent pour tous les teammates).
	matchOccurrences := make(map[string]int)
	for _, m := range allSquadRows {
		matchOccurrences[m.MatchID]++
	}
	sharedMatches := make([]string, 0)
	for mid, n := range matchOccurrences {
		if n >= len(teammates) {
			sharedMatches = append(sharedMatches, mid)
		}
	}
	if len(sharedMatches) == 0 {
		return nil
	}

	// 2. xuid map (main + teammates avec xuid résolu).
	xuidByPlayer := make(map[string]string)
	playersOrdered := make([]string, 0, 1+len(teammates))
	if mainXUID != "" {
		xuidByPlayer[mainGamertag] = mainXUID
		playersOrdered = append(playersOrdered, mainGamertag)
	}
	for _, tm := range teammates {
		if tm.XUID == nil || *tm.XUID == "" {
			continue
		}
		xuidByPlayer[tm.Gamertag] = *tm.XUID
		playersOrdered = append(playersOrdered, tm.Gamertag)
	}
	if len(playersOrdered) == 0 {
		return nil
	}

	xuids := make([]string, 0, len(playersOrdered))
	for _, p := range playersOrdered {
		xuids = append(xuids, xuidByPlayer[p])
	}

	// 3. 1 seul appel pour tous les xuids.
	rows, err := s.squadLoader.LoadWeaponKills(ctx, s.titleSlug, port.WeaponKillFilters{
		MatchIDs:            sharedMatches,
		XUIDs:               xuids,
		IncludeGrenadeMelee: true,
	})
	if err != nil {
		slog.WarnContext(ctx, "teammates_weapon_kills_load_failed", "err", err.Error())
		return nil
	}
	if len(rows) == 0 {
		return nil
	}

	// 4. Réindexation xuid → gamertag pour l'agrégation.
	gtByXUID := make(map[string]string, len(xuidByPlayer))
	for gt, x := range xuidByPlayer {
		gtByXUID[x] = gt
	}

	// 5. Group by weapon_id.
	type barAgg struct {
		weaponID       int64
		label          string
		isGrenadeMelee bool
		kills          map[string]int
		total          int
	}
	bars := make(map[int64]*barAgg)
	for _, r := range rows {
		gt, ok := gtByXUID[r.XUID]
		if !ok {
			continue
		}
		b, exists := bars[r.WeaponID]
		if !exists {
			b = &barAgg{
				weaponID:       r.WeaponID,
				label:          r.Label,
				isGrenadeMelee: r.IsGrenadeMelee,
				kills:          make(map[string]int),
			}
			bars[r.WeaponID] = b
		}
		b.kills[gt] += r.Kills
		b.total += r.Kills
		// Privilégier un label non-vide.
		if b.label == "" && r.Label != "" {
			b.label = r.Label
		}
	}
	if len(bars) == 0 {
		return nil
	}

	// 6. Tri ASC par TotalSquad (peu utilisées en haut), tie-break par label.
	out := make([]domain.SquadWeaponBar, 0, len(bars))
	for _, b := range bars {
		out = append(out, domain.SquadWeaponBar{
			WeaponID:       b.weaponID,
			Label:          b.label,
			IsGrenadeMelee: b.isGrenadeMelee,
			KillsByPlayer:  b.kills,
			TotalSquad:     b.total,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TotalSquad != out[j].TotalSquad {
			return out[i].TotalSquad < out[j].TotalSquad
		}
		return out[i].Label < out[j].Label
	})

	return &domain.SquadWeaponKills{
		Players: playersOrdered,
		Bars:    out,
	}
}

// ---------------------------------------------------------------------------
// teammates.16 — Charts de performance escouade (8 sous-charts par joueur)
// ---------------------------------------------------------------------------

// buildSquadPerformanceSeries construit la time-series par match × joueur
// sur l'INTERSECTION des matchs (= matchs où tous les coéquipiers
// sélectionnés + le main player étaient présents).
//
// Les MatchOrder sont alignés (0..N-1) sur les matchs triés chronologiquement
// par StartTime ASC. Tous les joueurs ont la même longueur de série.
func (s *TeammatesService) buildSquadPerformanceSeries(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag string,
	selectedGamertags []string,
) map[string][]domain.SquadPerformanceSeriesPoint {
	if len(allSquadRows) == 0 || len(selectedGamertags) == 0 {
		return nil
	}

	// 1. Matchs partagés (un match qui apparaît N=len(selectedGamertags) fois
	// dans allSquadRows = présent pour tous les coéquipiers + le main).
	matchOccurrences := make(map[string]int)
	startTimeByMatch := make(map[string]int64)
	for _, m := range allSquadRows {
		matchOccurrences[m.MatchID]++
		if _, ok := startTimeByMatch[m.MatchID]; !ok {
			startTimeByMatch[m.MatchID] = m.StartTime.Unix()
		}
	}
	sharedMatches := make([]string, 0)
	for mid, n := range matchOccurrences {
		if n >= len(selectedGamertags) {
			sharedMatches = append(sharedMatches, mid)
		}
	}
	if len(sharedMatches) == 0 {
		return nil
	}
	// Tri chronologique ASC.
	sort.SliceStable(sharedMatches, func(i, j int) bool {
		return startTimeByMatch[sharedMatches[i]] < startTimeByMatch[sharedMatches[j]]
	})
	matchOrderIndex := make(map[string]int, len(sharedMatches))
	for i, mid := range sharedMatches {
		matchOrderIndex[mid] = i
	}

	out := make(map[string][]domain.SquadPerformanceSeriesPoint, 1+len(selectedGamertags))

	// squadLoader.LoadFor résout les rows par gamertag (playerMatchesRepo est bound
	// au main et ignore l'arg gt → toutes les séries affichaient les stats du main).
	loadFor := func(gt string) []domain.SquadPerformanceSeriesPoint {
		if s.squadLoader == nil {
			return nil
		}
		rows, err := s.squadLoader.LoadFor(ctx, s.titleSlug, gt, port.PlayerMatchFilters{})
		if err != nil {
			slog.WarnContext(ctx, "teammates_perf_series_load_failed", "gamertag", gt, "err", err.Error())
			return nil
		}
		series := make([]domain.SquadPerformanceSeriesPoint, 0, len(sharedMatches))
		for _, r := range rows {
			idx, ok := matchOrderIndex[r.Summary.MatchID]
			if !ok {
				continue
			}
			pt := domain.SquadPerformanceSeriesPoint{
				MatchID:    r.Summary.MatchID,
				StartTime:  r.Summary.StartedAtUTC.Format("2006-01-02T15:04:05Z"),
				MatchOrder: idx,
				Kills:      intPtrOrZero(r.Self.Kills),
				Deaths:     intPtrOrZero(r.Self.Deaths),
				Assists:    intPtrOrZero(r.Self.Assists),
			}
			if r.Self.KDA != nil {
				v := round2(*r.Self.KDA)
				pt.KDA = &v
			}
			if r.Self.Accuracy != nil {
				v := round2(*r.Self.Accuracy)
				pt.Accuracy = &v
			}
			if r.Self.AvgLifeSeconds != nil {
				v := round2(*r.Self.AvgLifeSeconds)
				pt.AvgLifeSeconds = &v
			}
			if r.Enrichment.PerformanceScore != nil {
				v := round2(*r.Enrichment.PerformanceScore)
				pt.PerformanceScore = &v
			}
			if r.Self.MaxKillingSpree != nil {
				v := *r.Self.MaxKillingSpree
				pt.MaxKillingSpree = &v
			}
			if r.Self.HeadshotKills != nil {
				v := *r.Self.HeadshotKills
				pt.HeadshotKills = &v
			}
			if r.Self.PerfectKills != nil {
				v := *r.Self.PerfectKills
				pt.PerfectKills = &v
			}
			if r.Self.DamageDealt != nil && *r.Self.DamageDealt > 0 {
				v := round2(synergyOffensiveConversion(pt.Kills, pt.Assists, float64(*r.Self.DamageDealt)))
				pt.RendementOffensif = &v
			}
			if r.Self.DamageTaken != nil {
				v := round2(synergyDefensiveResistance(float64(*r.Self.DamageTaken), pt.Deaths))
				pt.ResistanceDefensive = &v
			}
			series = append(series, pt)
		}
		// Sort by MatchOrder pour garantir l'alignement avec les autres joueurs.
		sort.SliceStable(series, func(i, j int) bool {
			return series[i].MatchOrder < series[j].MatchOrder
		})
		return series
	}

	for _, gt := range append([]string{mainGamertag}, selectedGamertags...) {
		s := loadFor(gt)
		if len(s) > 0 {
			out[gt] = s
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ---------------------------------------------------------------------------
// teammates.06 — Radar synergie escouade (6 axes)
// ---------------------------------------------------------------------------

// synergyRadarThresholds retourne les seuils du radar scalés par nShared.
// Les axes absolus (combat/support/score/objective) sont proportionnels au
// nombre de matchs partagés. Impact et Survival sont des ratios agrégés →
// seuil fixe étiré au-dessus du P80 observé pour préserver la variance
// inter-joueurs (un squad de joueurs corrects dépasse tous le P80, ce qui
// aplatissait les axes à 100).
//
// Combat calibration : accuracy est stockée 0..100 en DB (voir transforms.go).
// Dans loadSynergyMateAxes/synergyMainFallbackAxes acc est normalisé /100
// avant usage → multiplicateur réel = 1 + (acc/100)×0.4 ≈ 1.2 pour 50%.
// Un bon joueur (~12K, 5HS, 2PK, 55% acc) produit ~19/match, seuil 25 ≈ 76%.
func synergyRadarThresholds(nShared int) narrative.ParticipationThresholds {
	n := float64(nShared)
	return narrative.ParticipationThresholds{
		Combat:    25.0 * n,                               // (kills+HS/2+PK/2)×(1+acc×0.4), ~25/match pour un excellent joueur
		Survival:  analysis.DefensiveResistanceP80 * 1.25, // ~1.99 ; étire le haut au-dessus du P80
		Support:   300.0 * n,                              // assists × 50, ~6 assists/match
		Score:     350.0 * n,                              // résiduel medals/streaks, ~350/match
		Objective: 350.0 * n,                              // PSA objectif, ~350/match
		Impact:    analysis.OffensiveConversionP80 * 1.25, // ~1.04 ; étire le haut au-dessus du P80
	}
}

// synergyOffensiveConversion calcule le rendement offensif agrégé sur l'ensemble
// des matchs : 225 × (ΣK + ΣA/3) / ΣDD. Retourne 0 si aucun dégât infligé.
func synergyOffensiveConversion(totalKills, totalAssists int, totalDamageDlt float64) float64 {
	if totalDamageDlt <= 0 {
		return 0
	}
	return 225.0 * (float64(totalKills) + float64(totalAssists)/3.0) / totalDamageDlt
}

// synergyDefensiveResistance calcule la résistance défensive agrégée :
// ΣDT / (225 × ΣD). Zéro mort avec damage positif → score parfait (au-delà du P80).
func synergyDefensiveResistance(totalDamageTkn float64, totalDeaths int) float64 {
	if totalDeaths == 0 {
		if totalDamageTkn > 0 {
			return analysis.DefensiveResistanceP80 * analysis.CombatYieldClipFactor
		}
		return 0
	}
	return totalDamageTkn / (225.0 * float64(totalDeaths))
}

// synergyMainFallbackAxes calcule combat et support depuis SquadMatchRow
// (fallback quand squadLoader est absent). Impact et Survival restent à 0
// car damage_dealt/damage_taken ne sont pas dans SquadMatchRow.
func synergyMainFallbackAxes(
	allSquadRows []domain.SquadMatchRow,
	sharedMatches map[string]struct{},
) map[narrative.ParticipationAxis]float64 {
	seen := make(map[string]struct{})
	raw := map[narrative.ParticipationAxis]float64{}
	for _, m := range allSquadRows {
		if _, ok := sharedMatches[m.MatchID]; !ok {
			continue
		}
		if _, dup := seen[m.MatchID]; dup {
			continue
		}
		seen[m.MatchID] = struct{}{}
		acc := 0.0
		if m.Accuracy != nil {
			acc = *m.Accuracy / 100.0 // DB stocke 0..100, formule attend 0..1
		}
		combat := (float64(m.Kills) + 0.5*float64(m.HeadshotKills) + 0.5*float64(m.PerfectKills)) *
			(1.0 + acc*0.4)
		raw[narrative.AxisCombat] += combat
		raw[narrative.AxisSupport] += float64(m.Assists) * 50.0
	}
	return raw
}

// loadSynergyMateAxes charge les 6 axes radar depuis canonical.PlayerMatchRow pour
// un gamertag, restreint aux matchs partagés. Formules :
//   - Combat  : (kills + HS/2 + PK/2) × (1 + accuracy × 0.4), somme sur les matchs
//   - Survival : résistance défensive agrégée ΣDT / (225 × ΣD)
//   - Support  : assists × 50, somme sur les matchs
//   - Score    : résiduel PS après kills×100 + assists×50 + objectif (medals/streaks)
//   - Objective: PSA catégorie "objective" via LoadObjectiveScores
//   - Impact   : rendement offensif agrégé 225×(ΣK+ΣA/3)/ΣDD
func (s *TeammatesService) loadSynergyMateAxes(
	ctx context.Context,
	gt string,
	sharedMatches map[string]struct{},
	sharedMatchIDs []string,
) map[narrative.ParticipationAxis]float64 {
	raw := map[narrative.ParticipationAxis]float64{}
	if s.squadLoader == nil {
		return raw
	}
	rows, err := s.squadLoader.LoadFor(ctx, s.titleSlug, gt, port.PlayerMatchFilters{})
	if err != nil {
		slog.WarnContext(ctx, "teammates_radar_load_failed", "gamertag", gt, "err", err.Error())
		return raw
	}

	var totalKills, totalAssists, totalDeaths int
	var totalDamageDlt, totalDamageTkn, totalPS float64
	for _, r := range rows {
		if _, ok := sharedMatches[r.Summary.MatchID]; !ok {
			continue
		}
		k := intPtrOrZero(r.Self.Kills)
		hs := intPtrOrZero(r.Self.HeadshotKills)
		pk := intPtrOrZero(r.Self.PerfectKills)
		a := intPtrOrZero(r.Self.Assists)
		acc := 0.0
		if r.Self.Accuracy != nil {
			acc = *r.Self.Accuracy / 100.0 // DB stocke 0..100, formule attend 0..1
		}
		ps := intPtrOrZero(r.Self.PersonalScore)
		if ps == 0 {
			ps = intPtrOrZero(r.Self.Score)
		}
		raw[narrative.AxisCombat] += (float64(k) + 0.5*float64(hs) + 0.5*float64(pk)) * (1.0 + acc*0.4)
		raw[narrative.AxisSupport] += float64(a) * 50.0
		totalKills += k
		totalAssists += a
		totalDeaths += intPtrOrZero(r.Self.Deaths)
		totalDamageDlt += float64(intPtrOrZero(r.Self.DamageDealt))
		totalDamageTkn += float64(intPtrOrZero(r.Self.DamageTaken))
		totalPS += float64(ps)
	}

	raw[narrative.AxisImpact] = synergyOffensiveConversion(totalKills, totalAssists, totalDamageDlt)
	raw[narrative.AxisSurvival] = synergyDefensiveResistance(totalDamageTkn, totalDeaths)

	// Objective via PSA — dégradation silencieuse si absent.
	var objTotal float64
	if objScores, err := s.squadLoader.LoadObjectiveScores(ctx, s.titleSlug, gt, sharedMatchIDs); err == nil {
		for _, v := range objScores {
			objTotal += float64(v)
		}
		raw[narrative.AxisObjective] = objTotal
	}

	// Score : résiduel après kills×100 + assists×50 + objectif (= medals/streaks).
	residual := totalPS - float64(totalKills)*100.0 - float64(totalAssists)*50.0 - objTotal
	if residual < 0 {
		residual = 0
	}
	raw[narrative.AxisScore] = residual
	return raw
}

// buildSquadSynergyRadar calcule un profil de participation 6 axes par joueur
// sur l'INTERSECTION des matchs (matchs où TOUS les coéquipiers sélectionnés
// + le main player étaient présents). Formules alignées sur participation_radar.py.
func (s *TeammatesService) buildSquadSynergyRadar(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag string,
	selectedGamertags []string,
) []domain.SquadSynergyRadarSeries {
	if len(allSquadRows) == 0 || len(selectedGamertags) == 0 {
		return nil
	}

	// Matchs PARTAGÉS : présents pour chaque coéquipier sélectionné.
	matchOccurrences := make(map[string]int)
	for _, m := range allSquadRows {
		matchOccurrences[m.MatchID]++
	}
	sharedMatches := make(map[string]struct{})
	for mid, n := range matchOccurrences {
		if n >= len(selectedGamertags) {
			sharedMatches[mid] = struct{}{}
		}
	}
	if len(sharedMatches) == 0 {
		return nil
	}

	sharedMatchIDs := make([]string, 0, len(sharedMatches))
	for mid := range sharedMatches {
		sharedMatchIDs = append(sharedMatchIDs, mid)
	}

	thresholds := synergyRadarThresholds(len(sharedMatches))

	mainRaw := s.loadSynergyMateAxes(ctx, mainGamertag, sharedMatches, sharedMatchIDs)
	if len(mainRaw) == 0 {
		mainRaw = synergyMainFallbackAxes(allSquadRows, sharedMatches)
	}

	out := make([]domain.SquadSynergyRadarSeries, 0, 1+len(selectedGamertags))
	toSeries := func(player string, raw map[narrative.ParticipationAxis]float64) domain.SquadSynergyRadarSeries {
		scores := narrative.ComputeParticipationProfile(raw, thresholds)
		axes := make([]domain.SquadSynergyRadarAxis, 0, len(scores))
		for _, sc := range scores {
			axes = append(axes, domain.SquadSynergyRadarAxis{
				Axis:  string(sc.Axis),
				Value: round2(sc.Value),
				Raw:   round2(sc.Raw),
			})
		}
		return domain.SquadSynergyRadarSeries{Player: player, Axes: axes}
	}

	out = append(out, toSeries(mainGamertag, mainRaw))
	for _, gt := range selectedGamertags {
		out = append(out, toSeries(gt, s.loadSynergyMateAxes(ctx, gt, sharedMatches, sharedMatchIDs)))
	}
	return out
}

// ---------------------------------------------------------------------------
// teammates.15 — Heatmap intensité kills par phase de match (10 buckets)
// ---------------------------------------------------------------------------

const intensityBuckets = 10
const intensityMinMatches = 3

// buildSquadIntensityProfile charge les kill events highlight pour les matchs
// du scope, calcule pour chaque option (all + main + teammates) un profil
// d'intensité 10-buckets × N matchs (normalisé par match).
//
// Renvoie nil si <3 matchs (section masquée), aucun kill event, ou aucune
// option ne produit de profil.
func (s *TeammatesService) buildSquadIntensityProfile(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag string,
	selectedGamertags []string,
	allTeamLabel string,
) *domain.SquadIntensityProfile {
	if s.repo == nil || len(allSquadRows) == 0 {
		return nil
	}

	// 1. Liste de matchs uniques + métadonnées affichage (date + carte).
	type matchMeta struct {
		startTime int64
		mapUI     string
		dateLabel string
	}
	metas := make(map[string]matchMeta)
	matchOrder := make([]string, 0, len(allSquadRows))
	for _, m := range allSquadRows {
		if _, ok := metas[m.MatchID]; ok {
			continue
		}
		metas[m.MatchID] = matchMeta{
			startTime: m.StartTime.Unix(),
			mapUI:     m.MapUI,
			dateLabel: m.StartTime.Format("02/01"),
		}
		matchOrder = append(matchOrder, m.MatchID)
	}
	if len(matchOrder) < intensityMinMatches {
		return nil
	}
	// Tri chronologique (oldest → newest).
	sort.SliceStable(matchOrder, func(i, j int) bool {
		return metas[matchOrder[i]].startTime < metas[matchOrder[j]].startTime
	})

	// 2. Charger les kill events. LoadImpactEvents retourne kills + deaths +
	//    parfois autres types ; on filtre côté calcul.
	events, err := s.repo.LoadImpactEvents(ctx, matchOrder)
	if err != nil || len(events) == 0 {
		if err != nil {
			slog.WarnContext(ctx, "teammates_intensity_events_load_failed", "err", err.Error())
		}
		return nil
	}

	// 3. Pour chaque match, calculer la durée approximée = max(time_ms) sur les events.
	maxTimeByMatch := make(map[string]int64, len(matchOrder))
	for _, e := range events {
		if e.TimeMS > maxTimeByMatch[e.MatchID] {
			maxTimeByMatch[e.MatchID] = e.TimeMS
		}
	}

	// 4. Résoudre xuid pour le main + chaque teammate. squadLoader.LoadFor est
	// obligatoire : playerMatchesRepo est bound au main, donc tous les toggles
	// "par joueur" affichaient le même xuid (celui du main) → mêmes kill events.
	xuidByGT := make(map[string]string)
	resolveXUID := func(gt string) {
		if _, ok := xuidByGT[gt]; ok {
			return
		}
		if s.squadLoader == nil {
			return
		}
		rows, err := s.squadLoader.LoadFor(ctx, s.titleSlug, gt, port.PlayerMatchFilters{})
		if err != nil || len(rows) == 0 {
			return
		}
		xuidByGT[gt] = rows[0].Self.Identity.XUID
	}
	resolveXUID(mainGamertag)
	for _, gt := range selectedGamertags {
		resolveXUID(gt)
	}

	// 5. Construire les options du toggle.
	options := []domain.SquadIntensityOption{{Key: "all", Label: allTeamLabel}}
	for _, gt := range append([]string{mainGamertag}, selectedGamertags...) {
		options = append(options, domain.SquadIntensityOption{Key: gt, Label: gt})
	}

	// 6. Builder per-option : 1 row par match avec 10 phases normalisées.
	buildRows := func(filterXUID string) []domain.SquadIntensityMatchRow {
		out := make([]domain.SquadIntensityMatchRow, 0, len(matchOrder))
		for _, mid := range matchOrder {
			meta := metas[mid]
			label := meta.dateLabel
			if meta.mapUI != "" {
				label = meta.mapUI + " — " + meta.dateLabel
			}
			row := domain.SquadIntensityMatchRow{MatchID: mid, Label: label}
			duration := maxTimeByMatch[mid]
			if duration <= 0 {
				out = append(out, row)
				continue
			}
			var counts [intensityBuckets]int
			for _, e := range events {
				if e.MatchID != mid {
					continue
				}
				if e.EventType != "kill" {
					continue
				}
				if filterXUID != "" && e.XUID != filterXUID {
					continue
				}
				bucket := int((e.TimeMS * intensityBuckets) / duration)
				if bucket < 0 {
					bucket = 0
				}
				if bucket >= intensityBuckets {
					bucket = intensityBuckets - 1
				}
				counts[bucket]++
			}
			// Normalisation par max bucket du match.
			maxC := 0
			for _, c := range counts {
				if c > maxC {
					maxC = c
				}
			}
			if maxC > 0 {
				for i, c := range counts {
					row.Phases[i] = round2(float64(c) / float64(maxC))
				}
			}
			out = append(out, row)
		}
		return out
	}

	rowsByOpt := make(map[string][]domain.SquadIntensityMatchRow, len(options))
	hasAny := false
	for _, opt := range options {
		filterXUID := ""
		if opt.Key != "all" {
			filterXUID = xuidByGT[opt.Key]
			if filterXUID == "" {
				// Joueur dont on n'a pas pu résoudre le xuid → ligne vide pour transparence
				rowsByOpt[opt.Key] = buildRows("__missing__")
				continue
			}
		}
		rows := buildRows(filterXUID)
		rowsByOpt[opt.Key] = rows
		// Tester si au moins une cellule non nulle pour marquer hasAny.
		for _, r := range rows {
			for _, p := range r.Phases {
				if p > 0 {
					hasAny = true
					break
				}
			}
			if hasAny {
				break
			}
		}
	}
	if !hasAny {
		return nil
	}

	return &domain.SquadIntensityProfile{Options: options, Rows: rowsByOpt}
}

// ---------------------------------------------------------------------------
// teammates.14 — Stats par minute (frags / morts / assists)
// ---------------------------------------------------------------------------

// buildSquadPerMinuteStats agrège sur le scope filtré, par joueur :
//   - main : depuis allSquadRows dédupliqué par match_id (Kills/Deaths/Assists/TimePlayedSecs
//     représentent le main player car c'est sa pipeline Q30).
//   - teammates : LoadPlayerMatches + filtre sur match_ids escouade, lit r.Self.{Kills,Deaths,Assists,TimePlayed}.
//
// Retourne nil si aucun joueur n'a de matchs avec time_played > 0.
func (s *TeammatesService) buildSquadPerMinuteStats(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag string,
	selectedGamertags []string,
	sessionMatchIDs map[string]bool,
) []domain.SquadPerMinuteEntry {
	if len(allSquadRows) == 0 {
		return nil
	}

	// 1. Set des match_ids du scope (post-filtres + dédup).
	matchIDsAllowed := make(map[string]struct{}, len(allSquadRows))
	for _, m := range allSquadRows {
		matchIDsAllowed[m.MatchID] = struct{}{}
	}
	if len(sessionMatchIDs) > 0 {
		filtered := make(map[string]struct{}, len(sessionMatchIDs))
		for id := range matchIDsAllowed {
			if sessionMatchIDs[id] {
				filtered[id] = struct{}{}
			}
		}
		matchIDsAllowed = filtered
	}

	// 2. Main : agrégat depuis allSquadRows dédup.
	type sums struct {
		k, d, a, secs, n int
	}
	mainSeen := make(map[string]struct{}, len(allSquadRows))
	mainAgg := sums{}
	for _, m := range allSquadRows {
		if _, dup := mainSeen[m.MatchID]; dup {
			continue
		}
		mainSeen[m.MatchID] = struct{}{}
		if _, ok := matchIDsAllowed[m.MatchID]; !ok {
			continue
		}
		mainAgg.k += m.Kills
		mainAgg.d += m.Deaths
		mainAgg.a += m.Assists
		mainAgg.secs += m.TimePlayedSecs
		mainAgg.n++
	}

	makeEntry := func(name string, agg sums) domain.SquadPerMinuteEntry {
		entry := domain.SquadPerMinuteEntry{Player: name, MatchCount: agg.n}
		if agg.secs > 0 {
			minutes := float64(agg.secs) / 60.0
			entry.KillsPerMinute = round2(float64(agg.k) / minutes)
			entry.DeathsPerMinute = round2(float64(agg.d) / minutes)
			entry.AssistsPerMinute = round2(float64(agg.a) / minutes)
		}
		return entry
	}

	entries := []domain.SquadPerMinuteEntry{makeEntry(mainGamertag, mainAgg)}

	// 3. Teammates : squadLoader.LoadFor (résolution per-gamertag) → filter par
	// match_ids → sum r.Self. squadLoader est obligatoire ici : playerMatchesRepo
	// est bound au gamertag principal et ignore l'arg gt (cf. doc TeammatesService),
	// l'utiliser ici réagrégeait les stats du main pour chaque coéquipier → toutes
	// les barres avaient la même valeur.
	for _, gt := range selectedGamertags {
		if s.squadLoader == nil {
			entries = append(entries, domain.SquadPerMinuteEntry{Player: gt})
			continue
		}
		mateRows, err := s.squadLoader.LoadFor(
			ctx, s.titleSlug, gt, port.PlayerMatchFilters{},
		)
		if err != nil {
			slog.WarnContext(ctx, "teammates_per_minute_load_failed",
				"gamertag", gt, "err", err.Error())
			entries = append(entries, domain.SquadPerMinuteEntry{Player: gt})
			continue
		}
		mateAgg := sums{}
		for _, r := range mateRows {
			if _, ok := matchIDsAllowed[r.Summary.MatchID]; !ok {
				continue
			}
			mateAgg.k += intPtrOrZero(r.Self.Kills)
			mateAgg.d += intPtrOrZero(r.Self.Deaths)
			mateAgg.a += intPtrOrZero(r.Self.Assists)
			mateAgg.secs += intPtrOrZero(r.Self.TimePlayed)
			mateAgg.n++
		}
		entries = append(entries, makeEntry(gt, mateAgg))
	}

	// Si tous les joueurs ont 0 secondes, retourner nil (insufficient data).
	hasData := false
	for _, e := range entries {
		if e.KillsPerMinute > 0 || e.DeathsPerMinute > 0 || e.AssistsPerMinute > 0 {
			hasData = true
			break
		}
	}
	if !hasData {
		return nil
	}
	return entries
}

// loadImpactEventsByMatch charge les events highlight pour la liste de
// match_ids et renvoie une map matchID → []ImpactEvent (kills/deaths
// horodatés). Utilise repo.LoadImpactEvents (Q32). Retourne map vide si
// échec — le scoreboard ne sera alors construit qu'à partir des stats
// participants.
func (s *TeammatesService) loadImpactEventsByMatch(
	ctx context.Context,
	matchIDs []string,
) map[string][]analysis.ImpactEvent {
	out := make(map[string][]analysis.ImpactEvent, len(matchIDs))
	if s.repo == nil || len(matchIDs) == 0 {
		return out
	}
	rows, err := s.repo.LoadImpactEvents(ctx, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "teammates_impact_events_load_failed",
			"err", err.Error(), "n_matches", len(matchIDs))
		return out
	}
	for _, r := range rows {
		// EventType de ImpactEventRow est le BadgeKey original ou un type kill/death.
		// analysis.ComputeMatchImpactFull attend EventType == "kill" ou "death".
		// On dérive depuis le BadgeKey si présent ; sinon on laisse passer tel quel.
		ev := analysis.ImpactEvent{
			TimeMS:    r.TimeMS,
			EventType: r.EventType,
			ActorXUID: r.XUID,
		}
		out[r.MatchID] = append(out[r.MatchID], ev)
	}
	return out
}

// ---------------------------------------------------------------------------
// MedalDigest — résumé narratif médailles par joueur (SquadSynergiesPage)
// ---------------------------------------------------------------------------

// buildMedalDigest agrège les médailles de chaque joueur sur les matchs
// partagés et retourne un []domain.MedalDigestEntry trié par gamertag
// (main player en tête). Retourne nil si squadLoader ou medalDefs sont absents.
func (s *TeammatesService) buildMedalDigest(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	teammates []domain.TeammateRow,
	locale string,
) []domain.MedalDigestEntry {
	if s.squadLoader == nil || len(allSquadRows) == 0 || len(teammates) == 0 {
		return nil
	}
	sharedMatches := collectSharedMatchIDsForDigest(allSquadRows, len(teammates))
	if len(sharedMatches) == 0 {
		return nil
	}
	players := collectDigestPlayerXUIDs(mainGamertag, mainXUID, teammates)
	if len(players) == 0 {
		return nil
	}
	xuids := make([]string, len(players))
	for i, p := range players {
		xuids[i] = p.xuid
	}
	rows, err := s.squadLoader.LoadMedals(ctx, s.titleSlug, port.MedalsByXUIDFilters{
		MatchIDs: sharedMatches,
		XUIDs:    xuids,
	})
	if err != nil || len(rows) == 0 {
		if err != nil {
			slog.WarnContext(ctx, "teammates_medal_digest_load_failed", "err", err.Error())
		}
		return nil
	}
	gamertags := make([]string, len(players))
	for i, p := range players {
		gamertags[i] = p.gamertag
	}
	emblems := s.squadLoader.LoadEmblemURLs(ctx, s.titleSlug, gamertags)
	defs := resolveMedalDigestDefs(ctx, s.medalDefs, rows, locale)
	return assembleMedalDigest(rows, players, defs, emblems, s.titleSlug)
}

// collectSharedMatchIDsForDigest retourne les matchs présents pour au moins
// minTeammates coéquipiers dans allSquadRows.
func collectSharedMatchIDsForDigest(allSquadRows []domain.SquadMatchRow, minTeammates int) []string {
	occ := make(map[string]int, len(allSquadRows))
	for _, m := range allSquadRows {
		occ[m.MatchID]++
	}
	out := make([]string, 0, len(occ))
	for mid, n := range occ {
		if n >= minTeammates {
			out = append(out, mid)
		}
	}
	return out
}

type digestPlayer struct {
	gamertag string
	xuid     string
}

// collectDigestPlayerXUIDs construit la liste ordonnée (main en tête + teammates
// avec xuid résolu) pour le chargement des médailles.
func collectDigestPlayerXUIDs(mainGT, mainXUID string, teammates []domain.TeammateRow) []digestPlayer {
	players := make([]digestPlayer, 0, 1+len(teammates))
	if mainXUID != "" {
		players = append(players, digestPlayer{mainGT, mainXUID})
	}
	for _, tm := range teammates {
		if tm.XUID == nil || *tm.XUID == "" {
			continue
		}
		players = append(players, digestPlayer{tm.Gamertag, *tm.XUID})
	}
	return players
}

// resolveMedalDigestDefs charge les définitions (label + description) pour les
// medal_ids présents dans rows. Tolère un repo nil (retourne map vide).
func resolveMedalDigestDefs(
	ctx context.Context,
	repo port.MedalDefinitionsRepository,
	rows []port.MedalRow,
	locale string,
) map[int64]port.MedalDefinitionRow {
	if repo == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(rows))
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		if _, ok := seen[r.MedalID]; !ok {
			seen[r.MedalID] = struct{}{}
			ids = append(ids, r.MedalID)
		}
	}
	defs, err := repo.LookupByIDs(ctx, ids, locale)
	if err != nil {
		slog.WarnContext(ctx, "teammates_medal_digest_defs_failed", "err", err.Error())
		return nil
	}
	return defs
}

// assembleMedalDigest construit les entrées digest par joueur à partir des
// rows de médailles + définitions résolues + emblèmes.
func assembleMedalDigest(
	rows []port.MedalRow,
	players []digestPlayer,
	defs map[int64]port.MedalDefinitionRow,
	emblems map[string]string,
	titleSlug string,
) []domain.MedalDigestEntry {
	type agg struct{ totalCount, matchCount int }
	perXUID := make(map[string]map[int64]*agg, len(players))
	perXUIDMatch := make(map[string]map[string]int, len(players))
	for _, r := range rows {
		if _, ok := perXUID[r.XUID]; !ok {
			perXUID[r.XUID] = make(map[int64]*agg)
			perXUIDMatch[r.XUID] = make(map[string]int)
		}
		a := perXUID[r.XUID]
		if _, ok := a[r.MedalID]; !ok {
			a[r.MedalID] = &agg{}
		}
		a[r.MedalID].totalCount += r.Count
		a[r.MedalID].matchCount++
		perXUIDMatch[r.XUID][r.MatchID] += r.Count
	}

	out := make([]domain.MedalDigestEntry, 0, len(players))
	for _, p := range players {
		byMedal := perXUID[p.xuid]
		if len(byMedal) == 0 {
			continue
		}
		items := make([]domain.MedalDigestItem, 0, len(byMedal))
		total := 0
		for medalID, ma := range byMedal {
			def := defs[medalID]
			imageURL := ""
			if titleSlug != "" {
				imageURL = fmt.Sprintf("/static/medals/%s/%d.png", titleSlug, medalID)
			}
			items = append(items, domain.MedalDigestItem{
				MedalID:       medalID,
				Label:         def.Label,
				Description:   def.Description,
				ImageURL:      imageURL,
				TotalCount:    ma.totalCount,
				MatchCount:    ma.matchCount,
				Category:      def.MedalType,
				Difficulty:    def.Difficulty,
				PersonalScore: def.PersonalScore,
			})
			total += ma.totalCount
		}
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].TotalCount != items[j].TotalCount {
				return items[i].TotalCount > items[j].TotalCount
			}
			return items[i].MedalID < items[j].MedalID
		})
		peak := 0
		for _, n := range perXUIDMatch[p.xuid] {
			if n > peak {
				peak = n
			}
		}
		avg := 0.0
		if nm := len(perXUIDMatch[p.xuid]); nm > 0 {
			avg = float64(total) / float64(nm)
		}
		top := items
		if len(top) > 5 {
			top = items[:5]
		}
		emblemURL := ""
		if emblems != nil {
			emblemURL = emblems[p.gamertag]
		}
		out = append(out, domain.MedalDigestEntry{
			Player:        p.gamertag,
			EmblemURL:     emblemURL,
			DistinctTypes: len(byMedal),
			TotalCount:    total,
			AvgPerMatch:   avg,
			PeakInMatch:   peak,
			TopMedals:     top,
			AllMedals:     items,
		})
	}
	return out
}
