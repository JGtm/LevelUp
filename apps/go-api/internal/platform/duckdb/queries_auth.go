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

// readSyncMetaValue lit une valeur sync_meta par clé depuis un *sql.DB. Source
// UNIQUE des lectures auth sync_meta (variantes *DB et *sql.DB délèguent ici).
// Retourne ("", nil) si la clé est absente ou en cas d'erreur (best-effort).
func readSyncMetaValue(ctx context.Context, sqlDB *sql.DB, key string) (string, error) {
	var v string
	if err := sqlDB.QueryRowContext(ctx,
		"SELECT value FROM sync_meta WHERE key = ?", key).Scan(&v); err != nil {
		return "", nil
	}
	return v, nil
}

// ReadMSALCacheJSON lit le cache MSAL sérialisé depuis sync_meta du joueur.
// Retourne ("", nil) si la clé est absente (joueur sans token persisté).
func ReadMSALCacheJSON(ctx context.Context, db *DB) (string, error) {
	return readSyncMetaValue(ctx, db.SQLDb(), "msal_token_cache")
}

// ReadMSALCacheJSONFromSQL est la variante *sql.DB : pour les lecteurs ouverts via
// OpenReadForQuery (qui retourne un *sql.DB réutilisant un handle en cache, JAMAIS
// un OpenReadOnly nu qui doublerait l'ouverture d'une DB tenue RW — E6, ADR 0016).
func ReadMSALCacheJSONFromSQL(ctx context.Context, sqlDB *sql.DB) (string, error) {
	return readSyncMetaValue(ctx, sqlDB, "msal_token_cache")
}

// ReadOAuthRefreshToken lit le refresh_token OAuth v2 depuis sync_meta du joueur.
// Clé utilisée : "oauth_refresh_token" (stockée par le sync Python legacy).
// Retourne ("", nil) si la clé est absente.
func ReadOAuthRefreshToken(ctx context.Context, db *DB) (string, error) {
	return readSyncMetaValue(ctx, db.SQLDb(), "oauth_refresh_token")
}

// ReadOAuthRefreshTokenFromSQL est la variante *sql.DB (cf. ReadMSALCacheJSONFromSQL, E6).
func ReadOAuthRefreshTokenFromSQL(ctx context.Context, sqlDB *sql.DB) (string, error) {
	return readSyncMetaValue(ctx, sqlDB, "oauth_refresh_token")
}
