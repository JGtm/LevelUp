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
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
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
			Class:       w.Class,
		})
	}

	events, canonicalEvents = applyKVSynthesisIfNeeded(events, canonicalEvents, kvPairs, matchID)

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

// applyKVSynthesisIfNeeded synthétise des events kill/death depuis les paires
// killer→victim quand les events highlight ne portent AUCUN kill/death.
//
// Title-agnostic : certains titres (Halo 5) ne portent PAS les kills dans
// highlight_events (= médailles seules) ; les kills horodatés vivent dans
// killer_victim_pairs. Quand events ne contient AUCUN kill/death mais que
// kvPairs est peuplé, on synthétise des events kill/death depuis les paires
// pour alimenter le kill-feed, les badges d'impact, la cadence et les rôles.
// Sur Infinite (kills dans highlight_events) ce chemin n'est jamais pris.
//
// Retourne (events, canonicalEvents) potentiellement modifiés : si la synthèse
// a lieu, canonicalEvents est forcé à nil (chemin EventRaw, medals-only canonique
// inutile ici). Sinon les entrées sont renvoyées inchangées.
func applyKVSynthesisIfNeeded(
	events []domain.EventRaw,
	canonicalEvents []canonical.HighlightEvent,
	kvPairs []domain.KVPairRaw,
	matchID string,
) ([]domain.EventRaw, []canonical.HighlightEvent) {
	if !eventsHaveKillOrDeath(events) {
		if synth := synthesizeEventRawFromKVPairs(kvPairs, matchID); len(synth) > 0 {
			events = mergeEventRawByTime(events, synth)
			canonicalEvents = nil // forcer le chemin EventRaw (medals-only canonique inutile ici)
		}
	}
	return events, canonicalEvents
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

// eventsHaveKillOrDeath indique si la liste d'events bruts contient au moins un
// kill ou death. Sert à décider du fallback synthétique kvPairs → events
// (titres dont highlight_events ne porte que des médailles, ex. Halo 5).
func eventsHaveKillOrDeath(events []domain.EventRaw) bool {
	for _, e := range events {
		if e.EventType == analysis.EventTypeKill || e.EventType == analysis.EventTypeDeath {
			return true
		}
	}
	return false
}

// synthesizeEventRawFromKVPairs reconstruit des EventRaw kill/death depuis les
// paires killer→victim. La règle de synthèse (1 paire → 1 kill + 1 death, acteur
// kill = tueur, acteur death = victime) est partagée avec les loaders engagement
// via analysis.SynthesizeKillEventsFromKVPairs (source unique). On reconvertit
// l'event canonique en EventRaw (XUID = acteur) pour les consommateurs legacy
// (evtList kill-feed, buildImpactInput, BuildMatchCadenceChart/Roles8).
func synthesizeEventRawFromKVPairs(kvPairs []domain.KVPairRaw, matchID string) []domain.EventRaw {
	if len(kvPairs) == 0 {
		return nil
	}
	inputs := make([]analysis.KVSyntheticInput, 0, len(kvPairs))
	for _, kv := range kvPairs {
		inputs = append(inputs, analysis.KVSyntheticInput{
			KillerXUID: kv.KillerXUID,
			VictimXUID: kv.VictimXUID,
			TimeMS:     kv.TimeMS,
			KillCount:  kv.KillCount,
		})
	}
	canon := analysis.SynthesizeKillEventsFromKVPairs(inputs, matchID)
	if len(canon) == 0 {
		return nil
	}
	out := make([]domain.EventRaw, 0, len(canon))
	for i := range canon {
		t := canon[i].TimeMS
		x := canon[i].XUID
		out = append(out, domain.EventRaw{
			EventType: canon[i].EventType,
			TimeMS:    &t,
			XUID:      &x,
		})
	}
	return out
}

// findViewerScoreboardRow retourne la ligne du viewer (is_me) du scoreboard, ou nil.
func findViewerScoreboardRow(rows []domain.MatchScoreboardRow) *domain.MatchScoreboardRow {
	for i := range rows {
		if rows[i].IsMe {
			return &rows[i]
		}
	}
	return nil
}

// buildViewerFragDistribution assemble la FragDistribution v2 du viewer (is_me) pour
// un match : classes gun depuis SES bulk weapon kills (registre, class/role résolus),
// melee/grenade/spartan + total depuis les compteurs natifs de SA ligne scoreboard.
// hasMechanics gate les Capacités spartanes + le niveau 2 de Mêlée (capability).
//
// RÉUTILISE le builder pur buildFragDistribution (P0) — aucune logique dupliquée
// (règle ≤2 copies). nil si le viewer est absent ou n'a aucun kill (le front rend null).
func buildViewerFragDistribution(
	me *domain.MatchScoreboardRow,
	bulkWeapons []domain.BulkWeaponKillRaw,
	hasMechanics bool,
) *domain.FragDistribution {
	if me == nil {
		return nil
	}
	total := derefInt(me.Kills)
	if total <= 0 {
		return nil
	}
	rows := make([]port.WeaponKillRow, 0, len(bulkWeapons))
	for _, w := range bulkWeapons {
		if w.XUID != me.XUID {
			continue
		}
		rows = append(rows, port.WeaponKillRow{
			Label: w.WeaponLabel, Kills: w.Kills, Class: w.Class, Role: w.Role,
		})
	}
	counts := domain.FragKillTypeCounts{
		Melee:         derefInt(me.MeleeKills),
		Grenade:       derefInt(me.GrenadeKills),
		Assassination: derefInt(me.AssassinationKills),
		GroundPound:   derefInt(me.GroundPoundKills),
		ShoulderBash:  derefInt(me.ShoulderBashKills),
		Total:         total,
	}
	fd := buildFragDistribution(rows, counts, hasMechanics)
	return &fd
}

// mergeEventRawByTime fusionne deux listes d'EventRaw et les trie par TimeMS
// croissant (nil TimeMS traité comme 0), stable. Aligne le kill-feed synthétique
// sur l'ordre chronologique attendu côté front.
func mergeEventRawByTime(a, b []domain.EventRaw) []domain.EventRaw {
	out := make([]domain.EventRaw, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	sort.SliceStable(out, func(i, j int) bool {
		var ti, tj int64
		if out[i].TimeMS != nil {
			ti = *out[i].TimeMS
		}
		if out[j].TimeMS != nil {
			tj = *out[j].TimeMS
		}
		return ti < tj
	})
	return out
}
