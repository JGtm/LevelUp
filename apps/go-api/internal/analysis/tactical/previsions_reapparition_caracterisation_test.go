package tactical

import (
	"math"
	"testing"
)

// poidsCaracterisation donne un poids non nul a chacune des six variables de position : la
// prevision qui en sort fige le score de chaque candidat.
var poidsCaracterisation = [NbVariables]float64{
	VarEnnemiProche:     1.2,
	VarEnnemisProches:   -0.4,
	VarEquipierProche:   -0.7,
	VarDistanceMort:     0.15,
	VarFrequence:        0.5,
	VarBarycentreEnnemi: -0.3,
}

// evenementsCaracterisation : cinq points candidats, vus un nombre de fois different dans trois
// autres matchs, dont un hors de toute zone.
func evenementsCaracterisation() []EvenementReapparition {
	pts := []struct {
		p Point3
		n int
	}{
		{Point3{X: 10, Y: 2}, 1}, {Point3{X: 20, Y: -3, Z: 1}, 2}, {Point3{X: 110, Y: 4}, 3},
		{Point3{X: 125, Y: -6}, 1}, {Point3{X: 500}, 2},
	}
	var out []EvenementReapparition
	for _, m := range []string{"x1", "x2", "x3"} {
		for _, pt := range pts {
			for i := 0; i < pt.n*2; i++ {
				out = append(out, EvenementReapparition{MatchID: m, CarteID: "carte", Reel: pt.p})
			}
		}
	}
	return out
}

func matchCaracterisation() EntreeOccupation {
	mort := Piste{XUID: "a", Equipe: 0, Points: []PointPiste{{T: 0, X: 50}, {T: 100, X: 50}}}
	vie := Piste{XUID: "a", Equipe: 0, Points: []PointPiste{{T: 340, X: 10, Y: 2}, {T: 500, X: 12}}}
	mate := Piste{XUID: "b", Equipe: 0, Points: []PointPiste{{T: 0, X: 30, Y: 5}, {T: 300, X: 90, Y: -2}, {T: 500, X: 90}}}
	e1 := Piste{XUID: "e1", Equipe: 1, Points: []PointPiste{{T: 0, X: 110, Y: 3}, {T: 200, X: 10, Y: 1}, {T: 500, X: 15}}}
	e2 := Piste{XUID: "e2", Equipe: 1, Points: []PointPiste{{T: 0, X: 70, Y: -8, Z: 2}, {T: 500, X: 40, Y: 8}}}
	e3 := Piste{XUID: "e3", Equipe: 1, Points: []PointPiste{{T: 0, X: 125}, {T: 250, X: 20, Y: -3}, {T: 500, X: 120}}}
	return EntreeOccupation{
		MatchID: "prevu", IntervalleFrameMs: 33, Pistes: []Piste{mort, vie, mate, e1, e2, e3},
		DebutsDeCompte: []DebutCompte{{XUID: "a", Frame: 100}},
	}
}

// TestCaracterisation_PrevisionParCandidat fige les probabilites de chaque candidat, a chaque
// echantillon, pour un match a trois adversaires et un equipier : toute modification du calcul
// des variables ou du score qui bouge un chiffre la fait rougir.
func TestCaracterisation_PrevisionParCandidat(t *testing.T) {
	r := PrevoirLesReapparitions(EntreePrevision{
		Match: matchCaracterisation(), MatchID: "prevu", Evenements: evenementsCaracterisation(),
		Poids: poidsCaracterisation, Zones: zonesPrevision(),
	})
	if r.Statut != PrevisionPrete || len(r.Morts) != 1 || r.Candidats != 5 {
		t.Fatalf("statut = %q, morts = %d, candidats = %d", r.Statut, len(r.Morts), r.Candidats)
	}
	var got [][]float64
	for _, e := range r.Morts[0].Echantillons {
		got = append(got, e.PointsP)
	}
	if len(got) != len(attenduCaracterisation) {
		t.Fatalf("%d echantillons, attendu %d : %v", len(got), len(attenduCaracterisation), got)
	}
	for i := range got {
		for j := range got[i] {
			if math.Abs(got[i][j]-attenduCaracterisation[i][j]) > 1e-9 {
				t.Errorf("echantillon %d, candidat %d : %v, attendu %v", i, j, got[i][j], attenduCaracterisation[i][j])
			}
		}
	}
}

var attenduCaracterisation = [][]float64{
	{0.003, 0.571, 0.134, 0.29, 0.002},
	{0.003, 0.571, 0.134, 0.29, 0.002},
	{0.003, 0.571, 0.134, 0.29, 0.002},
	{0.003, 0.571, 0.134, 0.29, 0.002},
	{0.06, 0.101, 0.82, 0.012, 0.008},
	{0.397, 0.024, 0.329, 0.009, 0.241},
	{0.397, 0.024, 0.329, 0.009, 0.241},
	{0.693, 0.002, 0.15, 0.001, 0.154},
}
