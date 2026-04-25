package prestige

import (
	"errors"
	"time"
)

// lifecycle.go — state machine et règles d'édition d'un défi.
//
// Référence : Axe 7 du plan conceptuel.
//
// State machine :
//   draft → active → (completed | expired | abandoned) → archived
//
// Règles d'édition :
//   - Mode libre  : la cible est éditable, recalcule le palier sur baseline courante
//   - Mode pilote : la cible est figée à la création (CanEditTarget = false)
//   - Statut != active : aucune édition de cible possible

// Erreurs de transition. Les handlers HTTP les traduisent en codes d'erreur API.
var (
	ErrInvalidTransition = errors.New("prestige: transition d'état invalide")
	ErrNotEditable       = errors.New("prestige: cible non éditable (mode pilote ou statut terminal)")
	ErrCooldownActive    = errors.New("prestige: cooldown actif sur cette métrique")
	ErrAlreadyTerminal   = errors.New("prestige: défi déjà en statut terminal")
)

// CanEditTarget retourne true si la cible du défi peut être modifiée.
//
// Critères :
//   - statut == active (hors active = pas d'édition de cible)
//   - mode == libre (le mode pilote a une cible figée par contrat)
func CanEditTarget(c Challenge) bool {
	if c.Status != StatusActive {
		return false
	}
	if c.Mode != ModeLibre {
		return false
	}
	return true
}

// CanAbandon retourne true si le défi peut être abandonné (statut active uniquement).
func CanAbandon(c Challenge) bool {
	return c.Status == StatusActive
}

// Commit fait passer un défi de draft à active.
//
// Précondition : statut == draft. Sinon ErrInvalidTransition.
func Commit(c Challenge, now time.Time) (Challenge, error) {
	if c.Status != StatusDraft {
		return c, ErrInvalidTransition
	}
	updated := c
	updated.Status = StatusActive
	updated.CommittedAt = &now
	return updated, nil
}

// MarkCompleted fait passer un défi à completed.
//
// Précondition : statut == active. Sinon ErrInvalidTransition.
func MarkCompleted(c Challenge, now time.Time) (Challenge, error) {
	if c.Status != StatusActive {
		return c, ErrInvalidTransition
	}
	updated := c
	updated.Status = StatusCompleted
	updated.CompletedAt = &now
	return updated, nil
}

// MarkExpired fait passer un défi à expired.
//
// Précondition : statut == active. Sinon ErrInvalidTransition.
func MarkExpired(c Challenge, now time.Time) (Challenge, error) {
	if c.Status != StatusActive {
		return c, ErrInvalidTransition
	}
	updated := c
	updated.Status = StatusExpired
	updated.ExpiredAt = &now
	return updated, nil
}

// MarkAbandoned fait passer un défi à abandoned.
//
// Précondition : statut == active. Sinon ErrInvalidTransition.
func MarkAbandoned(c Challenge, now time.Time) (Challenge, error) {
	if c.Status != StatusActive {
		return c, ErrInvalidTransition
	}
	updated := c
	updated.Status = StatusAbandoned
	updated.AbandonedAt = &now
	return updated, nil
}

// CooldownEndsAt retourne l'instant de fin du cooldown sur la métrique
// du défi terminé. Renvoie un time.Time zéro si pas de cooldown applicable
// (statut non terminal ou cooldown=0).
//
// Le cooldown s'applique uniquement aux modes pilote (le mode libre n'a
// pas de cooldown). L'appelant est responsable de filtrer.
func CooldownEndsAt(t Tuning, c Challenge) time.Time {
	if !c.Status.IsTerminal() {
		return time.Time{}
	}
	if c.Mode != ModePilote {
		return time.Time{}
	}
	d := t.CooldownDuration(c.Status)
	if d == 0 {
		return time.Time{}
	}
	end := terminalTime(c)
	if end.IsZero() {
		return time.Time{}
	}
	return end.Add(d)
}

// terminalTime extrait le timestamp de la dernière transition terminale.
func terminalTime(c Challenge) time.Time {
	switch c.Status {
	case StatusCompleted:
		if c.CompletedAt != nil {
			return *c.CompletedAt
		}
	case StatusExpired:
		if c.ExpiredAt != nil {
			return *c.ExpiredAt
		}
	case StatusAbandoned:
		if c.AbandonedAt != nil {
			return *c.AbandonedAt
		}
	}
	return time.Time{}
}

// IsCooldownActive retourne true si un nouveau défi sur la même métrique
// serait bloqué par le cooldown du défi précédent.
//
// previousChallenges = défis du joueur sur la même métrique, déjà terminés.
// now = timestamp du moment où on tente de créer un nouveau défi.
func IsCooldownActive(t Tuning, previousChallenges []Challenge, now time.Time) bool {
	for _, prev := range previousChallenges {
		if !prev.Status.IsTerminal() {
			continue
		}
		end := CooldownEndsAt(t, prev)
		if !end.IsZero() && now.Before(end) {
			return true
		}
	}
	return false
}
