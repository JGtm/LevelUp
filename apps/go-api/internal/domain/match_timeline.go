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

import "time"

// MatchTimeline porte les invariants temporels d'un match — source unique de
// vérité pour le "vrai" début, la "vraie" fin et la "vraie" durée de gameplay
// (countdown pré-match retranché).
//
// Modèle temporel (le countdown est au DÉBUT, la fin ne bouge pas) :
//
//	StartUTC                         GameplayEndUTC = StartUTC + DurationMs
//	|<--- T0Ms --->|<-- gameplay --->|
//	            GameplayStartUTC = StartUTC + T0Ms
//
//	DurationMs : durée totale du film (start_time → end_time), countdown inclus.
//	T0Ms       : durée du countdown pré-match (0 si inconnu ou non applicable).
//	StartUTC   : début du film (start_time_utc). Zero si non fourni — dans ce
//	             cas les helpers absolus (GameplayStartUTC/EndUTC) ne sont pas
//	             significatifs ; utiliser HasClock() pour discriminer.
//
// Construit via NewMatchTimeline (sans horloge absolue, ex. correction d'events
// relatifs) ou NewMatchTimelineAt (avec StartUTC, pour exposer début/fin réels).
// Les méthodes restent déterministes même pour les cas dégénérés ; les callers
// vérifient IsValid() / HasClock() s'ils ont besoin de discriminer.
type MatchTimeline struct {
	StartUTC   time.Time
	DurationMs int64
	T0Ms       int64
}

// NewMatchTimeline construit une MatchTimeline sans horloge absolue (StartUTC
// zero). Suffisant pour la correction d'events relatifs (CorrectEventTime) et
// les durées. Les valeurs négatives sont ramenées à 0.
func NewMatchTimeline(durationMs, t0Ms int64) MatchTimeline {
	return NewMatchTimelineAt(time.Time{}, durationMs, t0Ms)
}

// NewMatchTimelineAt construit une MatchTimeline avec son horloge absolue
// (StartUTC = début du film). Permet d'exposer le vrai début/fin/durée de
// gameplay. Les valeurs négatives sont ramenées à 0.
func NewMatchTimelineAt(startUTC time.Time, durationMs, t0Ms int64) MatchTimeline {
	if durationMs < 0 {
		durationMs = 0
	}
	if t0Ms < 0 {
		t0Ms = 0
	}
	return MatchTimeline{StartUTC: startUTC.UTC(), DurationMs: durationMs, T0Ms: t0Ms}
}

// HasClock indique si l'horloge absolue est renseignée (StartUTC non-zero) —
// condition pour que GameplayStartUTC / GameplayEndUTC soient significatifs.
func (t MatchTimeline) HasClock() bool {
	return !t.StartUTC.IsZero()
}

// GameplayStartUTC retourne le VRAI début du gameplay (countdown retranché) :
// StartUTC + T0Ms. Égal à StartUTC si T0 inconnu (0).
func (t MatchTimeline) GameplayStartUTC() time.Time {
	return t.StartUTC.Add(time.Duration(t.T0Ms) * time.Millisecond)
}

// GameplayEndUTC retourne la VRAIE fin du match (= fin du film, inchangée par
// le countdown) : StartUTC + DurationMs.
func (t MatchTimeline) GameplayEndUTC() time.Time {
	return t.StartUTC.Add(time.Duration(t.DurationMs) * time.Millisecond)
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
