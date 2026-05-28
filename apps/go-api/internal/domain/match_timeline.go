// Package domain — match_timeline.go : abstraction de la chronologie d'un match.
//
// MatchTimeline encapsule la durée totale du match (depuis le début du film
// incluant le countdown pré-match) et l'offset T0 (durée du countdown).
//
// Les événements `highlight_events.time_ms` sont stockés relatifs au début du
// film. CorrectEventTime() les ramène au début du gameplay (T0 = 0 dans le
// référentiel gameplay).
//
// Cf. .ai/PLAN_MATCH_TIMELINE_T0.md §4 et docs/adr/0024-match-timeline-t0.md.
package domain

// MatchTimeline porte les invariants temporels d'un match.
//
// DurationMs : durée totale (start_time → end_time), incluant le countdown.
// T0Ms       : durée du countdown pré-match (0 si inconnu ou non applicable).
//
// Construit via NewMatchTimeline. Les méthodes restent déterministes même
// pour les cas dégénérés ; les callers vérifient IsValid() s'ils ont besoin
// de discriminer.
type MatchTimeline struct {
	DurationMs int64
	T0Ms       int64
}

// NewMatchTimeline construit une MatchTimeline. Les valeurs négatives sont
// ramenées à 0 ; aucune erreur n'est levée pour rester utilisable comme
// fallback à chaque callsite.
func NewMatchTimeline(durationMs, t0Ms int64) MatchTimeline {
	if durationMs < 0 {
		durationMs = 0
	}
	if t0Ms < 0 {
		t0Ms = 0
	}
	return MatchTimeline{DurationMs: durationMs, T0Ms: t0Ms}
}

// IsValid vrai si T0 et Duration sont cohérents (T0 < Duration et Duration > 0).
func (t MatchTimeline) IsValid() bool {
	return t.DurationMs > 0 && t.T0Ms <= t.DurationMs
}

// GameplayDurationMs retourne la durée jouable (DurationMs − T0Ms), bornée à 0.
func (t MatchTimeline) GameplayDurationMs() int64 {
	gd := t.DurationMs - t.T0Ms
	if gd < 0 {
		return 0
	}
	return gd
}

// GameplayDurationSeconds retourne GameplayDurationMs converti en secondes (tronqué).
func (t MatchTimeline) GameplayDurationSeconds() int64 {
	return t.GameplayDurationMs() / 1000
}

// CorrectEventTime convertit un timestamp brut (relatif au début du film) en
// timestamp relatif au début du gameplay. Peut être négatif si l'événement
// précède T0 (rare, ex: événements de setup) — au caller de filtrer.
func (t MatchTimeline) CorrectEventTime(rawMs int64) int64 {
	return rawMs - t.T0Ms
}

// RawTimeFromCorrected fait l'opération inverse (utilisé pour reconstituer un
// timestamp film depuis un timestamp gameplay, ex: alignement avec un film).
func (t MatchTimeline) RawTimeFromCorrected(correctedMs int64) int64 {
	return correctedMs + t.T0Ms
}
