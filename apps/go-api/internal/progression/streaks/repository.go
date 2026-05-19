package streaks

import "context"

// repository.go — interface de persistance des streaks.
//
// Implémentation : internal/platform/duckdb/streaks_repo.go (stats.duckdb par joueur).
// La couche logique (evaluator.go) ne dépend que de cette interface.

// Repo gère la persistance des streaks pour un joueur.
type Repo interface {
	// GetActive retourne la streak active pour (user, title, type), ou nil
	// si aucune n'est active. Une seule streak peut être active par triplet.
	GetActive(ctx context.Context, userID, titleSlug string, st StreakType) (*Streak, error)

	// Upsert crée ou met à jour une streak. La clé est ID.
	Upsert(ctx context.Context, s Streak) error

	// List retourne toutes les streaks (active + historiques) pour un joueur
	// sur un titre, ordonnées par started_at DESC.
	List(ctx context.Context, userID, titleSlug string) ([]Streak, error)
}
