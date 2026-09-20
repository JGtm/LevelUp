package powerpos

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/tactical"
)

// accumuleBande fabrique une bande de cellules alimentees le long de la ligne `lig`, de
// `col0` a `col1` inclus, avec `kills` kills et `morts` morts par cellule, chacun venant de
// matchs distincts pour passer le plancher.
func accumuleBande(a *Accumulateur, lig, col0, col1, kills, morts int, denivele, portee float64) {
	pas := 0.5
	for col := col0; col <= col1; col++ {
		x := (float64(col) + 0.25) * pas
		y := (float64(lig) + 0.25) * pas
		for k := 0; k < kills; k++ {
			a.AjouteKill(KillSample{
				MatchID: "m" + string(rune('a'+k%26)) + string(rune('a'+k/26)),
				KillerX: x, KillerY: y, KillerZ: denivele,
				VictimX: x + portee, VictimY: y, VictimZ: 0,
			})
		}
		for k := 0; k < morts; k++ {
			// La victime est ici, le tueur est tres loin (hors de tout disque mesure).
			a.AjouteKill(KillSample{
				MatchID: "mort" + string(rune('a'+k%26)),
				KillerX: 900, KillerY: 900, KillerZ: 0,
				VictimX: x, VictimY: y, VictimZ: 0,
			})
		}
	}
}

// TestScorePlanchersEcartentLesCellulesPauvres : sans assez de matchs ni assez
// d'engagements dans le disque, une cellule n'est pas scorable DU TOUT.
func TestScorePlanchersEcartentLesCellulesPauvres(t *testing.T) {
	g := tactical.GrilleParDefaut()
	a := NouvelAccumulateur(g)
	// Une seule cellule, deux matchs : sous le plancher de trois.
	a.AjouteKill(kill("m1", 1, 1, 0, 5, 1, 0))
	a.AjouteKill(kill("m2", 1, 1, 0, 5, 1, 0))
	if got := Score(g, a.Cellules(), ReglageV1()); len(got) != 0 {
		t.Fatalf("cellules scorables = %d, attendu 0 (plancher de matchs)", len(got))
	}

	// Trois matchs mais trois engagements : sous le plancher du disque.
	a.AjouteKill(kill("m3", 1, 1, 0, 5, 1, 0))
	if got := Score(g, a.Cellules(), ReglageV1()); len(got) != 0 {
		t.Fatalf("cellules scorables = %d, attendu 0 (plancher d'engagements du disque)", len(got))
	}
}

// TestScoreAvantageSepareLesDeuxSens : a volume egal, une bande d'ou l'on tue score plus
// haut qu'une bande ou l'on meurt.
func TestScoreAvantageSepareLesDeuxSens(t *testing.T) {
	g := tactical.GrilleParDefaut()
	a := NouvelAccumulateur(g)
	accumuleBande(a, 0, 0, 20, 5, 1, 0, 6)   // on y tue
	accumuleBande(a, 100, 0, 20, 1, 5, 0, 6) // on y meurt
	scorees := Score(g, a.Cellules(), ReglageV1())
	if len(scorees) == 0 {
		t.Fatal("aucune cellule scorable")
	}
	forte := scoreDe(t, scorees, 5, 0)
	faible := scoreDe(t, scorees, 5, 100)
	if forte.Avantage <= 0.5 {
		t.Errorf("avantage de la bande gagnante = %.3f, attendu > 0,5", forte.Avantage)
	}
	if faible.Avantage >= 0.5 {
		t.Errorf("avantage de la bande perdante = %.3f, attendu < 0,5", faible.Avantage)
	}
	if forte.Score <= faible.Score {
		t.Errorf("score gagnant %.3f <= score perdant %.3f", forte.Score, faible.Score)
	}
}

// TestScoreRetrecitVersLaNeutralite : le retrecissement est ce qui empeche un petit
// echantillon de rendre un avantage extreme.
func TestScoreRetrecitVersLaNeutralite(t *testing.T) {
	r := ReglageV1()
	ref := etalon{killsP95: 100, porteeP50: 4, porteeP90: 10}
	// MEME rapport brut (0,75) des deux cotes, dix fois moins d'engagements d'un cote.
	petit := note(disque{kills: 30, morts: 10}, ref, r)
	grand := note(disque{kills: 300, morts: 100}, ref, r)
	if !(petit.Avantage < grand.Avantage) {
		t.Errorf("avantage retreci : petit %.3f, grand %.3f — a rapport egal, le gros"+
			" echantillon doit etre plus extreme", petit.Avantage, grand.Avantage)
	}
	if petit.Avantage >= 1 {
		t.Errorf("avantage d'un disque a 40 engagements = %.3f, attendu strictement < 1",
			petit.Avantage)
	}
}

