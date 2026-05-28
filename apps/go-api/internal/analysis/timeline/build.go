// Package timeline construit des domain.MatchTimeline depuis les rows DB.
//
// En Phase 1 du refactor T0 (cf. .ai/PLAN_MATCH_TIMELINE_T0.md), BuildFromRegistry
// retourne toujours T0=0 — le comportement reste identique à celui d'avant
// l'introduction de la couche d'abstraction. La bascule vers le vrai T0 se
// fait en Phase 3 en modifiant ce seul fichier.
//
// Pas de dépendance DB / HTTP — fonctions pures testables en isolation.
package timeline

import "levelup/go-api/internal/domain"

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
	return domain.NewMatchTimeline(durMs, 0)
}
