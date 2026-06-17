// Package service — builders pour l'onglet Combat de la Match View.
//
// Extrait de match_view_service.go (audit #1 god files). Couvre :
//   - buildCombatTabFull : assemblage de l'onglet (weapon kills, events, tug,
//     impact badges, KD timeline, killer/victim pairs, cadence, roles).
//   - buildKillerVictimPairs : agrégation kvPairs -> chart antagonistes.
//   - buildTugEvents / buildImpactInput / buildKDEvents : adapters vers la
//     couche analysis/.
package service

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// ---------------------------------------------------------------------------
// Combat Tab
// ---------------------------------------------------------------------------

func buildCombatTabFull(
	matchID string,
	bulkWeapons []domain.BulkWeaponKillRaw,
	events []domain.EventRaw,
	canonicalEvents []canonical.HighlightEvent,
	kvPairs []domain.KVPairRaw,
	scoreboard []domain.ScoreboardRaw,
	myXUID string,
	durationMS int64,
) domain.MatchCombatTab {
	wkList := make([]domain.MatchWeaponKill, 0)
	for _, w := range bulkWeapons {
		if w.XUID != myXUID {
			continue
		}
		wkList = append(wkList, domain.MatchWeaponKill{
			WeaponID:    w.WeaponID,
			WeaponLabel: w.WeaponLabel,
			KillCount:   w.Kills,
		})
	}

	evtList := make([]domain.MatchHighlightEvent, 0, len(events))
	for _, e := range events {
		// Skip des events pré-T0 (countdown) : après la correction T0 centrale,
		// event_time_ms peut être négatif. event_time_ms est l'axe X des charts
		// KD-cumul / frag-diff / scatter tug — un négatif y rendrait "-1m00s" et
		// un point hors-axe. Aligné sur le précédent skip<0 (first_events).
		if e.TimeMS != nil && *e.TimeMS < 0 {
			continue
		}
		evtList = append(evtList, domain.MatchHighlightEvent{
			EventType:     e.EventType,
			EventTimeMS:   e.TimeMS,
			ActorXUID:     e.XUID,
			ActorGamertag: e.Gamertag,
		})
	}

	// Tug-of-war
	tugEvents := buildTugEvents(kvPairs, myXUID)
	tugBins := analysis.ComputeTugOfWar(tugEvents, durationMS, 0)
	tugDomain := convertTugBinsToDomain(tugBins)

	// Impact badges : calculés en périmètre team-wide alliée (parité Python
	// _match_impact_events::compute_single_match_impact). Le filtre par
	// team_id du main est appliqué dans buildImpactInput → seuls les
	// participants alliés sont passés à l'analyse, mais les events restent
	// full (first_blood reste global toutes équipes).
	impactInput := buildImpactInput(events, scoreboard, myXUID)
	allBadges := analysis.ComputeMatchImpactFull(impactInput)
	badgesDomain := convertImpactBadgesToDomain(allBadges)

	// KD timeline
	kdEvents := buildKDEvents(kvPairs, myXUID)
	kdPoints := analysis.ComputeKDTimeline(kdEvents, myXUID)
	kdDomain := convertKDPointsToDomain(kdPoints)

	// Phase 1 méta-plan § 6.1.3 — pilote MatchView aligné fondations narrative.
	// Cadence intra-match + 8 rôles narratifs en parallèle des badges legacy.
	//
	// MV4.A : si canonicalEvents est peuplé (loader unifié actif), on l'utilise
	// directement. Sinon fallback sur la conversion à la volée depuis EventRaw.
	var cadence *domain.ChartSeries[domain.ChartPointStacked]
	var impactRoles []domain.MatchViewImpactRole
	if len(canonicalEvents) > 0 {
		cadence = BuildMatchCadenceChartFromCanonical(canonicalEvents, scoreboard, durationMS)
		impactRoles = BuildMatchImpactRoles8FromCanonical(canonicalEvents, scoreboard)
	} else {
		cadence = BuildMatchCadenceChart(events, scoreboard, matchID, durationMS)
		impactRoles = BuildMatchImpactRoles8(events, scoreboard, matchID)
	}

	// Killer-victim aggregation (chart match_view.18 — antagonistes).
	killerVictim := buildKillerVictimPairs(kvPairs, scoreboard)

	return domain.MatchCombatTab{
		WeaponKills:     wkList,
		HighlightEvents: evtList,
		TugOfWar:        tugDomain,
		ImpactBadges:    badgesDomain,
		KDTimeline:      kdDomain,
		NemesisDuels:    []domain.MatchNemesisRow{},
		KillerVictim:    killerVictim,
		ImpactRoles:     impactRoles,
		Cadence:         cadence,
	}
}

