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
	var startedAt sql.NullTime
	if err := db.QueryRowContext(ctx,
		`SELECT file_path, COALESCE(hls_path,''), COALESCE(transcode_status,''), transcode_started_at
		 FROM media_files WHERE id=1`).Scan(&fp, &hp, &ts, &startedAt); err != nil {
		t.Fatal(err)
	}
	if fp != "GT/hls/source/master.m3u8" || hp != fp || ts != TranscodeReady {
		t.Errorf("MKV row = (%q,%q,%q), want (GT/hls/source/master.m3u8, idem, ready)", fp, hp, ts)
	}
	// Le balayage a acquis le verrou 'processing' (+ horodatage) AVANT le
	// transcodage ; l'horodatage survit à la finalisation 'ready' → preuve que
	// MarkTranscodeProcessing a bien tourné côté sweep.
	if !startedAt.Valid {
		t.Error("transcode_started_at doit être horodaté (marquage 'processing' du balayage)")
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

// TestSelectPendingHLSCandidates : la sélection du balayage retient les vidéos
// sans hls_path SAUF celles dont le transcodage est déjà décidé/en cours. Exclus :
// images, clips transcodés (hls_path), 'direct' (web-natif), 'failed' (retry
// manuel seulement), 'processing' FRAIS (transcodage en cours). Ré-éligibles :
// transcode_status NULL (historique), 'processing' PÉRIMÉ (> transcodeStaleAfter,
// orphelin de crash) et 'processing' sans horodatage (orphelin legacy). Filtre
// slug honoré.
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
		if _, err := db.Exec(`CREATE TABLE media_files (
			id INTEGER, player_slug VARCHAR, file_path VARCHAR, hls_path VARCHAR, kind VARCHAR,
			transcode_status VARCHAR, transcode_started_at TIMESTAMPTZ)`); err != nil {
			t.Fatal(err)
		}
		// now() FRAIS vs now()-3h PÉRIMÉ (transcodeStaleAfter = 2 h) ; comparaison
		// timestamptz par instant → indépendante du fuseau.
		if _, err := db.Exec(`INSERT INTO media_files
			(id, player_slug, file_path, hls_path, kind, transcode_status, transcode_started_at) VALUES
			(1, 'A', 'A/clip.mkv',   NULL,                     'video', NULL,         NULL),
			(2, 'A', 'A/done.mkv',   'A/hls/done/master.m3u8', 'video', 'ready',      NULL),
			(3, 'B', 'B/other.mkv',  NULL,                     'video', NULL,         NULL),
			(4, 'A', 'A/img.png',    NULL,                     'image', NULL,         NULL),
			(5, 'A', 'A/direct.mkv', NULL,                     'video', 'direct',     NULL),
			(6, 'A', 'A/failed.mkv', NULL,                     'video', 'failed',     NULL),
			(7, 'A', 'A/fresh.mkv',  NULL,                     'video', 'processing', now()),
			(8, 'A', 'A/stale.mkv',  NULL,                     'video', 'processing', now() - INTERVAL 3 HOUR),
			(9, 'A', 'A/orphan.mkv', NULL,                     'video', 'processing', NULL)`); err != nil {
			t.Fatal(err)
		}
	}()

	all, err := selectPendingHLSCandidates(ctx, dbPath, "")
	if err != nil {
		t.Fatalf("selectPendingHLSCandidates: %v", err)
	}
	wantAll := map[string]bool{"A/clip.mkv": true, "B/other.mkv": true, "A/stale.mkv": true, "A/orphan.mkv": true}
	assertCandidateSet(t, "tous", all, wantAll)

	a, err := selectPendingHLSCandidates(ctx, dbPath, "A")
	if err != nil {
		t.Fatal(err)
	}
	wantA := map[string]bool{"A/clip.mkv": true, "A/stale.mkv": true, "A/orphan.mkv": true}
	assertCandidateSet(t, "slug=A", a, wantA)
}

