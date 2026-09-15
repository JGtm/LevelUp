// Package watcher — daemon_remove_test.go : retrait d'un joueur du suivi live
// (ADR 0035 D6, première étape d'une purge d'identité).
package watcher

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
)

func newRemovalDaemon(t *testing.T) *Daemon {
	t.Helper()
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})
	d.WithProfileGate(func(_ context.Context, _, _ string) bool { return true })
	return d
}

// Le retrait se fait PAR XUID : tous les titres suivis du joueur partent, et eux
// seuls — un homonyme (même gamertag, autre xuid) reste suivi.
func TestRemovePlayer_RetireTousLesTitresDuXUID(t *testing.T) {
	d := newRemovalDaemon(t)
	for _, p := range []domain.PlayerSummary{
		{Gamertag: "Spartan", XUID: "111", TitleSlug: "halo_infinite"},
		{Gamertag: "Spartan", XUID: "111", TitleSlug: "halo_5"},
		{Gamertag: "Autre", XUID: "222", TitleSlug: "halo_infinite"},
	} {
		if err := d.AddPlayer(context.Background(), p); err != nil {
			t.Fatalf("AddPlayer %+v: %v", p, err)
		}
	}

	removed := d.RemovePlayer(context.Background(), "111")
	if len(removed) != 2 {
		t.Fatalf("titres retires = %v, attendu 2", removed)
	}
	byTitle := map[string]bool{}
	for _, slug := range removed {
		byTitle[slug] = true
	}
	if !byTitle["halo_infinite"] || !byTitle["halo_5"] {
		t.Errorf("titres retires = %v", removed)
	}

	watched := d.WatchedPlayers()
	if len(watched) != 1 || watched[0].XUID != "222" {
		t.Fatalf("suivi restant = %+v, attendu le seul xuid 222", watched)
	}
}

// Le cancel du REST poller part avec le joueur : sans lui, sa goroutine survit et
// continue d'interroger la présence (fuite W2, corrigée pour UpdateSubscriptions
// et que ce chemin-ci ne doit pas réintroduire).
func TestRemovePlayer_RetireAussiLeCancelDuPoller(t *testing.T) {
	d := newRemovalDaemon(t)
	if err := d.AddPlayer(context.Background(), domain.PlayerSummary{
		Gamertag: "Spartan", XUID: "111", TitleSlug: "halo_infinite",
	}); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	// Le daemon n'étant pas démarré, aucun cancel n'est enregistré : on en pose un
	// à la main pour prouver que le retrait le consomme.
	key := playerKey("Spartan", "halo_infinite")
	cancelled := false
	d.playersMu.Lock()
	d.playerCancels[key] = func() { cancelled = true }
	d.playersMu.Unlock()

	d.RemovePlayer(context.Background(), "111")

	d.playersMu.RLock()
	_, stillThere := d.playerCancels[key]
	d.playersMu.RUnlock()
	if !cancelled {
		t.Error("le cancel du REST poller n'a pas ete appele")
	}
	if stillThere {
		t.Error("le cancel du REST poller est reste dans la map")
	}
}

// Idempotence et cas dégradés : un second retrait ne rend rien, un xuid vide et
// un récepteur nil (CLI, process sans watcher) ne paniquent pas.
func TestRemovePlayer_IdempotentEtNilSafe(t *testing.T) {
	d := newRemovalDaemon(t)
	if err := d.AddPlayer(context.Background(), domain.PlayerSummary{
		Gamertag: "Spartan", XUID: "111", TitleSlug: "halo_infinite",
	}); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	if got := d.RemovePlayer(context.Background(), "111"); len(got) != 1 {
		t.Fatalf("premier retrait = %v, attendu 1 titre", got)
	}
	if got := d.RemovePlayer(context.Background(), "111"); len(got) != 0 {
		t.Fatalf("second retrait = %v, attendu rien", got)
	}
	if got := d.RemovePlayer(context.Background(), ""); got != nil {
		t.Fatalf("xuid vide = %v, attendu nil", got)
	}
	var nilDaemon *Daemon
	if got := nilDaemon.RemovePlayer(context.Background(), "111"); got != nil {
		t.Fatalf("recepteur nil = %v, attendu nil", got)
	}
}

// Titre vide : normalisé sur le titre par défaut, comme le fait playerKey — sans
// quoi le rapport de purge nommerait un titre vide.
func TestRemovePlayer_TitreVideNormalise(t *testing.T) {
	d := newRemovalDaemon(t)
	if err := d.AddPlayer(context.Background(), domain.PlayerSummary{
		Gamertag: "SansTitre", XUID: "222",
	}); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	removed := d.RemovePlayer(context.Background(), "222")
	if len(removed) != 1 || removed[0] != title.DefaultSlug {
		t.Fatalf("titres retires = %v, attendu [%s]", removed, title.DefaultSlug)
	}
}
