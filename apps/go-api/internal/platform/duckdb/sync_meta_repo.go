// Package duckdb — sync_meta_repo.go : lecture/écriture de la table clé-valeur
// `sync_meta` d'une player DB. Extrait (dédup #6, K1c 2026-07-05) : le couple
// read + write ART-safe était copié-collé dans internal/api/notifications_title_ready.go
// et notifications_boot.go (readLastSeenAppVersion/writeLastSeenAppVersion).
//
// Écriture ART-safe : SELECT-then-UPDATE-or-INSERT (JAMAIS d'ON CONFLICT sur la PK
// `key`, qui réécrirait via l'index ART DuckDB — bug #23046). sync_meta est une
// table clé/valeur sans index secondaire.
//
// DETTE (ADR 0013) : WriteSyncMeta ouvre un handle RW direct via OpenReadWrite
// (comportement PRÉSERVÉ des 2 copies). Le durcissement « écriture SOUS LEASE
// dblease » (un seul writer par DB) reste un follow-up K1c — cf. plan.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// WriteSyncMeta upsert une clé sync_meta (ART-safe SELECT-then-UPDATE-or-INSERT).
func WriteSyncMeta(ctx context.Context, pdb *PlayerDB, key, value string) error {
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
