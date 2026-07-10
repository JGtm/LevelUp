package notifications

import (
	"context"
	"time"
)

// Repository est le port consommé par Service. Implémenté côté
// internal/platform/duckdb/notifications_repo.go.
//
// Toutes les méthodes scopent l'opération au PlayerDB encapsulé par l'impl
// (la sélection du joueur se fait à la construction du repo via le DI).
type Repository interface {
	// Insert persiste une notif déjà préparée (ID alloué, params encodés).
	// Retourne ErrCategoryDisabled si la pref est OFF.
	Insert(ctx context.Context, n *Notification) error

	// List paginé / filtré ; renvoie next_cursor à passer en BeforeID
	// pour la page suivante (nil si fin).
	List(ctx context.Context, f ListFilter) (ListResult, error)

	// UnreadCount retourne le total non-lu et la répartition par catégorie.
	UnreadCount(ctx context.Context) (UnreadCount, error)

	// MarkRead positionne read_at = now() sur les IDs fournis.
	// Idempotent (MarkRead sur déjà lu = no-op).
	MarkRead(ctx context.Context, ids []int64) (int, error)

	// MarkUnread remet read_at = NULL.
	MarkUnread(ctx context.Context, id int64) error

	// MarkAllRead positionne read_at sur toutes les notifs non-lues
	// (filtré par category si non vide).
	MarkAllRead(ctx context.Context, category Category) (int, error)

	// Delete supprime une notif (action "ignorer").
	Delete(ctx context.Context, id int64) error

	// CapAndSweep purge les notifs dépassant le cap de rétention (best-effort).
	CapAndSweep(ctx context.Context, max int) error

	// SweepStaleInfoRead marque lues les notifs severity='info' non lues plus
	// anciennes que cutoff (expiry douce DP8, best-effort). Idempotent.
	SweepStaleInfoRead(ctx context.Context, cutoff time.Time) error

	// GetPreferences charge l'état complet de notification_preferences.
	GetPreferences(ctx context.Context) ([]Preference, error)

	// UpsertPreferences applique des changements (par catégorie).
	UpsertPreferences(ctx context.Context, prefs []Preference) error

	// IsCategoryEnabled vérifie si une catégorie est activée (pour court-circuiter Emit).
	IsCategoryEnabled(ctx context.Context, c Category) (bool, error)
}
