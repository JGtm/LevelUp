package main

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

type passeVue struct{ titre, joueur string }

// Un passage par joueur de chaque titre, le titre posé sur le contexte, les profils réservés aux
// jetons ignorés, un titre illisible sauté sans arrêter les autres.
func TestRunMediaReconcile_UnPassageParJoueur(t *testing.T) {
	joueurs := map[string][]domain.PlayerSummary{
		"t1": {{Gamertag: "A"}, {Gamertag: "Jeton", AuthOnly: true}, {Gamertag: "B"}},
		"t3": {{Gamertag: "C"}},
	}
	charger := func(f ...string) ([]domain.PlayerSummary, error) {
		if f[0] == "t2" {
			return nil, errors.New("illisible")
		}
		return joueurs[f[0]], nil
	}
	var vues []passeVue
	runMediaReconcile(context.Background(), []string{"t1", "t2", "t3"}, charger,
		func(ctx context.Context, gamertag string) {
			vues = append(vues, passeVue{ctxkeys.TitleSlug(ctx), gamertag})
		})
	want := []passeVue{{"t1", "A"}, {"t1", "B"}, {"t3", "C"}}
	if len(vues) != len(want) {
		t.Fatalf("passes = %v, attendu %v", vues, want)
	}
	for i := range want {
		if vues[i] != want[i] {
			t.Fatalf("passes = %v, attendu %v", vues, want)
		}
	}
}

// L'annulation du contexte arrête la boucle entre deux joueurs.
func TestRunMediaReconcile_ArretEntreDeuxJoueurs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	charger := func(...string) ([]domain.PlayerSummary, error) {
		return []domain.PlayerSummary{{Gamertag: "A"}, {Gamertag: "B"}, {Gamertag: "C"}}, nil
	}
	n := 0
	runMediaReconcile(ctx, []string{"t1"}, charger, func(context.Context, string) {
		n++
		cancel()
	})
	if n != 1 {
		t.Fatalf("%d passes après annulation, attendu 1", n)
	}
}

// La tâche est coupée en démo et gardée sur les trois autres profils.
func TestMediaReconcileTask_PolitiqueParProfil(t *testing.T) {
	want := map[config.Deployment]taskPolicy{
		config.DeploymentHosted: taskKeep, config.DeploymentDev: taskKeep,
		config.DeploymentDemo: taskCut, config.DeploymentDesktop: taskKeep,
	}
	for profil, w := range want {
		if got := policyOf(taskMediaReconcile, profil); got != w {
			t.Errorf("%s : %v, attendu %v", profil, got, w)
		}
	}
}
