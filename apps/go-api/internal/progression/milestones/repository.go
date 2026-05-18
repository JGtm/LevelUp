package milestones

import "context"

// repository.go — interfaces de persistance pour le catalogue et les
// milestones débloqués.
//
// Le catalogue vit dans metadata.duckdb (cross-titres, statique). Les
// milestones débloqués vivent dans stats.duckdb (par joueur).

// CatalogRepo gère le référentiel des milestones (chargé du TOML).
type CatalogRepo interface {
	// Upsert insère ou met à jour une entrée. Utilisé par le loader TOML
	// au boot pour synchroniser le catalogue.
	Upsert(ctx context.Context, e CatalogEntry) error

	// ListByTitle retourne toutes les entrées pour un titre, triées par
	// (metric ASC, threshold ASC) pour faciliter l'ordre de progression.
	ListByTitle(ctx context.Context, titleSlug string) ([]CatalogEntry, error)
}

// EarnedRepo gère les milestones débloqués par un joueur.
type EarnedRepo interface {
	// IsEarned retourne true si le milestone est déjà débloqué pour ce joueur.
	IsEarned(ctx context.Context, userID, titleSlug, milestoneID string) (bool, error)

	// Append enregistre un déblocage. PK composite (user_id, title_slug,
	// milestone_id) garantit l'idempotence : un milestone ne peut être
	// débloqué qu'une fois.
	Append(ctx context.Context, e Earned) error

	// ListByUser retourne tous les milestones débloqués pour ce joueur sur
	// ce titre, triés par earned_at DESC (plus récents d'abord).
	ListByUser(ctx context.Context, userID, titleSlug string) ([]Earned, error)
}
