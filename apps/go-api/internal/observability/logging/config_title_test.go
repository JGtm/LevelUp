package logging

import (
	"os"
	"testing"
)

// TestConfig_WithTitleNamespace — MT-05 (PMT-10 PR-4) : namespacing du LogsDir
// par titre, copie (original non muté), no-op si vide.
func TestConfig_WithTitleNamespace(t *testing.T) {
	base := Config{LogsDir: "/var/logs"}

	ns := base.WithTitleNamespace("synthetic_title_b")
	want := "/var/logs" + string(os.PathSeparator) + "synthetic_title_b"
	if ns.LogsDir != want {
		t.Errorf("LogsDir namespacé = %q, want %q", ns.LogsDir, want)
	}
	if base.LogsDir != "/var/logs" {
		t.Errorf("la Config d'origine ne doit pas être mutée, got %q", base.LogsDir)
	}

	if got := base.WithTitleNamespace("").LogsDir; got != "/var/logs" {
		t.Errorf("titre vide doit être no-op, got %q", got)
	}
	if got := (Config{LogsDir: ""}).WithTitleNamespace("x").LogsDir; got != "" {
		t.Errorf("LogsDir vide doit rester vide, got %q", got)
	}
}