// assertCandidateSet vérifie que l'ensemble des file_path des candidats correspond
// EXACTEMENT à want (l'ordre de sélection n'étant pas garanti).
func assertCandidateSet(t *testing.T, label string, got []hlsCandidate, want map[string]bool) {
	t.Helper()
	seen := make(map[string]bool, len(got))
	for _, c := range got {
		seen[c.filePath] = true
		if !want[c.filePath] {
			t.Errorf("%s: candidat inattendu %q", label, c.filePath)
		}
	}
	for fp := range want {
		if !seen[fp] {
			t.Errorf("%s: candidat manquant %q", label, fp)
		}
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
		if _, err := db.Exec(`CREATE TABLE media_files (
			id INTEGER, player_slug VARCHAR, file_path VARCHAR, hls_path VARCHAR, kind VARCHAR,
			transcode_status VARCHAR, transcode_started_at TIMESTAMPTZ)`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO media_files
			(id, player_slug, file_path, hls_path, kind) VALUES (1, 'GT', 'GT/ghost.mkv', NULL, 'video')`); err != nil {
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

// TestEnsurePendingHLS_DirectMarkingPersists : une vidéo web-native mono-piste est
// marquée 'direct' au 1er balayage (servie en direct, pas de HLS) ; le 2e balayage
// ne la considère PLUS candidate — plus aucune re-probe ffprobe (avant persistance,
// SkippedDirect était recompté à l'infini, un ffprobe par sync). Nécessite ffmpeg/ffprobe.
func TestEnsurePendingHLS_DirectMarkingPersists(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent — TestEnsurePendingHLS_DirectMarkingPersists ignoré")
	}
	ctx := context.Background()
	base := t.TempDir()
	ownerDir := filepath.Join(base, "GT")
	if err := os.MkdirAll(ownerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	genMonoMP4(t, ownerDir, "mono.mp4") // web-natif mono-piste → servi en direct

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
			`INSERT INTO media_files (id, player_slug, file_path, kind) VALUES (1, 'GT', 'GT/mono.mp4', 'video')`); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}()

	// 1er balayage : servie direct + marquée 'direct'.
	st1, err := EnsurePendingHLS(ctx, EnsureHLSParams{DBPath: dbPath, CapturesBase: base})
	if err != nil {
		t.Fatalf("1er sweep: %v", err)
	}
	if st1.SkippedDirect != 1 {
		t.Fatalf("1er sweep: SkippedDirect = %d, want 1", st1.SkippedDirect)
	}
	func() {
		db, err := sql.Open("duckdb", dbPath)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() //nolint:errcheck
		var status string
		if err := db.QueryRowContext(ctx,
			`SELECT COALESCE(transcode_status,'') FROM media_files WHERE id=1`).Scan(&status); err != nil {
			t.Fatal(err)
		}
		if status != TranscodeDirect {
			t.Errorf("après 1er sweep: transcode_status = %q, want direct", status)
		}
	}()

	// 2e balayage : la ligne 'direct' n'est plus candidate → aucune re-probe.
	st2, err := EnsurePendingHLS(ctx, EnsureHLSParams{DBPath: dbPath, CapturesBase: base})
	if err != nil {
		t.Fatalf("2e sweep: %v", err)
	}
	if st2.SkippedDirect != 0 || st2.Transcoded != 0 || st2.Failed != 0 {
		t.Errorf("2e sweep: stats = %+v, want tout à 0 (ligne 'direct' exclue de la sélection)", st2)
	}
}

// TestResetFailedTranscodes : réarme les lignes 'failed' (→ NULL), scope --slug
// respecté ; les autres statuts sont intacts. Déterministe (pas de ffmpeg).
func TestResetFailedTranscodes(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared_social.duckdb")
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
			`INSERT INTO media_files (id, player_slug, file_path, kind, transcode_status) VALUES
			 (1, 'A', 'A/f1.mkv', 'video', 'failed'),
			 (2, 'A', 'A/ok.mkv', 'video', 'ready'),
			 (3, 'B', 'B/f2.mkv', 'video', 'failed')`); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}()

	statusOf := func(id int) sql.NullString {
		db, err := sql.Open("duckdb", dbPath)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() //nolint:errcheck
		var s sql.NullString
		if err := db.QueryRowContext(ctx, `SELECT transcode_status FROM media_files WHERE id=?`, id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	// Scope slug A : seule la ligne 1 ('failed', slug A) est réarmée.
	n, err := ResetFailedTranscodes(ctx, dbPath, "A")
	if err != nil {
		t.Fatalf("ResetFailedTranscodes(A): %v", err)
	}
	if n != 1 {
		t.Errorf("réarmées(A) = %d, want 1", n)
	}
	if s := statusOf(1); s.Valid {
		t.Errorf("ligne 1: transcode_status = %q, want NULL (réarmée)", s.String)
	}
	if s := statusOf(2); !s.Valid || s.String != TranscodeReady {
		t.Errorf("ligne 2: transcode_status = %v, want ready (intacte)", s)
	}
	if s := statusOf(3); !s.Valid || s.String != TranscodeFailed {
		t.Errorf("ligne 3 (slug B): transcode_status = %v, want failed (hors scope)", s)
	}

	// Sans scope : la ligne 3 (slug B) est réarmée à son tour.
	n, err = ResetFailedTranscodes(ctx, dbPath, "")
	if err != nil {
		t.Fatalf("ResetFailedTranscodes(all): %v", err)
	}
	if n != 1 {
		t.Errorf("réarmées(all) = %d, want 1 (seule B/f2 restait 'failed')", n)
	}
	if s := statusOf(3); s.Valid {
		t.Errorf("ligne 3: transcode_status = %q, want NULL (réarmée)", s.String)
	}
}
