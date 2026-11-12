package migration

// playlist_weights_schema.go : la table des poids de rotation des couples carte-mode d'une playlist
// (`playlist_map_mode_weights`, base metadata). Elle n'est PAS une étape du registre : elle naît à la
// première expansion des playlists du catalogue, qui l'écrit, et aucun lecteur de l'application ne la
// requiert. Le DDL vit ici pour qu'aucune couche d'appel ne porte de CREATE TABLE.

import (
	"context"
	"database/sql"
	"fmt"
)

// EnsurePlaylistWeightsTable crée la table des poids de rotation si elle est absente (idempotent). Clé
// primaire composite : la table n'est écrite que par l'expansion des playlists, jamais en concurrence
// (un seul écrivain, SELECT-then-write, jamais d'ON CONFLICT).
func EnsurePlaylistWeightsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS playlist_map_mode_weights (
			title_slug        VARCHAR NOT NULL,
			playlist_asset_id VARCHAR NOT NULL,
			pair_asset_id     VARCHAR NOT NULL,
			weight            DOUBLE,
			updated_at        TIMESTAMP,
			PRIMARY KEY (title_slug, playlist_asset_id, pair_asset_id)
		)`)
	if err != nil {
		return fmt.Errorf("create playlist_map_mode_weights: %w", err)
	}
	return nil
}
