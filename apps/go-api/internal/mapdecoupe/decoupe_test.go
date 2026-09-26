package mapdecoupe

import (
	"math"
	"testing"
)

func TestContoursCelluleSeuleRendUnAnneauOrienteExterieur(t *testing.T) {
	b := contours([]bool{false, false, false, true}, 2, 2)
	if len(b) != 1 {
		t.Fatalf("boucles = %d, attendu 1", len(b))
	}
	if len(b[0]) != 4 {
		t.Fatalf("sommets = %d, attendu 4", len(b[0]))
	}
	if a := aireSignee(b[0]); a != 2 {
		t.Errorf("aire signée = %d, attendu +2 (une cellule, frontière extérieure)", a)
	}
}

func TestContoursSepareLeTrouDeLExterieur(t *testing.T) {
	// Anneau 3x3 dont la cellule centrale est vide.
	dedans := make([]bool, 9)
	for k := range dedans {
		dedans[k] = k != 4
	}
	b := contours(dedans, 3, 3)
	if len(b) != 2 {
		t.Fatalf("boucles = %d, attendu 2 (extérieur + trou)", len(b))
	}
	var ext, trou int
	for _, l := range b {
		if aireSignee(l) > 0 {
			ext++
			continue
		}
		trou++
	}
	if ext != 1 || trou != 1 {
		t.Errorf("extérieurs = %d, trous = %d, attendu 1 et 1", ext, trou)
	}
}

func TestContoursPincementDiagonalRendDeuxAnneauxSimples(t *testing.T) {
	// (0,0) et (1,1) pleines, elles ne se touchent que par un coin.
	dedans := []bool{true, false, false, true}
	b := contours(dedans, 2, 2)
	if len(b) != 2 {
		t.Fatalf("boucles = %d, attendu 2 — le pincement doit rendre deux anneaux simples", len(b))
	}
	for _, l := range b {
		if len(l) != 4 || aireSignee(l) != 2 {
			t.Errorf("anneau inattendu : %v (aire %d)", l, aireSignee(l))
		}
	}
}

func TestSimplifieAnneauEcraseLEscalierDroitEtDemarreAuMemeSommet(t *testing.T) {
	// Rectangle 10 x 1 décrit cellule par cellule : 22 sommets, 4 suffisent.
	var pts [][2]float64
	for i := 0; i <= 10; i++ {
		pts = append(pts, [2]float64{float64(i), 0})
	}
	for i := 10; i >= 0; i-- {
		pts = append(pts, [2]float64{float64(i), 1})
	}
	out := simplifieAnneau(pts, 0.08)
	if len(out) != 4 {
		t.Fatalf("sommets = %d, attendu 4 : %v", len(out), out)
	}
	// Le point de départ est le plus petit lexicographiquement — indépendant du chaînage.
	tourne := append(append([][2]float64{}, pts[7:]...), pts[:7]...)
	autre := simplifieAnneau(tourne, 0.08)
	if len(autre) != len(out) || autre[0] != out[0] {
		t.Errorf("la simplification dépend du point de départ : %v vs %v", out, autre)
	}
}

func TestSimplifieAnneauGardeUnDecrochementPlusGrandQueLaTolerance(t *testing.T) {
	pts := [][2]float64{{0, 0}, {5, 0}, {5, 3}, {10, 3}, {10, 4}, {0, 4}}
	out := simplifieAnneau(pts, 0.08)
	if len(out) != 6 {
		t.Errorf("sommets = %d, attendu 6 : un décrochement de 3 m ne se lisse pas (%v)", len(out), out)
	}
}

func TestDecoupeRetireCeQuiTombeSurLeVide(t *testing.T) {
	// Masque 10x10 (1 m/cellule) : matière sur la moitié gauche seulement.
	dur := make([]bool, 100)
	for j := 0; j < 10; j++ {
		for i := 0; i < 5; i++ {
			dur[j*10+i] = true
		}
	}
	m := masqueTest(t, 10, 10, dur)
	// Pavé du designer : tout le cadre.
	brut := [][2]float64{{0, 0}, {10, 0}, {10, 10}, {0, 10}}

	r := Decoupe(brut, m, Options{SimplifieM: 0.08, AireMinM2: 1})
	if r.Degenere {
		t.Fatal("le découpage ne doit pas être dégénéré : la moitié de la zone est praticable")
	}
	if math.Abs(r.AireBrutM2-100) > 1e-9 {
		t.Errorf("aire brute = %v m², attendu 100", r.AireBrutM2)
	}
	if math.Abs(r.AireM2-50) > 1e-9 {
		t.Errorf("aire découpée = %v m², attendu 50", r.AireM2)
	}
	if math.Abs(r.PartGardee()-0.5) > 1e-9 {
		t.Errorf("part gardée = %v, attendu 0,5", r.PartGardee())
	}
	if len(r.Parties) != 0 || len(r.Trous) != 0 {
		t.Errorf("une moitié pleine rend un seul anneau : parties=%d trous=%d", len(r.Parties), len(r.Trous))
	}
	if len(r.Contour) != 4 {
		t.Errorf("contour = %d sommets, attendu 4 (un rectangle) : %v", len(r.Contour), r.Contour)
	}
}

