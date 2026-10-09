package migration

// steps_player_purge_legacy_auth_keys_test.go — purge_sync_meta_legacy_auth_keys_v1 retire les
// deux clés de credential héritées, garde toutes les autres et le schéma.

import (
	"testing"
)

func TestPurgeSyncMetaLegacyAuthKeys(t *testing.T) {
	db := openRepairIDsDB(t)
	execRepairIDs(t, db,
		`CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR,
			updated_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`,
		`INSERT INTO sync_meta (key, value) VALUES
			('xuid', '2533274000000000'), ('last_sync_at', '2026-10-08T00:00:00Z'),
			('oauth_refresh_token', 'factice'), ('msal_token_cache', 'factice')`,
	)
	ddl, err := ddlDeTable(t.Context(), db, "sync_meta")
	if err != nil {
		t.Fatal(err)
	}

	if err := applyPurgeSyncMetaLegacyAuthKeys(db); err != nil {
		t.Fatalf("purge: %v", err)
	}

	if n := scanInt(t, db, `SELECT COUNT(*) FROM sync_meta WHERE key IN ('oauth_refresh_token', 'msal_token_cache')`); n != 0 {
		t.Errorf("%d clé(s) de credential restantes", n)
	}
	var xuid string
	if err := db.QueryRow(`SELECT value FROM sync_meta WHERE key = 'xuid'`).Scan(&xuid); err != nil || xuid != "2533274000000000" {
		t.Errorf("clé xuid perdue ou modifiée : %q (%v)", xuid, err)
	}
	if n := scanInt(t, db, `SELECT COUNT(*) FROM sync_meta`); n != 2 {
		t.Errorf("lignes gardées = %d, attendu 2", n)
	}
	if err := verifierSchemaIdentique(t.Context(), db, "sync_meta", ddl, nil); err != nil {
		t.Errorf("schéma modifié : %v", err)
	}
	// Clé primaire et DEFAULT intacts : l'écriture courante (SELECT-then-INSERT) marche.
	execRepairIDs(t, db, `INSERT INTO sync_meta (key, value) VALUES ('spnkr_version', '1')`)
	if _, err := db.Exec(`INSERT INTO sync_meta (key, value) VALUES ('xuid', 'x')`); err == nil {
		t.Error("clé primaire non reposée : doublon de clé accepté")
	}
	if err := applyPurgeSyncMetaLegacyAuthKeys(db); err != nil {
		t.Errorf("second passage: %v", err)
	}
}

func TestPurgeSyncMetaLegacyAuthKeys_TableAbsente(t *testing.T) {
	db := openRepairIDsDB(t)
	if err := applyPurgeSyncMetaLegacyAuthKeys(db); err != nil {
		t.Errorf("table absente : %v, attendu no-op", err)
	}
}
