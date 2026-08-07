//go:build integration

// shared_social_dropidx_test.go — déplacé depuis internal/migration (Phase 1.5 b24).
// drop_idx_pn_xuid_unread (title-owned) : résolu via StepsFor(TargetSharedSocial) et appliqué
// directement (pas de RunForDB → provider non requis).
package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func ssIndexExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duckdb_indexes() WHERE index_name = ?`, name).Scan(&cnt); err != nil {
		t.Fatalf("indexExists(%q): %v", name, err)
	}
	return cnt > 0
}

func dropIdxPnMig(t *testing.T) *migration.Migration {
	t.Helper()
	steps := StepsFor(migration.TargetSharedSocial)
	for i := range steps {
		if steps[i].Name == "drop_idx_pn_xuid_unread" {
			return &steps[i]
		}
	}
	t.Fatal("migration drop_idx_pn_xuid_unread introuvable dans StepsFor(TargetSharedSocial)")
	return nil
}

func openSSMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestMigration_DropIdxPnXuidUnread_OnLegacyDB(t *testing.T) {
	db := openSSMemDB(t)

	if _, err := db.Exec(`
		CREATE TABLE player_notifications (
			xuid       VARCHAR NOT NULL,
			id         BIGINT NOT NULL,
			category   VARCHAR NOT NULL,
			severity   VARCHAR NOT NULL DEFAULT 'info',
			title_key  VARCHAR NOT NULL,
			source     VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			read_at    TIMESTAMP,
			PRIMARY KEY (xuid, id)
		);
		CREATE INDEX idx_pn_xuid_unread ON player_notifications(xuid, read_at);
	`); err != nil {
		t.Fatalf("seed legacy: %v", err)
	}
	if !ssIndexExists(t, db, "idx_pn_xuid_unread") {
		t.Fatal("seed legacy : idx_pn_xuid_unread devrait être créé")
	}

	drop := dropIdxPnMig(t)
	if err := drop.ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema drop_idx_pn_xuid_unread: %v", err)
	}
	if ssIndexExists(t, db, "idx_pn_xuid_unread") {
		t.Error("idx_pn_xuid_unread devrait être supprimé après ApplySchema")
	}

	if err := drop.ApplySchema(db); err != nil {
		t.Errorf("ApplySchema (2e passe, index déjà absent) doit être idempotent: %v", err)
	}
}

func TestMigration_DropIdxPnXuidUnread_OnFreshDB(t *testing.T) {
	db := openSSMemDB(t)

	if _, err := db.Exec(`
		CREATE TABLE player_notifications (
			xuid VARCHAR NOT NULL,
			id   BIGINT NOT NULL,
			PRIMARY KEY (xuid, id)
		);
	`); err != nil {
		t.Fatalf("seed fresh: %v", err)
	}

	drop := dropIdxPnMig(t)
	if err := drop.ApplySchema(db); err != nil {
		t.Errorf("ApplySchema doit être silencieuse sur DB sans index: %v", err)
	}
}
