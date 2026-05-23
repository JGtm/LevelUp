//go:build integration

// Package ops — seed_demo_cli_test.go : test E2E du binaire levelup seed-demo.
//
// Construit le binaire via `go build`, lance `levelup seed-demo` sur des
// fixtures DuckDB live, et vérifie les outputs (artifacts + anonymisation +
// configs). Pérennise le smoke test manuel de validation Phase 4.1.
//
// CGO_ENABLED=1 requis (driver duckdb + binaire compilé). Tag integration
// pour ne pas tourner sur les CI sans CGO.
package ops

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestSeedDemoCLI_E2E(t *testing.T) {
	const cliSourceXUID = "1111111111111111"

	// 1. Plante les fixtures via le helper partagé, puis déplace-les vers la
	//    layout PathResolver attendue par le CLI (data/titles/halo_infinite/
	//    {warehouse,players/JGtm}/). seedSourceDBs() retourne des paths temp
	//    plats qu'on relocate sous repoRoot.
	_, srcPlayer, srcShared, srcMeta := seedSourceDBs(t)

	repoRoot := t.TempDir()
	warehouseDir := filepath.Join(repoRoot, "data", "titles", "halo_infinite", "warehouse")
	playerDir := filepath.Join(repoRoot, "data", "titles", "halo_infinite", "players", "JGtm")
	for _, d := range []string{warehouseDir, playerDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustRename(t, srcPlayer, filepath.Join(playerDir, "stats.duckdb"))
	mustRename(t, srcShared, filepath.Join(warehouseDir, "shared_matches_v2.duckdb"))
	mustRename(t, srcMeta, filepath.Join(warehouseDir, "metadata.duckdb"))

	// 2. db_profiles.json v3.0 (format prod nested par titre).
	profiles := `{"version":"3.0","admin":"JGtm","profiles":{"halo_infinite":{` +
		`"JGtm":{"xuid":"` + cliSourceXUID + `","db_path":"data/titles/halo_infinite/players/JGtm/stats.duckdb"}}}}`
	if err := os.WriteFile(filepath.Join(repoRoot, "db_profiles.json"), []byte(profiles), 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. Build le binaire vers un tempdir séparé.
	binPath := filepath.Join(t.TempDir(), "levelup")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}
	build := exec.Command("go", "build", "-o", binPath, "levelup/go-api/cmd/levelup")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	// 4. Lance `levelup seed-demo` avec LEVELUP_REPO_ROOT=tmpDir.
	run := exec.Command(binPath, "seed-demo",
		"--gamertag", "JGtm",
		"--max-matches", "2",
		"--out", "out",
		"--service-tag", "SPTA",
		"--no-media",
	)
	run.Env = append(os.Environ(), "LEVELUP_REPO_ROOT="+repoRoot)
	if out, err := run.CombinedOutput(); err != nil {
		t.Fatalf("seed-demo failed: %v\n%s", err, out)
	}

	// 5. Vérifie présence des artifacts produits.
	outDir := filepath.Join(repoRoot, "out")
	for _, p := range []string{
		filepath.Join(outDir, "warehouse", "metadata.duckdb"),
		filepath.Join(outDir, "warehouse", "shared_matches_v2.duckdb"),
		filepath.Join(outDir, "players", DefaultDemoGamertag, "stats.duckdb"),
		filepath.Join(outDir, "db_profiles.json"),
		filepath.Join(outDir, "app_settings.json"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("missing artifact %s: %v", p, err)
		}
	}

	// 6. Vérifie contenu : tables filtrées, anonymisation, configs JSON.
	//    Réutilise les helpers existants du E2E direct (DRY).
	verifyMetaCopied(t, filepath.Join(outDir, "warehouse", "metadata.duckdb"))
	verifySharedExtracted(t, filepath.Join(outDir, "warehouse", "shared_matches_v2.duckdb"), cliSourceXUID)
	verifyPlayerExtracted(t, filepath.Join(outDir, "players", DefaultDemoGamertag, "stats.duckdb"), cliSourceXUID)
	verifyConfigsWritten(t, outDir, "JGtm", "SPTA", false)
}

func mustRename(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.Rename(src, dst); err != nil {
		t.Fatalf("rename %s -> %s: %v", src, dst, err)
	}
}
