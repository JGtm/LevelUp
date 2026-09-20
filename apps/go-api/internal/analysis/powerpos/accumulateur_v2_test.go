package powerpos

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/tactical"
)

// TestAccumulateurDirections : la direction sortante part de la cellule du tueur vers la
// victime ; la direction entrante part de la cellule de la victime vers le tueur.
func TestAccumulateurDirections(t *testing.T) {
	g := tactical.GrilleParDefaut()
	a := NouvelAccumulateur(g)
	// Tueur et victime aux CENTRES de deux cellules de la meme ligne (la direction part du
	// centre de la cellule, pas de la position exacte) : sortante vers +x, entrante vers -x.
	a.AjouteKill(kill("m1", 1.25, 1.25, 0, 9.25, 1.25, 0))
	tueur := celluleEn(t, a.Cellules(), g, 1.25, 1.25)
	victime := celluleEn(t, a.Cellules(), g, 9.25, 1.25)
	if tueur.DirSortante.N != 1 || tueur.DirSortante.Cos < 0.99 || math.Abs(tueur.DirSortante.Sin) > 0.01 {
		t.Errorf("direction sortante du tueur = %+v, attendu (1, 0)", tueur.DirSortante)
	}
	if victime.DirEntrante.N != 1 || victime.DirEntrante.Cos > -0.99 || math.Abs(victime.DirEntrante.Sin) > 0.01 {
		t.Errorf("direction entrante de la victime = %+v, attendu (-1, 0)", victime.DirEntrante)
	}
	if tueur.DirEntrante.N != 0 || victime.DirSortante.N != 0 {
		t.Error("les directions sont comptees du mauvais cote")
	}
}

// TestAccumulateurRangs : comptes ponderes, couverture et variante « fort ».
func TestAccumulateurRangs(t *testing.T) {
	g := tactical.GrilleParDefaut()
	a := NouvelAccumulateurPondere(g, ponderationTest())
	k := kill("m1", 1, 1, 0, 9, 1, 0)
	k.RangTueur = rang(1500) // borne haute : poids 1,5, fort
	a.AjouteKill(k)
	k2 := kill("m2", 1, 1, 0, 9, 1, 0)
	k2.RangTueur = rang(1000) // borne basse : poids 0,5, pas fort
	a.AjouteKill(k2)
	a.AjouteKill(kill("m3", 1, 1, 0, 9, 1, 0)) // rang inconnu : poids 1, ni fort ni connu

	tueur := celluleEn(t, a.Cellules(), g, 1, 1)
	if tueur.KillsDepuis != 3 {
		t.Fatalf("kills bruts = %d, attendu 3", tueur.KillsDepuis)
	}
	if math.Abs(tueur.KillsPonderes-3.0) > 1e-9 {
		t.Errorf("kills ponderes = %.3f, attendu 3,0 (1,5 + 0,5 + 1,0)", tueur.KillsPonderes)
	}
	if tueur.KillsRangConnu != 2 {
		t.Errorf("kills a rang connu = %d, attendu 2", tueur.KillsRangConnu)
	}
	if tueur.KillsTueurFort != 1 {
		t.Errorf("kills de tueur fort = %d, attendu 1", tueur.KillsTueurFort)
	}
	victime := celluleEn(t, a.Cellules(), g, 9, 1)
	if math.Abs(victime.MortsPonderees-3.0) > 1e-9 || victime.MortsRangConnu != 2 || victime.MortsTueurFort != 1 {
		t.Errorf("cote victime : ponderees %.3f, connu %d, fort %d — attendu 3,0 / 2 / 1",
			victime.MortsPonderees, victime.MortsRangConnu, victime.MortsTueurFort)
	}
}

// TestAccumulateurNeutreEgaleBrut : sans ponderation, les comptes ponderes valent les bruts.
func TestAccumulateurNeutreEgaleBrut(t *testing.T) {
	g := tactical.GrilleParDefaut()
	a := NouvelAccumulateur(g)
	for i := 0; i < 5; i++ {
		k := kill("m", 1, 1, 0, 9, 1, 0)
		k.RangTueur = rang(float64(1000 + 100*i))
		a.AjouteKill(k)
	}
	c := celluleEn(t, a.Cellules(), g, 1, 1)
	if c.KillsPonderes != float64(c.KillsDepuis) {
		t.Errorf("ponderes %.3f != bruts %d sous la ponderation neutre", c.KillsPonderes, c.KillsDepuis)
	}
	if c.KillsRangConnu != 5 {
		t.Errorf("la couverture du rang doit se compter meme sans ponderation : %d", c.KillsRangConnu)
	}
}

