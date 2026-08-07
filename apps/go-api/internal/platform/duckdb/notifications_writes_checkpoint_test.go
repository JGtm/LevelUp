//go:build cgo

// Package duckdb_test — notifications_writes_checkpoint_test.go (ADR 0022).
//
// Valide que les mutations notifications (mark-read, mark-all, delete) FLUSHENT
// le WAL via le CHECKPOINT immédiat des méthodes SocialPersister vers lesquelles
// NotificationsRepo route désormais.
//
// Oracle = TAILLE DU FICHIER WAL. Un CHECKPOINT écrit la mutation dans le
// .duckdb et tronque le .wal ; une écriture NON-checkpointée reste dans le .wal.
// C'est exactement ce .wal résiduel qui est perdu en prod quand la recovery
// #7659 quarantine un WAL orphelin au restart → « la notif marquée lue revient ».
//
// NB : un reopen normal REJOUE le WAL (donc ne discrimine pas) ; seule la
// quarantaine d'un WAL orphelin perd la donnée. La taille du WAL est l'oracle
// fidèle et déterministe du « CHECKPOINT a bien eu lieu ».
//
// Nommage `TestSet...PersistsAfterReopen` : matche la regex du gate CI
// (shared-social-gate.yml, `-run 'TestSet.*PersistsAfter'`) → ces tests gatent.
package duckdb_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb"
)

