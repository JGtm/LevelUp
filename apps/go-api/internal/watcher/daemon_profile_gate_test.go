// Package watcher — daemon_profile_gate_test.go : porte « profil suivi » sur
// AddPlayer (ADR 0035 D3).
//
// AddPlayer est la porte d'entrée du tracking live, et c'est par elle que le SSO
// Xbox a fait entrer un compte sans profil le 2026-07-23. Un refus ne laisse
// RIEN derrière : ni PlayerWatcher, ni cancel de poller.
package watcher

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	syncpkg "levelup/go-api/internal/sync"
)

func TestAddPlayer_RefuseSansProfilSuivi(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})

	var vuTitre, vuXUID string
	d.WithProfileGate(func(_ context.Context, titleSlug, xuid string) bool {
		vuTitre, vuXUID = titleSlug, xuid
		return false
	})

	err := d.AddPlayer(context.Background(), domain.PlayerSummary{
		Gamertag: "InconnuAuBataillon", XUID: "2533274796795729", TitleSlug: "halo_infinite",
	})

	if !errors.Is(err, ErrPlayerNotTracked) {
		t.Fatalf("AddPlayer doit rendre ErrPlayerNotTracked, reçu %v", err)
	}
	if vuTitre != "halo_infinite" || vuXUID != "2533274796795729" {
		t.Errorf("la porte doit recevoir (titre, xuid) = (halo_infinite, 2533274796795729), reçu (%q, %q)", vuTitre, vuXUID)
	}

	d.playersMu.RLock()
	defer d.playersMu.RUnlock()
	if len(d.players) != 0 {
		t.Errorf("aucun watcher ne doit être créé, reçu %d", len(d.players))
	}
	if len(d.playerCancels) != 0 {
		t.Errorf("aucun cancel de poller ne doit être enregistré, reçu %d", len(d.playerCancels))
	}
}

func TestAddPlayer_PorteOuverte(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})
	d.WithProfileGate(func(_ context.Context, _, _ string) bool { return true })

	if err := d.AddPlayer(context.Background(), domain.PlayerSummary{
		Gamertag: "JoueurDeclare", XUID: "1000000000000001", TitleSlug: "halo_infinite",
	}); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	d.playersMu.RLock()
	defer d.playersMu.RUnlock()
	if _, ok := d.players[playerKey("JoueurDeclare", "halo_infinite")]; !ok {
		t.Error("le joueur déclaré doit être pris en charge")
	}
}

// TestAddPlayer_TitreVideNormalise — un titre vide (les callers qui s'en
// remettent au défaut, dont le SSO) doit interroger la porte sur le titre PAR
// DÉFAUT et non sur « tous les titres » : sinon le profil d'un autre jeu
// ouvrirait le suivi de celui-ci.
func TestAddPlayer_TitreVideNormalise(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})

	var vuTitre string
	d.WithProfileGate(func(_ context.Context, titleSlug, _ string) bool {
		vuTitre = titleSlug
		return true
	})

	if err := d.AddPlayer(context.Background(), domain.PlayerSummary{
		Gamertag: "SansTitre", XUID: "1000000000000003",
	}); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	if vuTitre != title.DefaultSlug {
		t.Errorf("la porte doit être interrogée sur %q, reçu %q", title.DefaultSlug, vuTitre)
	}
}

// TestAddPlayer_PorteNilPasse — seam documenté : sans porte câblée, le
// comportement historique est conservé.
func TestAddPlayer_PorteNilPasse(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})

	if err := d.AddPlayer(context.Background(), domain.PlayerSummary{
		Gamertag: "SansPorte", XUID: "1000000000000002",
	}); err != nil {
		t.Fatalf("porte nil ⇒ AddPlayer doit réussir, reçu %v", err)
	}

	d.playersMu.RLock()
	defer d.playersMu.RUnlock()
	if len(d.players) != 1 {
		t.Errorf("1 watcher attendu, reçu %d", len(d.players))
	}
}

// TestWithProfileGate_PoseAussiLaPorteDuCoordinateur — le Coordinator est
// construit DANS NewDaemon et n'est exposé que derrière l'interface SyncGate :
// WithProfileGate est donc le point de câblage unique des deux portes du chemin
// watcher (prise en charge live, puis soumission du sync). Ce test garde ce lien.
func TestWithProfileGate_PoseAussiLaPorteDuCoordinateur(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})
	d.WithProfileGate(func(_ context.Context, _, _ string) bool { return false })

	ok := d.coordinator.Submit(context.Background(), syncpkg.CoordinatorRequest{
		Gamertag: "InconnuAuBataillon", XUID: "2533274796795729",
		MatchIDs: []string{"m1"}, TitleSlug: "halo_infinite",
	})
	if ok {
		t.Fatal("la porte posée sur le daemon doit AUSSI fermer Coordinator.Submit")
	}
}
