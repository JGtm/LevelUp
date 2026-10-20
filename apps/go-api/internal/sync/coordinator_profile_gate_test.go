// Package sync — coordinator_profile_gate_test.go : porte « profil suivi » sur
// Submit (ADR 0035 D3).
//
// Le 2026-07-23, un compte sans profil a traversé ce point : le coordinateur ne
// demandait que (gamertag, xuid), le moteur a dérivé un chemin de player DB du
// gamertag et écrit 25 matchs dans l'entrepôt partagé. La porte se ferme AVANT
// le claim in-flight : un joueur refusé n'occupe ni slot de dédup, ni sémaphore,
// ni goroutine.
package sync

import (
	"context"
	"testing"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
)

// TestCoordinator_Submit_RefuseSansProfilSuivi — porte fermée : Submit rend
// false, le runner n'est jamais appelé, le compteur expvar monte, et aucun claim
// n'est laissé derrière.
func TestCoordinator_Submit_RefuseSansProfilSuivi(t *testing.T) {
	observability.Reset()

	runner := &mockRunner{}
	var vuTitre, vuXUID string
	coord := NewCoordinator(runner, 2).
		WithProfileGate(func(_ context.Context, titleSlug, xuid string) bool {
			vuTitre, vuXUID = titleSlug, xuid
			return false
		})

	ok := coord.Submit(context.Background(), CoordinatorRequest{
		Gamertag: "InconnuAuBataillon", XUID: "2533274796795729",
		MatchIDs: []string{"m1"}, TitleSlug: "halo_infinite",
	})

	if ok {
		t.Fatal("Submit doit rendre false pour un joueur sans profil suivi")
	}
	if vuTitre != "halo_infinite" || vuXUID != "2533274796795729" {
		t.Errorf("la porte doit recevoir (titre, xuid) = (halo_infinite, 2533274796795729), reçu (%q, %q)", vuTitre, vuXUID)
	}
	if got := observability.LoadCounter("sync_refused_no_profile"); got != 1 {
		t.Errorf("compteur sync_refused_no_profile = %d, want 1", got)
	}

	// Aucun sync lancé, et aucun claim résiduel : un second Submit (porte ouverte
	// cette fois) doit pouvoir passer sur le MÊME joueur.
	coord.Wait()
	if n := runner.callCount.Load(); n != 0 {
		t.Fatalf("aucun RunSync ne doit partir, reçu %d appels", n)
	}
	if snap := coord.GateSnapshot(); snap.InflightWatcher != 0 {
		t.Errorf("aucun claim ne doit rester en vol, reçu %d", snap.InflightWatcher)
	}
}

// TestCoordinator_Submit_PorteOuverte — porte ouverte : comportement d'origine,
// et le compteur de refus ne bouge pas.
func TestCoordinator_Submit_PorteOuverte(t *testing.T) {
	observability.Reset()

	runner := &mockRunner{}
	coord := NewCoordinator(runner, 2).
		WithProfileGate(func(_ context.Context, _, _ string) bool { return true })

	done := make(chan struct{}, 1)
	coord.SetOnComplete(func(_ string, _ error) { done <- struct{}{} })

	if ok := coord.Submit(context.Background(), CoordinatorRequest{
		Gamertag: "JoueurDeclare", XUID: "1000000000000001", MatchIDs: []string{"m1"},
	}); !ok {
		t.Fatal("Submit doit rendre true quand la porte est ouverte")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("le sync n'a pas abouti en 2 s")
	}
	if n := runner.callCount.Load(); n != 1 {
		t.Errorf("RunSync appelé %d fois, want 1", n)
	}
	if got := observability.LoadCounter("sync_refused_no_profile"); got != 0 {
		t.Errorf("compteur sync_refused_no_profile = %d, want 0", got)
	}
}

// TestCoordinator_Submit_TitreVideNormalise — un titre vide (mono-titre
// historique) doit interroger la porte sur le titre PAR DÉFAUT et non sur « tous
// les titres » : sinon le profil d'un autre jeu ouvrirait ce sync-ci.
func TestCoordinator_Submit_TitreVideNormalise(t *testing.T) {
	observability.Reset()

	runner := &mockRunner{}
	var vuTitre string
	coord := NewCoordinator(runner, 2).
		WithProfileGate(func(_ context.Context, titleSlug, _ string) bool {
			vuTitre = titleSlug
			return false
		})

	coord.Submit(context.Background(), CoordinatorRequest{
		Gamertag: "SansTitre", XUID: "1000000000000003", MatchIDs: []string{"m1"},
	})

	if vuTitre != titlePkg.DefaultSlug {
		t.Errorf("la porte doit être interrogée sur %q, reçu %q", titlePkg.DefaultSlug, vuTitre)
	}
}

// TestCoordinator_Submit_PorteNilPasse — seam documenté : sans porte câblée, le
// comportement historique est conservé (montages de test / sans config).
func TestCoordinator_Submit_PorteNilPasse(t *testing.T) {
	observability.Reset()

	runner := &mockRunner{}
	coord := NewCoordinator(runner, 2)

	done := make(chan struct{}, 1)
	coord.SetOnComplete(func(_ string, _ error) { done <- struct{}{} })

	if ok := coord.Submit(context.Background(), CoordinatorRequest{
		Gamertag: "SansPorte", XUID: "1000000000000002", MatchIDs: []string{"m1"},
	}); !ok {
		t.Fatal("porte nil ⇒ Submit doit rendre true")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("le sync n'a pas abouti en 2 s")
	}
	if got := observability.LoadCounter("sync_refused_no_profile"); got != 0 {
		t.Errorf("compteur sync_refused_no_profile = %d, want 0", got)
	}
}
