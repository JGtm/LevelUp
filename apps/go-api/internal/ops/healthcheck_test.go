// Package ops — healthcheck_test.go : tests unitaires pour les checks d'intégrité.
//
//go:build integration

package ops

import (
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
