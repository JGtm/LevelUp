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
			liked BOOLEAN DEFAULT FALSE,
			liked_at TIMESTAMPTZ,
			discord_notified BOOLEAN DEFAULT FALSE,
			indexed_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE media_match_associations (
			media_file_id INTEGER,
			match_id VARCHAR,
			delta_seconds INTEGER,
			PRIMARY KEY (media_file_id, match_id)
		)`,
		`CREATE TABLE media_likes (
			media_path VARCHAR NOT NULL,
			liker_slug VARCHAR NOT NULL,
			liker_gamertag VARCHAR,
			liked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (media_path, liker_slug)
		)`,
		`CREATE TABLE match_favorites (
			player_slug VARCHAR NOT NULL,
			match_id VARCHAR NOT NULL,
			favorited_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (player_slug, match_id)
		)`,
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
			written_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
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
		{"media_likes", "SELECT COUNT(*) FROM media_likes WHERE liker_slug = 'friend1'", 1},
		{"match_favorites", "SELECT COUNT(*) FROM match_favorites WHERE player_slug = 'spartan'", 1},
		{"player_notifications", "SELECT COUNT(*) FROM player_notifications WHERE xuid = 'xuid-123'", 1},
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

	// Batch valide en partie 1 (media_files OK) + invalide en partie 2
	// (PK collision sur player_notifications : 2 fois la même row exacte).
	// La 2e insert sur notifications va FAIL avec PK constraint → rollback.
	now := time.Now().UTC()
	batch := &SharedSocialBatch{
		BatchID: "rollback-test",
		Source:  "unit_test",
		MediaFiles: []MediaFileInsert{
			{PlayerSlug: "p", FilePath: "/rb.mp4", FileName: "rb.mp4", FileHash: "rb", Kind: "video"},
		},
		Notifications: []NotificationInsert{
			{XUID: "x1", ID: 100, Category: "c", Severity: "info", TitleKey: "t", Source: "s", CreatedAt: now},
			{XUID: "x1", ID: 100, Category: "c", Severity: "info", TitleKey: "t", Source: "s", CreatedAt: now}, // PK duplicate
		},
	}

	err := p.Persist(ctx, batch)
	if err == nil {
		t.Fatal("Persist doit échouer (PK duplicate sur notification)")
	}

	// Vérifier que la media_file n'a PAS été persistée (rollback complet).
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM media_files WHERE file_path = '/rb.mp4'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("media_file persistée malgré rollback (count=%d, want 0)", count)
	}
	// Vérifier que la 1re notif n'a PAS été persistée non plus.
	if err := db.QueryRow("SELECT COUNT(*) FROM player_notifications WHERE xuid = 'x1' AND id = 100").Scan(&count); err != nil {
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
	_ = db.QueryRow("SELECT COUNT(*) FROM media_likes WHERE media_path = '/dup.mp4'").Scan(&count)
	if count != 1 {
		t.Errorf("dedup raté : count=%d, want 1", count)
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
	_ = db.QueryRow(`SELECT COUNT(*) FROM media_match_associations`).Scan(&count)
	if count != 2 {
		t.Errorf("INSERT OR IGNORE associations: count=%d, want 2 (1 dup ignoré)", count)
	}
	// Vérifier que la 1re wins (delta=30)
	var delta int
	_ = db.QueryRow(`SELECT delta_seconds FROM media_match_associations WHERE media_file_id = 1`).Scan(&delta)
	if delta != 30 {
		t.Errorf("delta du 1er INSERT: got %d, want 30 (le doublon doit être ignoré)", delta)
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
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_favorites WHERE player_slug = 'u1'`).Scan(&count)
	if count != 1 {
		t.Errorf("favorites: count=%d, want 1 (1 ajouté + 1 ajouté - 1 retiré)", count)
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
	_ = db.QueryRow(`SELECT read_at FROM player_notifications WHERE xuid = 'x' AND id = 1`).Scan(&read)
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
	_ = db.QueryRow("SELECT COUNT(*) FROM media_likes WHERE media_path = '/t.mp4'").Scan(&count)
	if count != 0 {
		t.Errorf("toggle remove KO: count=%d", count)
	}
}
