package himap

import (
	"math"
	"testing"
)

// renduTest rend une grille de `n` x `n` cellules d'un metre, dont toutes les cellules portent
// de la matiere : le masque est alors le seul discriminant.
func renduTest(n int) *Rendu {
	r := NewRendu([2]float64{0, 0}, [2]float64{float64(n - 1), float64(n - 1)}, 1)
	for k := range r.z {
		r.z[k] = 0
	}
	return r
}

func carre(x0, y0, cote float64) [][2]float64 {
	return [][2]float64{{x0, y0}, {x0 + cote, y0}, {x0 + cote, y0 + cote}, {x0, y0 + cote}}
}

func compte(m []bool) int {
	n := 0
	for _, b := range m {
		if b {
			n++
		}
	}
	return n
}

func TestMasqueZonesRemplitLePolygone(t *testing.T) {
	r := renduTest(40)
	m := MasqueZones(r, [][][2]float64{carre(10, 10, 10)}, 0)
	if n := compte(m); n < 90 || n > 130 {
		t.Fatalf("carre de 10x10 m sur une grille de 1 m : %d cellules, attendu ~100", n)
	}
	// Un point au centre est dedans, un point loin est dehors.
	if !m[15*r.NX+15] {
		t.Error("le centre du carre n'est pas dans le masque")
	}
	if m[35*r.NX+35] {
		t.Error("un point hors du carre est dans le masque")
	}
}

// LA DILATATION EST LA RAISON D'ETRE DE LA MARGE : sans elle, le masque coupe au ras du
// polygone et supprime le mur qui borde la zone.
func TestMasqueZonesDilateDeLaMarge(t *testing.T) {
	r := renduTest(60)
	sans := compte(MasqueZones(r, [][][2]float64{carre(20, 20, 10)}, 0))
	avec := compte(MasqueZones(r, [][][2]float64{carre(20, 20, 10)}, 4))
	if avec <= sans {
		t.Fatalf("la marge n'elargit rien : %d avec, %d sans", avec, sans)
	}
	// 10x10 dilate de 4 m de chaque cote -> ~18x18.
	if avec < 280 || avec > 380 {
		t.Fatalf("dilatation de 4 m sur un carre de 10 m : %d cellules, attendu ~324", avec)
	}
}

// DEUX POLYGONES DISJOINTS : l'union, pas l'intersection. Le piege inverse a deja coute une
// journee sur la coquille de mort (intersection de demi-espaces sur un maillage concave).
func TestMasqueZonesUnitLesPolygones(t *testing.T) {
	r := renduTest(60)
	m := MasqueZones(r, [][][2]float64{carre(5, 5, 8), carre(40, 40, 8)}, 0)
	if !m[8*r.NX+8] || !m[43*r.NX+43] {
		t.Fatal("les deux polygones disjoints doivent etre dans le masque")
	}
	if m[25*r.NX+25] {
		t.Error("l'espace entre les deux polygones ne doit pas etre dans le masque")
	}
}

func TestMesureEtEffaceHorsZones(t *testing.T) {
	r := renduTest(40)
	m := MasqueZones(r, [][][2]float64{carre(10, 10, 10)}, 0)
	matiere, dehors := r.MesureHorsZones(m)
	if matiere != 40*40 {
		t.Fatalf("matiere = %d, attendu %d", matiere, 40*40)
	}
	if dehors != matiere-compte(m) {
		t.Fatalf("dehors = %d, attendu %d", dehors, matiere-compte(m))
	}
	efface := r.EffaceHorsZones(m)
	if efface != dehors {
		t.Fatalf("efface = %d, attendu %d (la mesure doit predire l'effacement)", efface, dehors)
	}
	// Ce qui reste est exactement le masque.
	reste := 0
	for k := range r.z {
		if !math.IsInf(r.z[k], -1) {
			reste++
		}
	}
	if reste != compte(m) {
		t.Fatalf("reste = %d cellules, attendu %d", reste, compte(m))
	}
}
