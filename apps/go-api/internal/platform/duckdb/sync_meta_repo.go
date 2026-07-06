// Package duckdb — sync_meta_repo.go : lecture/écriture de la table clé-valeur
// `sync_meta` d'une player DB. Extrait (dédup #6, K1c 2026-07-05) : le couple
// read + write ART-safe était copié-collé dans internal/api/notifications_title_ready.go
// et notifications_boot.go (readLastSeenAppVersion/writeLastSeenAppVersion).
//
// Écriture ART-safe : SELECT-then-UPDATE-or-INSERT (JAMAIS d'ON CONFLICT sur la PK
// `key`, qui réécrirait via l'index ART DuckDB — bug #23046). sync_meta est une
// table clé/valeur sans index secondaire.
//
// ADR 0013 : WriteSyncMeta écrit SOUS LEASE dblease (KindPlayer, un seul writer par DB)
// — sérialise avec le post-sync / CLI / autres écrivains de la MÊME player DB. Sans lease,
// deux writers concurrents sur sync_meta ne sont sûrs que par l'effet de bord
// MaxOpenConns(1) du cache de connexion — fragile (K1c durcissement, 2026-07-06).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"levelup/go-api/internal/platform/dblease"
)

// ReadSyncMeta retourne sync_meta.value pour une clé, "" si absente. La table
// sync_meta est garantie présente après la migration "create_base_player_schema".
func ReadSyncMeta(ctx context.Context, pdb *PlayerDB, key string) (string, error) {
	var v sql.NullString
	err := pdb.ReadDB().QueryRow(ctx, `SELECT value FROM sync_meta WHERE key = ?`, key).Scan(&v)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", nil
	case err != nil:
		return "", err
	}
	return v.String, nil
}

// WriteSyncMeta upsert une clé sync_meta (ART-safe SELECT-then-UPDATE-or-INSERT),
// SOUS LEASE dblease (KindPlayer) : sérialise avec les autres écrivains de la player DB.
// ErrDBLocked remonte si le lease n'est pas obtenu dans le délai (mappé 503 côté handler).
func WriteSyncMeta(ctx context.Context, pdb *PlayerDB, key, value string) error {
	w, err := pdb.AcquirePlayerWriterTimeout(dblease.PlayerLeaseTimeout)
	if err != nil {
		return fmt.Errorf("lease: %w", err)
	}
	defer w.Release()

	rwDB, err := OpenReadWrite(pdb.Player.Path())
	if err != nil {
		return fmt.Errorf("open rw: %w", err)
	}
	defer rwDB.Close()
	return rwDB.UpsertNoConflict(ctx,
		`SELECT 1 FROM sync_meta WHERE key = ?`,
		[]any{key},
		`UPDATE sync_meta SET value = ?, updated_at = NOW() WHERE key = ?`,
		[]any{value, key},
		`INSERT INTO sync_meta (key, value, updated_at) VALUES (?, ?, NOW())`,
		[]any{key, value},
	)
}
