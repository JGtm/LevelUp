// Package duckdb — queries_auth.go : DERNIER lecteur de l'ancien credential
// store DuckDB (sync_meta.oauth_refresh_token).
//
// ADR 0023 Phase 5 (2026-08-25) : la source unique des tokens auth est
// MultiUserTokenStore. Ce fichier ne survit QUE pour la migration one-shot du
// boot (auth.MigrateLegacyTokens, cf. cmd/server/main.go), qui recopie un RT
// resté en sync_meta vers le store. Aucun autre appelant n'est autorisé —
// garde-rail : internal/platform/auth/sentinel_test.go.
//
// KILL-SWITCH DATÉ : retrait prévu le 2026-10-01, critère « 0 token migré au
// boot sur 30 j de logs prod » (cf. internal/platform/auth/migration.go). Le
// drop physique de la colonne suit la recette ADR 0026 au prochain rebuild.
package duckdb

import (
	"context"
	"database/sql"
)

// readSyncMetaValue lit une valeur sync_meta par clé depuis un *sql.DB.
// Retourne ("", nil) si la clé est absente ou en cas d'erreur (best-effort).
func readSyncMetaValue(ctx context.Context, sqlDB *sql.DB, key string) (string, error) {
	var v string
	if err := sqlDB.QueryRowContext(ctx,
		"SELECT value FROM sync_meta WHERE key = ?", key).Scan(&v); err != nil {
		return "", nil
	}
	return v, nil
}

// ReadOAuthRefreshToken lit le refresh_token OAuth v2 legacy depuis sync_meta du
// joueur (clé "oauth_refresh_token"). Retourne ("", nil) si la clé est absente.
//
// USAGE UNIQUE AUTORISÉ : la migration boot-time ADR 0023 (voir en-tête du
// fichier). Tout autre chemin doit lire MultiUserTokenStore.
func ReadOAuthRefreshToken(ctx context.Context, db *DB) (string, error) {
	return readSyncMetaValue(ctx, db.SQLDb(), "oauth_refresh_token")
}
