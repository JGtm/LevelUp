package powerpos

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/tactical"
)

// reglageHysteresisTest : amorce au p90, croissance au p50, fermeture de rayon 1, 8-connexite.
func reglageHysteresisTest() Reglage {
	r := ReglageV1()
	r.SeuilScoreMin = 0.5
	r.QuantileAmorce = 0.90
	r.QuantileCroissance = 0.50
	r.FermetureRayonCellules = 1
	r.Connexite8 = true
	r.TailleMiniComposante = 4
	r.MaxComposantes = 8
	return r
}

// TestHysteresisUneCelluleSousLAmorceEntreSiElleToucheUnGerme : c'est la distinction que le
// seuil unique de la v1 ne savait pas faire (Orange Pipes a 0,678 pour un seuil a 0,680).
func TestHysteresisUneCelluleSousLAmorceEntreSiElleToucheUnGerme(t *testing.T) {
	// Fond de 20 cellules a 0,3 (sous la croissance), un amas de 9 a 0,7 en (10, 10) dont
	// le centre est un germe a 0,95, et un amas isole de 9 a 0,7 en (40, 40) sans germe.
	// Scores tries : 20 x 0,3 | 17 x 0,7 | 1 x 0,95 — le quantile 0,6 tombe sur 0,7
	// (croissance), et le plancher absolu de 0,8 porte l'amorce au-dessus des 0,7.
	var scorees []CelluleScoree
	scorees = append(scorees, bloc(0, 0, 4, 5, 0.3)...)
	scorees = append(scorees, bloc(10, 10, 3, 3, 0.7)...)
	scorees = append(scorees, bloc(40, 40, 3, 3, 0.7)...)
	for i := range scorees {
		if scorees[i].Col == 11 && scorees[i].Lig == 11 {
			scorees[i].Score = 0.95
		}
	}
	r := reglageHysteresisTest()
	r.SeuilScoreMin = 0.8
	r.QuantileCroissance = 0.6
	got := Selectionne(tactical.GrilleParDefaut(), scorees, r)
	if len(got) != 1 {
		t.Fatalf("positions = %d, attendu 1 (l'amas avec germe seulement)", len(got))
	}
	if got[0].CentreX < 5.5 || got[0].CentreX > 6.5 {
		t.Errorf("centre x = %.2f, attendu ~5,75 (l'amas en (10..12, 10..12))", got[0].CentreX)
	}
	if got[0].NbCellulesMesurees != 9 {
		t.Errorf("cellules mesurees = %d, attendu 9", got[0].NbCellulesMesurees)
	}
}

// TestFermetureNEntrePasDansLesMoyennes : une cellule ajoutee par la fermeture, meme
// scorable avec un score sous le seuil, ne pese ni dans le score moyen ni dans les kills.
func TestFermetureNEntrePasDansLesMoyennes(t *testing.T) {
	r := reglageHysteresisTest()
	r.QuantileAmorce = 0.60
	r.QuantileCroissance = 0.60
	// Un pave 4x4 a 0,9 avec un trou d'une cellule a 0,1 au milieu (scorable, sous le
	// seuil), et un fond de 20 cellules a 0,2 pour poser les quantiles.
	var scorees []CelluleScoree
	for c := 0; c < 4; c++ {
		for l := 0; l < 4; l++ {
			s := 0.9
			if c == 1 && l == 1 {
				s = 0.1
			}
			scorees = append(scorees, scoree(c, l, s))
		}
	}
	scorees = append(scorees, bloc(50, 50, 4, 5, 0.2)...)
	got := Selectionne(tactical.GrilleParDefaut(), scorees, r)
	if len(got) != 1 {
		t.Fatalf("positions = %d, attendu 1", len(got))
	}
	p := got[0]
	if len(p.Cellules) != 16 {
		t.Errorf("cellules de la position = %d, attendu 16 (le trou est referme)", len(p.Cellules))
	}
	if p.NbCellulesMesurees != 15 {
		t.Errorf("cellules mesurees = %d, attendu 15 (le trou ne mesure pas)", p.NbCellulesMesurees)
	}
	if math.Abs(p.ScoreMoyen-0.9) > 1e-9 {
		t.Errorf("score moyen = %.4f, attendu 0,9 (le trou a 0,1 ne doit pas peser)", p.ScoreMoyen)
	}
	if p.Kills != 15*10 {
		t.Errorf("kills = %d, attendu 150 (15 cellules mesurees x 10)", p.Kills)
	}
}

