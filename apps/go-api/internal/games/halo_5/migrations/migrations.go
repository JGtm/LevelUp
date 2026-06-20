// Package migrations porte le jeu de migrations DuckDB de Halo 5.
//
// Phase 1a (live-only) : l'adapter Halo 5 lit l'API cryptum en direct, AUCUNE
// persistance DB. Le set est donc VIDE pour tous les targets. Son rôle est
// l'ISOLATION : enregistrer un set (même vide) pour halo_5 empêche le runner de
// retomber sur le fallback legacy de RunForTitleDB (= les migrations Halo
// Infinite), qui créerait des tables match_registry/match_participants/etc.
// parasites — jamais peuplées — dans le warehouse Halo 5.
//
// La durabilité des events (plan 4a) ajoutera ici la migration de la table
// highlight_events quand son schéma (identité gamertag vs xuid) sera tranché.
package migrations

import (
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/migration"
)

// Set retourne le jeu de migrations (vide en Phase 1a) de Halo 5. Steps retourne
// nil pour tout target → no-op propre ; CanonicalOrder vide.
func Set() migration.TitleMigrationSet {
	return migration.TitleMigrationSet{
		Slug:           halo5.TitleSlug,
		CanonicalOrder: nil,
		Steps:          func(migration.TargetDB) []migration.Migration { return nil },
	}
}

// Register enregistre le set auprès du runner de migrations. À appeler au boot
// AVANT tout RunForTitleDB(halo_5, ...) — donc avant provisionAdditionalActiveTitles.
// Inerte tant que halo_5 n'est pas provisionné (status != active) : peupler la map
// des sets n'a aucun effet de bord tant que le runner n'est pas invoqué pour ce slug.
func Register() {
	migration.RegisterMigrationSet(Set())
}
