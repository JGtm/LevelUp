//go:build cgo

// Package persist — tests SharedSocialPersister sur DuckDB réel.
//
// Couvre :
//   - Persist d'un batch avec toutes les tables (sanity full path)
//   - CHECKPOINT garanti (WAL vide après Persist)
//   - Atomicité : si 1 INSERT échoue, rollback complet
//   - IsEmpty → no-op (pas de TX inutile)
//   - Persist batch nil → no-op

package persist

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupSocialDB crée une shared_social.duckdb temporaire avec le schéma de
// base (migrations équivalentes appliquées en SQL inline).
func setupSocialDB(t *testing.T) (string, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "shared_social.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := []string{
		`CREATE SEQUENCE IF NOT EXISTS media_files_id_seq START 1`,
		`CREATE TABLE media_files (
			id INTEGER PRIMARY KEY DEFAULT nextval('media_files_id_seq'),
			player_slug VARCHAR,
			file_path VARCHAR UNIQUE,
			file_name VARCHAR,
			file_stem VARCHAR,
			file_ext VARCHAR,
			file_hash VARCHAR,
			kind VARCHAR,
			thumbnail_path VARCHAR,
			capture_start_utc TIMESTAMPTZ,
			capture_end_utc TIMESTAMPTZ,
			duration_seconds DOUBLE,
			status VARCHAR,
			mtime TIMESTAMPTZ,
			discord_notified BOOLEAN DEFAULT FALSE,
			indexed_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE media_match_associations (
			media_file_id INTEGER,
			match_id VARCHAR,
			delta_seconds INTEGER,
			PRIMARY KEY (media_file_id, match_id)
		)`,
		// Append-only (cf. shared_social_media_assoc_append_only_v1).
		`CREATE SEQUENCE IF NOT EXISTS media_match_associations_history_id_seq START 1`,
		`CREATE TABLE media_match_associations_history (
			id BIGINT PRIMARY KEY DEFAULT nextval('media_match_associations_history_id_seq'),
			media_file_id BIGINT NOT NULL, match_id VARCHAR NOT NULL, delta_seconds INTEGER,
			is_manual BOOLEAN NOT NULL DEFAULT FALSE, is_active BOOLEAN NOT NULL DEFAULT TRUE,
			associated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		`CREATE OR REPLACE VIEW media_match_associations_latest AS
			WITH lpp AS (
				SELECT media_file_id, match_id, delta_seconds, is_manual, is_active, associated_at, written_at,
					ROW_NUMBER() OVER (PARTITION BY media_file_id, match_id ORDER BY written_at DESC, id DESC) AS rn
				FROM media_match_associations_history),
			act AS (SELECT * FROM lpp WHERE rn = 1 AND is_active = TRUE),
			hm AS (SELECT media_file_id, bool_or(is_manual) AS has_manual FROM act GROUP BY media_file_id)
			SELECT a.media_file_id, a.match_id, a.delta_seconds, a.is_manual, a.associated_at, a.written_at
			FROM act a JOIN hm ON hm.media_file_id = a.media_file_id
			WHERE a.is_manual = hm.has_manual`,
		`CREATE TABLE media_likes (
			media_path VARCHAR NOT NULL,
			liker_slug VARCHAR NOT NULL,
			liker_gamertag VARCHAR,
			liked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (media_path, liker_slug)
		)`,
		// Append-only (cf. shared_social_likes_append_only_v1).
		`CREATE SEQUENCE IF NOT EXISTS media_likes_history_id_seq START 1`,
		`CREATE TABLE media_likes_history (
			id BIGINT PRIMARY KEY DEFAULT nextval('media_likes_history_id_seq'),
			media_path VARCHAR NOT NULL, liker_slug VARCHAR NOT NULL, liker_gamertag VARCHAR,
			is_liked BOOLEAN NOT NULL, liked_at TIMESTAMP,
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		`CREATE OR REPLACE VIEW media_likes_latest AS
			SELECT id, media_path, liker_slug, liker_gamertag, is_liked, liked_at, written_at
			FROM media_likes_history
			QUALIFY ROW_NUMBER() OVER (PARTITION BY media_path, liker_slug
				ORDER BY written_at DESC, id DESC) = 1`,
		`CREATE TABLE match_favorites (
			player_slug VARCHAR NOT NULL,
			match_id VARCHAR NOT NULL,
			favorited_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (player_slug, match_id)
		)`,
		// Append-only (cf. shared_social_favorites_append_only_v1) : persistFavorites
		// écrit ici, l'état courant se lit via match_favorites_latest.
		`CREATE SEQUENCE IF NOT EXISTS match_favorites_history_id_seq START 1`,
		`CREATE TABLE match_favorites_history (
			id BIGINT PRIMARY KEY DEFAULT nextval('match_favorites_history_id_seq'),
			player_slug VARCHAR NOT NULL, match_id VARCHAR NOT NULL,
			is_favorite BOOLEAN NOT NULL, favorited_at TIMESTAMP,
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		`CREATE OR REPLACE VIEW match_favorites_latest AS
			SELECT id, player_slug, match_id, is_favorite, favorited_at, written_at
			FROM match_favorites_history
			QUALIFY ROW_NUMBER() OVER (PARTITION BY player_slug, match_id
				ORDER BY written_at DESC, id DESC) = 1`,
		`CREATE TABLE player_notifications (
			xuid VARCHAR NOT NULL,
			id BIGINT NOT NULL,
			category VARCHAR NOT NULL,
			severity VARCHAR NOT NULL DEFAULT 'info',
			title_key VARCHAR NOT NULL,
			body_key VARCHAR,
			params VARCHAR,
			target_route VARCHAR,
			target_search VARCHAR,
			actor_xuid VARCHAR,
			actor_name VARCHAR,
			source VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			read_at TIMESTAMP,
			PRIMARY KEY (xuid, id)
		)`,
		// Append-only (cf. shared_social_notifications_append_only_v1) : les writers
		// notifs écrivent ici (event-log), l'état courant se lit via _latest.
		`CREATE SEQUENCE IF NOT EXISTS player_notifications_history_seq START 1`,
		`CREATE TABLE player_notifications_history (
			seq BIGINT PRIMARY KEY DEFAULT nextval('player_notifications_history_seq'),
			xuid VARCHAR NOT NULL, id BIGINT NOT NULL,
			category VARCHAR NOT NULL, severity VARCHAR NOT NULL DEFAULT 'info',
			title_key VARCHAR NOT NULL, body_key VARCHAR, params VARCHAR,
			target_route VARCHAR, target_search VARCHAR,
			actor_xuid VARCHAR, actor_name VARCHAR, source VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL, read_at TIMESTAMP,
			is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		`CREATE OR REPLACE VIEW player_notifications_latest AS
			SELECT xuid, id, category, severity, title_key, body_key, params,
			       target_route, target_search, actor_xuid, actor_name, source,
			       created_at, read_at, written_at
			FROM (
				SELECT *, ROW_NUMBER() OVER (PARTITION BY xuid, id
					ORDER BY written_at DESC, seq DESC) AS rn
				FROM player_notifications_history
			) ranked
			WHERE rn = 1 AND is_deleted = FALSE`,
		`CREATE TABLE notification_preferences (
			xuid VARCHAR NOT NULL,
			category VARCHAR NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			delivery VARCHAR NOT NULL DEFAULT 'both',
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (xuid, category)
		)`,
		// Append-only (cf. shared_social_notif_prefs_append_only_v1).
		`CREATE SEQUENCE IF NOT EXISTS notification_preferences_history_id_seq START 1`,
		`CREATE TABLE notification_preferences_history (
			id BIGINT PRIMARY KEY DEFAULT nextval('notification_preferences_history_id_seq'),
			xuid VARCHAR NOT NULL, category VARCHAR NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT TRUE, delivery VARCHAR NOT NULL DEFAULT 'both',
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		`CREATE OR REPLACE VIEW notification_preferences_latest AS
			SELECT id, xuid, category, enabled, delivery, updated_at, written_at
			FROM notification_preferences_history
			QUALIFY ROW_NUMBER() OVER (PARTITION BY xuid, category
				ORDER BY written_at DESC, id DESC) = 1`,
		// Pour le test, on utilise la table _legacy player_records (sans _history)
		// pour valider le fallback. La migration Phase 2 introduira _history.
		`CREATE TABLE player_records (
			xuid VARCHAR NOT NULL,
			metric VARCHAR NOT NULL,
			value DOUBLE NOT NULL,
			achieved_at TIMESTAMP,
			achieved_match_id VARCHAR,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (xuid, metric)
		)`,
	}
	for _, sql := range schema {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("schema setup: %v\nSQL: %s", err, sql)
		}
	}
	return path, db
}

// TestSharedSocialPersister_PlayerRecordsHistory_NominalPath valide le chemin
// NOMINAL append-only (player_records_history) — distinct du fallback legacy
// couvert par setupSocialDB. Vérifie le binding des 9 colonnes incluant
// previous_value/previous_achieved_at ajoutées au fix 2026-05-30, relues via la
// vue player_records_latest.
func TestSharedSocialPersister_PlayerRecordsHistory_NominalPath(t *testing.T) {
	_, db := setupSocialDB(t)
	for _, ddl := range []string{
		`CREATE SEQUENCE IF NOT EXISTS player_records_history_id_seq START 1`,
		`CREATE TABLE player_records_history (
			id BIGINT PRIMARY KEY DEFAULT nextval('player_records_history_id_seq'),
			xuid VARCHAR NOT NULL, metric VARCHAR NOT NULL,
			period VARCHAR NOT NULL DEFAULT 'all_time', value DOUBLE NOT NULL,
			achieved_at TIMESTAMP, achieved_match_id VARCHAR,
			previous_value DOUBLE, previous_achieved_at TIMESTAMP,
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		`CREATE OR REPLACE VIEW player_records_latest AS
			SELECT DISTINCT ON (xuid, metric, period)
				id, xuid, metric, period, value, achieved_at, achieved_match_id,
				previous_value, previous_achieved_at, written_at
			FROM player_records_history
			ORDER BY xuid, metric, period, written_at DESC, id DESC`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("setup _history: %v", err)
		}
	}

	p := NewSharedSocialPersister(db)
	ctx := context.Background()
	achieved := time.Now().UTC().Add(-time.Hour)
	prevAt := time.Now().UTC().Add(-48 * time.Hour)
	prev := 3.0
	mid := "match-x"
	if err := p.AppendPlayerRecord(ctx, "xuid-1", "kda_best", "all_time", 4.5, &achieved, &mid, &prev, &prevAt); err != nil {
		t.Fatalf("AppendPlayerRecord: %v", err)
	}

	var value, pv sql.NullFloat64
	var pat sql.NullTime
	var matchID sql.NullString
	if err := db.QueryRow(`
		SELECT value, previous_value, previous_achieved_at, achieved_match_id
		FROM player_records_latest WHERE xuid = 'xuid-1' AND metric = 'kda_best' AND period = 'all_time'
	`).Scan(&value, &pv, &pat, &matchID); err != nil {
		t.Fatalf("read latest: %v", err)
	}
	if value.Float64 != 4.5 {
		t.Errorf("value=%v want 4.5", value.Float64)
	}
	if !pv.Valid || pv.Float64 != 3.0 {
		t.Errorf("previous_value=%v want 3.0", pv)
	}
	if !pat.Valid {
		t.Error("previous_achieved_at should be set")
	}
	if matchID.String != "match-x" {
		t.Errorf("achieved_match_id=%v want match-x", matchID.String)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// IsEmpty + nil + no-op
// ─────────────────────────────────────────────────────────────────────────────

func TestSharedSocialBatch_IsEmpty(t *testing.T) {
	b := &SharedSocialBatch{}
	if !b.IsEmpty() {
		t.Error("batch vide doit être IsEmpty=true")
	}

	b.MediaFiles = []MediaFileInsert{{PlayerSlug: "p"}}
	if b.IsEmpty() {
		t.Error("batch avec media_files ne doit pas être IsEmpty")
	}
}

func TestSharedSocialPersister_NilBatch_NoOp(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	if err := p.Persist(context.Background(), nil); err != nil {
		t.Errorf("Persist(nil) doit être no-op, got: %v", err)
	}
}

func TestSharedSocialPersister_EmptyBatch_NoOp(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	if err := p.Persist(context.Background(), &SharedSocialBatch{BatchID: "empty"}); err != nil {
		t.Errorf("Persist(empty) doit être no-op, got: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Full path : insert sur toutes les tables
// ─────────────────────────────────────────────────────────────────────────────

func TestSharedSocialPersister_FullBatch_PersistsAllTables(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()

	captureStart := time.Now().UTC().Add(-1 * time.Hour)
	captureEnd := captureStart.Add(5 * time.Minute)
	duration := 300.0
	achievedAt := time.Now().UTC().Add(-30 * time.Minute)
	matchID := "match-xyz"

	batch := &SharedSocialBatch{
		BatchID: "test-full",
		Source:  "unit_test",
		MediaFiles: []MediaFileInsert{
			{
				PlayerSlug:      "spartan",
				FilePath:        "/cap/clip.mp4",
				FileName:        "clip.mp4",
				FileStem:        "clip",
				FileExt:         ".mp4",
				FileHash:        "deadbeef",
				Kind:            "video",
				CaptureStartUTC: &captureStart,
				CaptureEndUTC:   &captureEnd,
				DurationSeconds: &duration,
			},
		},
		// MediaAssociations testé après recup id auto-incrémenté
		Likes: []LikeInsert{
			{MediaPath: "/cap/clip.mp4", LikerSlug: "friend1", LikerGamertag: "Friend1", LikedAt: time.Now().UTC()},
		},
		Favorites: []FavoriteInsert{
			{PlayerSlug: "spartan", MatchID: matchID, FavoritedAt: time.Now().UTC()},
		},
		Notifications: []NotificationInsert{
			{
				XUID:      "xuid-123",
				ID:        1,
				Category:  "milestone",
				Severity:  "success",
				TitleKey:  "milestone.records.kda_best",
				Source:    "test",
				CreatedAt: time.Now().UTC(),
			},
		},
		PlayerRecordsAppend: []PlayerRecordAppend{
			{
				XUID:            "xuid-123",
				Metric:          "kda_best",
				Period:          "all_time",
				Value:           3.5,
				AchievedAt:      &achievedAt,
				AchievedMatchID: &matchID,
				WrittenAt:       time.Now().UTC(),
			},
		},
	}

	if err := p.Persist(ctx, batch); err != nil {
		t.Fatalf("Persist full batch: %v", err)
	}

	// Vérifier chaque table.
	expectations := []struct {
		name  string
		query string
		want  int
	}{
		{"media_files", "SELECT COUNT(*) FROM media_files WHERE file_path = '/cap/clip.mp4'", 1},
		{"media_likes", "SELECT COUNT(*) FROM media_likes_latest WHERE liker_slug = 'friend1' AND is_liked = TRUE", 1},
		{"match_favorites", "SELECT COUNT(*) FROM match_favorites_latest WHERE player_slug = 'spartan' AND is_favorite = TRUE", 1},
		{"player_notifications", "SELECT COUNT(*) FROM player_notifications_latest WHERE xuid = 'xuid-123'", 1},
		{"player_records", "SELECT COUNT(*) FROM player_records WHERE xuid = 'xuid-123' AND metric = 'kda_best'", 1},
	}
	for _, e := range expectations {
		var count int
		if err := db.QueryRow(e.query).Scan(&count); err != nil {
			t.Errorf("%s query: %v", e.name, err)
			continue
		}
		if count != e.want {
			t.Errorf("%s: count=%d, want %d", e.name, count, e.want)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Multi-files : plusieurs media_files dans un seul batch
// (Le CHECKPOINT n'est plus lancé depuis Persist — il se fait au shutdown via
// main.go pour éviter de bloquer la seule connexion MaxOpenConns(1).)
// ─────────────────────────────────────────────────────────────────────────────

func TestSharedSocialPersister_MultipleMediaFiles_AllInserted(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()

	captureStart := time.Now().UTC()
	batch := &SharedSocialBatch{
		BatchID: "multi-files-test",
		Source:  "unit_test",
		MediaFiles: []MediaFileInsert{
			{PlayerSlug: "p1", FilePath: "/m1.mp4", FileName: "m1.mp4", FileHash: "h1", Kind: "video", CaptureStartUTC: &captureStart},
			{PlayerSlug: "p1", FilePath: "/m2.mp4", FileName: "m2.mp4", FileHash: "h2", Kind: "video", CaptureStartUTC: &captureStart},
			{PlayerSlug: "p1", FilePath: "/m3.mp4", FileName: "m3.mp4", FileHash: "h3", Kind: "video", CaptureStartUTC: &captureStart},
		},
	}
	if err := p.Persist(ctx, batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM media_files WHERE player_slug = 'p1'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("media_files count=%d, want 3", count)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Atomicité : 1 INSERT échoue → rollback complet
// ─────────────────────────────────────────────────────────────────────────────

func TestSharedSocialPersister_Atomicity_RollbackOnFailure(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()

	// Depuis l'append-only, plus aucune collision PK exploitable sur les tables
	// d'état (toutes en INSERT pur + seq auto). On force donc l'échec en fin de
	// batch : persistPlayerRecords est le DERNIER helper — on droppe player_records
	// avant le Persist → l'INSERT legacy lève "table does not exist" → la TX doit
	// rollback INTÉGRALEMENT, y compris media_files (partie 1) et la notification.
	if _, err := db.Exec(`DROP TABLE player_records`); err != nil {
		t.Fatalf("drop player_records: %v", err)
	}

	now := time.Now().UTC()
	batch := &SharedSocialBatch{
		BatchID: "rollback-test",
		Source:  "unit_test",
		MediaFiles: []MediaFileInsert{
			{PlayerSlug: "p", FilePath: "/rb.mp4", FileName: "rb.mp4", FileHash: "rb", Kind: "video"},
		},
		Notifications: []NotificationInsert{
			{XUID: "x1", ID: 100, Category: "c", Severity: "info", TitleKey: "t", Source: "s", CreatedAt: now},
		},
		PlayerRecordsAppend: []PlayerRecordAppend{
			{XUID: "x1", Metric: "kda_best", Value: 4.2, WrittenAt: now},
		},
	}

	err := p.Persist(ctx, batch)
	if err == nil {
		t.Fatal("Persist doit échouer (player_records droppé en cours de batch)")
	}

	// Vérifier que la media_file n'a PAS été persistée (rollback complet).
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM media_files WHERE file_path = '/rb.mp4'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("media_file persistée malgré rollback (count=%d, want 0)", count)
	}
	// Vérifier que la notif (event _history) n'a PAS été persistée non plus.
	if err := db.QueryRow("SELECT COUNT(*) FROM player_notifications_history WHERE xuid = 'x1' AND id = 100").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("notification persistée malgré rollback (count=%d, want 0)", count)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Idempotence : INSERT OR IGNORE silencieux sur doublons
// ─────────────────────────────────────────────────────────────────────────────

func TestSharedSocialPersister_InsertOrIgnore_NoErrorOnDuplicate(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()
	now := time.Now().UTC()

	batch := &SharedSocialBatch{
		BatchID: "idemp",
		Source:  "unit_test",
		Likes: []LikeInsert{
			{MediaPath: "/dup.mp4", LikerSlug: "u1", LikedAt: now},
		},
	}
	if err := p.Persist(ctx, batch); err != nil {
		t.Fatalf("1st Persist: %v", err)
	}
	// 2e Persist du même like (INSERT OR IGNORE) ne doit pas fail.
	if err := p.Persist(ctx, batch); err != nil {
		t.Fatalf("2nd Persist (duplicate): %v", err)
	}
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM media_likes_latest WHERE media_path = '/dup.mp4' AND is_liked = TRUE").Scan(&count)
	if count != 1 {
		t.Errorf("dedup raté : count=%d, want 1 (2 events TRUE → vue _latest = 1)", count)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// PersistBatch (interface duckdb.SocialPersister) — cast any → typed
// ─────────────────────────────────────────────────────────────────────────────

func TestSharedSocialPersister_PersistBatch_CastsCorrectly(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()

	// Cast OK
	batch := &SharedSocialBatch{
		BatchID: "iface-test", Source: "test",
		Likes: []LikeInsert{{MediaPath: "/i.mp4", LikerSlug: "u", LikedAt: time.Now().UTC()}},
	}
	if err := p.PersistBatch(ctx, batch); err != nil {
		t.Fatalf("PersistBatch typed: %v", err)
	}

	// Cast KO sur mauvais type
	if err := p.PersistBatch(ctx, "not a batch"); err == nil {
		t.Error("PersistBatch sur string doit échouer avec erreur de cast")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Thumbnails + Associations + Favorites (couverture helpers)
// ─────────────────────────────────────────────────────────────────────────────

func TestSharedSocialPersister_MediaThumbnails_UpdatesOnlyNullPath(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()

	// Insert un media file sans thumbnail
	captureAt := time.Now().UTC()
	_ = p.Persist(ctx, &SharedSocialBatch{
		BatchID: "th1", Source: "test",
		MediaFiles: []MediaFileInsert{
			{PlayerSlug: "p", FilePath: "/m.mp4", FileName: "m.mp4", FileHash: "hth",
				Kind: "video", CaptureStartUTC: &captureAt},
		},
	})
	// Récup id auto-généré
	var id int64
	if err := db.QueryRow(`SELECT id FROM media_files WHERE file_hash = 'hth'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	// Update thumbnail via Persister
	if err := p.Persist(ctx, &SharedSocialBatch{
		BatchID: "th2", Source: "test",
		MediaThumbnails: []MediaThumbnailUpdate{{MediaFileID: id, ThumbnailPath: "/thumbs/m.webp"}},
	}); err != nil {
		t.Fatal(err)
	}
	var thumb sql.NullString
	_ = db.QueryRow(`SELECT thumbnail_path FROM media_files WHERE id = ?`, id).Scan(&thumb)
	if !thumb.Valid || thumb.String != "/thumbs/m.webp" {
		t.Errorf("thumbnail non mis à jour : %v", thumb)
	}
}

func TestSharedSocialPersister_MediaAssociations_InsertOrIgnore(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()

	// Insert 2 associations dont 1 doublon (même PK media_file_id+match_id)
	batch := &SharedSocialBatch{
		BatchID: "assoc1", Source: "test",
		MediaAssociations: []MediaAssociationInsert{
			{MediaFileID: 1, MatchID: "m-x", DeltaSeconds: 30},
			{MediaFileID: 1, MatchID: "m-x", DeltaSeconds: 99}, // duplicate → IGNORE
			{MediaFileID: 2, MatchID: "m-y", DeltaSeconds: 60},
		},
	}
	if err := p.Persist(ctx, batch); err != nil {
		t.Fatal(err)
	}
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM media_match_associations_latest`).Scan(&count)
	if count != 2 {
		t.Errorf("append-only associations: count=%d, want 2 (vue _latest dédup par (media,match))", count)
	}
	// Append-only : latest wins (le dernier event d'un (media,match) gagne via la vue).
	var delta int
	_ = db.QueryRow(`SELECT delta_seconds FROM media_match_associations_latest WHERE media_file_id = 1`).Scan(&delta)
	if delta != 99 {
		t.Errorf("delta append-only: got %d, want 99 (latest wins via vue _latest)", delta)
	}
}

func TestSharedSocialPersister_Favorites_AddAndRemove(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()

	now := time.Now().UTC()
	// Add 2 favorites
	if err := p.Persist(ctx, &SharedSocialBatch{
		BatchID: "fav1", Source: "test",
		Favorites: []FavoriteInsert{
			{PlayerSlug: "u1", MatchID: "m1", FavoritedAt: now},
			{PlayerSlug: "u1", MatchID: "m2", FavoritedAt: now},
		},
	}); err != nil {
		t.Fatal(err)
	}
	// Remove 1
	if err := p.Persist(ctx, &SharedSocialBatch{
		BatchID: "fav2", Source: "test",
		FavoritesToRemove: []FavoriteRemove{{PlayerSlug: "u1", MatchID: "m1"}},
	}); err != nil {
		t.Fatal(err)
	}
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_favorites_latest WHERE player_slug = 'u1' AND is_favorite = TRUE`).Scan(&count)
	if count != 1 {
		t.Errorf("favorites: count=%d, want 1 (m1+m2 ajoutés, m1 retiré → seul m2 favori)", count)
	}
}

func TestSharedSocialPersister_NotificationRead_UpdatesReadAt(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()

	now := time.Now().UTC()
	// Add notification
	_ = p.Persist(ctx, &SharedSocialBatch{
		BatchID: "n1", Source: "test",
		Notifications: []NotificationInsert{
			{XUID: "x", ID: 1, Category: "c", Severity: "info", TitleKey: "t", Source: "s", CreatedAt: now},
		},
	})
	// Mark as read
	readAt := now.Add(1 * time.Hour)
	if err := p.Persist(ctx, &SharedSocialBatch{
		BatchID: "n2", Source: "test",
		NotificationReads: []NotificationReadUpdate{{XUID: "x", ID: 1, ReadAt: readAt}},
	}); err != nil {
		t.Fatal(err)
	}
	var read sql.NullTime
	_ = db.QueryRow(`SELECT read_at FROM player_notifications_latest WHERE xuid = 'x' AND id = 1`).Scan(&read)
	if !read.Valid {
		t.Error("read_at non mis à jour")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Toggle : remove (DELETE)
// ─────────────────────────────────────────────────────────────────────────────

func TestSharedSocialPersister_ToggleLike_AddThenRemove(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()

	// Add
	if err := p.Persist(ctx, &SharedSocialBatch{
		BatchID: "toggle1", Source: "test",
		Likes: []LikeInsert{{MediaPath: "/t.mp4", LikerSlug: "u", LikedAt: time.Now().UTC()}},
	}); err != nil {
		t.Fatal(err)
	}
	// Remove
	if err := p.Persist(ctx, &SharedSocialBatch{
		BatchID: "toggle2", Source: "test",
		LikesToRemove: []LikeRemove{{MediaPath: "/t.mp4", LikerSlug: "u"}},
	}); err != nil {
		t.Fatal(err)
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM media_likes_latest WHERE media_path = '/t.mp4' AND is_liked = TRUE").Scan(&count)
	if count != 0 {
		t.Errorf("toggle remove KO: count=%d (add TRUE + remove FALSE → 0 like actif)", count)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Mutations notifications user-facing (MarkNotificationsRead, MarkUnread,
// MarkAll, Delete, CapAndSweep, UpsertPreferences) — ADR 0022 fermeture du gap.
// ─────────────────────────────────────────────────────────────────────────────

// seedNotif insère une notification minimale (event create) pour les tests de
// mutations. APPEND-ONLY : on écrit dans player_notifications_history, written_at
// sur l'HORLOGE UTC CANONIQUE — le même référentiel que la prod depuis le lot S5.
// Le seed doit partager l'horloge des events de mutation, sinon la comparaison est
// décidée par le fuseau de la machine : un seed en `CURRENT_TIMESTAMP` (donc deux
// heures dans le futur à UTC+2) resterait en tête de player_notifications_latest
// devant l'event UTC écrit ensuite, et le test échouerait hors UTC.
// Le tie-break seq DESC règle les égalités. L'état courant se lit via _latest.
func seedNotif(t *testing.T, db *sql.DB, xuid string, id int64, category string, createdAt time.Time) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO player_notifications_history
			(xuid, id, category, severity, title_key, source, created_at, read_at, is_deleted, written_at)
		VALUES (?, ?, ?, 'info', 'k', 's', ?, NULL, FALSE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`,
		xuid, id, category, createdAt); err != nil {
		t.Fatalf("seed notif id=%d: %v", id, err)
	}
}

func TestSharedSocialPersister_MarkNotificationsRead_Batch(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()
	now := time.Now().UTC()
	for _, id := range []int64{1, 2, 3} {
		seedNotif(t, db, "x", id, "c", now)
	}
	n, err := p.MarkNotificationsRead(ctx, "x", []int64{1, 2}, now)
	if err != nil {
		t.Fatalf("MarkNotificationsRead: %v", err)
	}
	if n != 2 {
		t.Errorf("n=%d want 2", n)
	}
	var unread int
	_ = db.QueryRow(`SELECT COUNT(*) FROM player_notifications_latest WHERE xuid='x' AND read_at IS NULL`).Scan(&unread)
	if unread != 1 {
		t.Errorf("unread=%d want 1", unread)
	}
	// Idempotent : re-marquer des déjà-lues → 0 ligne affectée.
	n2, _ := p.MarkNotificationsRead(ctx, "x", []int64{1, 2}, now)
	if n2 != 0 {
		t.Errorf("re-mark n=%d want 0", n2)
	}
}

func TestSharedSocialPersister_MarkNotificationUnread(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()
	now := time.Now().UTC()
	seedNotif(t, db, "x", 1, "c", now)
	if _, err := p.MarkNotificationsRead(ctx, "x", []int64{1}, now); err != nil {
		t.Fatalf("pre mark-read: %v", err)
	}
	n, err := p.MarkNotificationUnread(ctx, "x", 1)
	if err != nil {
		t.Fatalf("MarkNotificationUnread: %v", err)
	}
	if n != 1 {
		t.Errorf("n=%d want 1", n)
	}
	var read sql.NullTime
	_ = db.QueryRow(`SELECT read_at FROM player_notifications_latest WHERE xuid='x' AND id=1`).Scan(&read)
	if read.Valid {
		t.Error("read_at devrait être NULL après MarkNotificationUnread")
	}
	if n0, _ := p.MarkNotificationUnread(ctx, "x", 999); n0 != 0 {
		t.Errorf("id inconnu n=%d want 0", n0)
	}
}

func TestSharedSocialPersister_MarkAllNotificationsRead(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()
	now := time.Now().UTC()
	seedNotif(t, db, "x", 1, "A", now)
	seedNotif(t, db, "x", 2, "A", now)
	seedNotif(t, db, "x", 3, "B", now)
	nA, err := p.MarkAllNotificationsRead(ctx, "x", "A", now)
	if err != nil {
		t.Fatalf("MarkAll cat A: %v", err)
	}
	if nA != 2 {
		t.Errorf("nA=%d want 2", nA)
	}
	nAll, _ := p.MarkAllNotificationsRead(ctx, "x", "", now)
	if nAll != 1 {
		t.Errorf("nAll=%d want 1 (reste B)", nAll)
	}
}

// D5/DP8 : le sweep expiry douce ne touche que severity='info' non lues plus
// vieilles que cutoff.
func TestSharedSocialPersister_SweepStaleInfoNotificationsRead(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()
	now := time.Now().UTC()
	old := now.Add(-8 * 24 * time.Hour)
	cutoff := now.Add(-7 * 24 * time.Hour)

	seedNotifSeverity := func(id int64, severity string, createdAt time.Time) {
		t.Helper()
		if _, err := db.Exec(`
			INSERT INTO player_notifications_history
				(xuid, id, category, severity, title_key, source, created_at, read_at, is_deleted, written_at)
			VALUES ('x', ?, 'c', ?, 'k', 's', ?, NULL, FALSE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`,
			id, severity, createdAt); err != nil {
			t.Fatalf("seed notif id=%d: %v", id, err)
		}
	}
	seedNotifSeverity(1, "info", old)    // → swept (info + périmée)
	seedNotifSeverity(2, "success", old) // → intacte (pas info)
	seedNotifSeverity(3, "info", now)    // → intacte (récente)

	n, err := p.SweepStaleInfoNotificationsRead(ctx, "x", cutoff)
	if err != nil {
		t.Fatalf("SweepStaleInfoNotificationsRead: %v", err)
	}
	if n != 1 {
		t.Errorf("n=%d want 1 (seule l'info périmée)", n)
	}
	var unread int
	_ = db.QueryRow(`SELECT COUNT(*) FROM player_notifications_latest WHERE xuid='x' AND read_at IS NULL`).Scan(&unread)
	if unread != 2 {
		t.Errorf("unread=%d want 2 (success 8 j + info fraîche)", unread)
	}
	// Idempotent : re-sweep → 0.
	if n2, _ := p.SweepStaleInfoNotificationsRead(ctx, "x", cutoff); n2 != 0 {
		t.Errorf("re-sweep n=%d want 0", n2)
	}
}

func TestSharedSocialPersister_DeleteNotification(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()
	seedNotif(t, db, "x", 1, "c", time.Now().UTC())
	n, err := p.DeleteNotification(ctx, "x", 1)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if n != 1 {
		t.Errorf("n=%d want 1", n)
	}
	var cnt int
	_ = db.QueryRow(`SELECT COUNT(*) FROM player_notifications_latest WHERE xuid='x' AND id=1`).Scan(&cnt)
	if cnt != 0 {
		t.Errorf("cnt=%d want 0", cnt)
	}
	if n0, _ := p.DeleteNotification(ctx, "x", 999); n0 != 0 {
		t.Errorf("id inconnu n=%d want 0", n0)
	}
}

func TestSharedSocialPersister_CapAndSweepNotifications(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()
	base := time.Now().UTC().Add(-10 * time.Hour)
	for i := int64(1); i <= 5; i++ {
		seedNotif(t, db, "x", i, "c", base.Add(time.Duration(i)*time.Hour))
	}
	if err := p.CapAndSweepNotifications(ctx, "x", 3); err != nil {
		t.Fatalf("CapAndSweep: %v", err)
	}
	var cnt int
	_ = db.QueryRow(`SELECT COUNT(*) FROM player_notifications_latest WHERE xuid='x'`).Scan(&cnt)
	if cnt != 3 {
		t.Errorf("cnt=%d want 3", cnt)
	}
	// Les 3 plus récentes (ids 3,4,5) restent → MIN(id)=3.
	var minID int64
	_ = db.QueryRow(`SELECT MIN(id) FROM player_notifications_latest WHERE xuid='x'`).Scan(&minID)
	if minID != 3 {
		t.Errorf("minID=%d want 3 (plus récentes gardées)", minID)
	}
}

func TestSharedSocialPersister_UpsertNotificationPreferences(t *testing.T) {
	_, db := setupSocialDB(t)
	p := NewSharedSocialPersister(db)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := p.UpsertNotificationPreferences(ctx, "x",
		[]string{"A", "B"}, []bool{true, false}, []string{"both", "off"}, now); err != nil {
		t.Fatalf("upsert insert: %v", err)
	}
	var cnt int
	_ = db.QueryRow(`SELECT COUNT(*) FROM notification_preferences_latest WHERE xuid='x'`).Scan(&cnt)
	if cnt != 2 {
		t.Errorf("cnt=%d want 2", cnt)
	}
	// Update de A (ON CONFLICT) — pas de doublon, valeurs changées.
	if err := p.UpsertNotificationPreferences(ctx, "x",
		[]string{"A"}, []bool{false}, []string{"inapp"}, now.Add(time.Hour)); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	var enabled bool
	var delivery string
	_ = db.QueryRow(`SELECT enabled, delivery FROM notification_preferences_latest WHERE xuid='x' AND category='A'`).Scan(&enabled, &delivery)
	if enabled || delivery != "inapp" {
		t.Errorf("A enabled=%v delivery=%s want false/inapp", enabled, delivery)
	}
	_ = db.QueryRow(`SELECT COUNT(*) FROM notification_preferences_latest WHERE xuid='x'`).Scan(&cnt)
	if cnt != 2 {
		t.Errorf("après update cnt=%d want 2 (pas de doublon)", cnt)
	}
	// Slices de longueurs différentes → erreur.
	if err := p.UpsertNotificationPreferences(ctx, "x",
		[]string{"A", "B"}, []bool{true}, []string{"both"}, now); err == nil {
		t.Error("slices parallèles de longueurs différentes devraient retourner une erreur")
	}
}
