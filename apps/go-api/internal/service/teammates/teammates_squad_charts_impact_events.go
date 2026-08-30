// Package service - teammates_squad_charts_impact_events.go : builders
// teammates.07 (squad impact matrix) + teammates first events. Decoupe de
// teammates_squad_charts.go (god-file split, refactor 2026-05-27).
package teammates

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
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
	//    match le plus ancien (Q30SquadMatchesSharedQuery retourne DESC, et allSquadRows
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
// Premier frag / première mort — séries par joueur (chart lanes, onglet Dynamique)
// ---------------------------------------------------------------------------

// buildSquadFirstBlood charge les events highlight de l'escouade, les ramène au
// référentiel gameplay (T0) et produit UNE série par joueur avec les valeurs PAR
// MATCH (first_kill_sec / first_death_sec, nil si l'événement est absent).
//
// Aucun bucketing serveur : médianes, écart et fenêtre d'axe sont dérivés côté
// front (FirstBloodLanes). L'agrégation « premier événement » est celle de
// narrative.ComputeFirstEventsByActor — noyau partagé avec les surfaces solo
// (Timeseries / Session).
//
// Retourne nil si aucun event, aucun joueur résolu, ou aucune série exploitable.
func (s *TeammatesService) buildSquadFirstBlood(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	teammates []domain.TeammateRow,
) []domain.FirstBloodPlayerSeries {
	if s.repo == nil {
		return nil
	}
	// 1. Périmètre : matchs (chronologiques) + joueurs de l'escouade.
	matchIDs, xuidsOrdered, gtByXUID := firstBloodScope(allSquadRows, mainGamertag, mainXUID, teammates)
	if len(matchIDs) == 0 || len(xuidsOrdered) == 0 {
		return nil
	}
	// Métadonnées d'affichage (carte/mode/date) par match, pour le tooltip —
	// DEC-4 (retours utilisateur 2026-08-29) : plus jamais l'uuid du match.
	metaByMatch := squadFirstBloodMeta(allSquadRows)

	// 2. Charger les events, puis les ramener au référentiel gameplay (T0 /
	//    countdown pré-match retranché, §4.A-bis). CRITIQUE ici : le chart lit des
	//    valeurs absolues (secondes depuis le début du gameplay). T0 lu depuis
	//    allSquadRows (Q30.t0_ms) ; match sans T0 connu → identité.
	events, err := s.repo.LoadImpactEvents(ctx, matchIDs)
	if err != nil || len(events) == 0 {
		if err != nil {
			slog.WarnContext(ctx, "teammates_first_blood_load_failed", "err", err)
		}
		return nil
	}
	events = CorrectSquadImpactEvents(ctx, "squad.first_blood", events, timeline.BuildTimelinesFromSquadRows(allSquadRows))

	// 3. Agrégation partagée (min des TimeMS >= 0 par (joueur, match, type)).
	//    Les events pré-gameplay (TimeMS < 0 après correction T0) sont écartés
	//    par ComputeFirstEventsByActor : le « premier frag » est celui du
	//    GAMEPLAY, pas du countdown.
	actors := make([]narrative.FirstEventActor, 0, len(events))
	for _, e := range events {
		switch e.EventType {
		case analysis.EventTypeKill:
			actors = append(actors, narrative.FirstEventActor{
				MatchID: e.MatchID, XUID: e.XUID, IsKill: true, TimeMS: e.TimeMS,
			})
		case analysis.EventTypeDeath:
			actors = append(actors, narrative.FirstEventActor{
				MatchID: e.MatchID, XUID: e.XUID, IsKill: false, TimeMS: e.TimeMS,
			})
		}
	}
	byXUID := narrative.ComputeFirstEventsByActor(actors, xuidsOrdered, matchIDs)

	// 4. Projection en séries produit — un joueur sans aucun événement exploitable
	//    n'a pas de bande (pas de lane vide dans le chart).
	out := make([]domain.FirstBloodPlayerSeries, 0, len(xuidsOrdered))
	for _, xuid := range xuidsOrdered {
		points := make([]domain.FirstBloodMatchPoint, 0, len(matchIDs))
		for _, r := range byXUID[xuid] {
			points = append(points, domain.NewFirstBloodPoint(
				r.MatchID, r.FirstKillMS, r.FirstDeathMS, metaByMatch[r.MatchID]))
		}
		series := domain.FirstBloodPlayerSeries{Player: gtByXUID[xuid], Matches: points}
		if series.HasEvents() {
			out = append(out, series)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// squadFirstBloodMeta indexe carte/mode/date par match_id pour le tooltip du
// chart « premier frag / première mort » (DEC-4). MapUI est déjà résolu sur
// SquadMatchRow (Q30, enrichSquadMatchAssets tourne avant l'appel — cf.
// teammates_service.go) ; ModeUI réutilise squadModeUI, le résolveur canonique
// déjà partagé avec SquadMatchHistoryRow (teammates_service_assets.go) — ne
// pas dupliquer sa logique pair-sinon-variant. allSquadRows porte plusieurs
// lignes par match (une par coéquipier) : première occurrence retenue, comme
// firstBloodScope ci-dessous (les métadonnées de match sont invariantes par
// coéquipier).
func squadFirstBloodMeta(rows []domain.SquadMatchRow) map[string]domain.FirstBloodMatchMeta {
	meta := make(map[string]domain.FirstBloodMatchMeta, len(rows))
	for _, m := range rows {
		if _, ok := meta[m.MatchID]; ok {
			continue
		}
		meta[m.MatchID] = domain.FirstBloodMatchMeta{
			MapUI:     m.MapUI,
			ModeUI:    squadModeUI(m),
			StartTime: m.StartTime,
		}
	}
	return meta
}

// firstBloodScope dérive le périmètre du chart « premier frag / première mort » :
//   - matchIDs : identifiants uniques triés par start_time ASC (l'ordre ne change
//     pas les médianes mais rend le payload lisible et stable) ;
//   - xuidsOrdered : main puis coéquipiers, ordre canonique de la page ;
//   - gtByXUID : résolution xuid → gamertag pour l'étiquetage des bandes.
func firstBloodScope(
	allSquadRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	teammates []domain.TeammateRow,
) (matchIDs []string, xuidsOrdered []string, gtByXUID map[string]string) {
	startByMatch := make(map[string]time.Time, len(allSquadRows))
	matchIDs = make([]string, 0, len(allSquadRows))
	for _, m := range allSquadRows {
		if _, ok := startByMatch[m.MatchID]; ok {
			continue
		}
		startByMatch[m.MatchID] = m.StartTime
		matchIDs = append(matchIDs, m.MatchID)
	}
	slices.SortStableFunc(matchIDs, func(a, b string) int {
		return startByMatch[a].Compare(startByMatch[b])
	})

	gtByXUID = make(map[string]string, 1+len(teammates))
	xuidsOrdered = make([]string, 0, 1+len(teammates))
	if mainXUID != "" {
		xuidsOrdered = append(xuidsOrdered, mainXUID)
		gtByXUID[mainXUID] = mainGamertag
	}
	for _, tm := range teammates {
		if tm.XUID == nil || *tm.XUID == "" {
			continue
		}
		if _, dup := gtByXUID[*tm.XUID]; dup {
			continue
		}
		xuidsOrdered = append(xuidsOrdered, *tm.XUID)
		gtByXUID[*tm.XUID] = tm.Gamertag
	}
	return matchIDs, xuidsOrdered, gtByXUID
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
