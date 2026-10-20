package migration

// steps_player_purge_legacy_auth_keys.go — purge_sync_meta_legacy_auth_keys_v1 : retire de
// sync_meta (base joueur) les clés de l'ancien credential store, `oauth_refresh_token` et
// `msal_token_cache`.
//
// Pourquoi. Depuis la phase 5 de l'ADR 0023, la seule source des jetons est
// data/auth/watcher_tokens/{xuid}.json (MultiUserTokenStore) ; aucun code ne lit plus ces clés
// (ratchets auth/sentinel_test.go, sync/no_legacy_source_used_test.go). Leurs valeurs restaient
// pourtant dans les bases joueur : un refresh token résiduel copiable avec la base (sauvegarde,
// extraction) sans aucun usage.
//
// Remède SANS DELETE (la clé primaire `key` est un index ART, #23645) : swapKeepingRows
// reconstruit sync_meta à l'identique sans ces lignes, en une transaction. Idempotent : sans
// ces clés, aucune écriture. Aucune valeur n'est lue ni journalisée, seul le nombre de clés
// retirées l'est. Joué par la chaîne de migrations au démarrage, avant tout écrivain de
// sync_meta, comme les autres steps qui y écrivent.

import (
	"database/sql"
	"fmt"
	"log/slog"
)

// syncMetaLegacyAuthKeep : les lignes gardées — tout sauf les deux clés de credential retirées.
const syncMetaLegacyAuthKeep = `WHERE key IS NULL OR key NOT IN ('oauth_refresh_token', 'msal_token_cache')`

func applyPurgeSyncMetaLegacyAuthKeys(db *sql.DB) error {
	ctx := bootCtx()
	if err := recoverOrphanTable(ctx, db, "sync_meta", purgeSuffix); err != nil {
		return err
	}
	if ok, err := tableExists(db, "sync_meta"); err != nil || !ok {
		return err
	}
	var legacy int64
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sync_meta WHERE key IN ('oauth_refresh_token', 'msal_token_cache')`).Scan(&legacy); err != nil {
		return fmt.Errorf("sync_meta: compte des clés de credential héritées: %w", err)
	}
	if legacy == 0 {
		return nil
	}
	if _, err := swapKeepingRows(ctx, db, "sync_meta", "", syncMetaLegacyAuthKeep); err != nil {
		return err
	}
	slog.WarnContext(ctx, "migration: clés de credential héritées retirées de sync_meta (ADR 0023)",
		"removed_keys", legacy)
	return nil
}
