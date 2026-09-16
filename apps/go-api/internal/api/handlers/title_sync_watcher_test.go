// Package handlers — title_sync_watcher_test.go : le suivi live suit le profil
// quand un titre est mis en pause, réactivé ou purgé (revue adversariale du
// 2026-09-16, P1 : un poller fantôme faisait sonner le compteur d'intrusion).
package handlers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/service"
)

// fakeTitleWatcher enregistre les appels du handler.
type fakeTitleWatcher struct {
	running bool
	removed []string // "xuid|titre"
	added   []domain.PlayerSummary
	addErr  error
}

func (f *fakeTitleWatcher) IsRunning() bool { return f.running }
func (f *fakeTitleWatcher) RemovePlayerTitle(_ context.Context, xuid, titleSlug string) bool {
	f.removed = append(f.removed, xuid+"|"+titleSlug)
	return true
}
func (f *fakeTitleWatcher) AddPlayer(_ context.Context, p domain.PlayerSummary) error {
	if f.addErr != nil {
		return f.addErr
	}
	f.added = append(f.added, p)
	return nil
}

// newTitleSyncFixture : un joueur suivi sur DEUX titres (le store refuse de mettre
// en pause ou de purger le dernier titre actif), un watcher qui tourne.
func newTitleSyncFixture(t *testing.T) (*TitleSyncHandler, *fakeTitleWatcher) {
	t.Helper()
	root := t.TempDir()
	profilesPath := filepath.Join(root, "db_profiles.json")
	if err := os.WriteFile(profilesPath, []byte(`{
  "version": "3.0",
  "profiles": {
    "halo_infinite": {"Spartan": {"db_path": "x", "xuid": "111"}},
    "halo_5": {"Spartan": {"db_path": "y", "xuid": "111"}}
  }
}`), 0o644); err != nil {
		t.Fatalf("db_profiles: %v", err)
	}
	cfg := &config.AppConfig{RepoRoot: root, DBProfilesPath: profilesPath}
	w := &fakeTitleWatcher{running: true}
	h := NewTitleSyncHandler(service.NewProfileService(profilesPath, root)).
		WithWatcher(func() TitleWatcher { return w }).
		WithPlayerLookup(func(titleSlug, playerSlug string) (domain.PlayerSummary, bool) {
			players, err := cfg.LoadPlayers(titleSlug)
			if err != nil {
				return domain.PlayerSummary{}, false
			}
			for _, p := range players {
				if p.PlayerSlug == playerSlug {
					return p, true
				}
			}
			return domain.PlayerSummary{}, false
		})
	return h, w
}

func TestTitleSync_PauseRetireLeSuiviLive(t *testing.T) {
	h, w := newTitleSyncFixture(t)
	in := &titleSyncInput{PlayerSlug: "Spartan", Slug: "halo_5"}
	in.Body.Enabled = false
	if _, err := h.SetSync(context.Background(), in); err != nil {
		t.Fatalf("SetSync: %v", err)
	}
	if len(w.removed) != 1 || w.removed[0] != "111|halo_5" {
		t.Fatalf("retraits = %v, attendu [111|halo_5]", w.removed)
	}
	if len(w.added) != 0 {
		t.Fatalf("aucun ajout attendu sur une pause : %+v", w.added)
	}
}

func TestTitleSync_ReactivationRemetLeSuiviLive(t *testing.T) {
	h, w := newTitleSyncFixture(t)
	pause := &titleSyncInput{PlayerSlug: "Spartan", Slug: "halo_5"}
	if _, err := h.SetSync(context.Background(), pause); err != nil {
		t.Fatalf("pause: %v", err)
	}
	resume := &titleSyncInput{PlayerSlug: "Spartan", Slug: "halo_5"}
	resume.Body.Enabled = true
	if _, err := h.SetSync(context.Background(), resume); err != nil {
		t.Fatalf("reactivation: %v", err)
	}
	if len(w.added) != 1 {
		t.Fatalf("ajouts = %+v, attendu 1", w.added)
	}
	p := w.added[0]
	if p.XUID != "111" || p.TitleSlug != "halo_5" || p.Gamertag != "Spartan" || !p.SyncEnabled {
		t.Fatalf("joueur remis au watcher = %+v", p)
	}
}

func TestTitlePurge_RetireLeSuiviLiveAvecLeXuidResoluAvant(t *testing.T) {
	h, w := newTitleSyncFixture(t)
	if _, err := h.Purge(context.Background(), &titlePurgeInput{PlayerSlug: "Spartan", Slug: "halo_5"}); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if len(w.removed) != 1 || w.removed[0] != "111|halo_5" {
		t.Fatalf("retraits = %v, attendu [111|halo_5] (xuid résolu AVANT le retrait du profil)", w.removed)
	}
}

// Watcher absent, arrêté, ou refusant : la pause/réactivation reste acquise,
// rien ne bloque (le profil fait foi, le daemon relit au boot).
func TestTitleSync_WatcherAbsentOuEnEchec_NonBloquant(t *testing.T) {
	h, w := newTitleSyncFixture(t)
	w.running = false
	in := &titleSyncInput{PlayerSlug: "Spartan", Slug: "halo_5"}
	if _, err := h.SetSync(context.Background(), in); err != nil {
		t.Fatalf("SetSync watcher arrete: %v", err)
	}
	if len(w.removed) != 0 {
		t.Fatalf("un watcher arrete ne doit pas etre sollicite : %v", w.removed)
	}

	w.running = true
	w.addErr = errors.New("porte fermee")
	resume := &titleSyncInput{PlayerSlug: "Spartan", Slug: "halo_5"}
	resume.Body.Enabled = true
	out, err := h.SetSync(context.Background(), resume)
	if err != nil {
		t.Fatalf("SetSync AddPlayer en echec doit rester non bloquant : %v", err)
	}
	if !out.Body.SyncEnabled {
		t.Fatalf("la reactivation du profil reste acquise : %+v", out.Body)
	}

	sans := NewTitleSyncHandler(service.NewProfileService(
		filepath.Join(t.TempDir(), "db_profiles.json"), t.TempDir()))
	if sans.watcher != nil {
		t.Fatal("sans WithWatcher, aucun resolveur")
	}
}
