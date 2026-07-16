package ops

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestEnsurePendingHLS_TranscodesScannedVideo : une vidéo indexée sans HLS
// (hls_path NULL) est transcodée en HLS par le balayage, exactement comme à
// l'upload — c'est le correctif de l'asymétrie upload/scan (bug "media remux
// failed" sur HEVC). Une vidéo web-native mono-piste est laissée telle quelle
// (servie en direct, hls_path conservé NULL). Nécessite ffmpeg.
func TestEnsurePendingHLS_TranscodesScannedVideo(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent — TestEnsurePendingHLS ignoré")
	}
	ctx := context.Background()
	base := t.TempDir()
	ownerDir := filepath.Join(base, "GT")
	if err := os.MkdirAll(ownerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	genHLSSourceMKV(t, ownerDir, "source.mkv") // MKV multipiste → HLS requis
	genMonoMP4(t, ownerDir, "mono.mp4")        // MP4 mono web-natif → servi direct

	dbPath := filepath.Join(base, "shared_social.duckdb")
	setupSweepDB(t, ctx, dbPath)

	st, err := EnsurePendingHLS(ctx, EnsureHLSParams{DBPath: dbPath, CapturesBase: base, DeleteSource: true})
	if err != nil {
		t.Fatalf("EnsurePendingHLS: %v", err)
	}
	if st.Transcoded != 1 || st.SkippedDirect != 1 || st.Failed != 0 {
		t.Fatalf("stats = %+v, want Transcoded=1 SkippedDirect=1 Failed=0", st)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck

	// La vidéo MKV a basculé vers le HLS.
	var fp, hp, ts string
	if err := db.QueryRowContext(ctx,
		`SELECT file_path, COALESCE(hls_path,''), COALESCE(transcode_status,'')
		 FROM media_files WHERE id=1`).Scan(&fp, &hp, &ts); err != nil {
		t.Fatal(err)
	}
	if fp != "GT/hls/source/master.m3u8" || hp != fp || ts != TranscodeReady {
		t.Errorf("MKV row = (%q,%q,%q), want (GT/hls/source/master.m3u8, idem, ready)", fp, hp, ts)
	}
	if _, err := os.Stat(filepath.Join(ownerDir, "source.mkv")); !os.IsNotExist(err) {
		t.Errorf("source MKV non supprimé après HLS")
	}

	// La MP4 web-native est restée servie en direct (hls_path NULL).
	var hp2 sql.NullString
	if err := db.QueryRowContext(ctx,
		`SELECT hls_path FROM media_files WHERE id=2`).Scan(&hp2); err != nil {
		t.Fatal(err)
	}
	if hp2.Valid {
		t.Errorf("MP4 mono ne devrait pas avoir de hls_path, got %q", hp2.String)
	}
}

// setupSweepDB crée media_files avec 2 lignes sans HLS au départ : un MKV (HLS
// requis, miniature liée → la garde anti-perte autorise la suppression du
// source après transcoding) et un MP4 mono web-natif (servi direct).
func setupSweepDB(t *testing.T, ctx context.Context, dbPath string) {
	t.Helper()
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck
	if err := ensureMediaTables(ctx, db); err != nil {
		t.Fatalf("ensureMediaTables: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO media_files (id, player_slug, file_path, kind, thumbnail_path) VALUES
		 (1, 'GT', 'GT/source.mkv', 'video', 'GT/thumbs/source.webp'),
		 (2, 'GT', 'GT/mono.mp4', 'video', NULL)`); err != nil {
		t.Fatalf("insert media_files: %v", err)
	}
}

// TestEnsurePendingHLS_SingleFlight : si un balayage tourne déjà (mutex tenu),
// un second appel retourne immédiatement Busy=true sans rien transcoder.
// Déterministe — ni ffmpeg ni DB requis (retour avant tout accès disque).
func TestEnsurePendingHLS_SingleFlight(t *testing.T) {
	hlsSweepMu.Lock()
	defer hlsSweepMu.Unlock()

	st, err := EnsurePendingHLS(context.Background(), EnsureHLSParams{
		DBPath:       filepath.Join(t.TempDir(), "unused.duckdb"),
		CapturesBase: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("EnsurePendingHLS: erreur inattendue: %v", err)
	}
	if !st.Busy {
		t.Errorf("st.Busy = false, want true (balayage déjà en cours)")
	}
	if st.Transcoded != 0 {
		t.Errorf("st.Transcoded = %d, want 0 (ne doit rien faire)", st.Transcoded)
	}
}

// TestSelectPendingHLSCandidates : seules les vidéos sans hls_path sont
// candidates ; images et clips déjà transcodés exclus ; filtre slug honoré.
// (Migré depuis cmd/backfill-media-hls après centralisation de la logique.)
func TestSelectPendingHLSCandidates(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared_social.duckdb")
	func() {
		db, err := sql.Open("duckdb", dbPath)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() //nolint:errcheck
		if _, err := db.Exec(`CREATE TABLE media_files (id INTEGER, player_slug VARCHAR, file_path VARCHAR, hls_path VARCHAR, kind VARCHAR)`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO media_files VALUES
			(1, 'A', 'A/clip.mkv',  NULL,                     'video'),
			(2, 'A', 'A/done.mkv',  'A/hls/done/master.m3u8', 'video'),
			(3, 'B', 'B/other.mkv', NULL,                     'video'),
			(4, 'A', 'A/img.png',   NULL,                     'image')`); err != nil {
			t.Fatal(err)
		}
	}()

	all, err := selectPendingHLSCandidates(ctx, dbPath, "")
	if err != nil {
		t.Fatalf("selectPendingHLSCandidates: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("tous: %d candidats, want 2 (clip.mkv + other.mkv ; done.mkv et img.png exclus)", len(all))
	}

	a, err := selectPendingHLSCandidates(ctx, dbPath, "A")
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 1 || a[0].filePath != "A/clip.mkv" {
		t.Errorf("filtre slug A: got %+v, want [A/clip.mkv]", a)
	}
}

// TestEnsurePendingHLS_MissingSource : une ligne média dont le fichier source
// n'existe plus sur disque est comptée Missing, sans erreur et sans toucher la
// DB. Déterministe — pas de ffmpeg requis (os.Stat échoue avant le probe).
func TestEnsurePendingHLS_MissingSource(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	dbPath := filepath.Join(base, "shared_social.duckdb")
	func() {
		db, err := sql.Open("duckdb", dbPath)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() //nolint:errcheck
		if _, err := db.Exec(`CREATE TABLE media_files (id INTEGER, player_slug VARCHAR, file_path VARCHAR, hls_path VARCHAR, kind VARCHAR)`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO media_files VALUES (1, 'GT', 'GT/ghost.mkv', NULL, 'video')`); err != nil {
			t.Fatal(err)
		}
	}()

	st, err := EnsurePendingHLS(ctx, EnsureHLSParams{DBPath: dbPath, CapturesBase: base})
	if err != nil {
		t.Fatalf("EnsurePendingHLS: %v", err)
	}
	if st.Missing != 1 || st.Transcoded != 0 {
		t.Errorf("stats = %+v, want Missing=1 Transcoded=0", st)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck
	var hp sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT hls_path FROM media_files WHERE id=1`).Scan(&hp); err != nil {
		t.Fatal(err)
	}
	if hp.Valid {
		t.Errorf("hls_path ne devrait pas changer pour une source absente, got %q", hp.String)
	}
}

// TestEnsurePendingHLS_DryRun : en dry-run, les candidats à transcoder sont
// comptés mais NI le source supprimé NI la DB modifiée. Nécessite ffmpeg.
func TestEnsurePendingHLS_DryRun(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent — TestEnsurePendingHLS_DryRun ignoré")
	}
	ctx := context.Background()
	base := t.TempDir()
	ownerDir := filepath.Join(base, "GT")
	if err := os.MkdirAll(ownerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	genHLSSourceMKV(t, ownerDir, "source.mkv")

	dbPath := filepath.Join(base, "shared_social.duckdb")
	func() {
		db, err := sql.Open("duckdb", dbPath)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() //nolint:errcheck
		if err := ensureMediaTables(ctx, db); err != nil {
			t.Fatalf("ensureMediaTables: %v", err)
		}
		if _, err := db.ExecContext(ctx,
			`INSERT INTO media_files (id, player_slug, file_path, kind) VALUES (1, 'GT', 'GT/source.mkv', 'video')`); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}()

	st, err := EnsurePendingHLS(ctx, EnsureHLSParams{DBPath: dbPath, CapturesBase: base, DryRun: true})
	if err != nil {
		t.Fatalf("EnsurePendingHLS: %v", err)
	}
	if st.Transcoded != 1 {
		t.Errorf("dry-run: Transcoded = %d, want 1 (compté mais pas exécuté)", st.Transcoded)
	}
	if _, err := os.Stat(filepath.Join(ownerDir, "source.mkv")); err != nil {
		t.Errorf("dry-run: source supprimé à tort: %v", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck
	var fp string
	var hp sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT file_path, hls_path FROM media_files WHERE id=1`).Scan(&fp, &hp); err != nil {
		t.Fatal(err)
	}
	if fp != "GT/source.mkv" || hp.Valid {
		t.Errorf("dry-run: DB modifiée (fp=%q, hls valide=%v), attendu inchangée", fp, hp.Valid)
	}
}
