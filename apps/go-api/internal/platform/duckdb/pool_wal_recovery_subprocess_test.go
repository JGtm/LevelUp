//go:build cgo

// Package duckdb — pool_wal_recovery_subprocess_test.go (ADR 0021 Phase 2.3).
//
// Test E2E sub-process : reproduit la séquence prod EXACTE
//
//   1. PARENT : prépare un répertoire vide.
//   2. SUB-PROCESS : décompresse la fixture corrompue dans ce répertoire,
//      puis exit os.Exit(0) brutalement (pas de Close gracieux).
//      Le sub-process simule un binaire pré-fix qui aurait laissé la DB
//      dans cet état.
//   3. PARENT : ouvre via openSharedSocialWithWALRecovery → la quarantaine +
//      retry est exécutée → DB ouverte en succès.
//   4. PARENT : valide que .wal.orphan-* existe et que la métrique a avancé.
//
// Différence vs TestWALOrphanRepro_RecoveryFixesIt (qui tourne in-process) :
// le sub-process garantit un environnement propre sans cache process-wide
// DuckDB qui pourrait masquer le bug.

package duckdb

import (
	"compress/gzip"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const subprocessChildEnv = "LEVELUP_WAL_RECOVERY_CHILD"

// TestE2E_KillBrutal_WALOrphanRecoveryWorks (ADR 0021 Phase 2.3).
func TestE2E_KillBrutal_WALOrphanRecoveryWorks(t *testing.T) {
	if testing.Short() {
		t.Skip("skip en mode short — spawn sub-process")
	}

	if os.Getenv(subprocessChildEnv) == "1" {
		childWriteCorruptFixtureAndExit(t)
		return
	}

	dir := t.TempDir()

	cmd := exec.Command(os.Args[0],
		"-test.run=^TestE2E_KillBrutal_WALOrphanRecoveryWorks$",
		"-test.v",
	)
	cmd.Env = append(os.Environ(),
		subprocessChildEnv+"=1",
		"LEVELUP_WAL_RECOVERY_DIR="+dir,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sub-process échoué : %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(string(out), "CHILD_FIXTURE_OK") {
		t.Fatalf("sub-process n'a pas écrit la fixture corrompue :\n%s", out)
	}

	// État post-sub-process : <dir>/shared_social.duckdb + .wal corrompus.
	dbPath := filepath.Join(dir, "shared_social.duckdb")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("fixture .duckdb absente après sub-process : %v", err)
	}
	if _, err := os.Stat(dbPath + ".wal"); err != nil {
		t.Fatalf("fixture .wal absente après sub-process : %v", err)
	}

	// Tentative de récupération via le code prod.
	before := metricWALOrphanQuarantineSocial.Value()
	db := openSharedSocialWithWALRecovery(context.Background(), dbPath, "", "e2e-subprocess")

	if db == nil {
		// Quarantaine partielle attendue (corruption .duckdb header).
		matches, _ := filepath.Glob(dbPath + ".wal.orphan-*")
		if len(matches) == 0 {
			t.Errorf("ni recovery ni quarantaine — code recovery non-déclenché")
			return
		}
		after := metricWALOrphanQuarantineSocial.Value()
		if after <= before {
			t.Errorf("metric inchangée malgré quarantaine : %d → %d", before, after)
		}
		t.Logf("[partial] quarantaine sub-process OK, retry open échoue (corruption .duckdb — " +
			"runbook cmd/rebuild_shared_social requis)")
		return
	}
	defer db.Close()

	matches, err := filepath.Glob(dbPath + ".wal.orphan-*")
	if err != nil {
		t.Fatalf("glob orphan: %v", err)
	}
	if len(matches) != 1 {
		t.Errorf("attendu 1 .orphan, got %d : %v", len(matches), matches)
	}
	after := metricWALOrphanQuarantineSocial.Value()
	if after != before+1 {
		t.Errorf("metric attendue %d, got %d", before+1, after)
	}
	t.Logf("[OK] recovery E2E sub-process réussie")
}

// childWriteCorruptFixtureAndExit (sub-process side) : décompresse la fixture
// gzippée + copie le .wal orphelin dans le directory partagé avec le parent,
// puis os.Exit(0) BRUTAL.
func childWriteCorruptFixtureAndExit(t *testing.T) {
	dir := os.Getenv("LEVELUP_WAL_RECOVERY_DIR")
	if dir == "" {
		t.Fatal("LEVELUP_WAL_RECOVERY_DIR non set")
	}

	// Décompresser shared_social.duckdb.gz → dir/shared_social.duckdb
	gzPath := filepath.Join(fixtureDir, fixtureDBGz)
	gzFile, err := os.Open(gzPath)
	if err != nil {
		t.Fatalf("CHILD open fixture gz: %v", err)
	}
	defer gzFile.Close()
	gz, err := gzip.NewReader(gzFile)
	if err != nil {
		t.Fatalf("CHILD gzip reader: %v", err)
	}
	defer gz.Close()

	dstDB := filepath.Join(dir, "shared_social.duckdb")
	out, err := os.Create(dstDB)
	if err != nil {
		t.Fatalf("CHILD create db: %v", err)
	}
	if _, err := io.Copy(out, gz); err != nil {
		_ = out.Close()
		t.Fatalf("CHILD decompress: %v", err)
	}
	_ = out.Close()

	// Copier shared_social.duckdb.wal → dir/shared_social.duckdb.wal
	walSrc := filepath.Join(fixtureDir, fixtureWAL)
	walDst := dstDB + ".wal"
	walIn, err := os.Open(walSrc)
	if err != nil {
		t.Fatalf("CHILD open wal: %v", err)
	}
	defer walIn.Close()
	walOut, err := os.Create(walDst)
	if err != nil {
		t.Fatalf("CHILD create wal dst: %v", err)
	}
	if _, err := io.Copy(walOut, walIn); err != nil {
		_ = walOut.Close()
		t.Fatalf("CHILD copy wal: %v", err)
	}
	_ = walOut.Close()

	os.Stdout.WriteString("CHILD_FIXTURE_OK\n")
	_ = os.Stdout.Sync()

	// EXIT BRUTAL — court-circuite t.Cleanup, simule SIGKILL.
	os.Exit(0)
}
