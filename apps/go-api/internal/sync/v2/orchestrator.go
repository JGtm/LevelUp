// Package v2 — orchestrator.go : interface CycleOrchestrator et stub.
//
// L'implémentation des 6 phases est livrée incrémentalement (D1 à D5 du
// plan ADR 0027). D0 fournit l'interface et un stub qui retourne
// ErrNotImplemented pour câbler le scheduler et la suite contract sans
// casser la build.
package v2

import (
	"context"
	"errors"
	"time"
)

// ErrNotImplemented est retourné par le stub orchestrator tant que les
// phases ne sont pas livrées. Le scheduler tombe en fallback V1 quand il
// reçoit cette erreur (cf. scheduler/auto_sync.go Run loop).
var ErrNotImplemented = errors.New("sync.v2: CycleOrchestrator not yet implemented — fallback to V1")

// CycleOrchestrator orchestre un cycle de sync complet pour N joueurs.
// Une implémentation est process-wide singleton (instanciée par le
// scheduler au boot, réutilisée à chaque tick).
//
// Contract :
//   - Run est appelable concurremment seulement si l'implémentation le
//     déclare thread-safe (le stub ne l'est pas, à valider par phase).
//   - Run NE doit JAMAIS retourner d'erreur fatale pour un cycle complet
//     si un seul joueur échoue : le résultat partiel est dans CycleResult.
//     Une erreur de retour signifie un échec global (boot DB indispo,
//     etc.) qui devrait stopper le scheduler.
//   - PhaseDurations doit être peuplé même en cas d'échec partiel pour
//     diagnostic.
type CycleOrchestrator interface {
	Run(ctx context.Context, players []PlayerProfile) (CycleResult, error)
}

// stubOrchestrator est l'implémentation D0 : retourne ErrNotImplemented
// avec un CycleResult vide. Utile pour valider le câblage scheduler
// avant que les phases ne soient livrées.
type stubOrchestrator struct{}

// NewStubOrchestrator construit un orchestrator stub.
// À remplacer par NewCycleOrchestrator (D6) une fois Phases 1-5 livrées.
func NewStubOrchestrator() CycleOrchestrator {
	return &stubOrchestrator{}
}

// Run du stub retourne immédiatement ErrNotImplemented.
// Le scheduler doit traiter cette erreur comme un signal de fallback V1,
// pas comme un échec à logger en ERROR (cf. scheduler/auto_sync.go).
func (s *stubOrchestrator) Run(_ context.Context, _ []PlayerProfile) (CycleResult, error) {
	return CycleResult{
		StartedAt:      time.Now(),
		PerPlayer:      map[string]PlayerOutcome{},
		PhaseDurations: map[string]time.Duration{},
	}, ErrNotImplemented
}
