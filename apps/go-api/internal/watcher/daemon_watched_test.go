// Package watcher — daemon_watched_test.go : lecture du suivi live par
// l'annuaire des joueurs (ADR 0035 D2).
package watcher

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
)

func TestWatchedPlayers_RendLesCouplesJoueurTitre(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})
	d.WithProfileGate(func(_ context.Context, _, _ string) bool { return true })

	if err := d.AddPlayer(context.Background(), domain.PlayerSummary{
		Gamertag: "Spartan", XUID: "111", TitleSlug: "halo_infinite",
	}); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	// Titre vide : le watcher suit le titre par défaut, et l'annuaire doit le
	// rattacher à CE titre (même normalisation que playerKey).
	if err := d.AddPlayer(context.Background(), domain.PlayerSummary{
		Gamertag: "SansTitre", XUID: "222",
	}); err != nil {
		t.Fatalf("AddPlayer sans titre: %v", err)
	}

	got := d.WatchedPlayers()
	if len(got) != 2 {
		t.Fatalf("WatchedPlayers = %+v, want 2 entrées", got)
	}
	byXUID := make(map[string]domain.WatchedPlayerRef, len(got))
	for _, w := range got {
		byXUID[w.XUID] = w
	}
	if w := byXUID["111"]; w.Gamertag != "Spartan" || w.TitleSlug != "halo_infinite" {
		t.Errorf("xuid 111 = %+v", w)
	}
	if w := byXUID["222"]; w.Gamertag != "SansTitre" || w.TitleSlug != title.DefaultSlug {
		t.Errorf("xuid 222 = %+v (titre vide doit être normalisé sur le défaut)", w)
	}
}

// TestWatchedPlayers_DaemonNil : l'annuaire lit le watcher même quand il n'y en
// a pas (CLI, serveur sans watcher) — sans cas particulier chez l'appelant.
func TestWatchedPlayers_DaemonNil(t *testing.T) {
	var d *Daemon
	if got := d.WatchedPlayers(); got != nil {
		t.Fatalf("WatchedPlayers = %+v, want nil", got)
	}
}

// TestWatchedPlayers_AucunJoueur : daemon vivant mais sans joueur suivi → liste
// vide, jamais nil ambigu côté appelant.
func TestWatchedPlayers_AucunJoueur(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})
	if got := d.WatchedPlayers(); len(got) != 0 {
		t.Fatalf("WatchedPlayers = %+v, want vide", got)
	}
}
