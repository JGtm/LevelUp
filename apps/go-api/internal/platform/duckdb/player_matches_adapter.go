// Package duckdb — player_matches_adapter.go : adapteur per-player qui implémente
// `port.PlayerMatchesRepository` à partir d'un `*PlayerMatchesRepo` lié à un
// PlayerDB précis (P4.3 finale, ADR 0011).
//
// L'adapteur ignore les paramètres slug/gamertag de l'interface (déjà fixés au
// constructeur). Il existe pour faire le pont entre l'interface globale du
// port et l'implémentation per-player concrète. Permet aux services
// (HomeService, StatsService, etc.) de consommer canonical via le port unifié.
package duckdb

import (
	"context"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// PlayerMatchesAdapter wrappe un PlayerMatchesRepo (per-player) en implémentation
// de port.PlayerMatchesRepository (interface globale). Les paramètres
// slug/gamertag sont fixés au constructeur ; l'adapteur les ignore en arguments
// d'appel (ils sont déjà résolus côté pool/PlayerDB par l'appelant).
type PlayerMatchesAdapter struct {
	repo     *PlayerMatchesRepo
	slug     string
	gamertag string
}

// NewPlayerMatchesAdapter construit un adapter lié au PlayerMatchesRepo donné.
func NewPlayerMatchesAdapter(repo *PlayerMatchesRepo, slug, gamertag string) *PlayerMatchesAdapter {
	return &PlayerMatchesAdapter{repo: repo, slug: slug, gamertag: gamertag}
}

// LoadPlayerMatches délègue à PlayerMatchesRepo.Load. Les paramètres slug et
// gamertag sont ignorés (déjà capturés au constructeur).
func (a *PlayerMatchesAdapter) LoadPlayerMatches(
	ctx context.Context,
	_ string,
	_ string,
	filters port.PlayerMatchFilters,
) ([]canonical.PlayerMatchRow, error) {
	return a.repo.Load(ctx, filters)
}

// InvalidatePlayer est un no-op pour cette implémentation per-player. Le cache
// LRU n'est pas applicable ici car chaque PlayerDB est déjà résolu une fois par
// requête HTTP via le pool.
func (a *PlayerMatchesAdapter) InvalidatePlayer(_, _ string) {}
