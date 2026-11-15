package tactical

import (
	"sort"
	"testing"
)

// TestOccupation_LaRouteDUneVieSeJointAuSpawnParSonInstant — LA JOINTURE DU FILTRE « TOUTES LES
// VIES » : le service rattache la route d'une vie a son spawn par l'instant (`Route.DebutFrame`
// = `SpawnPiste.Frame`). Les deux sont le premier point retenu de la meme piste, sur la meme
// fenetre ; ce test tient l'egalite des deux ensembles d'instants, y compris quand les points
// ne sont pas tries et que certains tombent hors de la fenetre declaree de la piste.
func TestOccupation_LaRouteDUneVieSeJointAuSpawnParSonInstant(t *testing.T) {
	e := entreeTest(nil,
		// Points hors fenetre (5 et 120) et desordonnes : le premier point retenu est 12.
		Piste{XUID: "a", StartFrame: 10, EndFrame: 100, Points: []PointPiste{
			{T: 120, X: 4}, {T: 12, X: 1}, {T: 5, X: 0}, {T: 40, X: 2}, {T: 90, X: 3},
		}},
		pisteTest("a", 0, 150, 300, 50, nil),
		pisteTest("a", 0, 400, 460, 60, nil),
		pisteTest("b", 1, 0, 400, 20, nil),
	)
	joueurs := Occupation(GrilleParDefaut(), e, PasOccupationMs)
	if len(joueurs) != 2 {
		t.Fatalf("joueurs = %d, attendu 2", len(joueurs))
	}
	for _, j := range joueurs {
		var spawns, routes []int
		for _, s := range j.Spawns {
			spawns = append(spawns, s.Frame)
		}
		for _, r := range j.Routes {
			routes = append(routes, r.DebutFrame)
		}
		sort.Ints(spawns)
		sort.Ints(routes)
		if len(spawns) != len(routes) {
			t.Fatalf("joueur %s : %d spawns pour %d routes", j.XUID, len(spawns), len(routes))
		}
		for i := range spawns {
			if spawns[i] != routes[i] {
				t.Fatalf("joueur %s : instants des spawns %v, des routes %v — la jointure ne tient plus",
					j.XUID, spawns, routes)
			}
		}
	}
	if joueurs[0].XUID != "a" || joueurs[0].Spawns[0].Frame != 12 {
		t.Fatalf("premier spawn de a = %+v, attendu l'instant 12 (premier point retenu)", joueurs[0].Spawns)
	}
}
