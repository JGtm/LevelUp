// Package syncmeta lit et écrit les clés de la table `sync_meta` d'une base joueur.
//
// Paquet à part pour que `internal/sync` et ses sous-paquets (`historystate`) partagent la même
// lecture et la même écriture sans cycle d'import.
//
// Contrat d'écriture : [Set] est un SELECT puis UPDATE ou INSERT (jamais `ON CONFLICT`, règle
// ART n°6 : une base joueur ancienne peut ne pas porter la clé primaire). L'appelant tient le
// bail d'écriture du joueur (`dblease.AcquireWriterCtx` ou `AcquirePlayerWriterTimeout`, règle
// ART n°4) : ce paquet ne le prend pas.
package syncmeta

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Get lit une clé de sync_meta ; ok=false quand la clé est absente.
func Get(ctx context.Context, db *sql.DB, key string) (value string, ok bool, err error) {
	err = db.QueryRowContext(ctx, `SELECT value FROM sync_meta WHERE key = ?`, key).Scan(&value)
	switch {
	case err == nil:
		return value, true, nil
	case errors.Is(err, sql.ErrNoRows):
		return "", false, nil
	}
	return "", false, fmt.Errorf("syncmeta.Get(%s): %w", key, err)
}

// Set met à jour une clé de sync_meta, ou la crée si elle est absente.
func Set(ctx context.Context, db *sql.DB, key, value string) error {
	now := time.Now().UTC()
	var dummy int
	err := db.QueryRowContext(ctx, `SELECT 1 FROM sync_meta WHERE key = ?`, key).Scan(&dummy)
	switch {
	case err == nil:
		_, err = db.ExecContext(ctx, `UPDATE sync_meta SET value = ?, updated_at = ? WHERE key = ?`, value, now, key)
	case errors.Is(err, sql.ErrNoRows):
		_, err = db.ExecContext(ctx, `INSERT INTO sync_meta (key, value, updated_at) VALUES (?, ?, ?)`, key, value, now)
	}
	if err != nil {
		return fmt.Errorf("syncmeta.Set(%s): %w", key, err)
	}
	return nil
}
