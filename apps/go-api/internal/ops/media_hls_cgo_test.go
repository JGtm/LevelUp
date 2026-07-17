package ops

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHLSPathsFor(t *testing.T) {
	capturesDir := filepath.Join("base", "GT")
	src := filepath.Join(capturesDir, "Clip 01.mkv")
	outDir, hlsRel := HLSPathsFor(capturesDir, "base", "GT", src)

	wantOut := filepath.Join(capturesDir, "hls", "Clip 01")
	if outDir != wantOut {
		t.Errorf("outDir = %q, want %q", outDir, wantOut)
	}
	if hlsRel != "GT/hls/Clip 01/master.m3u8" {
		t.Errorf("hlsRel = %q, want GT/hls/Clip 01/master.m3u8", hlsRel)
	}
}

func TestHLSPathsFor_LegacyNoBase(t *testing.T) {
	// capturesBase vide → hlsRel retombe sur le chemin absolu du master.
	capturesDir := filepath.Join("base", "GT")
	src := filepath.Join(capturesDir, "clip.mkv")
	outDir, hlsRel := HLSPathsFor(capturesDir, "", "GT", src)
	if hlsRel != filepath.Join(outDir, "master.m3u8") {
		t.Errorf("hlsRel = %q, want %q", hlsRel, filepath.Join(outDir, "master.m3u8"))
	}
}

