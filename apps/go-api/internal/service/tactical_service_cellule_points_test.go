package service

// tactical_service_cellule_points_test.go — les positions que chaque contribution publie pour
// le plan : celle de l'événement et celle de l'autre joueur.

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func pointEgal(p *domain.TacticalPoint, x, y float64) bool {
	return p != nil && p.X == x && p.Y == y
}

// TestCellule_PointsDesFaces : une mort pose la victime puis le tueur, un frag le tueur puis la victime.
func TestCellule_PointsDesFaces(t *testing.T) {
	repo := repoDeuxFaces()
	got := celluleLue(t, NewTacticalService(repo, capsCompletes(), tsMoi),
		celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	if len(got.Contributions) != 2 {
		t.Fatalf("contributions = %+v, want une mort puis un frag", got.Contributions)
	}
	mort, frag := got.Contributions[0], got.Contributions[1]
	if mort.Face != domain.TacticalFaceMort || !pointEgal(mort.Point, 2.1, 2.1) || !pointEgal(mort.AutrePoint, 9, 9) {
		t.Errorf("mort : point %+v / autre %+v, want victime (2,1 ; 2,1) et tueur (9 ; 9)", mort.Point, mort.AutrePoint)
	}
	if frag.Face != domain.TacticalFaceFrag || !pointEgal(frag.Point, 2.2, 2.2) || !pointEgal(frag.AutrePoint, 9, 9) {
		t.Errorf("frag : point %+v / autre %+v, want tueur (2,2 ; 2,2) et victime (9 ; 9)", frag.Point, frag.AutrePoint)
	}
}

// TestCellule_IsolePoseLeTueur : « isole » publie la victime et la position du tueur quand elle est
// mesurée, et aucun point d'autre joueur quand elle ne l'est pas.
func TestCellule_IsolePoseLeTueur(t *testing.T) {
	cas := []struct {
		nom          string
		kx, ky       *float64
		wantAutre    bool
		wantX, wantY float64
	}{
		{"tueur mesure", m(12), m(13), true, 12, 13},
		{"tueur non mesure", nil, nil, false, 0, 0},
		{"tueur a moitie mesure", m(12), nil, false, 0, 0},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			univ := universVariantes(map[string]string{"m1": "Slayer:Arena"})
			mort := mortContexteAvecInstant("m1", tsMoi, 4.0, 4.0, m(19.0), 1, 0, 7000)
			mort.KillerX, mort.KillerY = c.kx, c.ky
			repo := &mockTacticalRepo{univ: univ, morts: domain.TacticalMortsContexte{Univers: univ, Morts: []domain.MortContexte{mort}}}
			svc := NewTacticalService(repo, capsCompletes(), tsMoi).WithRadarRange(map[string]int{"Slayer:Arena": 18})
			got := celluleLue(t, svc, celluleDemande(repo, domain.TacticalQuestionIsole, domain.TacticalQuiMoi, 8, 8))
			if len(got.Contributions) != 1 {
				t.Fatalf("contributions = %+v, want une mort", got.Contributions)
			}
			ct := got.Contributions[0]
			if !pointEgal(ct.Point, 4, 4) {
				t.Errorf("point = %+v, want la victime (4 ; 4)", ct.Point)
			}
			if c.wantAutre != (ct.AutrePoint != nil) || (c.wantAutre && !pointEgal(ct.AutrePoint, c.wantX, c.wantY)) {
				t.Errorf("autre_point = %+v, want présent=%v (%v ; %v)", ct.AutrePoint, c.wantAutre, c.wantX, c.wantY)
			}
		})
	}
}
