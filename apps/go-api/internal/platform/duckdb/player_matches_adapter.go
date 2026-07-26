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
	"log/slog"

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
	rows, err := a.repo.Load(ctx, filters)
	if err != nil {
		return nil, err
	}
	// Enrichissement FR/EN des labels (maps, modes, playlists) depuis
	// metadata.asset_translations + mode_name_tr. Les colonnes r.*_name_fr de
	// match_registry sont souvent NULL → sans cet enrichissement, les pages
	// canonical (Stats, Session, SessionCompare, etc.) affichent les libellés en
	// EN. Le Home/Synthesis l'appellent déjà via leurs propres repos ; on le
	// centralise ici pour tous les consommateurs du port. Best-effort : on
	// n'échoue pas le chargement si l'enrichissement échoue (DB metadata absente…).
	if err := NewHomeRepo(a.repo.pdb).EnrichCanonicalAssetTranslations(ctx, rows); err != nil {
		slog.WarnContext(ctx, "PlayerMatchesAdapter: FR asset enrichment failed", "err", err)
	}
	return rows, nil
}

// LobbySizesAtCompletion délègue à PlayerMatchesRepo (capability optionnelle
// consommée par SessionPageService pour le breakdown de placements). Le slug est
// ignoré (déjà capturé au constructeur).
func (a *PlayerMatchesAdapter) LobbySizesAtCompletion(
	ctx context.Context,
	_ string,
	matchIDs []string,
) (map[string]int, error) {
	return a.repo.LobbySizesAtCompletion(ctx, matchIDs)
}

// InvalidatePlayer est un no-op pour cette implémentation per-player. Le cache
// LRU n'est pas applicable ici car chaque PlayerDB est déjà résolu une fois par
// requête HTTP via le pool.
func (a *PlayerMatchesAdapter) InvalidatePlayer(_, _ string) {}