// buildKillerVictimPairs agrège les kvPairs par (killer_xuid, victim_xuid).
// Résout les gamertags via le scoreboard quand kv.{Killer,Victim}GT est vide.
func buildKillerVictimPairs(
	kvPairs []domain.KVPairRaw,
	scoreboard []domain.ScoreboardRaw,
) []domain.MatchKillerVictimPair {
	if len(kvPairs) == 0 {
		return nil
	}
	gtMap := make(map[string]string, len(scoreboard))
	for _, s := range scoreboard {
		if s.Gamertag != "" {
			gtMap[s.XUID] = s.Gamertag
		}
	}
	resolveGT := func(xuid, fallback string) string {
		if gt, ok := gtMap[xuid]; ok && gt != "" {
			return gt
		}
		if fallback != "" {
			return fallback
		}
		return xuid
	}

	type pairKey struct {
		killer, victim string
	}
	agg := make(map[pairKey]*domain.MatchKillerVictimPair)
	order := make([]pairKey, 0)

	for _, kv := range kvPairs {
		if kv.KillerXUID == "" || kv.VictimXUID == "" {
			continue
		}
		k := pairKey{killer: kv.KillerXUID, victim: kv.VictimXUID}
		count := kv.KillCount
		if count <= 0 {
			count = 1
		}
		if existing, ok := agg[k]; ok {
			existing.KillCount += count
			continue
		}
		agg[k] = &domain.MatchKillerVictimPair{
			KillerXUID:     kv.KillerXUID,
			KillerGamertag: resolveGT(kv.KillerXUID, kv.KillerGT),
			VictimXUID:     kv.VictimXUID,
			VictimGamertag: resolveGT(kv.VictimXUID, kv.VictimGT),
			KillCount:      count,
		}
		order = append(order, k)
	}

	pairs := make([]domain.MatchKillerVictimPair, 0, len(order))
	for _, k := range order {
		pairs = append(pairs, *agg[k])
	}
	return pairs
}

func buildTugEvents(kvPairs []domain.KVPairRaw, myXUID string) []analysis.TugOfWarEvent {
	events := make([]analysis.TugOfWarEvent, 0, len(kvPairs))
	for _, kv := range kvPairs {
		isAlly := kv.KillerXUID == myXUID
		events = append(events, analysis.TugOfWarEvent{
			TimeMS:    kv.TimeMS,
			IsAlly:    isAlly,
			EventType: analysis.EventTypeKill,
		})
	}
	return events
}

// buildImpactInput convertit les données brutes du match vers MatchImpactInput.
// Les events highlight_events (kill/death + horodatage + acteur) alimentent
// les badges event-based ; le scoreboard fournit les stats par joueur pour
// les badges stat-based (top_killer, silent_hero, false_brother).
//
// Périmètre team-wide alliée : Participants ne contient QUE les joueurs de la
// même team_id que myXUID (le main). Les events restent full (first_blood
// nécessite tous les kills toutes équipes confondues). Si myXUID n'est pas
// trouvé dans le scoreboard ou n'a pas de team_id, on dégrade en passant tous
// les participants pour ne pas casser silencieusement le calcul.
func buildImpactInput(events []domain.EventRaw, scoreboard []domain.ScoreboardRaw, myXUID string) analysis.MatchImpactInput {
	impactEvents := make([]analysis.ImpactEvent, 0, len(events))
	for _, ev := range events {
		if ev.TimeMS == nil || ev.XUID == nil {
			continue
		}
		// Skip des events pré-T0 (countdown) après correction T0 centrale :
		// aligne délibérément first_blood / top_gun / last_casualty sur le
		// gameplay (un frag de countdown ne doit pas gagner first_blood —
		// parité first_events) et évite un badge exposé à time_ms négatif.
		if *ev.TimeMS < 0 {
			continue
		}
		et := ev.EventType
		if et != analysis.EventTypeKill && et != analysis.EventTypeDeath {
			continue
		}
		impactEvents = append(impactEvents, analysis.ImpactEvent{
			TimeMS:    *ev.TimeMS,
			EventType: et,
			ActorXUID: *ev.XUID,
		})
	}

	var mainTeamID *int
	for _, p := range scoreboard {
		if p.XUID == myXUID && p.TeamID != nil {
			mainTeamID = p.TeamID
			break
		}
	}

	snaps := make([]analysis.ParticipantSnap, 0, len(scoreboard))
	for _, p := range scoreboard {
		if mainTeamID != nil && (p.TeamID == nil || *p.TeamID != *mainTeamID) {
			continue
		}
		snaps = append(snaps, analysis.ParticipantSnap{
			XUID:    p.XUID,
			Outcome: p.OutcomeCode,
			Kills:   p.Kills,
			Deaths:  p.Deaths,
			Assists: p.Assists,
		})
	}
	return analysis.MatchImpactInput{
		Events:       impactEvents,
		Participants: snaps,
	}
}

func buildKDEvents(kvPairs []domain.KVPairRaw, myXUID string) []analysis.KDEvent {
	events := make([]analysis.KDEvent, 0, len(kvPairs)*2)
	for _, kv := range kvPairs {
		if kv.KillerXUID == myXUID {
			events = append(events, analysis.KDEvent{
				TimeMS:    kv.TimeMS,
				IsKill:    true,
				ActorXUID: myXUID,
			})
		}
		if kv.VictimXUID == myXUID {
			events = append(events, analysis.KDEvent{
				TimeMS:    kv.TimeMS,
				IsKill:    false,
				ActorXUID: myXUID,
			})
		}
	}
	return events
}