// newNotifSocialDB crée une shared_social.duckdb fraîche avec le schéma notif et
// un CHECKPOINT initial (DDL flushé hors WAL).
func newNotifSocialDB(t *testing.T) *duckdb.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "shared_social.duckdb")
	db, err := duckdb.OpenReadWriteShared(path, "")
	if err != nil {
		t.Fatalf("OpenReadWriteShared: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ddl := `
		CREATE TABLE IF NOT EXISTS player_notifications (
			xuid VARCHAR NOT NULL, id BIGINT NOT NULL, category VARCHAR NOT NULL,
			severity VARCHAR NOT NULL DEFAULT 'info', title_key VARCHAR NOT NULL,
			body_key VARCHAR, params VARCHAR, target_route VARCHAR, target_search VARCHAR,
			actor_xuid VARCHAR, actor_name VARCHAR, source VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, read_at TIMESTAMP,
			PRIMARY KEY (xuid, id)
		);
		CREATE SEQUENCE IF NOT EXISTS player_notifications_history_seq START 1;
		CREATE TABLE IF NOT EXISTS player_notifications_history (
			seq BIGINT PRIMARY KEY DEFAULT nextval('player_notifications_history_seq'),
			xuid VARCHAR NOT NULL, id BIGINT NOT NULL, category VARCHAR NOT NULL,
			severity VARCHAR NOT NULL DEFAULT 'info', title_key VARCHAR NOT NULL,
			body_key VARCHAR, params VARCHAR, target_route VARCHAR, target_search VARCHAR,
			actor_xuid VARCHAR, actor_name VARCHAR, source VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL, read_at TIMESTAMP,
			is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE OR REPLACE VIEW player_notifications_latest AS
			SELECT xuid, id, category, severity, title_key, body_key, params,
			       target_route, target_search, actor_xuid, actor_name, source,
			       created_at, read_at, written_at
			FROM (SELECT *, ROW_NUMBER() OVER (PARTITION BY xuid, id
				ORDER BY written_at DESC, seq DESC) AS rn
				FROM player_notifications_history) ranked
			WHERE rn = 1 AND is_deleted = FALSE;
		CREATE TABLE IF NOT EXISTS notification_preferences (
			xuid VARCHAR NOT NULL, category VARCHAR NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT TRUE, delivery VARCHAR NOT NULL DEFAULT 'both',
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (xuid, category)
		);
	`
	if _, err := db.SQLDb().Exec(ddl); err != nil {
		t.Fatalf("seed notif ddl: %v", err)
	}
	if _, err := db.SQLDb().Exec("CHECKPOINT"); err != nil {
		t.Fatalf("checkpoint initial: %v", err)
	}
	return db
}

// seedFlushedNotif insère une notif non-lue puis CHECKPOINT (la rend durable et
// remet le WAL à ~0 pour isoler la durabilité de la MUTATION testée ensuite).
func seedFlushedNotif(t *testing.T, db *duckdb.DB, id int64) {
	t.Helper()
	if _, err := db.SQLDb().Exec(`
		INSERT INTO player_notifications_history
			(xuid, id, category, severity, title_key, source, created_at, read_at, is_deleted, written_at)
		VALUES ('x', ?, 'c', 'info', 'k', 's', NOW(), NULL, FALSE, NOW())`, id); err != nil {
		t.Fatalf("seed notif id=%d: %v", id, err)
	}
	if _, err := db.SQLDb().Exec("CHECKPOINT"); err != nil {
		t.Fatalf("seed checkpoint: %v", err)
	}
}

// walSize retourne la taille du fichier .wal (0 si absent). Oracle de flush.
func walSize(t *testing.T, dbPath string) int64 {
	t.Helper()
	info, err := os.Stat(dbPath + ".wal")
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatalf("stat wal: %v", err)
	}
	return info.Size()
}

// reopenNotifCount close la DB et la rouvre en READ_ONLY (sanity : la donnée
// est bien lisible). N'est PAS l'oracle de durabilité (un reopen rejoue le WAL).
func reopenNotifCount(t *testing.T, db *duckdb.DB, query string, args ...any) int64 {
	t.Helper()
	path := db.Path()
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	roDB, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
	if err != nil {
		t.Fatalf("reopen RO: %v", err)
	}
	defer roDB.Close()
	var n int64
	if err := roDB.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return n
}

func newNotifRepo(t *testing.T, socialDB *duckdb.DB) *duckdb.NotificationsRepo {
	t.Helper()
	pdb := &duckdb.PlayerDB{
		SharedSocial:    socialDB,
		SocialPersister: persist.NewSharedSocialPersister(socialDB.SQLDb()),
		XUID:            "x",
	}
	return duckdb.NewNotificationsRepo(pdb)
}

// TestSetNotificationRead_PersistsAfterReopen : MarkRead via Persister
// (CHECKPOINT immédiat) → WAL flushé (~0) + donnée lisible.
func TestSetNotificationRead_PersistsAfterReopen(t *testing.T) {
	socialDB := newNotifSocialDB(t)
	seedFlushedNotif(t, socialDB, 1)
	repo := newNotifRepo(t, socialDB)

	n, err := repo.MarkRead(context.Background(), []int64{1})
	if err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if n != 1 {
		t.Fatalf("MarkRead n=%d want 1", n)
	}
	if sz := walSize(t, socialDB.Path()); sz != 0 {
		t.Errorf("WAL non flushé après MarkRead (size=%d want 0) — CHECKPOINT manquant", sz)
	}
	got := reopenNotifCount(t, socialDB,
		`SELECT COUNT(*) FROM player_notifications_latest WHERE xuid='x' AND id=1 AND read_at IS NOT NULL`)
	if got != 1 {
		t.Errorf("read_at absent après reopen (got %d want 1)", got)
	}
}

// TestSetNotificationMarkAllRead_PersistsAfterReopen : MarkAllRead → WAL flushé.
func TestSetNotificationMarkAllRead_PersistsAfterReopen(t *testing.T) {
	socialDB := newNotifSocialDB(t)
	seedFlushedNotif(t, socialDB, 1)
	seedFlushedNotif(t, socialDB, 2)
	repo := newNotifRepo(t, socialDB)

	n, err := repo.MarkAllRead(context.Background(), "")
	if err != nil {
		t.Fatalf("MarkAllRead: %v", err)
	}
	if n != 2 {
		t.Fatalf("MarkAllRead n=%d want 2", n)
	}
	if sz := walSize(t, socialDB.Path()); sz != 0 {
		t.Errorf("WAL non flushé après MarkAllRead (size=%d want 0)", sz)
	}
}

// TestSetNotificationDelete_PersistsAfterReopen : Delete via Persister → WAL flushé.
func TestSetNotificationDelete_PersistsAfterReopen(t *testing.T) {
	socialDB := newNotifSocialDB(t)
	seedFlushedNotif(t, socialDB, 1)
	repo := newNotifRepo(t, socialDB)

	if err := repo.Delete(context.Background(), 1); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if sz := walSize(t, socialDB.Path()); sz != 0 {
		t.Errorf("WAL non flushé après Delete (size=%d want 0)", sz)
	}
	got := reopenNotifCount(t, socialDB,
		`SELECT COUNT(*) FROM player_notifications_latest WHERE xuid='x' AND id=1`)
	if got != 0 {
		t.Errorf("notif présente après Delete + reopen (got %d want 0)", got)
	}
}

// TestNotificationWrite_LegacyNoCheckpoint_LeavesWAL : prouve que l'oracle WAL
// discrimine — un event INSERT SANS CHECKPOINT (hors chemin Persister) laisse le
// WAL non-vide (donnée à risque de quarantaine #7659). Si ce test échoue (WAL==0),
// l'oracle ne discrimine pas et les tests positifs ne garantissent rien.
func TestNotificationWrite_LegacyNoCheckpoint_LeavesWAL(t *testing.T) {
	socialDB := newNotifSocialDB(t)
	seedFlushedNotif(t, socialDB, 1)

	// Écriture append-only SANS CHECKPOINT (event INSERT hors Persister).
	if _, err := socialDB.SQLDb().Exec(`
		INSERT INTO player_notifications_history
			(xuid, id, category, severity, title_key, source, created_at, read_at, is_deleted, written_at)
		VALUES ('x', 1, 'c', 'info', 'k', 's', NOW(), NOW(), FALSE, NOW())`); err != nil {
		t.Fatalf("insert sans checkpoint: %v", err)
	}
	if sz := walSize(t, socialDB.Path()); sz == 0 {
		t.Errorf("WAL vide après INSERT non-checkpointé (size=0) — l'oracle WAL ne discrimine pas")
	}
}
