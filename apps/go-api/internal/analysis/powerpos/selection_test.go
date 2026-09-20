package powerpos

import (
	"testing"

	"levelup/go-api/internal/analysis/tactical"
)

// scoree fabrique une cellule scoree a l'adresse donnee.
func scoree(col, lig int, score float64) CelluleScoree {
	return CelluleScoree{
		Cellule: Cellule{Col: col, Lig: lig,
			CentreX: (float64(col) + 0.5) * 0.5, CentreY: (float64(lig) + 0.5) * 0.5,
			KillsDepuis: 10, MortsDedans: 2, MatchsKills: 5},
		Score: score,
	}
}

// bloc fabrique un pave de cellules au meme score.
func bloc(col0, lig0, largeur, hauteur int, score float64) []CelluleScoree {
	var out []CelluleScoree
	for c := col0; c < col0+largeur; c++ {
		for l := lig0; l < lig0+hauteur; l++ {
			out = append(out, scoree(c, l, score))
		}
	}
	return out
}

func reglageTest() Reglage {
	r := ReglageV1()
	r.SeuilScoreMin = 0.5
	r.QuantileSeuil = 0.5
	r.TailleMiniComposante = 6
	r.MaxComposantes = 2
	return r
}

// TestSelectionneEcarteLesPetitesComposantes : une cellule chanceuse isolee n'est pas un
// lieu (cf. selection.go, filtre 2).
func TestSelectionneEcarteLesPetitesComposantes(t *testing.T) {
	g := tactical.GrilleParDefaut()
	scorees := append(bloc(0, 0, 3, 3, 0.9), scoree(50, 50, 0.95))
	got := Selectionne(g, scorees, reglageTest())
	if len(got) != 1 {
		t.Fatalf("positions = %d, attendu 1 (le pave de 9, pas la cellule isolee)", len(got))
	}
	if len(got[0].Cellules) != 9 {
		t.Errorf("cellules de la position = %d, attendu 9", len(got[0].Cellules))
	}
}

// TestSelectionneRespecteLePlancherAbsolu : sur une carte ou rien ne ressort, on ne publie
// RIEN — c'est la regle « pas de calque sans preuve ».
func TestSelectionneRespecteLePlancherAbsolu(t *testing.T) {
	g := tactical.GrilleParDefaut()
	r := reglageTest()
	scorees := bloc(0, 0, 5, 5, 0.30) // tout sous le plancher absolu de 0,5
	if got := Selectionne(g, scorees, r); len(got) != 0 {
		t.Fatalf("positions = %d, attendu 0 : le quantile ne doit pas sauver une carte plate", len(got))
	}
}

// TestSelectionnePlafonneLeNombreDePositions : colorier la carte n'est pas la lire.
func TestSelectionnePlafonneLeNombreDePositions(t *testing.T) {
	g := tactical.GrilleParDefaut()
	var scorees []CelluleScoree
	scorees = append(scorees, bloc(0, 0, 3, 3, 0.95)...)
	scorees = append(scorees, bloc(20, 0, 3, 3, 0.90)...)
	scorees = append(scorees, bloc(40, 0, 3, 3, 0.85)...)
	got := Selectionne(g, scorees, reglageTest())
	if len(got) != 2 {
		t.Fatalf("positions = %d, attendu 2 (MaxComposantes)", len(got))
	}
	if got[0].ScoreMoyen < got[1].ScoreMoyen {
		t.Errorf("positions non triees : %.3f puis %.3f", got[0].ScoreMoyen, got[1].ScoreMoyen)
	}
	if got[0].ScoreMoyen < 0.94 {
		t.Errorf("la meilleure position a un score moyen de %.3f, attendu ~0,95", got[0].ScoreMoyen)
	}
}

// TestSelectionneRemplitLaPosition : polygone, aire, centre et comptes sont renseignes.
func TestSelectionneRemplitLaPosition(t *testing.T) {
	g := tactical.GrilleParDefaut()
	got := Selectionne(g, bloc(0, 0, 4, 4, 0.9), reglageTest())
	if len(got) != 1 {
		t.Fatalf("positions = %d, attendu 1", len(got))
	}
	p := got[0]
	if len(p.Polygone) < 3 {
		t.Fatalf("polygone a %d sommets", len(p.Polygone))
	}
	// 4x4 cellules de 0,5 m = 2 m x 2 m, dilate d'un demi-pas = 2,5 m x 2,5 m = 6,25 m2.
	if p.AireM2 < 6.2 || p.AireM2 > 6.3 {
		t.Errorf("aire = %.3f m2, attendu ~6,25", p.AireM2)
	}
	if p.Kills != 16*10 || p.Morts != 16*2 {
		t.Errorf("comptes = %d kills / %d morts, attendu 160/32", p.Kills, p.Morts)
	}
	if p.Matchs != 5 {
		t.Errorf("matchs = %d, attendu 5 (le maximum des cellules)", p.Matchs)
	}
	if p.ScoreMax != 0.9 {
		t.Errorf("score max = %.3f, attendu 0,9", p.ScoreMax)
	}
	// Le barycentre du pave 4x4 ancre en (0, 0) tombe au centre de [0, 2] x [0, 2].
	if p.CentreX < 0.9 || p.CentreX > 1.1 || p.CentreY < 0.9 || p.CentreY > 1.1 {
		t.Errorf("centre = (%.2f, %.2f), attendu ~(1, 1)", p.CentreX, p.CentreY)
	}
}

// TestSelectionneVide : aucune cellule scoree, aucune position, aucune panique.
func TestSelectionneVide(t *testing.T) {
	if got := Selectionne(tactical.GrilleParDefaut(), nil, ReglageV1()); len(got) != 0 {
		t.Errorf("positions = %d, attendu 0", len(got))
	}
}
