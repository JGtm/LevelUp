// Package duckdb — squad_v2_adapter.go : adapteur production qui implémente
// service.SquadV2Loader en résolvant (titleSlug, gamertag) -> *PlayerDB via
// le pool global, puis en déléguant la lecture à PlayerMatchesRepo.
//
// L'adapteur s'appuie sur un TitlePlayerResolver injecté par le wiring (cf.
// internal/api/registry.go) pour éviter d'importer le package config — ce qui
// créerait une dépendance circulaire (config importe duckdb).
//
// Capability gating : si le PlayerDB est introuvable (joueur absent de
// db_profiles, fichier stats.duckdb manquant, slug titre inconnu), l'erreur est
// traduite en games.ErrCapabilityNotSupported pour que SquadServiceV2 puisse
// dégrader gracieusement (capability gap dans la réponse au lieu d'une 5xx).
package duckdb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// TitlePlayerResolver traduit une paire (titleSlug, gamertag) en *PlayerDB
// prêt à l'emploi (pool-cached). L'implémentation production est fournie par
// le wiring HTTP (registry.go) à partir de config.AppConfig.LoadPlayers +
// GetOrOpen.
//
// Doit retourner une erreur (idéalement matchant ErrPlayerNotFound) si la
// paire ne correspond à aucun profil ou si le fichier stats.duckdb est absent.
type TitlePlayerResolver func(ctx context.Context, titleSlug, gamertag string) (*PlayerDB, error)

// SquadV2LoaderAdapter implémente service.SquadV2Loader en s'appuyant sur le
// pool de PlayerDB et PlayerMatchesRepo.
type SquadV2LoaderAdapter struct {
	resolve TitlePlayerResolver
}

// NewSquadV2LoaderAdapter construit un adapteur production. Le resolver est
// injecté par le wiring (registry.go) — ne pas l'appeler avec resolver=nil.
func NewSquadV2LoaderAdapter(resolver TitlePlayerResolver) *SquadV2LoaderAdapter {
	return &SquadV2LoaderAdapter{resolve: resolver}
}

// LoadFor charge les matchs du joueur (titleSlug, gamertag) en passant par le
// pool DuckDB et PlayerMatchesRepo.
//
// Si la résolution échoue parce que le profil ou la base est absent, l'erreur
// est traduite en games.ErrCapabilityNotSupported. Les autres erreurs (DB
// corrompue, requête en échec) remontent telles quelles.
func (a *SquadV2LoaderAdapter) LoadFor(
	ctx context.Context,
	titleSlug string,
	gamertag string,
	filters port.PlayerMatchFilters,
) ([]canonical.PlayerMatchRow, error) {
	if a.resolve == nil {
		return nil, errors.New("SquadV2LoaderAdapter: resolver non câblé")
	}

	pdb, err := a.resolve(ctx, titleSlug, gamertag)
	if err != nil {
		if isPlayerCapabilityError(err) {
			return nil, fmt.Errorf("%w: title=%q gamertag=%q (%v)",
				games.ErrCapabilityNotSupported, titleSlug, gamertag, err)
		}
		return nil, fmt.Errorf("SquadV2LoaderAdapter.LoadFor: resolve %s/%s: %w",
			titleSlug, gamertag, err)
	}

	repo := NewPlayerMatchesRepo(pdb)
	rows, err := repo.Load(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("SquadV2LoaderAdapter.LoadFor: load %s: %w", gamertag, err)
	}
	return rows, nil
}

// isPlayerCapabilityError détecte les motifs d'erreur signifiant "ce joueur
// n'a pas la capability match.history" : profil introuvable dans
// db_profiles.json ou fichier stats.duckdb manquant. Tout autre échec
// (timeout, DB corrompue) reste une vraie erreur.
func isPlayerCapabilityError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	// Le message canonique côté config.ResolvePlayer est "joueur introuvable" ;
	// on reste défensif via une string-match prudente (le package duckdb ne
	// peut pas importer config sans créer un cycle).
	msg := err.Error()
	for _, marker := range []string{
		"joueur introuvable",
		"player_not_found",
		"no such file",
		"does not exist",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}