// TestRunHLSTranscode_Success : transcoding nominal → DB basculée vers le master,
// source supprimé, arbre HLS présent. Nécessite ffmpeg.
func TestRunHLSTranscode_Success(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent — test d'intégration RunHLSTranscode ignoré")
	}
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared_social.duckdb")

	// Setup DB puis FERMETURE : RunHLSTranscode rouvrira sa propre connexion
	// (DuckDB n'autorise qu'un handle RW par fichier dans le process). Miniature
	// liée → la garde anti-perte autorise la suppression du source.
	setupMediaDB(t, ctx, dbPath, "source.mkv", "processing", "GT/thumbs/source.webp")

	mkv := genHLSSourceMKV(t, dir, "source.mkv")
	outDir := filepath.Join(dir, "hls", "source")
	params := HLSTranscodeParams{
		SourceAbs:    mkv,
		OutDir:       outDir,
		DBPath:       dbPath,
		FileRel:      "source.mkv",
		HLSRel:       "GT/hls/source/master.m3u8",
		DeleteSource: true, // politique de rétention : supprimer après HLS prouvé
	}
	if err := RunHLSTranscode(ctx, params); err != nil {
		t.Fatalf("RunHLSTranscode: %v", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var fp, hp, ts string
	if err := db.QueryRowContext(ctx,
		`SELECT file_path, hls_path, transcode_status FROM media_files WHERE id=1`).Scan(&fp, &hp, &ts); err != nil {
		t.Fatal(err)
	}
	if fp != "GT/hls/source/master.m3u8" || hp != fp || ts != TranscodeReady {
		t.Errorf("DB = (%q, %q, %q), want (GT/hls/source/master.m3u8, idem, ready)", fp, hp, ts)
	}
	if _, err := os.Stat(mkv); !os.IsNotExist(err) {
		t.Errorf("source MKV non supprimé après transcoding")
	}
	if _, err := os.Stat(filepath.Join(outDir, "master.m3u8")); err != nil {
		t.Errorf("master.m3u8 absent: %v", err)
	}
}

// TestRunHLSTranscode_DeleteSourceFalse_KeepsSource : transcoding RÉUSSI avec
// miniature liée (les 3 gardes anti-perte passent) MAIS DeleteSource=false → le HLS
// est finalisé (status ready) et le source est CONSERVÉ (4e garde : politique de
// rétention). Défaut sûr en local. Nécessite ffmpeg.
func TestRunHLSTranscode_DeleteSourceFalse_KeepsSource(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent — test d'intégration RunHLSTranscode ignoré")
	}
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared_social.duckdb")
	// Miniature liée : sans le 4e garde, le source SERAIT supprimé.
	setupMediaDB(t, ctx, dbPath, "source.mkv", "processing", "GT/thumbs/source.webp")

	mkv := genHLSSourceMKV(t, dir, "source.mkv")
	outDir := filepath.Join(dir, "hls", "source")
	params := HLSTranscodeParams{
		SourceAbs:    mkv,
		OutDir:       outDir,
		DBPath:       dbPath,
		FileRel:      "source.mkv",
		HLSRel:       "GT/hls/source/master.m3u8",
		DeleteSource: false, // conservation demandée
	}
	if err := RunHLSTranscode(ctx, params); err != nil {
		t.Fatalf("RunHLSTranscode: %v", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var fp, ts string
	if err := db.QueryRowContext(ctx,
		`SELECT file_path, transcode_status FROM media_files WHERE id=1`).Scan(&fp, &ts); err != nil {
		t.Fatal(err)
	}
	if fp != "GT/hls/source/master.m3u8" || ts != TranscodeReady {
		t.Errorf("DB = (%q,%q), want (GT/hls/source/master.m3u8, ready)", fp, ts)
	}
	if _, err := os.Stat(mkv); err != nil {
		t.Errorf("source supprimé alors que DeleteSource=false: %v", err)
	}
}

// TestRunHLSTranscode_NoThumbnail_KeepsSource : transcoding RÉUSSI mais aucune
// miniature liée (échec ffmpeg miniature à l'ingestion) → le HLS est finalisé
// (file_path basculé, status ready) MAIS le source est CONSERVÉ pour permettre
// une régénération ultérieure de la miniature. Garde anti-perte de données
// (sinon : média lisible mais sans miniature, irréversible). Nécessite ffmpeg.
func TestRunHLSTranscode_NoThumbnail_KeepsSource(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent — test d'intégration RunHLSTranscode ignoré")
	}
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared_social.duckdb")
	setupMediaDB(t, ctx, dbPath, "source.mkv", "processing", "") // pas de miniature

	mkv := genHLSSourceMKV(t, dir, "source.mkv")
	outDir := filepath.Join(dir, "hls", "source")
	params := HLSTranscodeParams{
		SourceAbs: mkv,
		OutDir:    outDir,
		DBPath:    dbPath,
		FileRel:   "source.mkv",
		HLSRel:    "GT/hls/source/master.m3u8",
	}
	if err := RunHLSTranscode(ctx, params); err != nil {
		t.Fatalf("RunHLSTranscode: %v", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// HLS bien finalisé (le média reste lisible)…
	var fp, ts string
	if err := db.QueryRowContext(ctx,
		`SELECT file_path, transcode_status FROM media_files WHERE id=1`).Scan(&fp, &ts); err != nil {
		t.Fatal(err)
	}
	if fp != "GT/hls/source/master.m3u8" || ts != TranscodeReady {
		t.Errorf("DB = (%q,%q), want (GT/hls/source/master.m3u8, ready)", fp, ts)
	}
	// …mais le source est CONSERVÉ faute de miniature liée.
	if _, err := os.Stat(mkv); err != nil {
		t.Errorf("source supprimé alors qu'aucune miniature n'est liée: %v", err)
	}
}

// TestRunHLSTranscode_Failure_KeepsSource : un source invalide fait échouer
// BuildHLS → transcode_status='failed', source CONSERVÉ (fallback remux legacy).
func TestRunHLSTranscode_Failure_KeepsSource(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared_social.duckdb")
	setupMediaDB(t, ctx, dbPath, "bogus.mkv", "processing", "")

	bogus := filepath.Join(dir, "bogus.mkv")
	if err := os.WriteFile(bogus, []byte("not a video"), 0o644); err != nil {
		t.Fatal(err)
	}
	params := HLSTranscodeParams{
		SourceAbs: bogus,
		OutDir:    filepath.Join(dir, "hls", "bogus"),
		DBPath:    dbPath,
		FileRel:   "bogus.mkv",
		HLSRel:    "GT/hls/bogus/master.m3u8",
	}
	if err := RunHLSTranscode(ctx, params); err == nil {
		t.Fatal("RunHLSTranscode: erreur attendue (source invalide)")
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var ts string
	if err := db.QueryRowContext(ctx,
		`SELECT transcode_status FROM media_files WHERE id=1`).Scan(&ts); err != nil {
		t.Fatal(err)
	}
	if ts != TranscodeFailed {
		t.Errorf("transcode_status = %q, want failed", ts)
	}
	if _, err := os.Stat(bogus); err != nil {
		t.Errorf("source supprimé à tort sur échec: %v", err)
	}
}

// setupMediaDB crée la table media_files et insère une ligne (id=1), puis ferme
// la connexion pour libérer le fichier DuckDB au worker. thumbnailPath vide =>
// thumbnail_path NULL (la garde anti-perte conserve alors le source).
func setupMediaDB(t *testing.T, ctx context.Context, dbPath, fileRel, status, thumbnailPath string) {
	t.Helper()
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureMediaTables(ctx, db); err != nil {
		t.Fatalf("ensureMediaTables: %v", err)
	}
	var thumb any
	if thumbnailPath != "" {
		thumb = thumbnailPath
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO media_files (id, file_path, kind, transcode_status, thumbnail_path) VALUES (1, ?, 'video', ?, ?)`,
		fileRel, status, thumb); err != nil {
		t.Fatalf("insert media_files: %v", err)
	}
}

// genHLSSourceMKV génère un MKV synthétique H.264 + 2 pistes Opus.
func genHLSSourceMKV(t *testing.T, dir, name string) string {
	t.Helper()
	src := filepath.Join(dir, name)
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=15:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=880:duration=2",
		"-map", "0:v", "-map", "1:a", "-map", "2:a",
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "libopus",
		src)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("génération MKV: %v\n%s", err, out)
	}
	return src
}

// TestDetectHLSNeeded couvre le déclencheur : MKV multipiste → HLS requis ;
// MP4 H.264/AAC mono-piste → servi en direct.
func TestDetectHLSNeeded(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent — TestDetectHLSNeeded ignoré")
	}
	ctx := context.Background()
	dir := t.TempDir()

	mkv := genHLSSourceMKV(t, dir, "multi.mkv")
	if ok, err := DetectHLSNeeded(ctx, mkv); err != nil || !ok {
		t.Errorf("MKV multipiste: DetectHLSNeeded = (%v, %v), want (true, nil)", ok, err)
	}

	mp4 := genMonoMP4(t, dir, "mono.mp4")
	if ok, err := DetectHLSNeeded(ctx, mp4); err != nil || ok {
		t.Errorf("MP4 mono-piste: DetectHLSNeeded = (%v, %v), want (false, nil)", ok, err)
	}
}

// genMonoMP4 génère un MP4 web-natif mono-piste (H.264 + 1 AAC).
func genMonoMP4(t *testing.T, dir, name string) string {
	t.Helper()
	dst := filepath.Join(dir, name)
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=15:duration=1",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-map", "0:v", "-map", "1:a",
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac",
		"-movflags", "+faststart", dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("génération MP4: %v\n%s", err, out)
	}
	return dst
}

// TestMarkTranscodeProcessing_CompareAndSet : le verrou de transcodage est un
// compare-and-set. 1re acquisition sur une ligne non-'processing' → acquired=true
// + status/horodatage posés ; 2e acquisition immédiate ('processing' FRAIS, un
// autre worker transcode) → acquired=false et la ligne est INCHANGÉE ; après
// vieillissement artificiel du timestamp au-delà de transcodeStaleAfter (orphelin
// de crash simulé) → ré-acquisition acquired=true. Déterministe (pas de ffmpeg).
// Chaque accès direct ferme sa connexion avant l'appel suivant (un seul handle
// RW DuckDB par fichier).
func TestMarkTranscodeProcessing_CompareAndSet(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared_social.duckdb")
	setupMediaDB(t, ctx, dbPath, "clip.mkv", "", "") // status vide au départ

	// 1re acquisition : ligne pas 'processing' → verrou acquis, status + horodatage posés.
	acquired, err := MarkTranscodeProcessing(ctx, dbPath, "clip.mkv")
	if err != nil {
		t.Fatalf("MarkTranscodeProcessing (1re): %v", err)
	}
	if !acquired {
		t.Fatal("1re acquisition: acquired = false, want true")
	}
	status, startedAt := readTranscodeState(t, ctx, dbPath, "clip.mkv")
	if status != TranscodeProcessing {
		t.Errorf("transcode_status = %q, want processing", status)
	}
	if !startedAt.Valid {
		t.Fatal("transcode_started_at doit être horodaté après acquisition")
	}
	firstStamp := startedAt.Time

	// 2e acquisition immédiate : 'processing' frais → refusée, ligne inchangée.
	acquired, err = MarkTranscodeProcessing(ctx, dbPath, "clip.mkv")
	if err != nil {
		t.Fatalf("MarkTranscodeProcessing (2e): %v", err)
	}
	if acquired {
		t.Fatal("2e acquisition sur 'processing' frais: acquired = true, want false (worker déjà en cours)")
	}
	if _, after := readTranscodeState(t, ctx, dbPath, "clip.mkv"); !after.Time.Equal(firstStamp) {
		t.Errorf("transcode_started_at modifié par une acquisition refusée: %v -> %v", firstStamp, after.Time)
	}

	// Vieillissement artificiel au-delà de transcodeStaleAfter (orphelin de crash).
	func() {
		db, err := sql.Open("duckdb", dbPath)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() //nolint:errcheck
		if _, err := db.ExecContext(ctx,
			`UPDATE media_files SET transcode_started_at = now() - INTERVAL 3 HOUR WHERE file_path='clip.mkv'`); err != nil {
			t.Fatalf("vieillissement artificiel: %v", err)
		}
	}()

	// Ré-acquisition sur 'processing' périmé → verrou repris (récupération d'orphelin).
	acquired, err = MarkTranscodeProcessing(ctx, dbPath, "clip.mkv")
	if err != nil {
		t.Fatalf("MarkTranscodeProcessing (3e): %v", err)
	}
	if !acquired {
		t.Fatal("ré-acquisition sur 'processing' périmé: acquired = false, want true")
	}
}

// readTranscodeState lit (transcode_status, transcode_started_at) de la ligne
// media_files identifiée par file_path, en fermant sa connexion (handle RW unique).
func readTranscodeState(t *testing.T, ctx context.Context, dbPath, fileRel string) (string, sql.NullTime) {
	t.Helper()
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck
	var status string
	var startedAt sql.NullTime
	if err := db.QueryRowContext(ctx,
		`SELECT transcode_status, transcode_started_at FROM media_files WHERE file_path = ?`, fileRel).
		Scan(&status, &startedAt); err != nil {
		t.Fatal(err)
	}
	return status, startedAt
}
