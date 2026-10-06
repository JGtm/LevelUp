package service

// tactical_service_cellule_faces_test.go — ce que chaque contribution du détail d'une zone dit
// d'elle-même (sa face, l'autre joueur de l'événement) et le nom de la zone, lu au centre de la
// cellule avec la hauteur des événements (règle V6, analysis/tactical.NommerZone).

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// celluleLue sert la demande et échoue le test sur une erreur.
func celluleLue(t *testing.T, svc *TacticalService, req domain.TacticalCelluleRequest) domain.TacticalCelluleReponse {
	t.Helper()
	got, err := svc.Cellule(context.Background(), req)
	if err != nil {
		t.Fatalf("Cellule: %v", err)
	}
	return got
}

// carre rend le contour d'un carré [a, b]².
func carre(a, b float64) [][2]float64 {
	return [][2]float64{{a, a}, {b, a}, {b, b}, {a, b}}
}

// repoDeuxFaces : dans m1, je meurs en (2,1 ; 2,1) à 4 m de haut, tué par « Rival » posté au sol en
// (9, 9) ; puis je tue « Cible » depuis (2,2 ; 2,2), au sol (1 m), elle en hauteur (5 m) en (9, 9).
func repoDeuxFaces() *mockTacticalRepo {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = universUnMatch("m1", domain.OutcomeWin)
	repo.pos.Points = []domain.TacticalKillPosition{
		{MatchID: "m1", KillerXUID: tsAdv, VictimXUID: tsMoi, KillerGamertag: "Rival", VictimGamertag: "Moi",
			KillerX: 9, KillerY: 9, KillerZ: m(0.5), VictimX: 2.1, VictimY: 2.1, VictimZ: m(4), TimeMs: 1000},
		{MatchID: "m1", KillerXUID: tsMoi, VictimXUID: tsAdv, KillerGamertag: "Moi", VictimGamertag: "Cible",
			KillerX: 2.2, KillerY: 2.2, KillerZ: m(1), VictimX: 9, VictimY: 9, VictimZ: m(5), TimeMs: 2000},
	}
	return repo
}

// TestCellule_FacesEtAutreJoueur : une mort nomme son tueur, un frag sa victime.
func TestCellule_FacesEtAutreJoueur(t *testing.T) {
	repo := repoDeuxFaces()
	got := celluleLue(t, NewTacticalService(repo, capsCompletes(), tsMoi),
		celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	want := []struct{ face, autre string }{{domain.TacticalFaceMort, "Rival"}, {domain.TacticalFaceFrag, "Cible"}}
	if len(got.Contributions) != len(want) {
		t.Fatalf("contributions = %+v, want une mort puis un frag", got.Contributions)
	}
	for i, w := range want {
		if c := got.Contributions[i]; c.Face != w.face || c.AutreGamertag != w.autre {
			t.Errorf("contribution %d = face %q / autre %q, want %q / %q", i, c.Face, c.AutreGamertag, w.face, w.autre)
		}
	}
}

// TestCellule_IsoleNommeLeTueur : la lecture « isole » nomme aussi le tueur de chaque mort.
func TestCellule_IsoleNommeLeTueur(t *testing.T) {
	univ := universVariantes(map[string]string{"m1": "Slayer:Arena"})
	mort := mortContexteAvecInstant("m1", tsMoi, 4.0, 4.0, m(19.0), 1, 0, 7000)
	mort.KillerGamertag = "Rival"
	repo := &mockTacticalRepo{univ: univ, morts: domain.TacticalMortsContexte{Univers: univ, Morts: []domain.MortContexte{mort}}}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi).WithRadarRange(map[string]int{"Slayer:Arena": 18})
	got := celluleLue(t, svc, celluleDemande(repo, domain.TacticalQuestionIsole, domain.TacticalQuiMoi, 8, 8))
	if len(got.Contributions) != 1 || got.Contributions[0].AutreGamertag != "Rival" {
		t.Fatalf("contributions = %+v, want une mort tuée par « Rival »", got.Contributions)
	}
}

// zonesEmpilees : deux zones de même contour [0, 5]², l'une au sol (0 à 2 m), l'autre à l'étage
// (3 à 6 m) — le centre de la cellule (4, 4) au pas de 0,5 m, (2,25 ; 2,25), est dans les deux.
func zonesEmpilees() *mockCallouts {
	return &mockCallouts{zones: []domain.ZoneNommee{
		{NomFR: "Rez", NomEN: "Ground", X: 2.5, Y: 2.5, Polygone: carre(0, 5), ZBas: 0, ZHaut: 2, VolumeIndex: 1},
		{NomFR: "Étage", NomEN: "Upper", X: 2.5, Y: 2.5, Polygone: carre(0, 5), ZBas: 3, ZHaut: 6, VolumeIndex: 2},
	}}
}

// TestCellule_ZoneNommeeParLaHauteurDeLaFace : la hauteur d'une MORT est celle de la victime, celle
// d'un FRAG celle du tueur — la même cellule se nomme à l'étage pour ma mort, au rez pour mon frag.
func TestCellule_ZoneNommeeParLaHauteurDeLaFace(t *testing.T) {
	repo := repoDeuxFaces()
	svc := NewTacticalService(repo, capsCompletes(), tsMoi).WithCalloutsStore(zonesEmpilees())
	cas := []struct {
		question, fr, en string
	}{
		{domain.TacticalQuestionMorts, "Étage", "Upper"},
		{domain.TacticalQuestionKills, "Rez", "Ground"},
	}
	for _, c := range cas {
		got := celluleLue(t, svc, celluleDemande(repo, c.question, domain.TacticalQuiMoi, 4, 4))
		if got.Zone == nil || got.Zone.NomFR != c.fr || got.Zone.NomEN != c.en {
			t.Errorf("%s : zone = %+v, want %s / %s", c.question, got.Zone, c.fr, c.en)
		}
	}
}

// TestCellule_ZoneSansNom : aucune zone ne contient le centre ni n'en passe à moins de 2 m — la
// réponse ne porte aucun nom (le web écrit « Zone sans nom »), jamais un nom de repli.
func TestCellule_ZoneSansNom(t *testing.T) {
	repo := repoDeuxFaces()
	loin := &mockCallouts{zones: []domain.ZoneNommee{
		{NomFR: "Loin", NomEN: "Far", X: 40, Y: 40, Polygone: carre(38, 42), ZBas: 0, ZHaut: 6, VolumeIndex: 1},
	}}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi).WithCalloutsStore(loin)
	if got := celluleLue(t, svc, celluleDemande(repo, domain.TacticalQuestionMorts, domain.TacticalQuiMoi, 4, 4)); got.Zone != nil {
		t.Errorf("zone = %+v, want aucune", got.Zone)
	}
}
