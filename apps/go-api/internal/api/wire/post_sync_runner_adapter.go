// Package api — post_sync_runner_adapter.go : adapter qui expose
// BuildPostSyncDeltaHook comme une implémentation de port.PostSyncRunner.
//
// Phase 4 plan stabilisation 2026-05-22 — solution B1 (étape 2/3).
//
// Permet à SyncEngine (couche sync, hors api) d'invoquer le hook post-sync
// sans importer api directement. La résolution du PlayerDB via ServiceRegistry
// reste interne à api ; seul le wrapper conforme à port.PostSyncRunner traverse
// la frontière de package.
package wire

import (
	"context"

	"levelup/go-api/internal/port"
)

// postSyncRunnerAdapter adapte la closure handlers.PostSyncDeltaHook au
// contrat port.PostSyncRunner.
type postSyncRunnerAdapter struct {
	reg *ServiceRegistry
}

// BeforeSync délègue à BuildPostSyncDeltaHook(reg) qui capture le snapshot
// before via ServiceRegistry.resolve(slug) puis retourne la closure after.
func (a *postSyncRunnerAdapter) BeforeSync(ctx context.Context, slug string) port.PostSyncFinalizer {
	// On recrée la closure à chaque appel pour capturer correctement la
	// snapshot par invocation (state machine du hook = capture+emit).
	hook := BuildPostSyncDeltaHook(a.reg)
	after := hook(ctx, slug)
	if after == nil {
		return nil
	}
	// after est de type func(ctx context.Context) — compatible avec
	// port.PostSyncFinalizer par signature.
	return port.PostSyncFinalizer(after)
}

// NewPostSyncRunner construit un port.PostSyncRunner depuis un ServiceRegistry.
// À appeler dans cmd/server/main.go au boot, puis injecter sur :
//   - SyncHandler.WithPostSyncRunner (HTTP)
//   - scheduler.AutoSyncScheduler.WithPostSyncRunner (auto-sync 15 min)
//   - cmd/levelup CLI sync subcommands (si applicable)
//
// Toutes les entry points utilisent le MÊME runner — garantie de cohérence
// (les 3 paths invoquent le même hook).
func NewPostSyncRunner(reg *ServiceRegistry) port.PostSyncRunner {
	if reg == nil {
		return nil
	}
	return &postSyncRunnerAdapter{reg: reg}
}
