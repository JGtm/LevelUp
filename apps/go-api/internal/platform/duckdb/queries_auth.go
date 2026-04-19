// Package duckdb — queries_auth.go : lecture du cache MSAL depuis sync_meta.
package duckdb

import "context"

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
