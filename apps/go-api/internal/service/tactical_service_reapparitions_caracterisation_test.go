package service

// tactical_service_reapparitions_caracterisation_test.go — TEMOIN de la regle « reapparition adverse » :
// les observations que la lecture tire de sidecars poses a la main, figees champ par champ.

import (
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// sidecarAdverseMixte : deux adversaires d'equipes differentes (1 et 2), un equipier, un joueur sans
// equipe, des debuts de manche, des premieres vies et des pistes sans drapeau.
func sidecarAdverseMixte(id string) *domain.TacticalRasterSidecar {
	sc := sidecarPose(id)
	sc.Joueurs = []domain.TacticalRasterJoueur{
		{XUID: tsMoi, Equipe: 0, Spawns: []domain.TacticalRasterSpawn{
			{Frame: 0, X: 1, Y: 1, PremiereVie: true},
			reappAdverse(500, 2, 2, 0), // une reapparition du joueur lui-meme : jamais adverse
		}},
		{XUID: tsAmi, Equipe: 0, Spawns: []domain.TacticalRasterSpawn{reappAdverse(600, 3, 3, 1)}},
		{XUID: tsAdv, Equipe: 1, Spawns: []domain.TacticalRasterSpawn{
			{Frame: 0, X: 40, Y: 40, PremiereVie: true},
			reappAdverse(1000, 25.5, 5.25, 2),
			{Frame: 2000, X: 26, Y: 6, DebutDeManche: true, Instantane: []domain.TacticalRasterVivant{{Equipe: 0, X: 2, Y: 2}}},
			{Frame: 3000, X: 27, Y: 7},
			reappAdverse(4000, 28, 8, 3),
		}},
		{XUID: tsAdv2, Equipe: 2, Spawns: []domain.TacticalRasterSpawn{reappAdverse(1200, 22, 2, 1)}},
		{XUID: "sans-equipe", Equipe: -1, Spawns: []domain.TacticalRasterSpawn{reappAdverse(1300, 21, 1, 2)}},
	}
	return sc
}

func decrireObservations(obs []string) string { return strings.Join(obs, "\n") }

// TestCaracterisation_ObservationsAdverses fige, pour deux matchs (dont un ou l'equipe du joueur est
// inconnue), les observations adverses et les comptes de matchs.
func TestCaracterisation_ObservationsAdverses(t *testing.T) {
	univ, _ := matchsAssassin(2, 0, 0)
	inconnu := sidecarAdverseMixte("m2")
	inconnu.Joueurs[0].Equipe = -1
	sidecars := map[string]*domain.TacticalRasterSidecar{"m1": sidecarAdverseMixte("m1"), "m2": inconnu}
	svc, _ := svcAdverses(t, univ, &mockRasterStore{sidecars: sidecars})

	obs, matchs := svc.observationsAdverses(univ, sidecars)
	var lignes []string
	for _, o := range obs {
		ligne := fmt.Sprintf("%s (%.2f, %.2f, %.2f) equipe=%d instantane=%d", o.MatchID, o.X, o.Y, o.Z, o.EquipeActive, len(o.Instantane))
		for _, v := range o.Instantane {
			ligne += fmt.Sprintf(" [%s e%d %.0f,%.0f]", v.XUID, v.Equipe, v.X, v.Y)
		}
		lignes = append(lignes, ligne)
	}
	got := fmt.Sprintf("retenus=%d sansEquipe=%d\n%s", matchs.retenus, matchs.sansEquipe, decrireObservations(lignes))
	if got != attenduObservationsAdverses {
		t.Errorf("observations :\n%s\nattendu :\n%s", got, attenduObservationsAdverses)
	}
}

const attenduObservationsAdverses = `retenus=1 sansEquipe=1
m1 (25.50, 5.25, 0.00) equipe=0 instantane=3 [2533274000000001 e0 2,2] [2533274000000001 e0 3,2] [ e1 60,60]
m1 (28.00, 8.00, 0.00) equipe=0 instantane=4 [2533274000000001 e0 2,2] [2533274000000001 e0 3,2] [2533274000000001 e0 4,2] [ e1 60,60]
m1 (22.00, 2.00, 0.00) equipe=0 instantane=2 [2533274000000001 e0 2,2] [ e1 60,60]`
