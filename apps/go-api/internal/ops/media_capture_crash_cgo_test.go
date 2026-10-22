//go:build cgo

package ops

// media_capture_crash_cgo_test.go — une base shared_social ANTÉRIEURE à la colonne
// capture_time_source (données + index sur media_files) reçoit la colonne à la première
// indexation. Ce test coupe le processus juste après la préparation de la base (ajout de la
// colonne et recalcul des fenêtres, le début d'IndexMedia), sans Close ni CHECKPOINT final, puis
// rouvre la base : elle doit s'ouvrir, avec la colonne et, fuseau connu, les fenêtres recalculées.
// Un ALTER resté dans le WAL jusqu'au CHECKPOINT de fin d'indexation (hash, ffprobe, ffmpeg entre
// les deux) rend la base illisible à la réouverture (même famille que DuckDB #7659, cf.
// IndexMedia). Le scénario « fuseau inconnu » (recalcul sauté, aucune écriture, donc aucun
// CHECKPOINT du recalcul) isole le CHECKPOINT de changement de schéma de ensureMediaTables.

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	platform_duckdb "levelup/go-api/internal/platform/duckdb"
)

const (
	envCrashChildDir = "LEVELUP_MEDIA_COLUMN_CRASH_DIR"
	// envCrashChildNoTZ : le sous-processus indexe sans fuseau, le recalcul est sauté et n'écrit
	// rien : seul le CHECKPOINT de changement de schéma protège alors la base.
	envCrashChildNoTZ = "LEVELUP_MEDIA_COLUMN_CRASH_NOTZ"
)

func TestIndexMediaPrepare_NewColumnSurvivesCrash(t *testing.T) {
	if dir := os.Getenv(envCrashChildDir); dir != "" {
		crashChildPrepareAndExit(t, filepath.Join(dir, "shared_social.duckdb"), os.Getenv(envCrashChildNoTZ) == "1")
		return
	}
	if testing.Short() {
		t.Skip("sous-processus : hors mode short")
	}
	t.Run("recalcul fait", func(t *testing.T) { runCrashScenario(t, false) })
	t.Run("recalcul sauté, fuseau inconnu", func(t *testing.T) { runCrashScenario(t, true) })
}

// runCrashScenario : base antérieure, sous-processus coupé après la préparation, réouverture.
func runCrashScenario(t *testing.T, noTZ bool) {
	t.Helper()
	dir := t.TempDir()
	socialPath := filepath.Join(dir, "shared_social.duckdb")
	seedPreColumnSocialDB(t, socialPath)

	cmd := exec.Command(os.Args[0], "-test.run=^TestIndexMediaPrepare_NewColumnSurvivesCrash$", "-test.v")
	flag := "0"
	if noTZ {
		flag = "1"
	}
	cmd.Env = append(os.Environ(), envCrashChildDir+"="+dir, envCrashChildNoTZ+"="+flag)
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "CHILD_PREPARED_OK") {
		t.Fatalf("le sous-processus n'a pas préparé la base :\n%s", out)
	}

	db := openWithTimezone(t, socialPath, "Europe/Paris")
	defer db.Close()
	ctx := context.Background()
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_files WHERE capture_time_source IS NOT NULL`).Scan(&n); err != nil {
		t.Fatalf("RÉOUVERTURE après arrêt brutal : %v", err)
	}
	if noTZ {
		if n != 0 {
			t.Errorf("sans fuseau, %d ligne(s) marquée(s), want 0", n)
		}
		return
	}
	if n != 2 {
		t.Errorf("après arrêt brutal, %d ligne(s) marquée(s), want 2 (recalcul perdu)", n)
	}
	assertWindow(t, db, "Replay 2026-09-01 21-46-53.mkv", hReplay.Add(-28872*time.Millisecond), hReplay, captureSourceOBSReplay)
}

// seedPreColumnSocialDB : media_files au schéma d'avant la colonne, avec ses index de migration
// et deux lignes antérieures, base fermée proprement.
func seedPreColumnSocialDB(t *testing.T, path string) {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureMediaTables(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`ALTER TABLE media_files DROP COLUMN capture_time_source`,
		`CREATE INDEX IF NOT EXISTS idx_mf_player_slug ON media_files(player_slug)`,
		`CREATE INDEX IF NOT EXISTS idx_mf_player_stem ON media_files(player_slug, file_stem)`,
		`INSERT INTO media_files (player_slug, file_path, file_name, file_stem, kind,
			capture_start_utc, capture_end_utc, duration_seconds)
		 VALUES ('spartan', 'spartan/a.mkv', 'Replay 2026-09-01 21-46-53.mkv', 'Replay 2026-09-01 21-46-53', 'video',
			TIMESTAMPTZ '2026-09-01 19:46:53+00', TIMESTAMPTZ '2026-09-01 19:47:21.872+00', 28.872),
		        ('spartan', 'spartan/b.png', 'shot.png', 'shot', 'image',
			TIMESTAMPTZ '2026-07-16 20:18:27+00', TIMESTAMPTZ '2026-07-16 20:18:27+00', 0)`,
		`CHECKPOINT`,
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("préparation de la base antérieure (%.40s…) : %v", q, err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

// crashChildPrepareAndExit : le début d'IndexMedia (même handle, même préparation), puis arrêt
// sans Close ni CHECKPOINT.
func crashChildPrepareAndExit(t *testing.T, socialPath string, noTZ bool) {
	ctx := context.Background()
	handle, err := platform_duckdb.OpenReadWriteShared(socialPath, "Europe/Paris")
	if err != nil {
		t.Fatalf("CHILD ouverture : %v", err)
	}
	db := handle.SQLDb()
	if err := ensureMediaTables(ctx, db); err != nil {
		t.Fatalf("CHILD ensureMediaTables : %v", err)
	}
	loc := mustParis(t)
	if noTZ {
		loc = nil
	}
	if err := recalcBeforeIndex(ctx, db, loc, "spartan"); err != nil {
		t.Fatalf("CHILD recalcul : %v", err)
	}
	_, _ = os.Stdout.WriteString("CHILD_PREPARED_OK\n")
	_ = os.Stdout.Sync()
	os.Exit(0)
}
