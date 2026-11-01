// Package ops — healthcheck_test.go : tests unitaires pour les checks d'intégrité.
//
//go:build integration

package ops

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckDirExists_ExistingDir(t *testing.T) {
	dir := t.TempDir()
	check := checkDirExists("test-dir", dir)
	if !check.OK {
		t.Errorf("expected OK for existing dir, got: %s", check.Message)
	}
}

func TestCheckDirExists_MissingDir(t *testing.T) {
	check := checkDirExists("test-dir", "/nonexistent/path/xyz")
	if check.OK {
		t.Error("expected NOT OK for missing dir")
	}
}

func TestCheckFileExists_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	check := checkFileExists("test-file", f)
	if !check.OK {
		t.Errorf("expected OK for existing file, got: %s", check.Message)
	}
}

func TestCheckFileExists_MissingFile(t *testing.T) {
	check := checkFileExists("test-file", "/nonexistent/file.txt")
	if check.OK {
		t.Error("expected NOT OK for missing file")
	}
}

func TestHealthReport_Summary(t *testing.T) {
	report := HealthReport{
		OK: true,
		Checks: []HealthCheck{
			{Name: "check1", OK: true, Message: "ok"},
			{Name: "check2", OK: true, Message: "ok"},
		},
	}
	s := report.Summary()
	if s == "" {
		t.Error("expected non-empty summary")
	}
}

func TestHealthReport_Summary_WithFailures(t *testing.T) {
	report := HealthReport{
		OK: false,
		Checks: []HealthCheck{
			{Name: "check1", OK: true, Message: "ok"},
			{Name: "check2", OK: false, Message: "database missing"},
		},
	}
	s := report.Summary()
	if s == "" {
		t.Error("expected non-empty summary with failures")
	}
}

// TestRunHealthcheck_FichiersDeConfigurationExplicites — les chemins résolus par la configuration
// (LEVELUP_DB_PROFILES / LEVELUP_APP_SETTINGS, fichiers montés hors de DataRoot) priment sur ceux
// du PathResolver : le contrôle porte sur le fichier réellement lu.
func TestRunHealthcheck_FichiersDeConfigurationExplicites(t *testing.T) {
	root, elsewhere := t.TempDir(), t.TempDir()
	dbp := filepath.Join(elsewhere, "db_profiles.json")
	app := filepath.Join(elsewhere, "app_settings.json")
	for _, p := range []string{dbp, app} {
		if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	report := RunHealthcheck(context.Background(), HealthcheckOptions{RepoRoot: root, DBProfilesPath: dbp, AppSettingsPath: app})
	for _, c := range report.Checks {
		if (c.Name == "db_profiles.json" || c.Name == "app_settings.json") && !c.OK {
			t.Errorf("%s : %s — le chemin explicite %s n'a pas été utilisé", c.Name, c.Message, elsewhere)
		}
	}
	report = RunHealthcheck(context.Background(), HealthcheckOptions{RepoRoot: root})
	var absent int
	for _, c := range report.Checks {
		if (c.Name == "db_profiles.json" || c.Name == "app_settings.json") && !c.OK {
			absent++
		}
	}
	if absent != 2 {
		t.Errorf("sans chemin explicite, les deux fichiers doivent manquer sous %s (absents=%d)", root, absent)
	}
}
