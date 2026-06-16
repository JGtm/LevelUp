//go:build integration

// Package migration — steps_player_notifications_test.go : tests dédiés à la
// suite TargetSharedSocial avec un focus sur la migration de drop d'index
// `drop_idx_pn_xuid_unread` (2026-05-15).
//
// Tag `integration` car ces tests nécessitent le driver DuckDB (CGO).

package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// indexExists vérifie via la table système duckdb_indexes() qu'un index est
// présent. DuckDB n'expose pas les indexes via information_schema standard.
func indexExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var cnt int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM duckdb_indexes() WHERE index_name = ?`, name,
	).Scan(&cnt)
	if err != nil {
		t.Fatalf("indexExists(%q): %v", name, err)
	}
	return cnt > 0
}

// TestRunForDB_SharedSocial_CreatesNotificationsTables vérifie que la suite
// TargetSharedSocial crée bien les tables notifications (sanity check sur
// le pipeline avant de tester le drop d'index).
func TestRunForDB_SharedSocial_CreatesNotificationsTables(t *testing.T) {
	db := openMemDB(t)

	if !sharedSocialBaseIsGlobal() {
		t.Skip("schémas shared_social title-owned (Phase 1.5 b24) — couverture : TestTitleStepsRunEndToEnd_SharedSocial")
	}

	if err := RunForDB(db, TargetSharedSocial); err != nil {
		t.Fatalf("RunForDB(SharedSocial): %v", err)
	}

	tables := []string{
		"player_notifications",
		"notification_preferences",
		"player_records",
	}
	for _, tbl := range tables {
		if !assertTableExists(t, db, tbl) {
			t.Errorf("table %q absente après migration shared_social", tbl)
		}
	}
}

// TestRunForDB_SharedSocial_DropsIdxPnXuidUnread vérifie que la migration
// `drop_idx_pn_xuid_unread` retire l'index ART (xuid, read_at) tout en
// laissant les 2 autres index secondaires en place.
func TestRunForDB_SharedSocial_DropsIdxPnXuidUnread(t *testing.T) {
	db := openMemDB(t)

	if !sharedSocialBaseIsGlobal() {
		t.Skip("schémas shared_social title-owned (Phase 1.5 b24) — couverture : TestTitleStepsRunEndToEnd_SharedSocial")
	}

	if err := RunForDB(db, TargetSharedSocial); err != nil {
		t.Fatalf("RunForDB(SharedSocial): %v", err)
	}

	if indexExists(t, db, "idx_pn_xuid_unread") {
		t.Error("idx_pn_xuid_unread devrait avoir été supprimé par la migration drop_idx_pn_xuid_unread")
	}

	// Les 2 autres indexes (colonnes NOT NULL) doivent rester.
	for _, idx := range []string{"idx_pn_xuid_created_desc", "idx_pn_xuid_category"} {
		if !indexExists(t, db, idx) {
			t.Errorf("index %q devrait être conservé (colonne NOT NULL, pas concerné par le bug ART/NULL)", idx)
		}
	}
}

// TestRunForDB_SharedSocial_Idempotent vérifie l'idempotence de la suite
// TargetSharedSocial complète sur 3 passes (incluant drop_idx_pn_xuid_unread).
func TestRunForDB_SharedSocial_Idempotent(t *testing.T) {
	db := openMemDB(t)

	if !sharedSocialBaseIsGlobal() {
		t.Skip("schémas shared_social title-owned (Phase 1.5 b24) — couverture : TestTitleStepsRunEndToEnd_SharedSocial")
	}

	for pass := 1; pass <= 3; pass++ {
		if err := RunForDB(db, TargetSharedSocial); err != nil {
			t.Fatalf("RunForDB(SharedSocial) passe %d: %v", pass, err)
		}
	}

	var notDone int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM schema_migrations WHERE schema_done = FALSE",
	).Scan(&notDone); err != nil {
		t.Fatalf("query schema_done: %v", err)
	}
	if notDone > 0 {
		t.Errorf("%d migration(s) shared_social avec schema_done=FALSE après 3 passes", notDone)
	}

	// Vérification spécifique : la migration drop_idx est appliquée.
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations
		 WHERE name = 'drop_idx_pn_xuid_unread' AND schema_done = TRUE`,
	).Scan(&n); err != nil {
		t.Fatalf("query drop_idx_pn_xuid_unread: %v", err)
	}
	if n != 1 {
		t.Errorf("migration drop_idx_pn_xuid_unread non appliquée (count=%d)", n)
	}
}

// TestMigration_DropIdxPnXuidUnread_OnLegacyDB + _OnFreshDB ont été déplacés vers
// internal/games/halo_infinite/migrations/shared_social_dropidx_test.go (Phase 1.5 b24) :
// drop_idx_pn_xuid_unread est title-owned, résolu via StepsFor(TargetSharedSocial).
