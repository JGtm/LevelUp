package duckdb

// playlist_expansion_store.go : la lecture de l'expansion des playlists du catalogue, derrière
// `port.PlaylistsSeenReader`. Les écritures (file de récupération, poids de rotation) vivent dans
// persist.PlaylistChildrenPersister, le DDL dans migration.EnsurePlaylistWeightsTable.

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// signalUnreadableRow journalise et compte une ligne dont le Scan a échoué dans une lecture best-effort ;
// la ligne est ignorée, la lecture continue. Une lecture qui perd des lignes sans le dire sert un résultat
// partiel pour un résultat complet.
func signalUnreadableRow(ctx context.Context, titleSlug, lecture string, err error) {
	observability.IncCounterT(titleSlug, "wire_rows_scan_failed_total")
	slog.WarnContext(ctx, "wire: ligne illisible ignorée", "lecture", lecture, "titleSlug", titleSlug, "err", err)
}

// PlaylistsSeenReader lit les playlists du registre des matchs d'un titre.
type PlaylistsSeenReader struct {
	Bases TitleBases
}

var _ port.PlaylistsSeenReader = PlaylistsSeenReader{}

// PlaylistsSeenInMatches rend les playlists distinctes de match_registry (avec version), dans une lecture de
// la base partagée rendue en sortant.
func (r PlaylistsSeenReader) PlaylistsSeenInMatches(ctx context.Context, titleSlug string) ([]port.PlaylistRef, error) {
	sharedSQL, release, err := r.Bases.ReadShared(ctx, titleSlug)
	if err != nil {
		return nil, err
	}
	defer release()
	rows, err := sharedSQL.QueryContext(ctx, `
		SELECT DISTINCT playlist_id, playlist_version_id
		FROM match_registry
		WHERE playlist_id IS NOT NULL AND playlist_id != ''
		  AND playlist_version_id IS NOT NULL AND playlist_version_id != ''`)
	if err != nil {
		return nil, fmt.Errorf("query playlists: %w", err)
	}
	defer rows.Close()
	var playlists []port.PlaylistRef
	for rows.Next() {
		var p port.PlaylistRef
		if err := rows.Scan(&p.ID, &p.Version); err != nil {
			signalUnreadableRow(ctx, titleSlug, "catalog_expand_playlists", err)
			continue
		}
		playlists = append(playlists, p)
	}
	return playlists, rows.Err()
}
