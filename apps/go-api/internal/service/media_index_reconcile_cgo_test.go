//go:build cgo

// media_index_reconcile_cgo_test.go — suppression d'une vidéo convertie par DeleteMedia,
// puis indexation : le média ne revient pas tant que l'original est absent, et revient
// avec ses likes et une nouvelle miniature quand l'original est remis.
package service

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
)

const likesDDL = `
	CREATE SEQUENCE IF NOT EXISTS media_likes_history_id_seq START 1;
	CREATE TABLE IF NOT EXISTS media_likes_history (
		id BIGINT PRIMARY KEY DEFAULT nextval('media_likes_history_id_seq'),
		media_path VARCHAR NOT NULL, liker_slug VARCHAR NOT NULL, liker_gamertag VARCHAR,
		is_liked BOOLEAN NOT NULL, liked_at TIMESTAMP,
		written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	);
	CREATE OR REPLACE VIEW media_likes_latest AS
		SELECT id, media_path, liker_slug, liker_gamertag, is_liked, liked_at, written_at
		FROM media_likes_history
		QUALIFY ROW_NUMBER() OVER (PARTITION BY media_path, liker_slug
			ORDER BY written_at DESC, id DESC) = 1;`

func TestDeleteConvertedMediaThenIndex_ComesBackWithOriginal(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent")
	}
	ctx := context.Background()
	root := t.TempDir()
	const gamertag = "JGtm"
	dbPath := filepath.Join(root, "shared_social.duckdb")
	base := filepath.Join(root, "captures")
	capDir := filepath.Join(base, gamertag)
	if err := os.MkdirAll(capDir, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := duckdb.OpenReadWriteShared(dbPath, "")
	if err != nil {
		t.Fatalf("OpenReadWriteShared: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	index := func() ops.MediaIndexResult {
		t.Helper()
		res, err := ops.IndexMedia(ctx, ops.MediaIndexOptions{
			SharedSocialDBPath: dbPath, CapturesDir: capDir, CapturesBase: base, Gamertag: gamertag,
			Timezone: "Europe/Paris", TitleSlug: "test_title", ForeignFilenamePrefixes: []string{},
			MissingPolicy: ops.MissingMark,
		})
		if err != nil {
			t.Fatalf("IndexMedia: %v", err)
		}
		return res
	}

	// Une vidéo convertie en HLS dont l'original est conservé, avec un like.
	mkv := genServiceMKV(t, capDir, "multi.mkv")
	original, err := os.ReadFile(mkv)
	if err != nil {
		t.Fatal(err)
	}
	if res := index(); res.NewFiles != 1 || res.Thumbnails != 1 {
		t.Fatalf("première passe : %+v, want 1 fichier et 1 miniature", res)
	}
	// Ce que les migrations ajoutent au schéma d'indexation : les likes et updated_at.
	if _, err := db.SQLDb().Exec(likesDDL + `ALTER TABLE media_files ADD COLUMN updated_at TIMESTAMP;`); err != nil {
		t.Fatal(err)
	}
	var id int64
	var filePath string
	if err := db.SQLDb().QueryRow(`SELECT id, file_path FROM media_files`).Scan(&id, &filePath); err != nil {
		t.Fatal(err)
	}
	outDir, hlsRel := ops.HLSPathsFor(capDir, base, gamertag, mkv)
	if err := ops.RunHLSTranscode(ctx, ops.HLSTranscodeParams{
		SourceAbs: mkv, OutDir: outDir, DBPath: dbPath, FileRel: filePath, HLSRel: hlsRel, DeleteSource: false,
	}); err != nil {
		t.Fatalf("RunHLSTranscode: %v", err)
	}
	if _, err := db.SQLDb().Exec(`INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
		VALUES (?, 'Chocoboflor', 'Chocoboflor', TRUE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`, hlsRel); err != nil {
		t.Fatal(err)
	}

	// Suppression depuis l'app : l'original part avec la version convertie.
	pdb := &duckdb.PlayerDB{SharedSocial: db, Gamertag: gamertag}
	svc := NewMediaService(duckdb.NewMediaRepo(pdb), "",
		WithMediaWriterAcquirer(func() (*dblease.LeasedWriter, error) {
			return pdb.AcquireSharedSocialWriterTimeout(dblease.SharedLeaseTimeout)
		}),
		WithMediaFileRemover(ops.NewOSMediaFileRemover(func() string { return base })))
	if _, err := svc.DeleteMedia(ctx, domain.MediaDeleteRequest{FilePath: hlsRel, RequesterSlug: gamertag}); err != nil {
		t.Fatalf("DeleteMedia: %v", err)
	}
	if _, err := os.Stat(mkv); !os.IsNotExist(err) {
		t.Fatal("l'original devait être retiré avec le média")
	}

	// Rien ne revient tant que l'original est absent.
	index()
	index()
	if n := visibleRows(t, db.SQLDb()); n != 0 {
		t.Fatalf("%d fiche(s) visible(s) après suppression, want 0", n)
	}

	// L'original remis au même endroit : la fiche revient, avec ses likes et une miniature.
	if err := os.WriteFile(mkv, original, 0o644); err != nil {
		t.Fatal(err)
	}
	index()
	index()
	var gotID int64
	var gotPath string
	var hls, thumb sql.NullString
	if err := db.SQLDb().QueryRow(`SELECT id, file_path, hls_path, thumbnail_path FROM media_files
		WHERE COALESCE(status, '') NOT IN ('deleted', 'missing')`).Scan(&gotID, &gotPath, &hls, &thumb); err != nil {
		t.Fatalf("fiche visible attendue : %v", err)
	}
	if gotID != id || gotPath != gamertag+"/multi.mkv" || hls.Valid {
		t.Errorf("fiche = (%d, %q, hls %v), want (%d, %q, lecture directe)", gotID, gotPath, hls, id, gamertag+"/multi.mkv")
	}
	if !thumb.Valid {
		t.Fatal("miniature non liée")
	}
	if _, err := os.Stat(filepath.Join(base, filepath.FromSlash(thumb.String))); err != nil {
		t.Errorf("miniature %q absente du disque : %v", thumb.String, err)
	}
	var likes int
	if err := db.SQLDb().QueryRow(`SELECT count(*) FROM media_likes_latest WHERE media_path = ? AND is_liked`, gotPath).Scan(&likes); err != nil || likes != 1 {
		t.Errorf("likes sur %q = %d (err %v), want 1", gotPath, likes, err)
	}
}

func visibleRows(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM media_files WHERE COALESCE(status, '') NOT IN ('deleted', 'missing')`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
