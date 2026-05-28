// Package timeline construit des domain.MatchTimeline depuis les rows DB.
//
// En Phase 1 du refactor T0 (cf. .ai/PLAN_MATCH_TIMELINE_T0.md), BuildFromRegistry
// retourne toujours T0=0 — le comportement reste identique à celui d'avant
// l'introduction de la couche d'abstraction. La bascule vers le vrai T0 se
// fait en Phase 3 en modifiant ce seul fichier.
//
// Pas de dépendance DB / HTTP — fonctions pures testables en isolation.
package timeline

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// BuildFromRegistry construit une MatchTimeline depuis une row match_registry.
//
// Phase 1 : T0Ms = 0 (comportement de fallback identique au code pré-refacto).
// Phase 3 : T0Ms sera dérivé de reg.RealStartTime (repurposé) ou d'un champ
// dédié dans MatchRegistryRow.
func BuildFromRegistry(reg domain.MatchRegistryRow) domain.MatchTimeline {
	var durMs int64
	if reg.DurationSeconds != nil {
		durMs = int64(*reg.DurationSeconds) * 1000
	}
	return domain.NewMatchTimeline(durMs, phase1T0Ms())
}

// BuildTimelinesFromPlayerMatches indexe une MatchTimeline par match_id depuis
// les rows canoniques chargées par un service (Timeseries, Squad, MatchView).
//
// Phase 1 : T0=0 pour tous (via phase1T0Ms) → CorrectEvents est une identité.
// Phase 3 : la résolution du T0 lira la valeur stockée (real_start_time
// repurposé) — un seul point à modifier, cf. phase1T0Ms.
func BuildTimelinesFromPlayerMatches(rows []canonical.PlayerMatchRow) map[string]domain.MatchTimeline {
	out := make(map[string]domain.MatchTimeline, len(rows))
	for _, r := range rows {
		var durMs int64
		if r.Summary.DurationSeconds != nil {
			durMs = int64(*r.Summary.DurationSeconds) * 1000
		}
		out[r.Summary.MatchID] = domain.NewMatchTimeline(durMs, phase1T0Ms())
	}
	return out
}

// BuildForMatchMs construit la MatchTimeline d'un match unique depuis sa durée
// en millisecondes. Utilisé par MatchViewService qui charge un seul match.
// Phase 1 : T0=0 via phase1T0Ms.
func BuildForMatchMs(durationMs int64) domain.MatchTimeline {
	return domain.NewMatchTimeline(durationMs, phase1T0Ms())
}

// phase1T0Ms est le point de bascule du strangler fig. En Phase 1 il retourne
// toujours 0 (comportement historique préservé). En Phase 3, la résolution du
// T0 par match remplacera les appels à cette fonction par la lecture de la
// valeur stockée. Centralisé ici pour rendre la bascule explicite et greppable.
func phase1T0Ms() int64 { return 0 }
