//go:build cgo

package ops

// media_capture_recalc_cgo_test.go — recalcul des fenêtres des médias indexés avant la règle par
// source : simulation sans écriture, écriture idempotente, lignes déjà marquées intactes, et
// passage par IndexMedia.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	platform_duckdb "levelup/go-api/internal/platform/duckdb"
)

// legacyFixture : une ligne telle que l'ancienne indexation l'écrivait (début = H, fin = H + D).
type legacyFixture struct {
	name     string
	kind     string
	h        time.Time
	duration *float64
	source   *string // non nil = ligne déjà marquée
}

func f64(v float64) *float64 { return &v }
func strp(v string) *string  { return &v }

var (
	hReplay      = time.Date(2026, 9, 1, 19, 46, 53, 0, time.UTC)
	hReplayNoDur = time.Date(2026, 9, 1, 20, 54, 51, 0, time.UTC)
	hRecord      = time.Date(2026, 4, 19, 15, 10, 54, 0, time.UTC)
	hXbox        = time.Date(2019, 12, 12, 21, 49, 0, 0, time.UTC)
	hFileDate    = time.Date(2026, 7, 16, 20, 18, 27, 0, time.UTC)
)

func legacyFixtures() []legacyFixture {
	return []legacyFixture{
		{"Replay 2026-09-01 21-46-53.mkv", mediaKindVideo, hReplay, f64(28.872), nil},
		{"2026-04-19 17-10-54.mkv", mediaKindVideo, hRecord, f64(600), nil},
		{"Halo_5_Guardians-2019-12-12_22h49.mp4", mediaKindVideo, hXbox, f64(30), nil},
		{"clip.mp4", mediaKindVideo, hFileDate, f64(27), nil},
		{"shot.png", mediaKindImage, hFileDate, f64(0), nil},
		// Nom daté mais heure stockée différente : H venait de la date du fichier.
		{"Replay 2026-07-03 21-38-44.mkv", mediaKindVideo, hFileDate, f64(27.727), nil},
		// Durée inconnue : la fin devient H.
		{"Replay 2026-09-01 22-54-51.mkv", mediaKindVideo, hReplayNoDur, nil, nil},
		// Déjà marquée : jamais reprise.
		{"Replay 2026-09-02 10-00-00.mkv", mediaKindVideo, hReplay, f64(30), strp(captureSourceOBSReplay)},
	}
}

func insertLegacyFixtures(t *testing.T, db *sql.DB, rows []legacyFixture) {
	t.Helper()
	for _, r := range rows {
		var end *time.Time
		switch {
		case r.kind == mediaKindImage:
			end = &r.h
		case r.duration != nil:
			e := r.h.Add(time.Duration(*r.duration * float64(time.Second)))
			end = &e
		}
		if _, err := db.ExecContext(context.Background(), `
			INSERT INTO media_files (player_slug, file_path, file_name, kind,
				capture_start_utc, capture_end_utc, duration_seconds, capture_time_source)
			VALUES ('spartan', ?, ?, ?, ?, ?, ?, ?)`,
			"spartan/"+r.name, r.name, r.kind, r.h, end, r.duration, r.source); err != nil {
			t.Fatalf("insert %s: %v", r.name, err)
		}
	}
}

type storedWindow struct {
	start, end sql.NullTime
	source     sql.NullString
}

func readWindow(t *testing.T, db *sql.DB, name string) storedWindow {
	t.Helper()
	var w storedWindow
	if err := db.QueryRowContext(context.Background(), `
		SELECT capture_start_utc, capture_end_utc, capture_time_source
		FROM media_files WHERE file_name = ?`, name).Scan(&w.start, &w.end, &w.source); err != nil {
		t.Fatalf("lecture %s: %v", name, err)
	}
	return w
}

func assertWindow(t *testing.T, db *sql.DB, name string, start, end time.Time, source string) {
	t.Helper()
	w := readWindow(t, db, name)
	if !w.start.Valid || !w.start.Time.Equal(start) || !w.end.Valid || !w.end.Time.Equal(end) {
		t.Errorf("%s : fenêtre [%v, %v], want [%v, %v]", name, w.start.Time, w.end.Time, start, end)
	}
	if !w.source.Valid || w.source.String != source {
		t.Errorf("%s : source %v, want %s", name, w.source, source)
	}
}

