// Package api — registry_notifications.go : factories pour le système de
// notifications in-app (per-player).
//
// Externalisé de registry.go pour respecter la limite 500L par module
// (CLAUDE.md §14).
package api

import (
	"context"
	"sync"

	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

// notifServicesByXUID cache les *notifications.Service par xuid.
//
// Important : la monotonicité des IDs (snowflake-like) est garantie *au sein
// d'une instance de Service*. Si deux instances coexistaient pour le même
// joueur, leurs générateurs d'ID pourraient collisionner sub-milliseconde.
// On cache donc une instance unique par xuid pour la durée de vie du process.
var notifServicesByXUID sync.Map // map[string]*notifications.Service

// Notifications retourne le *notifications.Service associé au joueur identifié
// par slug. Cache process-level par xuid (cf. notifServicesByXUID).
func (r *ServiceRegistry) Notifications(ctx context.Context, slug string) (*notifications.Service, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return notifServiceFor(pdb), nil
}

// NotificationsEmitter retourne l'interface Emitter (sous-ensemble de Service)
// pour l'injection dans les hooks d'émission (sync engine, media handler, etc.).
//
// Variante minimaliste : ce qu'attend un consommateur est l'interface Emitter,
// pas le Service complet. Le compile-time check `var _ Emitter = (*Service)(nil)`
// dans le package notifications garantit la compatibilité.
func (r *ServiceRegistry) NotificationsEmitter(ctx context.Context, slug string) (notifications.Emitter, error) {
	return r.Notifications(ctx, slug)
}

// notifServiceFor construit ou retourne l'instance cachée pour ce PlayerDB.
func notifServiceFor(pdb *duckdb.PlayerDB) *notifications.Service {
	if v, ok := notifServicesByXUID.Load(pdb.XUID); ok {
		return v.(*notifications.Service)
	}
	repo := duckdb.NewNotificationsRepo(pdb)
	svc := notifications.NewService(repo)
	// LoadOrStore évite la race : si un autre goroutine a inséré
	// entre Load et Store, on retourne celle-là.
	if existing, loaded := notifServicesByXUID.LoadOrStore(pdb.XUID, svc); loaded {
		return existing.(*notifications.Service)
	}
	return svc
}
