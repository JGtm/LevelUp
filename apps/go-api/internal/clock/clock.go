// Package clock — interface horloge injectable pour testabilité.
//
// Centralise l'accès à l'heure courante de façon à pouvoir mocker `time.Now()`
// dans les tests qui dépendent du temps. ADR : préférer `clock.Clock` aux
// appels directs `time.Now()` dans `internal/analysis/` et `internal/service/`
// quand le résultat dépend de la date courante.
//
// Sites identifiés en revue 2026-04-29 (P3.7 / test manquant IV) :
//   - analysis/sessions.go::IsSessionPotentiallyActive
//
// Usage :
//
//	type SessionService struct {
//	    Clock clock.Clock
//	}
//
//	func (s *SessionService) Active(...) bool {
//	    now := s.Clock.Now()
//	    ...
//	}
//
// En prod : `clock.System{}`. En test : `clock.Fake{T: time.Date(...)}`.
package clock

import "time"

// Clock est l'interface horloge minimale injectable.
type Clock interface {
	Now() time.Time
}

// System est l'implémentation prod — wrap `time.Now()`.
type System struct{}

// Now retourne `time.Now()`.
func (System) Now() time.Time { return time.Now() }

// Fake est l'implémentation test — retourne une heure fixe.
type Fake struct {
	T time.Time
}

// Now retourne f.T (heure mockée).
func (f Fake) Now() time.Time { return f.T }
