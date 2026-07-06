// Package service - teammates_squad_charts_impact_events.go : builders
// teammates.07 (squad impact matrix) + teammates first events. Decoupe de
// teammates_squad_charts.go (god-file split, refactor 2026-05-27).
package teammates

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
)

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

	// 2. Charger les events impact pour tous les matchs, ramenés au référentiel
	//    gameplay (T0 / countdown pré-match retranché, §4.A-bis). Les gagnants
	//    de badges sont invariants par T0, mais on corrige pour la cohérence des
	//    TimeMS event-based et la robustesse future.
	eventsByMatch := s.loadImpactEventsByMatch(
		ctx, matchIDOrder, timeline.BuildTimelinesFromSquadRows(allSquadRows))

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
				"main_xuid", mainXUID, "err", err)
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
//
//nolint:funlen // chart-builder cohésif (load events → first kill/death → 15s bins).
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

	// 3. Charger les events, puis les ramener au référentiel gameplay (T0 /
	//    countdown pré-match retranché, §4.A-bis). CRITIQUE ici : ce chart
	//    dépend des valeurs absolues (first_kill_s / first_death_s, bins 15s).
	//    T0 lu depuis allSquadRows (Q30.t0_ms) ; match sans T0 connu → identité.
	events, err := s.repo.LoadImpactEvents(ctx, matchIDs)
	if err != nil || len(events) == 0 {
		if err != nil {
			slog.WarnContext(ctx, "teammates_first_events_load_failed", "err", err)
		}
		return nil
	}
	events = CorrectSquadImpactEvents(ctx, "teammates.17", events, timeline.BuildTimelinesFromSquadRows(allSquadRows))

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
		if e.TimeMS < 0 {
			// Event pré-gameplay (countdown) après correction T0 — ignoré.
			// Évite la collision avec le sentinel -1 de firstKillS/firstDeathS.
			continue
		}
		k := keyOf(e.MatchID, e.XUID)
		ft := firsts[k]
		if ft == nil {
			ft = &firstTimes{firstKillS: -1, firstDeathS: -1}
			firsts[k] = ft
		}
		secs := e.TimeMS / 1000
		switch e.EventType {
		case analysis.EventTypeKill:
			if ft.firstKillS == -1 || secs < ft.firstKillS {
				ft.firstKillS = secs
			}
		case analysis.EventTypeDeath:
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
// Match set : union des matchs où au moins un coéquipier sélectionné a joué
// avec le main (cf. spec Python `load_weapon_kills_data` qui passe les
// match_ids par joueur). Le SQL filtre par (xuid, match_id) — un joueur
// absent d'un match n'apparaît tout simplement pas dans les rows agrégés
// pour ce match. L'intersection stricte excluait tout coéquipier qui n'a
// pas joué exactement les mêmes matchs que les autres → chart absent.
//
// Renvoie nil si :
//   - squadLoader == nil (DI non câblée)
//   - aucun match commun avec au moins un coéquipier
//   - aucun joueur avec xuid résolu
//   - le repo ne renvoie aucune donnée (capability absente ou tables vides)
//
//nolint:funlen // chart-builder cohésif (load weapons → resolve xuid → kill counts).
