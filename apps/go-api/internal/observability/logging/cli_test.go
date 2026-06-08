package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallCLI_RoutesLeaderboardModuleToFile vérifie qu'un log taggé
// module=leaderboard émis après InstallCLI atterrit dans logs/leaderboard.log.
func TestInstallCLI_RoutesLeaderboardModuleToFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LEVELUP_LOGS_DIR", dir)
	t.Setenv("LEVELUP_LOGS_ENABLED", "true")
	t.Setenv("LEVELUP_LOGS_FILE_LEVEL", "info")

	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	closer := InstallCLI("")
	slog.Default().With("module", ModuleLeaderboard, "job", "snapshot-test").
		Info("snapshot world leaderboard terminé", "rows_inserted", 42)
	closer() // flush + close des fichiers

	path := filepath.Join(dir, ModuleLeaderboard+".log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture %s: %v", path, err)
	}
	content := string(data)
	if !strings.Contains(content, "snapshot world leaderboard terminé") {
		t.Errorf("message absent de %s : %q", path, content)
	}
	if !strings.Contains(content, "rows_inserted") {
		t.Errorf("attribut rows_inserted absent de %s", path)
	}
}
