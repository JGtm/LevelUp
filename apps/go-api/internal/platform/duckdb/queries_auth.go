// Package duckdb — queries_auth.go : lecture/écriture du cache auth depuis sync_meta.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
)

// WriteOAuthRefreshToken persiste le refresh_token OAuth v2 dans sync_meta
// (clé 'oauth_refresh_token'). À appeler après chaque OAuth refresh réussi pour
// rotater le token (Microsoft tourne le refresh_token à chaque usage : sans
// persistance, le tick suivant utilise un RT révoqué et échoue).
//
// Pattern SELECT-then-UPDATE-or-INSERT (pas d'ON CONFLICT) : certaines player DB
// legacy créées par l'ancien sync Python ont un sync_meta SANS contrainte
// PRIMARY KEY/UNIQUE sur `key`, ce qui faisait échouer `ON CONFLICT(key)` avec
// « The specified columns as conflict target are not referenced by a
// UNIQUE/PRIMARY KEY CONSTRAINT » (cf. CLAUDE.md, règle écritures legacy auth).
// Ce write DuckDB est un double-write de compat (ADR 0023) : la source unique
// reste le MultiUserTokenStore, écrit en premier par le callback onRotated.
//
// Le db doit être ouvert en mode read-write.
func WriteOAuthRefreshToken(ctx context.Context, db *DB, token string) error {
	if token == "" {
		return nil
	}
	var existing string
	err := db.QueryRow(ctx,
		"SELECT value FROM sync_meta WHERE key = 'oauth_refresh_token'").Scan(&existing)
	switch {
	case err == nil:
		_, err = db.Exec(ctx,
			"UPDATE sync_meta SET value = ? WHERE key = 'oauth_refresh_token'", token)
		return err
	case errors.Is(err, sql.ErrNoRows):
		_, err = db.Exec(ctx,
			"INSERT INTO sync_meta(key, value) VALUES ('oauth_refresh_token', ?)", token)
		return err
	default:
		return err
	}
}

// ReadMSALCacheJSON lit le cache MSAL sérialisé depuis sync_meta du joueur.
// Retourne ("", nil) si la clé est absente (joueur sans token persisté).
func ReadMSALCacheJSON(ctx context.Context, db *DB) (string, error) {
	var cacheJSON string
	err := db.QueryRow(ctx, "SELECT value FROM sync_meta WHERE key = 'msal_token_cache'").Scan(&cacheJSON)
	if err != nil {
		return "", nil // absent ou erreur → pas de cache
	}
	return cacheJSON, nil
}

// ReadOAuthRefreshToken lit le refresh_token OAuth v2 depuis sync_meta du joueur.
// Clé utilisée : "oauth_refresh_token" (stockée par le sync Python legacy).
// Retourne ("", nil) si la clé est absente.
func ReadOAuthRefreshToken(ctx context.Context, db *DB) (string, error) {
	var token string
	err := db.QueryRow(ctx, "SELECT value FROM sync_meta WHERE key = 'oauth_refresh_token'").Scan(&token)
	if err != nil {
		return "", nil // absent → pas de token legacy
	}
	return token, nil
}
