//go:build cgo

// Package ops — test E2E ULTIME du bug WAL DuckDB #7659 sur shared_social.
//
// Reproduit le scénario prod EXACT :
//  1. Sub-process : ouvre shared_social via NewConnector(timezone), fait
//     IndexMedia (avec CHECKPOINT explicite Phase 3), exit avec os.Exit(0)
//     (PAS de Close gracieux).
//  2. Parent : relit shared_social via NewConnector(timezone) (= comme le
//     pool prod au boot).
//  3. Vérifie : reopen doit réussir, les data sont préservées (CHECKPOINT
//     a fusionné le WAL dans le fichier principal).
//
// Différence avec media_checkpoint_test.go : ce test utilise un SUB-PROCESS
// (équivalent kill brutal — pas de defer Close, pas de cleanup Go) au lieu
// d'un Close gracieux. C'est la condition la plus proche de la réalité prod
// (Air SIGKILL Windows) atteignable en test.

package ops

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_KillBrutal_ReopenAfterCrashedIndexMedia simule un crash brutal
// (sub-process exit sans Close gracieux) puis vérifie que le reopen via
// NewConnector(timezone) marche.
//
// Ce test n'est pas un sub-process Go classique (exec.Command sur soi-même
// avec une variable d'env pour discriminer le mode child) — pour rester
// simple et auto-contenu. Cf. testing.T.RunParallel + helper exit non-graceful.
func TestE2E_KillBrutal_ReopenAfterCrashedIndexMedia(t *testing.T) {
	if testing.Short() {
		t.Skip("skip en mode short — sub-process spawn")
	}

	// Le mode "child" est sélectionné via une env var. Le parent lance
	// `go test -run TestE2E_KillBrutal_ReopenAfterCrashedIndexMedia` avec
	// cette env var positionnée → le sub-process exécute le helper d'écriture
	// puis exit os.Exit(0) brutalement (sans defer Close).
	if os.Getenv("LEVELUP_KILL_BRUTAL_CHILD") == "1" {
		childWriteAndExitBrutal(t)
		return
	}

	// Parent : prépare un dir partagé, lance le child, vérifie l'état post-crash.
	dir := t.TempDir()
	socialPath := filepath.Join(dir, "shared_social.duckdb")
	matchesPath := filepath.Join(dir, "shared_matches.duckdb")

	setupE2EMatchRegistry(t, dir, "match-kill-brutal", 17)
	_ = matchesPath

	capturesDir := filepath.Join(dir, "captures", "spartan")
	if err := os.MkdirAll(capturesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(capturesDir, "Replay 2026-04-19 17-10-54.mp4"), []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Spawn sub-process avec le test self-call.
	cmd := exec.Command(os.Args[0],
		"-test.run=^TestE2E_KillBrutal_ReopenAfterCrashedIndexMedia$",
		"-test.v",
	)
	cmd.Env = append(os.Environ(),
		"LEVELUP_KILL_BRUTAL_CHILD=1",
		"LEVELUP_KILL_BRUTAL_DIR="+dir,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Le sub-process appelle os.Exit(0) après son boulot — exit code 0
		// est attendu. Si exit != 0 c'est qu'il a planté pour autre raison.
		t.Logf("sub-process output:\n%s", string(out))
		// Tolérance : tant que le boulot d'écriture a eu lieu (vérifié plus
		// bas), l'exit code peut différer (testing harness exit hook).
		// On loggue uniquement pour debug.
		_ = err
	}
	// Mais le sub-process doit avoir écrit dans la DB
	if !strings.Contains(string(out), "CHILD_WROTE_OK") {
		t.Fatalf("sub-process n'a pas écrit (output sans marqueur):\n%s", string(out))
	}

	// Parent : reopen via NewConnector(timezone) — comme le pool prod au boot.
	db := openWithTimezone(t, socialPath, "Europe/Paris")
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("REOPEN PING FAILED après sub-process crash :\n  %v\n"+
			"Le CHECKPOINT explicite d'IndexMedia n'a pas suffi à vider le WAL.\n"+
			"Bug DuckDB #7659 toujours actif.", err)
	}

	// Vérifier que les data sont là.
	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM media_files").Scan(&n); err != nil {
		t.Fatalf("query media_files: %v", err)
	}
	if n < 1 {
		t.Errorf("media perdu après crash : count=%d", n)
	}
}

// childWriteAndExitBrutal est appelé dans le sub-process. Il ouvre
// shared_social via NewConnector(timezone), fait IndexMedia, puis fait
// os.Exit(0) BRUTAL — pas de defer Close, simule un kill.
//
// La fonction PRINT "CHILD_WROTE_OK" sur stdout pour signaler au parent que
// l'écriture a réussi.
func childWriteAndExitBrutal(t *testing.T) {
	dir := os.Getenv("LEVELUP_KILL_BRUTAL_DIR")
	if dir == "" {
		t.Fatal("LEVELUP_KILL_BRUTAL_DIR non set dans le child")
	}
	socialPath := filepath.Join(dir, "shared_social.duckdb")
	matchesPath := filepath.Join(dir, "shared_matches.duckdb")
	capturesDir := filepath.Join(dir, "captures", "spartan")

	ctx := context.Background()

	// IndexMedia complet → fait CHECKPOINT à la fin grâce au fix Phase 3.
	if _, err := IndexMedia(ctx, MediaIndexOptions{
		PlayerDBPath:        filepath.Join(dir, "player.duckdb"),
		SharedSocialDBPath:  socialPath,
		SharedMatchesDBPath: matchesPath,
		CapturesDir:         capturesDir,
		CapturesBase:        filepath.Join(dir, "captures"),
		Gamertag:            "spartan",
		Timezone:            "Europe/Paris",
	}); err != nil {
		t.Fatalf("CHILD IndexMedia: %v", err)
	}

	// Marqueur pour le parent.
	os.Stdout.WriteString("CHILD_WROTE_OK\n")
	os.Stdout.Sync()

	// EXIT BRUTAL — pas de defer Close, simule un SIGKILL.
	// os.Exit court-circuite le harness testing.T.Cleanup → cohérent avec
	// un kill par Air sur Windows.
	os.Exit(0)
}

// Garantit que sql.DB est importé pour l'usage ailleurs (lint).
var _ = sql.ErrNoRows
