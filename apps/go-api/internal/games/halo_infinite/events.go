package halo_infinite

// events.go — reconstruction de la timeline canonique d'événements (MatchEvent)
// pour Halo Infinite, à partir des highlight_events filmés bruts.
//
// Contrairement à Halo 5 (timeline native servie par l'API, cf. halo_5/events.go),
// Infinite ne dispose PAS d'une timeline d'events propre côté API : on la
// reconstitue depuis `shared.highlight_events` (parser film → kill/death/medal/mode)
// au référentiel T0. C'est la Phase 2 du plan `.ai/PLAN_CANONICAL_MATCH_EVENTS.md`.
//
// Dégradations assumées (vs Halo 5 natif), reportées via infiniteEventLimitations
// et les capabilities match.killfeed.per_kill (degraded) / match.events.spatial
// (not_exposed) :
//   - arme-par-kill et mécanique (headshot/melee) : ABSENTES de highlight_events
//     (RE film non câblé côté serveur) → Kind/Headshot zéro-value, Weapon nil ;
//   - positions monde : non extraites par le parser → KillerLoc/VictimLoc nil ;
//   - identifiants médaille/impulsion : non portés par highlight_events (vivent
//     dans medals_earned, non aligné temporellement) → RefID nil.

import (
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// killPairToleranceMs — fenêtre d'appariement kill↔death (ms). Aligné sur la
// reconstruction killer_victim_pairs du reste de l'app (analysis, tolérance 5 ms).
const killPairToleranceMs = 5

// mapInfiniteEvents reconstitue la timeline canonique d'un match Infinite depuis
// les highlight_events bruts (kill/death/medal/mode), ramenée au référentiel T0.
//
// Reconstruction :
//   - kill + death (2 rows, 1 xuid chacune) → 1 MatchEventKill par appariement
//     temporel (analysis.ComputeKillerVictimPairs, tolérance killPairToleranceMs).
//     Les events simultanés non appariés (double-kill) sont OMIS — limitation
//     reportée par infiniteEventLimitations.
//   - medal → MatchEventMedal ; mode → MatchEventImpulse (Player = xuid centrant
//     l'event ; RefID nil, cf. en-tête de fichier).
//
// T0 : chaque TimeMS brut (relatif au film) est ramené au gameplay via
// tl.CorrectEventTime ; les events de countdown (corrigé < 0) sont skippés
// (même convention que les builders narrative, cf. timeline.CorrectEvents).
// Le résultat est trié par TimeMs croissant (stable).
func mapInfiniteEvents(
	raw []canonical.HighlightEvent,
	tl domain.MatchTimeline,
	opts canonical.MatchEventOptions,
) []canonical.MatchEvent {
	out := make([]canonical.MatchEvent, 0, len(raw))

	// 1. Kills : appariement temporel killer↔victim sur les rows kill/death.
	if opts.Wants(canonical.MatchEventKill) {
		for _, p := range analysis.ComputeKillerVictimPairs(toRawEvents(raw), killPairToleranceMs) {
			corrected := tl.CorrectEventTime(p.TimeMS)
			if corrected < 0 {
				continue
			}
			out = append(out, canonical.MatchEvent{
				Type:   canonical.MatchEventKill,
				TimeMs: int(corrected),
				// Identité XUID-seule : le gamertag est résolu au CHOKEPOINT canonique
				// (jamais gamertag||xuid ici, cf. events.go canonical + règle projet).
				Killer: &canonical.PlayerIdentity{XUID: p.KillerXUID},
				Victim: &canonical.PlayerIdentity{XUID: p.VictimXUID},
				// Kind/Headshot/Weapon/Loc volontairement zéro-value : dégradation Infinite.
			})
		}
	}

	// 2. Events 1:1 centrés joueur (medal / mode → impulse). kill/death sont déjà
	// consommés par l'appariement ; type inconnu → skip.
	for _, e := range raw {
		ct, ok := infiniteHighlightTypeToCanonical(e.EventType)
		if !ok || !opts.Wants(ct) {
			continue
		}
		corrected := tl.CorrectEventTime(e.TimeMS)
		if corrected < 0 {
			continue
		}
		ev := canonical.MatchEvent{Type: ct, TimeMs: int(corrected)}
		if e.XUID != "" {
			ev.Player = &canonical.PlayerIdentity{XUID: e.XUID}
		}
		out = append(out, ev)
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].TimeMs < out[j].TimeMs })
	return out
}

// toRawEvents projette les highlight_events canoniques en analysis.RawEvent pour
// l'appariement killer↔victim. Seuls kill/death sont pertinents ; les autres
// types sont ignorés par ComputeKillerVictimPairs. Le gamertag n'est pas porté
// (résolu au chokepoint) : RawEvent.Gamertag reste vide → fallback XUID interne
// inoffensif (on ne lit que les XUID des paires retournées).
func toRawEvents(raw []canonical.HighlightEvent) []analysis.RawEvent {
	out := make([]analysis.RawEvent, 0, len(raw))
	for _, e := range raw {
		out = append(out, analysis.RawEvent{
			EventType: e.EventType,
			XUID:      e.XUID,
			TimeMS:    e.TimeMS,
		})
	}
	return out
}

// infiniteHighlightTypeToCanonical mappe un event_type highlight_events (base
// Infinite réellement stockée : kill/death/medal/mode, cf. analysis.inferEventType)
// vers son MatchEventType canonique. kill/death sont traités à part (appariement)
// → ok=false ici ; type inconnu → ok=false (skip).
func infiniteHighlightTypeToCanonical(eventType string) (canonical.MatchEventType, bool) {
	switch eventType {
	case analysis.EventTypeMedal:
		return canonical.MatchEventMedal, true
	case analysis.EventTypeMode:
		return canonical.MatchEventImpulse, true
	}
	return "", false
}

// infiniteEventLimitations décrit les dégradations connues de la timeline Infinite
// reconstruite (vs la timeline native Halo 5). Reporté dans MatchEventTimeline.Limitations
// pour que le consommateur (kill-feed/timeline front) sache cadrer l'affichage.
func infiniteEventLimitations() []canonical.CapabilityGap {
	return []canonical.CapabilityGap{
		{
			CapabilityKey: string(games.CapMatchKillfeedPerKill),
			ReasonCode:    "weapon_per_kill_unavailable",
			Severity:      "info",
			Message:       "arme-par-kill et mécanique (headshot/melee) non disponibles : reconstruites depuis highlight_events (RE film non câblé)",
		},
		{
			CapabilityKey: string(games.CapMatchEventsSpatial),
			ReasonCode:    "world_positions_not_extracted",
			Severity:      "info",
			Message:       "positions monde non extraites par le parser film Infinite",
		},
		{
			CapabilityKey: string(games.CapMatchEventsTimeline),
			ReasonCode:    "kills_temporally_paired",
			Severity:      "info",
			Message:       "kills appariés temporellement (5 ms) : doubles simultanés possiblement omis ; médailles/impulsions sans identifiant",
		},
	}
}
