// Package timeline construit des domain.MatchTimeline depuis les rows DB.
//
// Phase 3 du refactor T0 (cf. .ai/PLAN_MATCH_TIMELINE_T0.md + docs/adr/0024) :
// les builders lisent désormais le vrai T0 (countdown pré-match) propagé depuis
// match_registry.real_start_time jusqu'au canonical (MatchSummary.T0Ms) et au
// MatchView (MatchMetaRaw.T0Ms). Quand le T0 est indisponible (real_start_time
// NULL → T0Ms nil), le fallback est T0=0 (chronologie brute identique au
// comportement pré-Phase 3). NewMatchTimeline ramène tout T0 négatif à 0.
//
// Pas de dépendance DB / HTTP — fonctions pures testables en isolation.
package timeline

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// BuildFromRegistry construit une MatchTimeline depuis une row match_registry.
//
// T0Ms = real_start_time − start_time (countdown pré-match), 0 si real_start_time
// absent. NewMatchTimeline borne les valeurs négatives à 0.
//
// Note : ce builder n'a pas de caller runtime (les services consomment des
// PlayerMatchRow / MatchMetaRaw) ; conservé comme helper symétrique testable.
func BuildFromRegistry(reg domain.MatchRegistryRow) domain.MatchTimeline {
	var durMs int64
	if reg.DurationSeconds != nil {
		durMs = int64(*reg.DurationSeconds) * 1000
	}
	var t0Ms int64
	if reg.RealStartTime != nil {
		t0Ms = reg.RealStartTime.Sub(reg.StartTime).Milliseconds()
	}
	// Horloge absolue fournie (StartUTC) → vrai début/fin/durée accessibles via
	// les méthodes GameplayStartUTC / GameplayEndUTC / GameplayDurationSeconds.
	return domain.NewMatchTimelineAt(reg.StartTime, durMs, t0Ms)
}

// BuildTimelinesFromPlayerMatches indexe une MatchTimeline par match_id depuis
// les rows canoniques chargées par un service (Timeseries, Squad, MatchView).
//
// Le T0 est lu depuis Summary.T0Ms (propagé par projectMatchSummary, Phase 3) ;
// nil → T0=0 (fallback chronologie brute).
func BuildTimelinesFromPlayerMatches(rows []canonical.PlayerMatchRow) map[string]domain.MatchTimeline {
	out := make(map[string]domain.MatchTimeline, len(rows))
	for _, r := range rows {
		var durMs int64
		if r.Summary.DurationSeconds != nil {
			durMs = int64(*r.Summary.DurationSeconds) * 1000
		}
		out[r.Summary.MatchID] = domain.NewMatchTimeline(durMs, derefInt64(r.Summary.T0Ms))
	}
	return out
}

// BuildForMatchMs construit la MatchTimeline d'un match unique depuis sa durée
// et son offset T0 (countdown pré-match) en millisecondes. Utilisé par
// MatchViewService qui charge un seul match (t0Ms vient de MatchMetaRaw.T0Ms,
// 0 si indisponible).
func BuildForMatchMs(durationMs, t0Ms int64) domain.MatchTimeline {
	return domain.NewMatchTimeline(durationMs, t0Ms)
}

// derefInt64 déréférence un *int64 en retournant 0 si nil.
func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// GameplayDurationsMS projette une map de MatchTimeline en map match_id → durée
// de gameplay en ms (countdown retranché). Sert de dénominateur canonique aux
// builders intensité / cadence (au lieu d'inférer la fin depuis les events).
// Les durées ≤ 0 sont omises (le builder retombe alors sur son proxy maxTime).
func GameplayDurationsMS(timelines map[string]domain.MatchTimeline) map[string]int64 {
	if len(timelines) == 0 {
		return nil
	}
	out := make(map[string]int64, len(timelines))
	for id, tl := range timelines {
		if d := tl.GameplayDurationMs(); d > 0 {
			out[id] = d
		}
	}
	return out
}
