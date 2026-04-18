// Package port — fanout.go : ports hexagonaux pour l'enrichissement fanout.
package port

import "context"

// FanoutRepository expose les opérations de persistance nécessaires au fanout.
// Implémenté par platform/duckdb.FanoutRepo.
type FanoutRepository interface {
	// CountCommonMatchesForXUID compte les matchs de matchIDs où targetXUID était participant.
	CountCommonMatchesForXUID(ctx context.Context, targetXUID string, matchIDs []string) (int, error)

	// LoadExistingEnrichments retourne l'ensemble des match_ids déjà enrichis.
	LoadExistingEnrichments(ctx context.Context, matchIDs []string) (map[string]bool, error)

	// InsertStubEnrichments insère des stubs player_match_enrichment pour les matchs manquants.
	InsertStubEnrichments(ctx context.Context, xuid string, matchIDs []string) (int, error)
}

// FanoutPlayerFactory ouvre un FanoutRepository pour un joueur donné.
// Implémenté par config.FanoutFactory.
type FanoutPlayerFactory interface {
	OpenForPlayer(ctx context.Context, gamertag, titleSlug string) (FanoutRepository, error)
}
