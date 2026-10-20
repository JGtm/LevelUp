package handlers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
)

// TestSessionsRecalcPaths : un profil Halo 5 se recalcule sur les bases de Halo 5 (joueur ET
// partagée), un profil sans titre sur celles du titre par défaut.
func TestSessionsRecalcPaths(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	for _, c := range []struct{ titre, attendu string }{
		{"halo_5", "halo_5"},
		{titlePkg.DefaultSlug, titlePkg.DefaultSlug},
		{"", titlePkg.DefaultSlug},
	} {
		player, shared := sessionsRecalcPaths(cfg, domain.PlayerSummary{Gamertag: "JGtm", TitleSlug: c.titre})
		if want := pr.PlayerDBPath(c.attendu, "JGtm"); player != want {
			t.Errorf("titre %q : base joueur %q, attendu %q", c.titre, player, want)
		}
		if want := pr.SharedDBPath(c.attendu); shared != want {
			t.Errorf("titre %q : base partagée %q, attendu %q", c.titre, shared, want)
		}
	}
}

// TestPostRecalculateSessions_ProfilHalo5SurLesBasesDeHalo5 : la route ouvre la base joueur du
// titre du profil. Avant, tout profil se recalculait sur la base du titre par défaut : un joueur
// suivi seulement sur Halo 5 voyait créer une base Infinite vide, et ses propres sessions
// n'étaient jamais recalculées.
func TestPostRecalculateSessions_ProfilHalo5SurLesBasesDeHalo5(t *testing.T) {
	root := t.TempDir()
	profilesPath := filepath.Join(root, "db_profiles.json")
	profiles := `{"version":"3.0","admin":"H5Only","profiles":{"halo_5":{"H5Only":{"xuid":"2001"}}}}`
	if err := os.WriteFile(profilesPath, []byte(profiles), 0o600); err != nil {
		t.Fatalf("db_profiles: %v", err)
	}
	cfg := &config.AppConfig{RepoRoot: root, DBProfilesPath: profilesPath}
	store := jobs.NewStore(t.TempDir())
	h := NewSettingsHandler(cfg, settings_platform.NewStore(filepath.Join(root, "app_settings.json")), store)

	out, err := h.handlePostRecalculateSessions(context.Background(), nil)
	if err != nil {
		t.Fatalf("handlePostRecalculateSessions: %v", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if j := store.Get(out.Body.JobID); j != nil && j.IsTerminal() {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if j := store.Get(out.Body.JobID); j == nil || !j.IsTerminal() {
		t.Fatalf("job non terminé: %+v", j)
	}
	pr := titlePkg.NewPathResolver(root)
	if _, err := os.Stat(pr.PlayerDBPath("halo_5", "H5Only")); err != nil {
		t.Errorf("la base joueur de Halo 5 devait être ouverte: %v", err)
	}
	if _, err := os.Stat(pr.PlayerDBPath(titlePkg.DefaultSlug, "H5Only")); !os.IsNotExist(err) {
		t.Errorf("aucune base du titre par défaut ne devait être créée pour ce profil (stat: %v)", err)
	}
}