// TestScoreAxesAngulaires : couverture et abri repondent dans le bon sens et retombent a
// 0,5 sous le minimum de directions.
func TestScoreAxesAngulaires(t *testing.T) {
	r := ReglageV1()
	r.MinDirections = 4
	ref := etalon{killsP95: 100, porteeP50: 4, porteeP90: 10}
	eventail := sommeDe(0, math.Pi/2, math.Pi, 3*math.Pi/2, 0.3, 2.0)
	faisceau := sommeDe(0.1, 0.1, 0.1, 0.1, 0.1, 0.1)

	ouvert := note(disque{kills: 50, morts: 50, dirSortante: eventail, dirEntrante: eventail}, ref, r)
	ferme := note(disque{kills: 50, morts: 50, dirSortante: faisceau, dirEntrante: faisceau}, ref, r)
	if ouvert.Couverture <= ferme.Couverture {
		t.Errorf("couverture : eventail %.3f <= faisceau %.3f", ouvert.Couverture, ferme.Couverture)
	}
	if ouvert.Abri >= ferme.Abri {
		t.Errorf("abri : eventail %.3f >= faisceau %.3f", ouvert.Abri, ferme.Abri)
	}
	peu := note(disque{kills: 50, morts: 50, dirSortante: sommeDe(0, 1, 2), dirEntrante: sommeDe(0, 1, 2)}, ref, r)
	if peu.Couverture != 0.5 || peu.Abri != 0.5 {
		t.Errorf("sous MinDirections : couverture %.3f / abri %.3f, attendu 0,5 / 0,5", peu.Couverture, peu.Abri)
	}
}

// TestScoreAvantagePondere : a comptes bruts egaux, un disque dont les kills viennent de
// tueurs forts a plus d'avantage qu'un disque dont les morts viennent de tueurs forts.
func TestScoreAvantagePondere(t *testing.T) {
	r := ReglageV1()
	r.UtiliseRangPondere = true
	ref := etalon{killsP95: 100, porteeP50: 4, porteeP90: 10}
	forts := note(disque{kills: 50, morts: 50, killsPond: 70, mortsPond: 30}, ref, r)
	faibles := note(disque{kills: 50, morts: 50, killsPond: 30, mortsPond: 70}, ref, r)
	if forts.Avantage <= faibles.Avantage {
		t.Errorf("avantage pondere : forts %.3f <= faibles %.3f", forts.Avantage, faibles.Avantage)
	}
	r.UtiliseRangPondere = false
	brut := note(disque{kills: 50, morts: 50, killsPond: 70, mortsPond: 30}, ref, r)
	if math.Abs(brut.Avantage-0.5) > 1e-9 {
		t.Errorf("sans ponderation, l'avantage d'un disque 50/50 = %.3f, attendu 0,5", brut.Avantage)
	}
}

// TestReglageV2PoidsSommentAUn : le score v2 reste une note dans [0, 1].
func TestReglageV2PoidsSommentAUn(t *testing.T) {
	r := ReglageV2()
	somme := r.PoidsAvantage + r.PoidsIntensite + r.PoidsHauteur + r.PoidsPortee +
		r.PoidsCouverture + r.PoidsAbri
	if math.Abs(somme-1) > 1e-9 {
		t.Errorf("somme des poids v2 = %.3f, attendu 1", somme)
	}
	if r.QuantileAmorce <= r.QuantileCroissance {
		t.Errorf("amorce %.2f <= croissance %.2f", r.QuantileAmorce, r.QuantileCroissance)
	}
	v1 := ReglageV1()
	if v1.PoidsCouverture != 0 || v1.PoidsAbri != 0 || v1.QuantileAmorce != 0 || v1.FermetureRayonCellules != 0 || v1.Connexite8 || v1.UtiliseRangPondere {
		t.Error("ReglageV1 a ete touche : il doit rester la preuve datee du 2026-09-20")
	}
}