// TestScoreHauteurEtPortee : les deux axes secondaires repondent dans le bon sens.
func TestScoreHauteurEtPortee(t *testing.T) {
	r := ReglageV1()
	ref := etalon{killsP95: 100, porteeP50: 4, porteeP90: 10}
	bas := note(disque{kills: 50, morts: 50, deniveleP: -1.5, porteeP: 4, killsPourMoyenne: 50}, ref, r)
	haut := note(disque{kills: 50, morts: 50, deniveleP: 1.5, porteeP: 4, killsPourMoyenne: 50}, ref, r)
	if haut.Hauteur <= bas.Hauteur {
		t.Errorf("hauteur : surplomb %.3f <= fosse %.3f", haut.Hauteur, bas.Hauteur)
	}
	court := note(disque{kills: 50, morts: 50, deniveleP: 0, porteeP: 4, killsPourMoyenne: 50}, ref, r)
	long := note(disque{kills: 50, morts: 50, deniveleP: 0, porteeP: 10, killsPourMoyenne: 50}, ref, r)
	if long.Portee <= court.Portee {
		t.Errorf("portee : longue %.3f <= courte %.3f", long.Portee, court.Portee)
	}
}

// TestScoreBorne : le score reste dans [0, 1], quelles que soient les entrees.
func TestScoreBorne(t *testing.T) {
	r := ReglageV1()
	ref := etalon{killsP95: 100, porteeP50: 4, porteeP90: 10}
	cas := []disque{
		{kills: 100000, morts: 0, deniveleP: 50, porteeP: 200, killsPourMoyenne: 1},
		{kills: 0, morts: 100000, deniveleP: -50, porteeP: -200, killsPourMoyenne: 1},
		{kills: 40, morts: 40},
	}
	for _, d := range cas {
		got := note(d, ref, r)
		if got.Score < 0 || got.Score > 1 {
			t.Errorf("score = %.3f hors [0, 1] pour %+v", got.Score, d)
		}
	}
}

// TestOffsetsDisque : le disque de 2 m au pas de 0,5 m couvre le bon voisinage, centre
// inclus.
func TestOffsetsDisque(t *testing.T) {
	offsets := offsetsDisque(0.5, 2.0)
	centre := false
	for _, o := range offsets {
		if o.Col == 0 && o.Lig == 0 {
			centre = true
		}
		if math.Hypot(float64(o.Col)*0.5, float64(o.Lig)*0.5) > 2.0+1e-9 {
			t.Fatalf("offset hors du disque : %+v", o)
		}
	}
	if !centre {
		t.Error("le centre n'est pas dans son propre disque")
	}
	// Un disque de rayon 4 cellules contient 49 cellules (compte exact de la grille).
	if len(offsets) != 49 {
		t.Errorf("cellules du disque = %d, attendu 49", len(offsets))
	}
}

// TestScoreTriDecroissantEtStable : le CSV et le catalogue en dependent.
func TestScoreTriDecroissantEtStable(t *testing.T) {
	g := tactical.GrilleParDefaut()
	a := NouvelAccumulateur(g)
	accumuleBande(a, 0, 0, 30, 5, 1, 1.0, 9)
	accumuleBande(a, 60, 0, 30, 2, 4, 0, 3)
	scorees := Score(g, a.Cellules(), ReglageV1())
	for i := 1; i < len(scorees); i++ {
		if scorees[i-1].Score < scorees[i].Score {
			t.Fatalf("tri non decroissant en %d : %.4f puis %.4f", i, scorees[i-1].Score, scorees[i].Score)
		}
	}
}

// scoreDe retrouve la cellule scoree contenant (col, lig) en unites de grille.
func scoreDe(t *testing.T, scorees []CelluleScoree, col, lig int) CelluleScoree {
	t.Helper()
	for _, c := range scorees {
		if c.Col == col && c.Lig == lig {
			return c
		}
	}
	t.Fatalf("cellule (%d, %d) non scorable", col, lig)
	return CelluleScoree{}
}
