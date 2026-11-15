package persist

// playlist_children_persister.go : les écritures de l'expansion des playlists du catalogue dans metadata
// — la file de récupération des couples carte-mode (`catalog_fetch_queue`) et leurs poids de rotation
// (`playlist_map_mode_weights`).
//
// ART-SAFETY (bug DuckDB #23645) : aucune de ces tables n'est écrite par UPSERT.
//   - `catalog_fetch_queue` n'a plus de clé primaire : INSERT OR IGNORE échouerait en dur. L'enfilage est
//     un INSERT ... SELECT ... WHERE NOT EXISTS, comme ops/catalog_queue.go et
//     platform/duckdb.CatalogWriterDB.
//   - `playlist_map_mode_weights` s'écrit par SELECT d'existence puis UPDATE ou INSERT
//     (duckdbkeys.Upsert), jamais par ON CONFLICT.
//
// L'appelant tient metadata en écriture partagée du processus (un seul écrivain) : rien ici ne protège la
// course entre le SELECT et l'INSERT.

import (
	"context"
	"database/sql"
	"time"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/port"
	"levelup/go-api/pkg/duckdbkeys"
)

// PlaylistChildrenPersister écrit dans metadata les couples enfants d'une playlist.
type PlaylistChildrenPersister struct {
	db *sql.DB
}

var _ port.PlaylistChildStore = (*PlaylistChildrenPersister)(nil)

// NewPlaylistChildrenPersister lie le persister à la connexion metadata.
func NewPlaylistChildrenPersister(db *sql.DB) *PlaylistChildrenPersister {
	return &PlaylistChildrenPersister{db: db}
}

// EnsureSchema crée la table des poids de rotation si elle est absente (idempotent).
func (p *PlaylistChildrenPersister) EnsureSchema(ctx context.Context) error {
	return migration.EnsurePlaylistWeightsTable(ctx, p.db)
}

// EnqueuePair enfile le couple carte-mode `pairID` (version `versionID`) dans la file de récupération du
// titre, sans doublon sur (titre, type, identifiant).
func (p *PlaylistChildrenPersister) EnqueuePair(ctx context.Context, titleSlug, pairID, versionID string) error {
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO catalog_fetch_queue (title_slug, asset_type, asset_id, version_id)
				 SELECT ?, 'pair', ?, ?
				 WHERE NOT EXISTS (
				   SELECT 1 FROM catalog_fetch_queue
				   WHERE title_slug = ? AND asset_type = 'pair' AND asset_id = ?
				 )`,
		titleSlug, pairID, versionID,
		titleSlug, pairID,
	)
	return err
}

// RecordWeight inscrit le poids de rotation du couple `pairID` dans la playlist `playlistID` (insère, ou
// met à jour le poids et l'horodatage UTC d'une ligne existante).
func (p *PlaylistChildrenPersister) RecordWeight(ctx context.Context, titleSlug, playlistID, pairID string, weight float64) error {
	now := time.Now().UTC()
	return duckdbkeys.Upsert{
		ExistsSQL:  `SELECT 1 FROM playlist_map_mode_weights WHERE title_slug=? AND playlist_asset_id=? AND pair_asset_id=?`,
		ExistsArgs: []any{titleSlug, playlistID, pairID},
		UpdateSQL:  `UPDATE playlist_map_mode_weights SET weight=?, updated_at=? WHERE title_slug=? AND playlist_asset_id=? AND pair_asset_id=?`,
		UpdateArgs: []any{weight, now, titleSlug, playlistID, pairID},
		InsertSQL:  `INSERT INTO playlist_map_mode_weights (title_slug, playlist_asset_id, pair_asset_id, weight, updated_at) VALUES (?,?,?,?,?)`,
		InsertArgs: []any{titleSlug, playlistID, pairID, weight, now},
	}.Exec(ctx, p.db)
}
