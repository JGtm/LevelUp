package service

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

// TestUploadHLSTranscoding_EndToEnd valide tout l'assemblage Phase 3 :
// launchHLSTranscoding détecte le MKV multipiste, le marque, crée un job, et le
// worker async bascule la DB vers le master HLS + supprime le source + notifie
// la galerie. Nécessite ffmpeg. Cas AVEC miniature → source supprimé.
func TestUploadHLSTranscoding_EndToEnd(t *testing.T) {
	// thumbnail non-NULL : reflète la miniature générée par IndexMedia AVANT le transcode.
	// DeleteSource=true : les 4 gardes passent → suppression attendue.
	mkv, dbPath, capturesDir := runHLSTranscodeFixture(t, "GT/thumbs/multi.jpg", true)

	assertHLSFinalized(t, dbPath, capturesDir)
	if _, err := os.Stat(mkv); !os.IsNotExist(err) {
		t.Error("source MKV non supprimé après transcoding (miniature liée → suppression attendue)")
	}
}

// TestUploadHLSTranscoding_NoThumbnail_ConservesSource couvre la garde anti-perte de
// ed1b1e982 : sans miniature liée, le source MKV est CONSERVÉ (seul moyen de régénérer
// la miniature plus tard), tout en finalisant le HLS en DB.
func TestUploadHLSTranscoding_NoThumbnail_ConservesSource(t *testing.T) {
	// thumbnail vide → SQL NULL → garde miniature déclenchée → source conservé, MÊME
	// avec DeleteSource=true (la garde anti-perte miniature précède la politique de rétention).
	mkv, dbPath, capturesDir := runHLSTranscodeFixture(t, "", true)

	assertHLSFinalized(t, dbPath, capturesDir)
	if _, err := os.Stat(mkv); err != nil {
		t.Errorf("source MKV doit être CONSERVÉ sans miniature (régénération ultérieure), got err=%v", err)
	}
}

// runHLSTranscodeFixture monte le décor commun (MKV multipiste + media_files avec
// thumbnailValue, "" = NULL) et exécute le transcode async jusqu'à complétion.
func runHLSTranscodeFixture(t *testing.T, thumbnailValue string, deleteSource bool) (mkv, dbPath, capturesDir string) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent — test d'intégration transcoding ignoré")
	}
	ctx := context.Background()
	base := t.TempDir()
	const gamertag = "GT"
	capturesDir = filepath.Join(base, gamertag)
	if err := os.MkdirAll(capturesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mkv = genServiceMKV(t, capturesDir, "multi.mkv")

	dbPath = filepath.Join(base, "shared_social.duckdb")
	func() {
		db, err := sql.Open("duckdb", dbPath)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		if _, err := db.ExecContext(ctx,
			`CREATE TABLE media_files (id INTEGER, file_path VARCHAR, hls_path VARCHAR, transcode_status VARCHAR, kind VARCHAR, thumbnail_path VARCHAR)`); err != nil {
			t.Fatal(err)
		}
		var thumb interface{} // nil → SQL NULL (cas sans miniature)
		if thumbnailValue != "" {
			thumb = thumbnailValue
		}
		if _, err := db.ExecContext(ctx,
			`INSERT INTO media_files VALUES (1, 'GT/multi.mkv', NULL, NULL, 'video', ?)`, thumb); err != nil {
			t.Fatal(err)
		}
	}()

	jobStore := jobs.NewStore(filepath.Join(base, "jobs.json"))
	done := make(chan struct{}, 1)
	feedBump := func() {
		select {
		case done <- struct{}{}:
		default:
		}
	}
	svc := NewMediaService(nil, "", WithMediaTranscoding(jobStore, feedBump))

	req := domain.UploadRequest{
		CapturesDir:        capturesDir,
		CapturesBase:       base,
		Gamertag:           gamertag,
		SharedSocialDBPath: dbPath,
		DeleteSource:       deleteSource,
	}
	svc.launchHLSTranscoding(ctx, req, []string{mkv})

	select {
	case <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("timeout : transcoding non terminé (feed-bump non émis)")
	}
	return mkv, dbPath, capturesDir
}

// assertHLSFinalized vérifie que la DB a basculé vers le master HLS (status ready) et que
// le master.m3u8 existe — invariant commun aux deux cas (miniature ou non).
func assertHLSFinalized(t *testing.T, dbPath, capturesDir string) {
	t.Helper()
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var fp, hp, ts string
	if err := db.QueryRowContext(context.Background(),
		`SELECT file_path, hls_path, transcode_status FROM media_files WHERE id=1`).Scan(&fp, &hp, &ts); err != nil {
		t.Fatal(err)
	}
	if fp != "GT/hls/multi/master.m3u8" || hp != fp || ts != "ready" {
		t.Errorf("DB = (%q, %q, %q), want (GT/hls/multi/master.m3u8, idem, ready)", fp, hp, ts)
	}
	if _, err := os.Stat(filepath.Join(capturesDir, "hls", "multi", "master.m3u8")); err != nil {
		t.Errorf("master.m3u8 absent: %v", err)
	}
}

func genServiceMKV(t *testing.T, dir, name string) string {
	t.Helper()
	src := filepath.Join(dir, name)
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=15:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=880:duration=2",
		"-map", "0:v", "-map", "1:a", "-map", "2:a",
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "libopus",
		"-metadata:s:a:0", "title=Game", "-metadata:s:a:1", "title=Mic",
		src)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("génération MKV: %v\n%s", err, out)
	}
	return src
}