func TestDecoupePubliLeTrouDuMasqueCommeEvidement(t *testing.T) {
	m := masqueTest(t, 10, 10, masqueTroue(4, 5))
	brut := [][2]float64{{0, 0}, {10, 0}, {10, 10}, {0, 10}}

	r := Decoupe(brut, m, Options{SimplifieM: 0.08, AireMinM2: 1})
	if len(r.Trous) != 1 {
		t.Fatalf("trous = %d, attendu 1", len(r.Trous))
	}
	if math.Abs(r.AireM2-96) > 1e-9 {
		t.Errorf("aire découpée = %v m², attendu 96", r.AireM2)
	}
}

// TestDecoupeRendALaZoneLeVideQuElleEnferme fige la règle de l'item 9.2 : un vide ENTOURÉ de
// décor est un trou de reconstruction, pas un débordement — il revient à la zone.
func TestDecoupeRendALaZoneLeVideQuElleEnferme(t *testing.T) {
	m := masqueTest(t, 10, 10, masqueTroue(4, 5))
	brut := [][2]float64{{0, 0}, {10, 0}, {10, 10}, {0, 10}}

	r := Decoupe(brut, m, Options{SimplifieM: 0.08, AireMinM2: 1, RendLesEnclaves: true})
	if len(r.Trous) != 0 {
		t.Errorf("trous = %d, attendu 0 : le vide enfermé revient à la zone", len(r.Trous))
	}
	if math.Abs(r.AireM2-100) > 1e-9 {
		t.Errorf("aire découpée = %v m², attendu 100", r.AireM2)
	}
}

// TestDecoupeRetireLeVideQuiCommuniqueAvecLeDehors : l'autre moitié de la règle — une
// échancrure ouverte sur le bord de la zone est un débordement, elle se coupe.
func TestDecoupeRetireLeVideQuiCommuniqueAvecLeDehors(t *testing.T) {
	dur := make([]bool, 100)
	for k := range dur {
		dur[k] = true
	}
	for j := 0; j <= 5; j++ { // une entaille qui part du bord haut de la grille
		for i := 4; i <= 5; i++ {
			dur[j*10+i] = false
		}
	}
	m := masqueTest(t, 10, 10, dur)
	brut := [][2]float64{{0, 0}, {10, 0}, {10, 10}, {0, 10}}

	r := Decoupe(brut, m, Options{SimplifieM: 0.08, AireMinM2: 1, RendLesEnclaves: true})
	if math.Abs(r.AireM2-88) > 1e-9 {
		t.Errorf("aire découpée = %v m², attendu 88 : l'entaille ouverte se coupe", r.AireM2)
	}
}

// masqueTroue rend une grille 10x10 pleine, évidée sur le carré [i0..i1]².
func masqueTroue(i0, i1 int) []bool {
	dur := make([]bool, 100)
	for k := range dur {
		dur[k] = true
	}
	for j := i0; j <= i1; j++ {
		for i := i0; i <= i1; i++ {
			dur[j*10+i] = false
		}
	}
	return dur
}

func TestDecoupeDeclareDegenereCeQuiNeLaissePresqueRien(t *testing.T) {
	dur := make([]bool, 100) // aucune matière
	m := masqueTest(t, 10, 10, dur)
	brut := [][2]float64{{0, 0}, {10, 0}, {10, 10}, {0, 10}}

	r := Decoupe(brut, m, Options{SimplifieM: 0.08, AireMinM2: AireMinParDefaut})
	if !r.Degenere {
		t.Fatal("une zone entièrement sur le vide doit être déclarée dégénérée")
	}
	if r.AireBrutM2 <= 0 {
		t.Error("l'aire brute reste mesurée, même quand le découpage ne laisse rien")
	}
}

func TestDecoupeRefuseUnPolygoneOuUnMasqueAbsent(t *testing.T) {
	m := masqueTest(t, 2, 2, []bool{true, true, true, true})
	if !Decoupe([][2]float64{{0, 0}, {1, 1}}, m, Options{}).Degenere {
		t.Error("un polygone de moins de trois sommets doit être dégénéré")
	}
	if !Decoupe([][2]float64{{0, 0}, {2, 0}, {2, 2}}, nil, Options{}).Degenere {
		t.Error("un masque absent doit être dégénéré, jamais un découpage deviné")
	}
}

func TestDecoupeHorsCadreNeDevinRien(t *testing.T) {
	m := masqueTest(t, 4, 4, make([]bool, 16))
	// Un pavé entièrement à gauche du cadre.
	if !Decoupe([][2]float64{{-50, 0}, {-40, 0}, {-40, 4}, {-50, 4}}, m, Options{AireMinM2: 1}).Degenere {
		t.Error("un pavé hors cadre doit être dégénéré")
	}
}