func TestRecalcCaptureTimes_DryRunThenWriteIsIdempotent(t *testing.T) {
	ctx := context.Background()
	loc := mustParis(t)
	_, db := openDiagDB(t)
	if err := ensureMediaTables(ctx, db); err != nil {
		t.Fatalf("ensureMediaTables: %v", err)
	}
	insertLegacyFixtures(t, db, legacyFixtures())

	dry, err := recalcCaptureTimes(ctx, db, loc, true)
	if err != nil {
		t.Fatalf("simulation: %v", err)
	}
	if dry.Examined != 7 || dry.Shifted != 5 {
		t.Errorf("simulation : %d examinées / %d décalées, want 7 / 5", dry.Examined, dry.Shifted)
	}
	wantBySource := map[string]CaptureRecalcCount{
		captureSourceOBSReplay:    {Examined: 2, Shifted: 2, NoDuration: 1},
		captureSourceOBSRecording: {Examined: 1},
		captureSourceXboxConsole:  {Examined: 1, Shifted: 1},
		captureSourceFileDate:     {Examined: 3, Shifted: 2},
	}
	for k, want := range wantBySource {
		if got := dry.BySource[k]; got != want {
			t.Errorf("simulation, source %s : %+v, want %+v", k, got, want)
		}
	}
	if w := readWindow(t, db, "Replay 2026-09-01 21-46-53.mkv"); w.source.Valid || !w.start.Time.Equal(hReplay) {
		t.Fatalf("la simulation a écrit : %+v", w)
	}

	if _, err := recalcCaptureTimes(ctx, db, loc, false); err != nil {
		t.Fatalf("écriture: %v", err)
	}
	assertWindow(t, db, "Replay 2026-09-01 21-46-53.mkv", hReplay.Add(-28872*time.Millisecond), hReplay, captureSourceOBSReplay)
	assertWindow(t, db, "2026-04-19 17-10-54.mkv", hRecord, hRecord.Add(10*time.Minute), captureSourceOBSRecording)
	assertWindow(t, db, "Halo_5_Guardians-2019-12-12_22h49.mp4", hXbox.Add(-30*time.Second), hXbox, captureSourceXboxConsole)
	assertWindow(t, db, "clip.mp4", hFileDate.Add(-27*time.Second), hFileDate, captureSourceFileDate)
	assertWindow(t, db, "shot.png", hFileDate, hFileDate, captureSourceFileDate)
	assertWindow(t, db, "Replay 2026-07-03 21-38-44.mkv", hFileDate.Add(-27727*time.Millisecond), hFileDate, captureSourceFileDate)
	assertWindow(t, db, "Replay 2026-09-01 22-54-51.mkv", hReplayNoDur, hReplayNoDur, captureSourceOBSReplay)
	assertWindow(t, db, "Replay 2026-09-02 10-00-00.mkv", hReplay, hReplay.Add(30*time.Second), captureSourceOBSReplay)

	again, err := recalcCaptureTimes(ctx, db, loc, false)
	if err != nil {
		t.Fatalf("second passage: %v", err)
	}
	if again.Examined != 0 || again.Shifted != 0 {
		t.Errorf("second passage : %d examinées / %d décalées, want 0 / 0 (idempotent)", again.Examined, again.Shifted)
	}
}

// TestRecalcCaptureTimes_WithoutMarkerColumn : une base jamais indexée depuis la règle se
// simule (toutes ses lignes sont antérieures) mais ne s'écrit pas sans ensureMediaTables.
func TestRecalcCaptureTimes_WithoutMarkerColumn(t *testing.T) {
	ctx := context.Background()
	_, db := openDiagDB(t)
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE media_files (id INTEGER, file_name VARCHAR, kind VARCHAR,
			capture_start_utc TIMESTAMPTZ, capture_end_utc TIMESTAMPTZ, duration_seconds DOUBLE);
		INSERT INTO media_files VALUES (1, 'Replay 2026-09-01 21-46-53.mkv', 'video',
			TIMESTAMPTZ '2026-09-01 19:46:53+00', TIMESTAMPTZ '2026-09-01 19:47:21+00', 28)`); err != nil {
		t.Fatal(err)
	}
	dry, err := recalcCaptureTimes(ctx, db, mustParis(t), true)
	if err != nil || dry.Examined != 1 || dry.Shifted != 1 {
		t.Errorf("simulation sans colonne : %+v, err %v ; want 1 examinée, 1 décalée", dry, err)
	}
	if _, err := recalcCaptureTimes(ctx, db, mustParis(t), false); err == nil {
		t.Error("écriture sans colonne : want une erreur")
	}
}

// TestIndexMedia_RecalcsLegacyWindows : l'indexation corrige les lignes antérieures de la base,
// y compris celles d'un autre joueur ou dont le fichier n'est plus dans le dossier.
func TestIndexMedia_RecalcsLegacyWindows(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	socialPath := filepath.Join(dir, "shared_social.duckdb")
	seed, err := sql.Open("duckdb", socialPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureMediaTables(ctx, seed); err != nil {
		t.Fatal(err)
	}
	insertLegacyFixtures(t, seed, legacyFixtures()[:1])
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}
	capturesDir := filepath.Join(dir, "captures", "spartan")
	if err := os.MkdirAll(capturesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := IndexMedia(ctx, MediaIndexOptions{
		PlayerDBPath:       filepath.Join(dir, "player.duckdb"),
		SharedSocialDBPath: socialPath,
		CapturesDir:        capturesDir,
		CapturesBase:       filepath.Join(dir, "captures"),
		Gamertag:           "spartan",
		Timezone:           "Europe/Paris",
	}); err != nil {
		t.Fatalf("IndexMedia: %v", err)
	}
	report, err := RecalcMediaCaptureTimes(ctx, CaptureRecalcOptions{DBPath: socialPath, Timezone: "Europe/Paris", DryRun: true})
	if err != nil {
		t.Fatalf("simulation après IndexMedia: %v", err)
	}
	if report.Examined != 0 {
		t.Errorf("après IndexMedia, %d ligne(s) encore à recalculer, want 0", report.Examined)
	}
	check, release, err := platform_duckdb.OpenReadForQuery(socialPath)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	assertWindow(t, check, "Replay 2026-09-01 21-46-53.mkv", hReplay.Add(-28872*time.Millisecond), hReplay, captureSourceOBSReplay)
}
