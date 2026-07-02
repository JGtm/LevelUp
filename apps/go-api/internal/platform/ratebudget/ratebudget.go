// Package ratebudget — registre process-wide de rate.Limiter PAR COMPTE Xbox
// (xuid). Sujet 2 throttling, incrément T1 (unification du budget).
//
// PROBLÈME : trois familles de consommateurs dépensaient le quota API du MÊME
// compte Xbox avec des limiteurs INDÉPENDANTS — le pool de tokens (limiter
// par-slot), career_live (limiter local par-requête), worldenrich (limiter
// local par-source). Le pool ne voyait jamais la vraie pression par compte →
// 429 « surprises » malgré un PerTokenRPS respecté localement.
//
// SOLUTION : un limiteur UNIQUE par xuid, partagé par construction — tous les
// clients HaloAPIClient d'un même compte attendent sur le même token bucket.
// Package FEUILLE (x/time/rate uniquement) : importable par pool, service,
// worldenrich sans cycle.
//
// AIMD (incrément T2) : SetRPS permet au pool d'abaisser le débit d'un compte
// sur 429 (÷2, plancher) et de le restaurer progressivement — l'ajustement
// profite immédiatement à TOUS les consommateurs du compte.
package ratebudget

import (
	"sync"

	"golang.org/x/time/rate"
)

// minRPS est le plancher AIMD : un compte n'est jamais ralenti sous ce débit
// (sinon la convergence s'arrête de fait).
const minRPS = 1.0

type entry struct {
	limiter *rate.Limiter
	baseRPS float64 // débit nominal (plafond de restauration AIMD)
}

var registry = struct {
	mu sync.Mutex
	m  map[string]*entry
}{m: make(map[string]*entry)}

// ForXUID retourne le limiteur PARTAGÉ du compte xuid, en le créant au premier
// appel avec le débit rps (les appels suivants ignorent rps — le débit courant,
// éventuellement ajusté par AIMD, est conservé). xuid vide → limiteur local
// non partagé (impossible d'attribuer ; comportement historique).
func ForXUID(xuid string, rps float64) *rate.Limiter {
	if rps <= 0 {
		rps = minRPS
	}
	if xuid == "" {
		return rate.NewLimiter(rate.Limit(rps), 1)
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if e, ok := registry.m[xuid]; ok {
		return e.limiter
	}
	e := &entry{limiter: rate.NewLimiter(rate.Limit(rps), 1), baseRPS: rps}
	registry.m[xuid] = e
	return e.limiter
}

// HalveRPS divise le débit du compte par 2 (plancher minRPS) — appelé par le
// pool sur un 429 imputé à ce compte (AIMD : multiplicative decrease). Retourne
// le nouveau débit (0 si le compte est inconnu — no-op).
func HalveRPS(xuid string) float64 {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	e, ok := registry.m[xuid]
	if !ok {
		return 0
	}
	next := float64(e.limiter.Limit()) / 2
	if next < minRPS {
		next = minRPS
	}
	e.limiter.SetLimit(rate.Limit(next))
	return next
}

// RestoreStep remonte le débit du compte de step (additive increase), plafonné
// à son débit nominal. Appelé périodiquement (refresher pool) pour les comptes
// sains. Retourne le débit courant après ajustement (0 si compte inconnu).
func RestoreStep(xuid string, step float64) float64 {
	if step <= 0 {
		return 0
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	e, ok := registry.m[xuid]
	if !ok {
		return 0
	}
	cur := float64(e.limiter.Limit())
	if cur >= e.baseRPS {
		return cur
	}
	next := cur + step
	if next > e.baseRPS {
		next = e.baseRPS
	}
	e.limiter.SetLimit(rate.Limit(next))
	return next
}

// CurrentRPS retourne le débit courant du compte (0 si inconnu) — observabilité.
func CurrentRPS(xuid string) float64 {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if e, ok := registry.m[xuid]; ok {
		return float64(e.limiter.Limit())
	}
	return 0
}