// TestSelectionV1Inchangee : avec un reglage sans hysteresis ni fermeture, la selection est
// celle de la v1, cellule pour cellule (D12 : le verdict v1 doit rester rejouable).
func TestSelectionV1Inchangee(t *testing.T) {
	g := tactical.GrilleParDefaut()
	r := reglageTest()
	scorees := append(bloc(0, 0, 3, 3, 0.9), scoree(50, 50, 0.95))
	// Un semis en damier au-dessus du seuil : en v1, chaque cellule est seule (4-connexite
	// sans fermeture) et rien ne passe la taille minimale.
	for c := 20; c < 26; c++ {
		for l := 0; l < 6; l++ {
			if (c+l)%2 == 0 {
				scorees = append(scorees, scoree(c, l, 0.9))
			}
		}
	}
	got := Selectionne(g, scorees, r)
	if len(got) != 1 {
		t.Fatalf("positions v1 = %d, attendu 1 (le damier reste eclate)", len(got))
	}
	if got[0].NbCellulesMesurees != 9 || len(got[0].Cellules) != 9 {
		t.Errorf("v1 : cellules = %d / mesurees = %d, attendu 9 / 9", len(got[0].Cellules), got[0].NbCellulesMesurees)
	}
	d := Diagnostique(scorees, r)
	if d.SeuilAmorce != d.SeuilCroissance {
		t.Errorf("v1 : deux seuils differents (%.3f / %.3f)", d.SeuilAmorce, d.SeuilCroissance)
	}
	if d.CellulesRetenues != d.CellulesApresFermeture {
		t.Errorf("v1 : la fermeture a ajoute des cellules (%d -> %d)", d.CellulesRetenues, d.CellulesApresFermeture)
	}
}

// TestDiagnostiqueCompteCommeSelectionne : le diagnostic et la selection voient la meme
// chose.
func TestDiagnostiqueCompteCommeSelectionne(t *testing.T) {
	r := reglageHysteresisTest()
	var scorees []CelluleScoree
	scorees = append(scorees, bloc(0, 0, 10, 10, 0.3)...)
	// Damier a 0,9 : 8 cellules retenues, eclatees avant fermeture, soudees apres.
	for c := 20; c < 24; c++ {
		for l := 0; l < 4; l++ {
			if (c+l)%2 == 0 {
				scorees = append(scorees, scoree(c, l, 0.9))
			}
		}
	}
	d := Diagnostique(scorees, r)
	if d.CellulesAmorce != 8 || d.CellulesRetenues != 8 {
		t.Errorf("amorce / retenues = %d / %d, attendu 8 / 8", d.CellulesAmorce, d.CellulesRetenues)
	}
	if len(d.TaillesAvantFermeture) != 8 {
		t.Errorf("composantes avant fermeture = %d, attendu 8 (damier)", len(d.TaillesAvantFermeture))
	}
	if d.Composantes != 1 || d.ComposantesRetenues != 1 {
		t.Errorf("composantes apres fermeture = %d (retenues %d), attendu 1 / 1", d.Composantes, d.ComposantesRetenues)
	}
	if d.CellulesApresFermeture <= 8 {
		t.Errorf("cellules apres fermeture = %d, attendu > 8", d.CellulesApresFermeture)
	}
	got := Selectionne(tactical.GrilleParDefaut(), scorees, r)
	if len(got) != d.ComposantesRetenues {
		t.Errorf("selection = %d positions, diagnostic = %d", len(got), d.ComposantesRetenues)
	}
	if len(got) == 1 && len(got[0].Cellules) != d.CellulesApresFermeture {
		t.Errorf("cellules de la position = %d, diagnostic = %d", len(got[0].Cellules), d.CellulesApresFermeture)
	}
}

// TestDiagnostiqueVide : aucune cellule, aucun compte, aucune panique.
func TestDiagnostiqueVide(t *testing.T) {
	if d := Diagnostique(nil, ReglageV1()); d.CellulesRetenues != 0 || d.Composantes != 0 {
		t.Errorf("diagnostic du vide = %+v", d)
	}
}
