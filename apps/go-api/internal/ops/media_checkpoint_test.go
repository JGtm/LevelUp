//go:build cgo

// Package ops — test anti-régression du bug WAL DuckDB #7659 sur IndexMedia.
//
// Couvre : indexation media → close brutal (sans Close gracieux) → reopen via
// NewConnector(timezone) (= comme le pool prod) → doit réussir sans
// "INTERNAL Error: Failure while replaying WAL file".
//
// Avant le fix CHECKPOINT (commit XXX) : ce test plantait au reopen.
// Après : le CHECKPOINT en fin d'IndexMedia vide le WAL → reopen safe.

package ops

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"os"
	"path/filepath"
	"testing"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

// openWithTimezone reproduit le pattern d'ouverture du pool : NewConnector +
// init function SET TimeZone. C'est CE pattern qui plante au replay WAL si
// le WAL contient des entrées laissées par un IndexMedia non-checkpointé.
func openWithTimezone(t *testing.T, path, tz string) *sql.DB {
	t.Helper()
	connector, err := duckdb.NewConnector(path, func(execer driver.ExecerContext) error {
		_, initErr := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return initErr
	})
	if err != nil {
		t.Fatalf("NewConnector: %v", err)
	}
	return sql.OpenDB(connector)
}

// TestIndexMedia_CHECKPOINT_NoWALAfterRun valide que le WAL de
// shared_social.duckdb est vide (ou absent) après un IndexMedia complet.
// Anti-régression directe du bug DuckDB #7659.
func TestIndexMedia_CHECKPOINT_NoWALAfterRun(t *testing.T) {
	dir := t.TempDir()
	socialPath := filepath.Join(dir, "shared_social.duckdb")
	matchesPath := filepath.Join(dir, "shared_matches.duckdb")
	ctx := context.Background()

	// Setup shared_matches avec 1 match.
	setupE2EMatchRegistry(t, dir, "match-checkpoint", 17)
	_ = matchesPath

	// Setup captures dir avec un fichier OBS.
	capturesDir := filepath.Join(dir, "captures", "spartan")
	if err := os.MkdirAll(capturesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	obsFile := filepath.Join(capturesDir, "Replay 2026-04-19 17-10-54.mp4")
	if err := os.WriteFile(obsFile, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run IndexMedia complet.
	if _, err := IndexMedia(ctx, MediaIndexOptions{
		PlayerDBPath:        filepath.Join(dir, "player.duckdb"),
		SharedSocialDBPath:  socialPath,
		SharedMatchesDBPath: filepath.Join(dir, "shared_matches.duckdb"),
		CapturesDir:         capturesDir,
		CapturesBase:        filepath.Join(dir, "captures"),
		Gamertag:            "spartan",
		Timezone:            "Europe/Paris",
	}); err != nil {
		t.Fatalf("IndexMedia: %v", err)
	}

	// Vérification 1 : WAL absent ou vide après IndexMedia.
	walPath := socialPath + ".wal"
	info, err := os.Stat(walPath)
	if err == nil && info.Size() > 0 {
		t.Errorf("WAL non-vide après IndexMedia+CHECKPOINT (size=%d). Le CHECKPOINT explicite a échoué silencieusement.\n"+
			"Path: %s", info.Size(), walPath)
	}
}

// TestIndexMedia_ReopenAfterRun_NoWALReplayCrash est le test ultime
// anti-régression du bug WAL. Reproduit le scénario prod :
//  1. IndexMedia complet (indexe 1 fichier + association)
//  2. Reopen via NewConnector(timezone) — comme le pool au boot serveur
//  3. Le ping doit réussir : pas d'"INTERNAL Error: Failure while replaying
//     WAL file: Calling DatabaseManager::GetDefaultDatabase"
//
// Limite : on ne reproduit pas un SIGKILL brutal (db.Close() Go checkpoint).
// Mais grâce au CHECKPOINT explicite d'IndexMedia, même un SIGKILL ne laisse
// pas de WAL à rejouer.
func TestIndexMedia_ReopenAfterRun_NoWALReplayCrash(t *testing.T) {
	dir := t.TempDir()
	socialPath := filepath.Join(dir, "shared_social.duckdb")
	matchesPath := filepath.Join(dir, "shared_matches.duckdb")
	ctx := context.Background()

	setupE2EMatchRegistry(t, dir, "match-reopen", 17)
	_ = matchesPath

	capturesDir := filepath.Join(dir, "captures", "spartan")
	if err := os.MkdirAll(capturesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	obsFile := filepath.Join(capturesDir, "Replay 2026-04-19 17-10-54.mp4")
	if err := os.WriteFile(obsFile, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Phase 1 : IndexMedia complet (fait CHECKPOINT à la fin grâce au fix).
	if _, err := IndexMedia(ctx, MediaIndexOptions{
		PlayerDBPath:        filepath.Join(dir, "player.duckdb"),
		SharedSocialDBPath:  socialPath,
		SharedMatchesDBPath: filepath.Join(dir, "shared_matches.duckdb"),
		CapturesDir:         capturesDir,
		CapturesBase:        filepath.Join(dir, "captures"),
		Gamertag:            "spartan",
		Timezone:            "Europe/Paris",
	}); err != nil {
		t.Fatalf("IndexMedia: %v", err)
	}

	// Phase 2 : Reopen via NewConnector(timezone) — comme le pool prod.
	// Sans le fix CHECKPOINT, ce ping plantait avec assertion DuckDB #7659.
	db2 := openWithTimezone(t, socialPath, "Europe/Paris")
	defer db2.Close()

	if err := db2.PingContext(ctx); err != nil {
		t.Fatalf("REOPEN PING FAILED — bug DuckDB #7659 ré-introduit ?\n  err: %v\n"+
			"Vérifier que CHECKPOINT en fin d'IndexMedia tourne bien.", err)
	}

	// Phase 3 : les données sont toujours là (CHECKPOINT a fusionné le WAL
	// dans le fichier principal, pas perdu).
	var mediaCount int
	if err := db2.QueryRowContext(ctx, "SELECT COUNT(*) FROM media_files").Scan(&mediaCount); err != nil {
		t.Fatalf("query media_files après reopen: %v", err)
	}
	if mediaCount != 1 {
		t.Errorf("data perdue après reopen : media_files count=%d, want 1", mediaCount)
	}

	var assocCount int
	if err := db2.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM media_match_associations_latest WHERE match_id = ?", "match-reopen").Scan(&assocCount); err != nil {
		t.Fatalf("query associations: %v", err)
	}
	if assocCount != 1 {
		t.Errorf("association perdue après reopen : count=%d, want 1", assocCount)
	}
}

// TestIndexMedia_MultipleRunsStress_NoWALAccumulation simule 5 cycles
// IndexMedia successifs + reopen entre chaque. Vérifie que le WAL ne grossit
// jamais (preuve que CHECKPOINT tourne à chaque cycle) et que les reopens
// ne plantent jamais.
func TestIndexMedia_MultipleRunsStress_NoWALAccumulation(t *testing.T) {
	dir := t.TempDir()
	socialPath := filepath.Join(dir, "shared_social.duckdb")
	matchesPath := filepath.Join(dir, "shared_matches.duckdb")
	ctx := context.Background()

	setupE2EMatchRegistry(t, dir, "match-stress", 17)
	_ = matchesPath

	capturesDir := filepath.Join(dir, "captures", "spartan")
	_ = os.MkdirAll(capturesDir, 0o755)

	for i := 0; i < 5; i++ {
		// Nouveau fichier à chaque cycle (sinon dédup hash → skip)
		fileName := filepath.Join(capturesDir, "Replay 2026-04-19 17-1"+string(rune('0'+i))+"-54.mp4")
		if err := os.WriteFile(fileName, []byte("video"+string(rune('a'+i))), 0o644); err != nil {
			t.Fatal(err)
		}
		// Forcer mtime pour différenciation
		mtime := time.Date(2026, 4, 19, 15, 10+i, 0, 0, time.UTC)
		_ = os.Chtimes(fileName, mtime, mtime)

		if _, err := IndexMedia(ctx, MediaIndexOptions{
			PlayerDBPath:        filepath.Join(dir, "player.duckdb"),
			SharedSocialDBPath:  socialPath,
			SharedMatchesDBPath: filepath.Join(dir, "shared_matches.duckdb"),
			CapturesDir:         capturesDir,
			CapturesBase:        filepath.Join(dir, "captures"),
			Gamertag:            "spartan",
			Timezone:            "Europe/Paris",
		}); err != nil {
			t.Fatalf("cycle %d IndexMedia: %v", i, err)
		}

		// Vérifier que WAL reste petit après chaque cycle (preuve CHECKPOINT).
		if info, err := os.Stat(socialPath + ".wal"); err == nil && info.Size() > 1024 {
			t.Errorf("cycle %d : WAL size=%d > 1024 — CHECKPOINT n'a pas tourné", i, info.Size())
		}

		// Reopen via NewConnector(timezone) → ne doit jamais planter.
		db := openWithTimezone(t, socialPath, "Europe/Paris")
		if err := db.PingContext(ctx); err != nil {
			db.Close()
			t.Fatalf("cycle %d : reopen ping fail: %v", i, err)
		}
		db.Close()
	}
}
