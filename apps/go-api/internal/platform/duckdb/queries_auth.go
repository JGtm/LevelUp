// Package duckdb — queries_auth.go : lecture du cache auth depuis sync_meta.
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
